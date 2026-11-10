# infra (AWS CDK em Go)

App CDK do `gestor-gastos-infra-base`. Região fixa: `us-east-1`. Ambientes `dev` e `prod` ficam na mesma conta, separados pelo prefixo nos nomes e pelo caminho `/gestor-gastos/<env>/...` no SSM. CloudFormation exports nunca são usados ([ADR 0007](../docs/adr/0007-esteira-ci-cd.md)).

## Stacks

| Stack | Escopo | O que cria |
|---|---|---|
| `gestor-gastos-cicd` | conta | OIDC provider do GitHub (`token.actions.githubusercontent.com`, audience `sts.amazonaws.com`) e 12 roles `gh-deploy-<repo>-<env>` |
| `gestor-gastos-budget` | conta | Budget mensal `gestor-gastos-mensal` de US$ 5, com alerta por e-mail a 80% do custo real e a 100% do previsto |

As stacks por ambiente (auth, events, api) entram na tarefa 6.4.

## Comandos

Pela raiz do repositório:

```sh
task infra:test         # go vet + testes de assertion do CDK, sem AWS
task infra:synth:cicd   # template da stack cicd em infra/cdk.out, sem AWS
BUDGET_EMAIL=voce@exemplo.com task infra:synth   # todas as stacks
```

Os testes e o synth rodam sem credenciais. Nesse caso a stack fica agnóstica de conta e as ARNs usam `${AWS::AccountId}`. O CLI do CDK é chamado por `npx` com versão fixa (ver `infra/Taskfile.yml`), então só é preciso ter o Node instalado.

## Roles de deploy

Uma role por repositório implantável e ambiente:

| Repositório | dev | prod |
|---|---|---|
| `gestor-gastos-infra-base` | `gh-deploy-gestor-gastos-infra-base-dev` | `gh-deploy-gestor-gastos-infra-base-prod` |
| `gestor-gastos-users-service` | `gh-deploy-gestor-gastos-users-service-dev` | `gh-deploy-gestor-gastos-users-service-prod` |
| `gestor-gastos-categories-service` | `gh-deploy-gestor-gastos-categories-service-dev` | `gh-deploy-gestor-gastos-categories-service-prod` |
| `gestor-gastos-expenses-service` | `gh-deploy-gestor-gastos-expenses-service-dev` | `gh-deploy-gestor-gastos-expenses-service-prod` |
| `gestor-gastos-bff` | `gh-deploy-gestor-gastos-bff-dev` | `gh-deploy-gestor-gastos-bff-prod` |
| `gestor-gastos-frontend` | `gh-deploy-gestor-gastos-frontend-dev` | `gh-deploy-gestor-gastos-frontend-prod` |

ARN: `arn:aws:iam::<conta>:role/gh-deploy-<repo>-<env>`. O nome é previsível, então os workflows montam o ARN a partir de uma variável `AWS_ACCOUNT_ID` do GitHub, sem depender de saída de stack.

**Trust policy.** Ação `sts:AssumeRoleWithWebIdentity` vinda do OIDC provider, com `StringEquals` exato em duas claims:

- `token.actions.githubusercontent.com:aud` = `sts.amazonaws.com`
- `token.actions.githubusercontent.com:sub` = `repo:Rossete-Inc/<repo>:environment:<env>`

O GitHub só emite esse `sub` quando o job declara `environment: dev` ou `environment: prod`. Um job sem environment, de outro repositório ou de outro ambiente não consegue assumir a role. A regra de que `dev` só aceita a `main` e `prod` só aceita tags `v*.*.*` fica nas regras de deployment dos GitHub Environments.

**Permissões.** Todas as roles recebem só:

- `sts:AssumeRole` e `sts:TagSession` em `arn:aws:iam::<conta>:role/cdk-*`, as roles criadas pelo `cdk bootstrap` (deploy, file-publishing, image-publishing e lookup). Quem cria os recursos é a `cloudformation-execution-role` do bootstrap, assumida pelo CloudFormation.
- `ssm:GetParameter`, `ssm:GetParameters` e `ssm:GetParametersByPath` em `/gestor-gastos/<env>` e `/gestor-gastos/<env>/*`, para a esteira checar pré-requisitos antes do deploy e falhar com mensagem clara.

A role do **frontend** recebe também permissões no Amplify Hosting, porque o site não é publicado pelo CDK: a esteira envia o build do `expo export -p web` com `aws amplify create-deployment` e `start-deployment` e acompanha o job. As ações são `amplify:CreateDeployment`, `StartDeployment`, `GetApp`, `GetBranch`, `GetJob`, `ListJobs` e `StopJob`, em `arn:aws:amplify:us-east-1:<conta>:apps/*`. O escopo é `apps/*` porque o app só será criado na tarefa 6.6. Depois disso dá para restringir ao app id.

**Limite conhecido.** As roles de `dev` e de `prod` assumem as mesmas roles `cdk-*` do bootstrap, que têm poder de administrador via CloudFormation. Na mesma conta, a separação entre ambientes vem da trust por environment e das regras dos GitHub Environments, não do IAM. Contas separadas ficam para quando houver usuários reais (ADR 0007).

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

3. Faça o deploy. O CDK mostra as mudanças de IAM e pede confirmação; confira as 12 roles e o OIDC provider antes de aceitar:

   ```sh
   npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-cicd
   ```

   Se a conta já tiver um OIDC provider para `token.actions.githubusercontent.com` (criado à mão antes), o deploy falha porque só pode existir um por URL. Apague o antigo ou importe-o na stack antes de repetir.

4. Verifique:

   ```sh
   aws iam list-open-id-connect-providers
   aws iam list-roles --query "Roles[?starts_with(RoleName, 'gh-deploy-')].RoleName"
   aws iam get-role --role-name gh-deploy-gestor-gastos-users-service-dev --query "Role.AssumeRolePolicyDocument"
   ```

5. No GitHub, crie a variável `AWS_ACCOUNT_ID` na organização Rossete-Inc e os Environments `dev` (só a branch `main`) e `prod` (só tags `v*.*.*`, com aprovação) em cada repositório implantável.

### Stack budget

O e-mail dos alertas **não fica no código**, porque o repositório é público. Informe-o pelo contexto do CDK ou pela variável de ambiente `BUDGET_EMAIL`. Sem ele, o synth e o deploy da stack budget falham com uma mensagem explicando o que fazer. As outras stacks não são afetadas.

```sh
cd infra
npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-budget -c budgetEmail=voce@exemplo.com
# ou
BUDGET_EMAIL=voce@exemplo.com npx --yes aws-cdk@2.1148.0 deploy gestor-gastos-budget
```

Na esteira, o e-mail vem de uma variável do GitHub Environment (`BUDGET_EMAIL`).

## Convenções para as próximas stacks

- Nome de stack por ambiente: `gestor-gastos-<env>-<nome>` (ex.: `gestor-gastos-dev-auth`). Stacks da conta: `gestor-gastos-<nome>`.
- Parâmetros publicados no SSM em `/gestor-gastos/<env>/<...>`. As roles de deploy já podem ler esse caminho.
- Sem `exportName` em `CfnOutput`. Quem consome lê do SSM.
- Constantes compartilhadas (org, região, ambientes, nomes de role, prefixo do SSM) ficam em `infra/config.go`.
