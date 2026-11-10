# Plano — Gestor de Gastos (MVP)

## Contexto
Projeto novo, organizado em **repositórios separados** (sem monorepo, ver ADR 0006), criados na organização GitHub [Rossete-Inc](https://github.com/orgs/Rossete-Inc/repositories). Objetivo: plataforma simples para o usuário lançar gastos de **cartão** (por tipo: alimentação, transporte…) e **gastos fixos** (aluguel, luz, internet…), ver **listas e gráficos de pizza por grupo/categoria**, com login, permissões e um **admin** que gerencia clientes. Restrições: AWS com custo mínimo (serverless), front React como PWA (Android/iOS), back em microserviços + BFF que fornece dados de tela, design no Figma.

Decisões confirmadas:
- **Back-end: Go** — cold start ~100 ms, Lambdas de 128–256 MB em arm64 (Graviton, ~20% mais barato); custo Lambda é GB-s, então a JVM do Kotlin custaria mais e exigiria SnapStart.
- **Front-end: Expo + React Native + Expo Router**, seguindo a skill `app-multiplataforma-guardrails` (PWA no Amplify agora, lojas depois).
- **Banco: DynamoDB on-demand**, uma tabela por microserviço.

## Arquitetura

Diagramas de arquitetura, fluxos, telas e dados: [diagramas.md](diagramas.md).

```
PWA (Expo web, Amplify Hosting)
   │  login (SRP)            │  HTTPS + JWT
   ▼                         ▼
Cognito User Pool ──► API Gateway HTTP API (JWT authorizer)  ← único ponto público
 grupos: admin, cliente      │
                             ▼
                      BFF (Lambda Go)  — endpoints por tela, agrega dados
               ┌─────────────┼──────────────┐   (Lambda Invoke via IAM, sem rede pública)
               ▼             ▼              ▼
        users-svc     categories-svc   expenses-svc   (Lambdas Go)
        DDB users     DDB categories   DDB expenses
               └──── EventBridge (ex.: UserDeleted → expenses-svc limpa dados)
```

- **API Gateway HTTP API** (≈70% mais barato que REST API) com authorizer JWT do Cognito. Só o BFF fica exposto; microserviços são invocados diretamente (`lambda:InvokeFunction`), sem API própria — mais barato e sem superfície pública. Trade-off: acoplamento ao transporte Lambda; isolar num client interno para trocar por HTTP depois se preciso.
- **Cognito** (tier Lite, grátis até 10k MAU): cadastro com confirmação por e-mail, grupos `admin`/`cliente` (vêm na claim `cognito:groups`). Trigger *Post Confirmation* adiciona ao grupo `cliente` e publica `UserConfirmed` no EventBridge; o users-svc consome o evento e cria o perfil (o trigger vive no infra-base, que sobe antes dos serviços). O admin inicial é criado via script/CLI.
- **Autorização em 2 camadas**: BFF valida grupo para rotas `/admin/*`; cada serviço recebe `userId` do BFF e **sempre** filtra por ele (dono do dado).
- **Custo**: tudo dentro do free tier em baixo volume; CloudWatch Logs com retenção 7 dias; **AWS Budget com alerta** (ex.: US$ 5) criado na fase 0.

## Microserviços e dados (DynamoDB, single-table por serviço)

| Serviço | Responsabilidade | Chaves |
|---|---|---|
| **users-svc** | perfil, listar/ativar/bloquear usuários, alterar papel (via Cognito Admin API) | `PK=USER#id`; GSI para listagem admin |
| **categories-svc** | categorias padrão do sistema + personalizadas do usuário. Grupo `CARTAO` (alimentação, transporte, lazer, saúde, compras, outros) e `FIXO` (aluguel, luz, água, internet, telefone, condomínio) | `PK=SYSTEM` ou `USER#id`, `SK=CAT#grupo#id` |
| **expenses-svc** | CRUD de lançamentos; agregação mensal por grupo/categoria | `PK=USER#id`, `SK=EXP#AAAA-MM-DD#ulid`; atributos: `grupo`, `categoriaId`, `valorCentavos` (int64, nunca float), `descricao`, `data`, `recorrente` (bool, fixos) |

- Agregação para gráficos: Query por usuário + prefixo do mês e soma em memória (volume pessoal é pequeno). Se crescer, materializar totais via DynamoDB Streams (registrar em ADR).
- Gastos fixos recorrentes: MVP lança manualmente com flag `recorrente`; geração automática mensal (EventBridge Scheduler) fica para v2.

## BFF — endpoints orientados a tela (contrato OpenAPI)
`openapi.yaml` no repositório `gestor-gastos-bff` é a fonte da verdade, publicado como asset de cada release; o front gera tipos com `orval` em `shared/api/generated` a partir de uma versão fixada.

- `GET /me`
- `GET /telas/dashboard?mes=AAAA-MM` → total do mês, total por grupo (cartão × fixo), pizza por categoria, últimos 5 lançamentos
- `GET /telas/gastos?mes=&grupo=&categoria=&cursor=` → lista paginada com nome/cor da categoria já resolvidos
- `GET /telas/gasto-form` → categorias agrupadas para o formulário
- `POST /gastos`, `PUT /gastos/{id}`, `DELETE /gastos/{id}` (escritas também passam pelo BFF por ser o único ponto público; ele só delega)
- `GET/POST /categorias` (personalizadas)
- `GET /admin/telas/usuarios`, `PATCH /admin/usuarios/{id}` (status, papel)

## Repositórios (separados, sem monorepo — ADR 0006)

Um repositório por deployable, mais uma lib Go compartilhada, todos na organização [Rossete-Inc](https://github.com/orgs/Rossete-Inc/repositories) com URL `https://github.com/Rossete-Inc/<repo>`. Módulos Go usam o path `github.com/Rossete-Inc/<repo>`. Cada repositório tem seu próprio CI/CD (GitHub Actions) e seu próprio `cdk deploy`.

| Repositório | Conteúdo | Publica / consome |
|---|---|---|
| `gestor-gastos-infra-base` (este) | Docs transversais (plano, ADRs), CDK de Cognito (+ trigger), HTTP API + authorizer, barramento EventBridge, Budget | Publica no SSM: user pool id, client id, API id, bus name |
| `gestor-gastos-go-commons` | Módulo Go versionado por tag semver: claims/auth, erros, logging `slog`, envelope e client de Lambda Invoke | Consumido via `go get github.com/Rossete-Inc/gestor-gastos-go-commons@vX.Y.Z` |
| `gestor-gastos-users-service` | Lambda + tabela DynamoDB + CDK próprio | Publica ARN da função no SSM; lê pool id |
| `gestor-gastos-categories-service` | Lambda + tabela DynamoDB + CDK próprio | Publica ARN da função no SSM |
| `gestor-gastos-expenses-service` | Lambda + tabela DynamoDB + CDK próprio + regra EventBridge (`UserDeleted`) | Publica ARN no SSM; lê bus name |
| `gestor-gastos-bff` | Lambda BFF + `openapi.yaml` + CDK que pluga as rotas na HTTP API | Lê ARNs dos serviços e API id do SSM; publica `openapi.yaml` na release |
| `gestor-gastos-frontend` | App Expo (guardrails) + `amplify.yml` | Gera tipos com `orval` do `openapi.yaml` de uma release do BFF |

**Integração entre repositórios** só por três vias:
- **SSM Parameter Store** em `/gestor-gastos/<env>/...` (nunca CloudFormation exports, que travam deploys independentes);
- **contrato OpenAPI versionado** (releases do BFF);
- **tags semver** da `gestor-gastos-go-commons`.

**Ordem de deploy:** infra-base → serviços → BFF → frontend. ADRs específicos de um repositório vivem nele; os transversais ficam aqui.

Estrutura interna:
```
gestor-gastos-infra-base/   docs/{plano.md,adr/}, infra/ (CDK Go: auth, api, events, budget), .github/workflows/
gestor-gastos-go-commons/   auth/, errs/, logging/, invoke/ (envelope + client), go.mod
gestor-gastos-<nome>-service/
                            cmd/lambda/main.go, internal/{domain,app,store/dynamo,handler}, infra/ (CDK Go), .github/workflows/
gestor-gastos-bff/          cmd/lambda/main.go, internal/{handler,screens,clients}, openapi.yaml, infra/, .github/workflows/
gestor-gastos-frontend/     src/app/ rotas finas: (auth)/login, cadastro, confirmar; (app)/dashboard, gastos, gastos/[id], categorias, perfil; (admin)/usuarios
                            src/features/{auth,expenses,categories,dashboard,admin}/
                            src/shared/{ui,api,config,platform}, amplify.yml, .github/workflows/
```
- Lambdas: runtime `provided.al2023`, arm64, binário `bootstrap`.
- Front: auth com `aws-amplify` (módulo Auth apenas) isolado em `features/auth`; token injetado no cliente HTTP de `shared/api`. Gráfico de pizza com `react-native-gifted-charts` (funciona web e nativo via `react-native-svg`) — lib nova ⇒ ADR, conforme guardrails.

## Esteira CI/CD (ADR 0007)

Todo deploy na AWS é automático, feito pelo **GitHub Actions + AWS CDK**. Ninguém faz deploy manual depois do bootstrap.

| Gatilho | O que roda | Destino |
|---|---|---|
| Pull request | Go: `go vet`, `golangci-lint`, `go test`, build arm64, `cdk synth` + `cdk diff` comentado no PR. Front: `tsc`, `eslint`, `jest`, `check_boundaries.py`. BFF: lint do `openapi.yaml` | Nenhum deploy |
| Merge na `main` | Build + `cdk deploy` + canary CodeDeploy nas Lambdas + smoke test | **dev** automático |

- **Credenciais:** OIDC do GitHub na AWS, sem access keys. Cada repositório tem a role de deploy `gh-deploy-<repo>-dev`, com trust restrita a `repo:Rossete-Inc/<repo>:environment:dev`, e a role só leitura `gh-diff-<repo>` para o `cdk diff` do PR, com trust `repo:Rossete-Inc/<repo>:pull_request`.
- **Bootstrap (único passo manual, uma vez por conta):** `cdk bootstrap` + deploy local da stack `cicd` do infra-base, que cria o OIDC provider e as roles.
- **Workflows reutilizáveis** no infra-base (`.github/workflows/go-service.yml`, `frontend.yml`), chamados pelos outros repositórios por tag.
- **Rollback:** Lambdas publicadas por alias `live`. O deploy usa `Canary10Percent5Minutes` e volta automaticamente se o alarme de erros do CloudWatch disparar.
- **Ambiente único:** só `dev`, em us-east-1, com recursos prefixados e parâmetros em `/gestor-gastos/dev/` no SSM. Tags `vX.Y.Z` não fazem deploy. É um projeto de estudo e precisa ficar barato; um segundo ambiente fica para uma decisão futura. Um `concurrency` group por repositório evita deploys simultâneos.
- **Casos específicos:** `go-commons` só testa e publica a tag. O frontend faz `expo export -p web` e deploy no Amplify via `aws amplify create-deployment`.

## Design no Figma
Criar arquivo novo via Figma MCP (skills `figma-create-new-file` + `figma-generate-library` + `figma-generate-design`), mobile-first 390 px:
1. Tokens (cores, tipografia, espaçamento, cores por categoria) e componentes base (Button, Input, Select, Card, ListItem, Badge, TabBar, valor monetário).
2. Telas: Login, Cadastro, Confirmar e-mail, Dashboard (totais + pizza + últimos), Lista de gastos (filtros mês/grupo/categoria), Novo/Editar gasto, Categorias, Perfil, Admin › Usuários.
Os tokens do Figma viram `shared/ui/tokens` no front.

## Fases de entrega
0. **Fundação** — criar os repositórios, ADRs, conta AWS + Budget, bootstrap (CDK + stack `cicd` com OIDC e roles), workflows reutilizáveis no infra-base e esteira em cada repositório; primeira tag da `go-commons`.
1. **Figma** — design system + telas (acima).
2. **Infra base** (`gestor-gastos-infra-base`) — Cognito (pool, client, grupos, trigger), HTTP API + authorizer, EventBridge, parâmetros no SSM. Tabelas DynamoDB ficam no repositório de cada serviço.
3. **Serviços Go** — categories (com seed das padrão) → expenses → users; testes unitários do domínio + testes de store com DynamoDB Local.
4. **BFF** — OpenAPI, handlers por tela, autorização admin, testes com clients mockados.
5. **Front** — auth → formulário/lista de gastos → dashboard com pizza → categorias → admin; MSW para mock até o BFF subir.
6. **PWA e deploy** — manifest, service worker, Amplify Hosting (`amplify.yml`, rewrite SPA, cache conforme guardrails).
7. **E2E** — Playwright: cadastro → login → lançar gasto → ver na pizza; admin bloqueia usuário → login falha.

## Verificação
- Em cada repositório Go: `go test ./...`, `golangci-lint run` e `cdk synth` (no CI do próprio repositório).
- Front: `npx tsc --noEmit`, `npx eslint .`, `npx jest`, e `python <skill>/scripts/check_boundaries.py .` no `gestor-gastos-frontend`.
- Ambiente `dev` na AWS: merge na `main` de cada repositório na ordem infra-base → serviços → BFF → frontend, com a esteira fazendo o deploy; criar admin via CLI, rodar o E2E Playwright contra a URL do Amplify.
- Instalar o PWA num Android e num iPhone (Adicionar à tela inicial) e validar login/lançamento.
- Conferir no Cost Explorer após 1 semana que o custo está ~US$ 0.
