# infra (AWS CDK em Go)

App CDK do `gestor-gastos-infra-base`. Região fixa: `us-east-1`. Por custo, há um único ambiente na AWS, `dev`, implantado a cada push na `main`. Os nomes continuam levando o ambiente (`gestor-gastos-dev-*`, `/gestor-gastos/dev/...` no SSM), então um segundo ambiente no futuro não exige renomear nada. CloudFormation exports nunca são usados ([ADR 0007](../docs/adr/0007-esteira-ci-cd.md)).

## Stacks

| Stack | Escopo | O que cria |
|---|---|---|
| `gestor-gastos-cicd` | conta | OIDC provider do GitHub (`token.actions.githubusercontent.com`, audience `sts.amazonaws.com`) 6 roles de deploy `gh-deploy-<repo>-dev` e 5 roles só leitura `gh-diff-<repo>` |
| `gestor-gastos-budget` | conta | Budget mensal `gestor-gastos-mensal` de US$ 5, com alerta por e-mail a 80% do custo real e a 100% do previsto |
| `gestor-gastos-dev-events` | ambiente | Barramento do EventBridge `gestor-gastos-dev`, DLQ SQS `gestor-gastos-dev-events-dlq` para os targets das regras dos serviços e alarme quando a DLQ tem mensagens |
| `gestor-gastos-dev-auth` | ambiente | User Pool `gestor-gastos-dev-users` (plano Lite), grupos `admin` e `cliente`, app client público `gestor-gastos-dev-frontend` (SRP) e a Lambda do trigger Post Confirmation |
| `gestor-gastos-dev-api` | ambiente | HTTP API `gestor-gastos-dev-api` com authorizer JWT do Cognito, CORS, throttling no stage `$default` e a rota de verificação `GET /infra/auth-check` |

As stacks de ambiente usam o contexto `env` (padrão `dev`, o mesmo `--context env=dev` que o `go-service.yml` passa). Elas só se comunicam pelo SSM: a `api` lê o issuer e o client id que a `auth` publicou, resolvidos pelo CloudFormation no deploy. A ordem de deploy é declarada com `addStackDependency`, que não cria exports: **events → auth → api**.

### Parâmetros no SSM

Todos do tipo `String`, tier Standard (sem custo), em `/gestor-gastos/dev/`. Os outros repositórios leem estes caminhos; não os renomeie sem atualizar quem consome.

| Parâmetro | Stack | Valor | Quem usa |
|---|---|---|---|
| `/gestor-gastos/dev/auth/user-pool-id` | auth | Id do User Pool | users-service (operações de admin) |
| `/gestor-gastos/dev/auth/user-pool-arn` | auth | ARN do User Pool | users-service (IAM restrito ao pool, tarefa 10.5) |
| `/gestor-gastos/dev/auth/user-pool-client-id` | auth | Id do app client público | frontend (`EXPO_PUBLIC_COGNITO_CLIENT_ID`), BFF |
| `/gestor-gastos/dev/auth/issuer-url` | auth | `https://cognito-idp.us-east-1.amazonaws.com/<pool-id>` | stack api (authorizer), BFF |
| `/gestor-gastos/dev/auth/jwks-url` | auth | `<issuer>/.well-known/jwks.json` | BFF (validação do JWT) |
| `/gestor-gastos/dev/events/bus-name` | events | `gestor-gastos-dev` | serviços que publicam e criam regras |
| `/gestor-gastos/dev/events/bus-arn` | events | ARN do barramento | regras e permissões `events:PutEvents` |
| `/gestor-gastos/dev/events/dlq-arn` | events | ARN da DLQ | target das regras dos serviços (`deadLetterQueue`) |
| `/gestor-gastos/dev/events/dlq-url` | events | URL da DLQ | inspeção e redrive |
| `/gestor-gastos/dev/api/id` | api | Id da HTTP API | BFF (adiciona as rotas) |
| `/gestor-gastos/dev/api/url` | api | `https://<api-id>.execute-api.us-east-1.amazonaws.com` | frontend (`EXPO_PUBLIC_API_URL`), smoke tests |
| `/gestor-gastos/dev/api/authorizer-id` | api | Id do authorizer JWT | BFF (protege as rotas) |

