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

O deploy dos serviços, do BFF e do frontend é automático, feito pelo **GitHub Actions + AWS CDK**. As stacks do próprio infra-base (`cicd`, `budget`, `auth`, `events`, `web`, `api`) ainda têm deploy manual, feito por um admin; o porquê está em `docs/workflows.md`.

| Gatilho | O que roda | Destino |
|---|---|---|
| Pull request | Go: `go vet`, `golangci-lint`, `go test`, build arm64, `cdk synth` + `cdk diff` comentado no PR. Front: `tsc`, `eslint`, `jest`, `check_boundaries.py`. BFF: lint do `openapi.yaml` | Nenhum deploy |
| Merge na `main` | Build + `cdk deploy` + canary CodeDeploy nas Lambdas + smoke test. Frontend: deploy no Amplify + smoke da PWA + E2E Playwright | **dev** automático |

- **Credenciais:** OIDC do GitHub na AWS, sem access keys. Cada repositório tem a role de deploy `gh-deploy-<repo>-dev`, com trust restrita a `repo:Rossete-Inc@339323229/<repo>@<repo-id>:environment:dev`, e a role só leitura `gh-diff-<repo>` para o `cdk diff` do PR, com trust `repo:Rossete-Inc@339323229/<repo>@<repo-id>:pull_request`.
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

A entrega é por **fatias verticais** (ADR 0008): cada fatia junta serviço + rotas do BFF + telas do front e vai de ponta a ponta antes da próxima. A lista de tarefas está em `openspec/changes/mvp-gestor-gastos/tasks.md`.

0. **Fundação** — repositórios, ADRs, workflows reutilizáveis e esteira em cada repositório; ambiente local em Docker (`local/`, `docs/local.md`); `go-commons` com a primeira tag.
1. **Figma** — design system + telas (acima). As telas que faltam seguem os PNGs de referência enquanto o plano Starter do Figma limitar o acesso.
2. **Fatia 1, Autenticação e perfil** — users-service, `/me` no BFF, telas de login, cadastro e perfil.
3. **Base na nuvem** — bootstrap, stack `cicd` (OIDC e roles), Budget, Cognito com trigger, HTTP API, EventBridge e SSM; deploy da fatia 1 em `dev`.
4. **Fatias 2 a 5** — Categorias → Gastos → Painel (pizza com `react-native-gifted-charts`) → Administração. Cada uma: PR → merge → deploy em `dev` → validação do usuário em `dev`.
5. **PWA** — manifest, service worker, headers de cache no Amplify.
6. **Integração ponta a ponta** — E2E Playwright na esteira do frontend (cadastro → gasto → painel; admin bloqueia, reativa e exclui); versão final de todos os repositórios com smoke e canary; custo conferido após 7 dias.

## Verificação
- Em cada repositório Go: `go test ./...`, `golangci-lint run` e `cdk synth` (no CI do próprio repositório).
- Front: `npx tsc --noEmit`, `npx eslint .`, `npx jest`, e `python <skill>/scripts/check_boundaries.py .` no `gestor-gastos-frontend`.
- Ambiente `dev` na AWS: merge na `main` de cada repositório, com a esteira fazendo o deploy (no primeiro deploy, na ordem infra-base → serviços → BFF → frontend); o E2E Playwright roda na esteira do frontend contra a URL do Amplify, depois de cada deploy.
- Cada fatia é validada pelo usuário em `dev` depois do merge (ADR 0008).
- Instalar o PWA num Android e num iPhone (Adicionar à tela inicial) e validar login/lançamento.
- Conferir no Cost Explorer após 1 semana que o custo está ~US$ 0.
