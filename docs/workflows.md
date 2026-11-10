# Workflows reutilizáveis da esteira

Os repositórios do Gestor de Gastos não escrevem a própria esteira: eles chamam os workflows reutilizáveis deste repositório ([ADR 0007](adr/0007-esteira-ci-cd.md)).

| Workflow | Para quem | O que faz |
|---|---|---|
| [`go-service.yml`](../.github/workflows/go-service.yml) | users-service, categories-service, expenses-service, BFF, go-commons e, mais tarde, o próprio infra-base | `go mod tidy`, `gofmt`, `go vet`, `golangci-lint`, `go test` com cobertura, build arm64 das Lambdas, `cdk synth` + `cdk diff` comentado no PR e `cdk deploy` em dev |
| [`frontend.yml`](../.github/workflows/frontend.yml) | frontend | `tsc`, `eslint`, `prettier`, `check_boundaries.py`, `jest`, `expo export -p web` e deploy no Amplify Hosting |
| [`ci.yml`](../.github/workflows/ci.yml) | só este repositório | `actionlint` (com `shellcheck`) em todos os workflows, checagem de "Validado localmente" e `go mod tidy`, `gofmt`, `go vet` e testes da Lambda do trigger e do CDK (`infra/`) |

## Gatilhos

A AWS tem um ambiente só, **dev** ([ADR 0007](adr/0007-esteira-ci-cd.md), atualização de 2026-11-10). Quem decide os gatilhos é o workflow que chama. Os workflows reutilizáveis olham o evento e escolhem o que rodar:

| Evento | Jobs |
|---|---|
| `pull_request` | Validado localmente, qualidade, build e `cdk synth` + `cdk diff` contra dev (comentário no PR, atualizado a cada push). No front: as verificações e o `expo export`. Sem deploy |
| `push` na `main` | Qualidade, build e deploy em **dev** (Environment `dev`, sem aprovação manual) + smoke test |

Tags `vX.Y.Z` não fazem deploy. A `go-commons` continua publicando tags semver, mas como biblioteca.

- **Concorrência:** o job de deploy usa o grupo `deploy-<repo>-dev`, com `cancel-in-progress: false`. Um deploy novo espera o anterior terminar, sem cancelá-lo.
- **Credenciais:** OIDC na conta `vars.AWS_ACCOUNT_ID`, região `us-east-1`. O `<repo>` vem de `github.event.repository.name`. Nenhum segredo é usado.
  - **Deploy:** role `gh-deploy-<repo>-dev`, cuja trust aceita só `repo:Rossete-Inc/<repo>:environment:dev`. Por isso o job de deploy usa o Environment `dev`.
  - **`cdk diff` no PR:** role só leitura `gh-diff-<repo>`, cuja trust aceita só `repo:Rossete-Inc/<repo>:pull_request`. É o `sub` que o GitHub emite em `pull_request` para um job **sem** `environment:`, por isso o job de diff não usa Environment. A role só assume a lookup role do bootstrap do CDK (`cdk-*-lookup-role-*`) e lê templates do CloudFormation e o SSM `/gestor-gastos/*`. Sem permissão para criar change set, o diff roda com `--method=template`, comparando com o template implantado.
- **Validado localmente:** o job reprova o PR que tenha **Funcionalidade** ou **Correção** marcada (ou nenhum tipo marcado) sem `- [x] Validado localmente` no corpo. PR só de **Infraestrutura / CI** ou **Documentação** fica dispensado. Para o job rodar de novo quando o corpo é editado, inclua `edited` nos `types` do `pull_request`.

## Serviço Go

`.github/workflows/ci.yml` em cada repositório Go:

```yaml
name: ci

on:
  pull_request:
    types: [opened, synchronize, reopened, edited]
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  esteira:
    uses: Rossete-Inc/gestor-gastos-infra-base/.github/workflows/go-service.yml@main
    permissions:
      contents: read
      id-token: write       # OIDC na AWS
      pull-requests: write  # comentário do cdk diff
    with:
      build-tags: integration
      dynamodb-local: true
      lambdas: ./cmd/lambda
      cdk-dir: infra
      smoke-test: ./scripts/smoke.sh
```

Para a `go-commons`, que não tem deploy: `lambdas: ""` e `cdk-dir: ""`. Ficam só qualidade e testes. As tags semver dela são publicadas à parte e não disparam deploy.