Os nomes ficam em `infra/config.go` (`Param*`).

### Stack auth

- **User Pool** igual ao do cognito-local (`local/seed/cognito.go`): e-mail como login (sem diferenciar maiúsculas), cadastro aberto, verificação do e-mail por código, atributos obrigatórios `email` e `name`, senha de no mínimo 8 caracteres com maiúscula, minúscula e número, recuperação só por e-mail. E-mail padrão do Cognito (limite de ~50/dia), sem SES. Plano **Lite**, o mais barato, que já tem SRP, grupos e triggers. `RemovalPolicy: RETAIN`: apagar a stack não apaga o pool nem os usuários.
- **App client** público, sem secret, só `ALLOW_USER_SRP_AUTH` e `ALLOW_REFRESH_TOKEN_AUTH`, sem OAuth/Hosted UI, com `PreventUserExistenceErrors` e revogação de token. Access token e id token de **15 minutos**, refresh token de **30 dias**. Localmente o cognito-local usa `USER_PASSWORD_AUTH`, porque não implementa SRP (design.md).
- **Trigger Post Confirmation:** Lambda Go `gestor-gastos-dev-post-confirmation`, com código em `cmd/post-confirmation` e `internal/postconfirmation`, na raiz do repositório (módulo `github.com/Rossete-Inc/gestor-gastos-infra-base`). Ao confirmar o cadastro, adiciona o usuário ao grupo `cliente` e publica `UserConfirmed` (source `gestor-gastos.auth`) no barramento, com a `go-commons/events`. Ignora a confirmação de "esqueci a senha". Se a publicação falhar, só registra o erro: o `/me` do BFF cria o perfil sob demanda. arm64, `provided.al2023`, 128 MB, timeout de 5 s (o limite do Cognito), logs por 7 dias.
- **Dependência circular evitada:** o pool referencia a Lambda (trigger), e a Lambda depende da própria política. Se a política citasse o ARN do pool, o CloudFormation teria pool → Lambda → política → pool. Por isso a permissão `cognito-idp:AdminAddUserToGroup` é dada em `arn:aws:cognito-idp:us-east-1:<conta>:userpool/*`. O `events:PutEvents` é restrito ao ARN do barramento, montado pelo nome.
- **Binário da Lambda:** segue a convenção do `go-service.yml`: `dist/post-confirmation/bootstrap` na raiz, gerado por `task infra:build`. O app CDK empacota esse binário em `dist/post-confirmation.zip` com permissão de execução (0755) e data fixa. O zip é feito pelo app, e não pelo CLI do CDK, porque no Windows o CLI geraria o zip sem o bit de execução e a Lambda falharia com `permission denied`. Sem o binário, o synth da stack auth falha com uma mensagem dizendo para rodar `task infra:build`; as outras stacks não são afetadas.

### Stack events

- Barramento custom `gestor-gastos-dev`. Sem arquivo nem replay, por custo.
- DLQ `gestor-gastos-dev-events-dlq` (SQS, criptografia gerenciada, só TLS, retenção de 14 dias). As regras ficam no CDK de cada serviço e usam esta fila como `deadLetterQueue` do target. A política da fila já autoriza o EventBridge a enviar mensagens de qualquer regra do barramento (`aws:SourceArn` = `arn:aws:events:us-east-1:<conta>:rule/gestor-gastos-dev/*`), porque as regras ainda não existem quando a fila é criada.
- Alarme `gestor-gastos-dev-events-dlq-com-mensagens` quando há pelo menos uma mensagem visível. Não tem ação (SNS exigiria pôr um e-mail no código público): ele fica vermelho no console do CloudWatch. Está dentro dos 10 alarmes gratuitos.

### Stack api