### Inputs do `go-service.yml`

| Input | Tipo | Padrão | Uso |
|---|---|---|---|
| `go-version-file` | string | `go.mod` | De onde sai a versão do Go |
| `golangci-lint-version` | string | `v2.15` | Versão do golangci-lint |
| `build-tags` | string | `""` | Build tags de vet, lint e testes (ex.: `integration`) |
| `dynamodb-local` | boolean | `false` | Sobe o `amazon/dynamodb-local:2.6.1` na porta 8000 e define `DYNAMODB_ENDPOINT` |
| `min-coverage` | number | `80` | Cobertura mínima por pacote, em %. `0` desliga |
| `lambdas` | string | `./cmd/lambda` | Pacotes `main` das Lambdas, separados por espaço. Cada um vira `<dist-dir>/<pasta>/bootstrap` (linux/arm64, `-tags lambda.norpc`). Vazio: nada a compilar |
| `dist-dir` | string | `dist` | Pasta, a partir da raiz, com os binários que o CDK empacota |
| `cdk-dir` | string | `infra` | Pasta do app CDK em Go. Vazio: sem synth, diff e deploy |
| `cdk-version` | string | `2` | Versão do pacote npm `aws-cdk` |
| `cdk-args` | string | `""` | Argumentos extras de `cdk synth`, `diff` e `deploy` |
| `node-version` | string | `24` | Node do CLI do CDK |
| `smoke-test` | string | `""` | Comando bash rodado na raiz após o deploy, com as credenciais do ambiente e `AMBIENTE=dev` |
| `exigir-validacao-local` | boolean | `true` | Liga a checagem de "Validado localmente" |

### Contrato com o CDK do serviço

- O app CDK lê o ambiente do contexto `env` (`cdk deploy --all --context env=dev`). A variável `AMBIENTE` também fica disponível.
- O binário de cada Lambda está em `<raiz>/<dist-dir>/<pasta>/bootstrap`. Com `cdk-dir: infra`, use `awslambda.Code_FromAsset(jsii.String("../dist/lambda"), nil)`, runtime `provided.al2023` e arquitetura `ARM_64`.
- O canary do CodeDeploy, o alias `live` e a falha clara quando faltar parâmetro no SSM ficam no CDK de cada serviço (tarefa 6.5), não no workflow.

## Frontend

`.github/workflows/ci.yml` no `gestor-gastos-frontend`:

```yaml
name: ci

on:
  pull_request:
    types: [opened, synchronize, reopened, edited]
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  esteira:
    uses: Rossete-Inc/gestor-gastos-infra-base/.github/workflows/frontend.yml@main
    permissions:
      contents: read
      id-token: write
    with:
      verificacoes-extras: |
        npm run api:gerar
        git diff --exit-code src/shared/api/generated
      # Publicados pelas stacks api e auth do infra-base (infra/README.md):
      env-ssm: |
        EXPO_PUBLIC_API_URL=/gestor-gastos/{env}/api/url
        EXPO_PUBLIC_COGNITO_CLIENT_ID=/gestor-gastos/{env}/auth/user-pool-client-id
```

### Inputs do `frontend.yml`

| Input | Tipo | Padrão | Uso |
|---|---|---|---|
| `node-version` | string | `24` | Versão do Node |
| `python-version` | string | `3.12` | Python do `scripts/check_boundaries.py` |
| `verificacoes-extras` | string | `""` | Comandos bash extras, antes do jest |
| `env-build-pr` | string | valores locais (`EXPO_PUBLIC_APP_ENV=local`, API em `127.0.0.1:8080`, cognito-local) | Linhas `NOME=valor` do `expo export` de verificação |
| `env-ssm` | string | `""` | Linhas `NOME=/caminho/ssm` lidas antes do build de deploy. `{env}` vira `dev`. Parâmetro ausente reprova o deploy com mensagem clara |
| `env-deploy` | string | `EXPO_PUBLIC_APP_ENV={env}` e `EXPO_PUBLIC_COGNITO_REGION=us-east-1` | Linhas `NOME=valor` fixas do build de deploy |
| `dist-dir` | string | `dist` | Saída do `expo export` |
| `amplify-app-id-ssm` | string | `/gestor-gastos/{env}/frontend/amplify-app-id` | Parâmetro do SSM com o id do app no Amplify |
| `amplify-branch` | string | `main` | Branch do app no Amplify que recebe o deploy |
| `smoke-test` | string | `""` | Comando bash após o deploy, com `AMBIENTE` e `APP_URL` |
| `exigir-validacao-local` | boolean | `true` | Liga a checagem de "Validado localmente" |