- HTTP API `gestor-gastos-dev-api`. As rotas de negócio são do BFF (tarefa 6.5), que as adiciona a esta API lendo `api/id` e `api/authorizer-id` do SSM.
- **Authorizer JWT** `cognito-jwt`: issuer = `auth/issuer-url`, audience = `auth/user-pool-client-id`, token no header `Authorization`. Para access tokens do Cognito, que não têm `aud`, a HTTP API compara a audience com a claim `client_id`.
- **CORS:** sempre libera `http://localhost:8081` (`expo start --web`). A origem do site no Amplify é configurável pelo contexto `amplifyOrigin` ou pela variável `AMPLIFY_ORIGIN` (ex.: `https://main.d1abc.amplifyapp.com`, sem barra final). O app do Amplify só existe a partir da tarefa 6.6; quando existir, grave a origem em `infra/cdk.json` (`"context": { "amplifyOrigin": "..." }`), para todo deploy usá-la. Métodos `GET`, `POST`, `PUT`, `PATCH`, `DELETE` e `OPTIONS`; headers `Authorization` e `Content-Type`; cache do preflight de 1 hora.
- **Throttling** no stage `$default`, para todas as rotas: burst 20 e 10 requisições/s. Baixo de propósito, para limitar o custo em caso de abuso.
- **Rota de verificação `GET /infra/auth-check`:** protegida pelo authorizer JWT. Uma HTTP API sem rotas responde 404, não 401, e HTTP API não tem integração mock. Por isso a rota repassa (HTTP proxy) para o JWKS público do próprio pool: sem Lambda e sem custo parada. Sem token, o authorizer responde **401** antes de chegar à integração. Com um access token válido, ela devolve as chaves públicas do pool (que já são públicas), o que também serve de smoke test do login.
- Sem logs de acesso, por custo.

## Comandos

Pela raiz do repositório:

```sh
task infra:test         # go vet + testes de assertion do CDK e da Lambda do trigger, sem AWS e sem build
task infra:build        # compila a Lambda do trigger em dist/post-confirmation/bootstrap (linux/arm64)
task infra:synth:cicd   # template da stack cicd em infra/cdk.out, sem AWS
BUDGET_EMAIL=voce@exemplo.com task infra:synth   # compila a Lambda e gera todas as stacks
```

Os testes e o synth rodam sem credenciais. Nesse caso a stack fica agnóstica de conta e as ARNs usam `${AWS::AccountId}`. O CLI do CDK é chamado por `npx` com versão fixa (ver `infra/Taskfile.yml`), então só é preciso ter o Node instalado.

## Roles de deploy

Uma role de deploy por repositório implantável e uma role só leitura, para o `cdk diff` em PRs, por repositório que usa CDK:

| Repositório | Deploy (`dev`) | `cdk diff` em PR |
|---|---|---|
| `gestor-gastos-infra-base` | `gh-deploy-gestor-gastos-infra-base-dev` | `gh-diff-gestor-gastos-infra-base` |
| `gestor-gastos-users-service` | `gh-deploy-gestor-gastos-users-service-dev` | `gh-diff-gestor-gastos-users-service` |
| `gestor-gastos-categories-service` | `gh-deploy-gestor-gastos-categories-service-dev` | `gh-diff-gestor-gastos-categories-service` |
| `gestor-gastos-expenses-service` | `gh-deploy-gestor-gastos-expenses-service-dev` | `gh-diff-gestor-gastos-expenses-service` |
| `gestor-gastos-bff` | `gh-deploy-gestor-gastos-bff-dev` | `gh-diff-gestor-gastos-bff` |
| `gestor-gastos-frontend` | `gh-deploy-gestor-gastos-frontend-dev` | (sem CDK) |

ARN: `arn:aws:iam::<conta>:role/gh-deploy-<repo>-<env>`. O nome é previsível, então os workflows montam o ARN a partir de uma variável `AWS_ACCOUNT_ID` do GitHub, sem depender de saída de stack.

**Trust policy.** Ação `sts:AssumeRoleWithWebIdentity` vinda do OIDC provider, com `StringEquals` exato em duas claims:

- `token.actions.githubusercontent.com:aud` = `sts.amazonaws.com`
- `token.actions.githubusercontent.com:sub` = `repo:Rossete-Inc@339323229/<repo>@<repo-id>:environment:<env>`

O GitHub só emite esse `sub` quando o job declara `environment: dev`. Um job sem environment ou de outro repositório não consegue assumir a role. A regra de que `dev` só aceita a `main` fica nas regras de deployment do GitHub Environment.

**Permissões.** Todas as roles recebem só:

- `sts:AssumeRole` e `sts:TagSession` em `arn:aws:iam::<conta>:role/cdk-*`, as roles criadas pelo `cdk bootstrap` (deploy, file-publishing, image-publishing e lookup). Quem cria os recursos é a `cloudformation-execution-role` do bootstrap, assumida pelo CloudFormation.
- `ssm:GetParameter`, `ssm:GetParameters` e `ssm:GetParametersByPath` em `/gestor-gastos/<env>` e `/gestor-gastos/<env>/*`, para a esteira checar pré-requisitos antes do deploy e falhar com mensagem clara.

A role do **frontend** recebe também permissões no Amplify Hosting, porque o site não é publicado pelo CDK: a esteira envia o build do `expo export -p web` com `aws amplify create-deployment` e `start-deployment` e acompanha o job. As ações são `amplify:CreateDeployment`, `StartDeployment`, `GetApp`, `GetBranch`, `GetJob`, `ListJobs` e `StopJob`, em `arn:aws:amplify:us-east-1:<conta>:apps/*`. O escopo é `apps/*` porque o app só será criado na tarefa 6.6. Depois disso dá para restringir ao app id.

**Limite conhecido.** As roles de deploy assumem as roles `cdk-*` do bootstrap, que têm poder de administrador via CloudFormation. O que protege a conta é a trust por environment somada à regra do Environment `dev` (só a `main`, que é protegida e exige PR).

### Roles de `cdk diff`

O job de `cdk diff` roda em PRs de qualquer branch, então não pode usar a role de deploy. Cada repositório com CDK tem uma role `gh-diff-<repo>` com trust em `sub` = `repo:Rossete-Inc@339323229/<repo>@<repo-id>:pull_request`, o valor que o GitHub emite para um job de `pull_request` **sem** `environment:`. PRs de fork não recebem token OIDC.

**Formato do `sub`.** Os repositórios da Rossete-Inc emitem a claim no formato de subject imutável do GitHub (`use_immutable_subject: true`, conferir com `gh api repos/Rossete-Inc/<repo>/actions/oidc/customization/sub`): `repo:Rossete-Inc@339323229/<repo>@<repo-id>:...`. O id da org e os ids dos repositórios ficam em `infra/config.go` (`OrgID` e `RepoIDs`). Um repositório novo precisa entrar ali antes de ganhar role.

Permissões, todas de leitura:

- `sts:AssumeRole` e `sts:TagSession` só em `cdk-*-lookup-role-*`, a role do bootstrap com `ReadOnlyAccess`. As roles de deploy e de publicação de assets ficam de fora.
- `cloudformation:DescribeStacks` e `GetTemplate` nas stacks `gestor-gastos-*`.
- Leitura do SSM em `/gestor-gastos/*` e `/cdk-bootstrap/*`.

Por isso o diff roda com `cdk diff --method=template`: comparar por change set exigiria a role de deploy.

## Deploy local único da stack cicd

Feito **uma única vez por conta**, por um admin, depois do `cdk bootstrap` (tarefa 6.1). Depois disso a stack é atualizada pela esteira do próprio infra-base, com a role `gh-deploy-gestor-gastos-infra-base-<env>`.

1. Confirme que o bootstrap existe e que você está na conta certa:

   ```sh
   aws sts get-caller-identity
   aws cloudformation describe-stacks --stack-name CDKToolkit --region us-east-1 --query "Stacks[0].StackStatus"
   ```

2. Rode os testes e veja o que vai ser criado:

   ```sh
   task infra:test
   cd infra
   npx --yes aws-cdk@2.1148.0 diff gestor-gastos-cicd
   ```

3. Faça o deploy. O CDK mostra as mudanças de IAM e pede confirmação; confira as 11 roles (6 de deploy e 5 de diff) e o OIDC provider antes de aceitar:

   ```sh
   npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-cicd
   ```

   Se a conta já tiver um OIDC provider para `token.actions.githubusercontent.com` (criado à mão antes), o deploy falha porque só pode existir um por URL. Apague o antigo ou importe-o na stack antes de repetir.