Os scripts esperados no `package.json` são `typecheck`, `lint`, `check:boundaries` e `test:ci`. O `format:check` roda se existir.

### De onde vem o id do app no Amplify

Em ordem:

1. A variável `AMPLIFY_APP_ID` do **Environment** `dev` do GitHub, se existir. Serve para o primeiro deploy ou para apontar para outro app sem mexer no SSM.
2. O parâmetro `/gestor-gastos/dev/frontend/amplify-app-id` no SSM, publicado pelo CDK do frontend (tarefa 6.6). É o caminho normal.

Se nenhum dos dois existir, o deploy falha com mensagem clara. O deploy segue o fluxo manual do Amplify (app sem repositório conectado): `create-deployment`, upload do zip para a URL pré-assinada, `start-deployment` e espera o job terminar (até 10 minutos). A URL publicada é `https://<branch>.<app-id>.amplifyapp.com`.

## O que cada repositório precisa configurar

| Onde | O quê |
|---|---|
| Organização (ou repositório) → Variables | `AWS_ACCOUNT_ID`: id da conta AWS. Não é segredo, mas fica fora do código |
| Repositório → Environments → `dev` | Sem revisores, com a regra de deploy restrita à branch `main`. É o único Environment: o `cdk diff` do PR não usa Environment |
| Environment `dev` → Variables (só o front, opcional) | `AMPLIFY_APP_ID`, se não quiser ler do SSM |
| AWS (stack `cicd`, tarefa 6.2) | Role de deploy `gh-deploy-<repo>-dev` e, nos repositórios com CDK (infra-base, serviços e BFF), a role só leitura `gh-diff-<repo>` |
| Branch protection da `main` | Exigir os checks da esteira (`esteira / vet, lint e testes`, `esteira / Validado localmente` etc.) |

### Limitações conhecidas

- **PR de fork:** não recebe token OIDC, então o `cdk diff` é pulado. Os demais jobs rodam.
- **Diff só de leitura:** quem abre PR a partir de uma branch do repositório pode, em tese, alterar o workflow do PR, mas só alcança a role `gh-diff-<repo>`, que não altera nada na conta. O `--method=template` não mostra o que o CloudFormation faria com recursos alterados fora do CDK (drift); para o MVP isso basta.
- **Versão:** `@main` acompanha a última versão dos workflows. Quando a esteira estabilizar, crie tags neste repositório (ex.: `workflows-v1`) e troque o `@main` por elas, para que uma mudança aqui não quebre todos os repositórios de uma vez.
- **Variáveis em workflow reutilizável:** o `vars` lido dentro do workflow é o do repositório que chama (e do Environment do job). Por isso `AWS_ACCOUNT_ID` e `AMPLIFY_APP_ID` ficam no repositório de cada serviço ou na organização.

## Deploy do próprio infra-base

O `ci.yml` deste repositório roda `go vet` e os testes da Lambda do trigger (`cmd/post-confirmation`) e dos testes de assertion do CDK (`infra/`), sem AWS, mas ainda não faz `cdk diff` nem deploy. As stacks `events`, `auth` e `api` são implantadas à mão por um admin, como descrito em [`infra/README.md`](../infra/README.md).

O deploy ainda não chama o `go-service.yml` porque ele faz `cdk deploy --all`, o que inclui as stacks da conta: a `cicd`, que cria as próprias roles que a esteira assume, e a `budget`, que exige o e-mail dos alertas (`BUDGET_EMAIL`, variável que ainda não existe no GitHub). Quando isso for resolvido, o job fica como está comentado no topo do `ci.yml` (`lambdas: ./cmd/post-confirmation`, `cdk-dir: infra`). A Lambda já segue a convenção do `go-service.yml` (`dist/post-confirmation/bootstrap`). A stack `cicd` continua com deploy local, feito por um admin.

## Verificação local

```sh
go install github.com/rhysd/actionlint/cmd/actionlint@latest
actionlint            # usa o shellcheck do PATH, se houver
```