4. Verifique:

   ```sh
   aws iam list-open-id-connect-providers
   aws iam list-roles --query "Roles[?starts_with(RoleName, 'gh-')].RoleName"
   aws iam get-role --role-name gh-deploy-gestor-gastos-users-service-dev --query "Role.AssumeRolePolicyDocument"
   ```

5. No GitHub, crie a variável `AWS_ACCOUNT_ID` na organização Rossete-Inc e o Environment `dev` (regra de deployment: só a branch `main`) em cada repositório implantável.

### Stack budget

O e-mail dos alertas **não fica no código**, porque o repositório é público. Informe-o pelo contexto do CDK ou pela variável de ambiente `BUDGET_EMAIL`. Sem ele, o synth e o deploy da stack budget falham com uma mensagem explicando o que fazer. As outras stacks não são afetadas.

```sh
cd infra
npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-budget -c budgetEmail=voce@exemplo.com
# ou
BUDGET_EMAIL=voce@exemplo.com npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-budget
```

Na esteira, o e-mail vem da variável `BUDGET_EMAIL` do GitHub Environment `dev`.

## Deploy das stacks do ambiente (events, auth, api)

Pré-requisitos: bootstrap do CDK feito e stack `cicd` implantada. Enquanto o deploy do infra-base não estiver na esteira (ver o topo de `.github/workflows/ci.yml`), ele é feito à mão por um admin. Confira sempre o diff antes:

```sh
task infra:test
task infra:build
cd infra
npx --yes aws-cdk@2.1148.0 diff gestor-gastos-dev-events gestor-gastos-dev-auth gestor-gastos-dev-api -c env=dev
npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-dev-events gestor-gastos-dev-auth gestor-gastos-dev-api -c env=dev
```

O CDK implanta na ordem events → auth → api e pede confirmação das mudanças de IAM (role do trigger e permissão para o Cognito invocá-lo). As três stacks são nomeadas porque `--all` incluiria as stacks da conta.

Verificação:

```sh
# Todos os parâmetros publicados
aws ssm get-parameters-by-path --path /gestor-gastos/dev --recursive --region us-east-1 \
  --query "Parameters[].[Name,Value]" --output table

# Chamada sem token: 401
API_URL=$(aws ssm get-parameter --name /gestor-gastos/dev/api/url --region us-east-1 --query Parameter.Value --output text)
curl -i "$API_URL/infra/auth-check"   # 401 {"message":"Unauthorized"}

# Preflight do CORS a partir do localhost
curl -i -X OPTIONS "$API_URL/infra/auth-check" -H "Origin: http://localhost:8081" -H "Access-Control-Request-Method: GET"
```

O fluxo completo (cadastro, código por e-mail, grupo `cliente` e `UserConfirmed` consumido) só é exercitado de ponta a ponta com o users-service, na tarefa 6.5. Para ver o trigger antes disso, cadastre e confirme um usuário de teste (`aws cognito-idp sign-up` e `confirm-sign-up`, com o código recebido por e-mail) e confira o log em `/aws/lambda/gestor-gastos-dev-post-confirmation` e o grupo com `aws cognito-idp admin-list-groups-for-user`.

## Convenções para as próximas stacks

- Nome de stack por ambiente: `gestor-gastos-<env>-<nome>` (ex.: `gestor-gastos-dev-auth`). Stacks da conta: `gestor-gastos-<nome>`.
- Parâmetros publicados no SSM em `/gestor-gastos/<env>/<...>`. As roles de deploy já podem ler esse caminho.
- Sem `exportName` em `CfnOutput`. Quem consome lê do SSM.
- Constantes compartilhadas (org, região, ambientes, nomes de role e de stack, prefixo e nomes dos parâmetros do SSM) ficam em `infra/config.go`.
- Uma stack lê o que outra publicou com `lerParametro` (`infra/ssm.go`), que vira um parâmetro do CloudFormation resolvido no deploy, e declara a ordem com `AddStackDependency`.
