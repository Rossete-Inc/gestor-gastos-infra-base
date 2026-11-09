# Tasks

> Cada grupo indica o repositório da org Rossete-Inc onde o trabalho acontece. A entrega é feita em **fatias verticais** (design.md, seção "Execução local e entrega em fatias"). Cada fatia roda localmente, é **validada pelo usuário** e só depois sobe para `dev`. Nenhuma tarefa de fatia toca a AWS antes da sua validação local.

## 1. Fundação GitHub (sem AWS)

- [x] 1.1 Criar os 7 repositórios **públicos** em Rossete-Inc (`gestor-gastos-infra-base`, `-go-commons`, `-users-service`, `-categories-service`, `-expenses-service`, `-bff`, `-frontend`) com a branch `main` protegida; verificar com `gh repo list Rossete-Inc`
  - Públicos porque a org está no plano Free (ver design.md, "Repositórios públicos e região"). A proteção da `main` exige PR (0 aprovações, já que o desenvolvedor é único), histórico linear e conversas resolvidas, vale também para admins e bloqueia force push e exclusão. Ela já está aplicada nos 6 repositórios iniciados com README.
  - O `infra-base` foi criado vazio para receber o push da 1.2. A proteção dele é aplicada ao fim da 1.2.
- [x] 1.2 Ligar este diretório ao remoto `gestor-gastos-infra-base`, fazer commit e push dos docs e do OpenSpec, e aplicar a proteção da `main` do mesmo jeito da 1.1; verificar o README renderizado no GitHub
- [x] 1.3 Criar os GitHub Environments `dev` e `prod` em cada repositório implantável, com aprovação obrigatória em `prod`, e o template de PR com o item "Validado localmente"; verificar nas configurações de cada repositório
  - Environments em `infra-base`, `users-service`, `categories-service`, `expenses-service`, `bff` e `frontend`. O `go-commons` fica de fora porque não é implantável.
  - `dev` aceita deploy só da branch `main`. `prod` aceita deploy só de tags `v*.*.*` e exige aprovação de `rrossete`, com autoaprovação permitida porque o desenvolvedor é único.
  - O `.github/pull_request_template.md` foi adicionado por PR nos 7 repositórios.

## 2. Design (Figma)

- [x] 2.1 Criar o arquivo Figma com tokens (cores, tipografia, espaçamento, cores de categoria) e componentes base (Button, Input, Select, Card, ListItem, Badge, TabBar, valor monetário, PieChart); verificar com captura de tela da página de componentes
  - Arquivo: https://www.figma.com/design/4d4L6ZY0vCBjriiN5bPjDx. 37 primitivas, 43 cores semânticas, 13 medidas, 8 estilos de texto (Inter), 2 sombras, 23 ícones e 18 componentes. A pizza está desenhada como rosca na tela Painel.
  - Pendente: modo escuro, porque o plano Starter do Figma permite 1 modo por coleção. As cores semânticas já permitem adicioná-lo como segundo modo.
- [x] 2.2a Desenhar as telas mobile 390 px Login, Login – erro, Cadastro, Confirmar e-mail, Painel, Painel – mês vazio, Gastos, Novo gasto e Categorias; verificado por captura de tela contra as specs
- [ ] 2.2b Desenhar Perfil, Admin › Usuários (+ folha de ações), Editar gasto (+ confirmação de exclusão) e Novo gasto – sem conexão; ajustar a altura de "Painel – mês vazio" para 844 px e a margem do Header para `spacing/xl`; verificar por captura de tela contra as specs
  - Bloqueada pelo limite de chamadas do Figma MCP no plano Starter. O script de Perfil e Admin já está pronto (`figma-perfil-admin.js` no scratchpad da sessão). Precisa estar pronta antes das telas correspondentes nas fatias 1, 3 e 5.

## 3. Ambiente local (`gestor-gastos-infra-base`, pasta `local/`)

- [x] 3.1 Criar o `docker-compose.yml` com DynamoDB Local (8000), dynamodb-admin (8001) e cognito-local (9229). O cognito-local replica o User Pool: e-mail como login, política de senha, grupos `admin` e `cliente` e app client sem secret com `USER_PASSWORD_AUTH`. Criar o Taskfile com `task local:up`, `local:seed`, `local:reset` e `local:down`; verificar que `task local:up` deixa os três containers saudáveis
- [x] 3.2 Criar o seed idempotente: tabelas de cada serviço com as mesmas chaves do CDK, as 12 categorias padrão e os usuários confirmados `admin@local.test` (grupo `admin`) e `cliente@local.test` (grupo `cliente`); verificar no dynamodb-admin e com `aws cognito-idp initiate-auth --endpoint-url http://localhost:9229`
  - O seed é um programa Go em `local/seed`, que só aceita endpoints locais. As chaves das tabelas definidas nele são o contrato que o CDK de cada serviço deve seguir (ver `docs/local.md`, "Tabelas").
  - As cores das categorias padrão ainda precisam ser conferidas com os tokens `category/*` do Figma.
- [x] 3.3 Escrever `docs/local.md` com os pré-requisitos (Docker, Go, Node, go-task), as portas, como subir cada repositório localmente, os usuários de teste, onde ver o código de confirmação e as diferenças entre local e nuvem; verificar seguindo os passos num clone limpo

## 4. Biblioteca compartilhada (`gestor-gastos-go-commons`)

- [x] 4.1 Criar o módulo `github.com/Rossete-Inc/gestor-gastos-go-commons` com os pacotes `auth` (claims, validação de JWT por JWKS configurável), `errs` (erros tipados → problem+json), `logging` (slog JSON sem PII) e `config` (`RUNTIME=local|aws`); testes unitários com cobertura ≥ 80%
- [x] 4.2 Criar o pacote `invoke` (envelope operação + payload + userId) com dois transportes atrás da mesma interface: Lambda Invoke (aws) e HTTP (local); testes dos dois transportes com servidor e client fakes
- [x] 4.3 Criar o pacote `events` com dois publishers: EventBridge (aws) e local, que faz POST aos consumidores de `LOCAL_EVENT_TARGETS`; testes cobrindo fan-out e falha de um consumidor
- [ ] 4.4 Incluir helpers de paginação com cursor assinado por HMAC e de soma em centavos; testes cobrindo cursor adulterado e somas sem perda de precisão
- [ ] 4.5 Criar a esteira só de testes (GitHub Actions, sem AWS) e publicar a tag `v0.1.0`; verificar com `go list -m github.com/Rossete-Inc/gestor-gastos-go-commons@v0.1.0`

## 5. Fatia 1 — Autenticação e perfil (local)

- [ ] 5.1 users-service: domínio de perfil, papéis e máquina de estados do ciclo de vida, store DynamoDB com GSI de listagem, consumidor idempotente de `UserConfirmed` e operação `ObterPerfil`/`CriarPerfil`; testes unitários e de integração com DynamoDB Local
- [ ] 5.2 users-service: `cmd/local` (servidor HTTP chamando o mesmo handler da Lambda); verificar com `curl` no ambiente local
- [ ] 5.3 BFF: escrever o `openapi.yaml` completo (`/me`, `/telas/dashboard`, `/telas/gastos`, `/telas/gasto-form`, `/gastos`, `/categorias`, `/admin/telas/usuarios`, `/admin/usuarios/{id}`) com erros problem+json; verificar com lint do OpenAPI (Spectral) no CI
- [ ] 5.4 BFF: validação do JWT (JWKS do cognito-local ou do Cognito), extração do `userId` e dos grupos, checagem de `admin` em `/admin/*`, `/me` com criação do perfil sob demanda e `cmd/local`; testes cobrindo 401, 403 e perfil ausente
- [ ] 5.5 frontend: criar o projeto Expo + Expo Router (TS strict) com a estrutura `app/features/shared`, ESLint de fronteiras, Prettier, Husky, env validado com Zod e `.env.local` apontando para o BFF e o cognito-local; verificar com `tsc`, `eslint` e `check_boundaries.py` sem erros
- [ ] 5.6 frontend: `shared/ui` com os tokens e componentes do Figma e `shared/api` com tipos gerados pelo `orval`, cliente HTTP com token, TanStack Query e MSW; testes de componentes e teste de que um 401 dispara a renovação de sessão
- [ ] 5.7 frontend: feature `auth` (cadastro, confirmação, login, logout, renovação) e tela Perfil; testes cobrindo os cenários da spec `autenticacao`
- [ ] 5.8 **Validação local pelo usuário:** com `task local:up`, users-service, BFF e `expo start --web`, seguir o roteiro cadastro → confirmação (código no log do cognito-local) → login → Perfil → logout, incluindo senha fraca e login inválido. O usuário aprova antes do grupo 6

## 6. Base na nuvem + deploy da fatia 1 (após 5.8)

- [ ] 6.1 Instalar o AWS CDK e rodar `cdk bootstrap` em us-east-1; verificar a stack `CDKToolkit` no CloudFormation
- [ ] 6.2 infra-base: stack `cicd` com o OIDC provider do GitHub e roles `gh-deploy-<repo>-<env>` com trust `repo:Rossete-Inc/<repo>:environment:<env>`, e stack `budget` com US$ 5 e alertas a 80% do real e 100% do previsto; testes de assertion do CDK e deploy local único
- [ ] 6.3 infra-base: workflows reutilizáveis `go-service.yml` (vet, golangci-lint, test, build arm64, synth/diff no PR, checagem de "Validado localmente", deploy dev na `main`, deploy prod na tag com environment, `concurrency` por ambiente) e `frontend.yml` (tsc, eslint, jest, boundaries, `expo export -p web`, `amplify create-deployment`); verificar com `actionlint` e rodando no próprio infra-base
- [ ] 6.4 infra-base: stacks `auth` (User Pool com SRP, access token de 15 min, grupos), `events` (barramento com DLQ), `api` (HTTP API, authorizer JWT, CORS, throttling), trigger Post Confirmation em Go e parâmetros no SSM; testes de assertion, verificar que uma chamada sem token recebe 401 e `aws ssm get-parameters-by-path`
- [ ] 6.5 users-service e BFF: CDK (tabela on-demand com RETAIN em prod, Lambda arm64 256 MB com alias `live`, CodeDeploy canary em prod, logs 7/30 dias, regra do EventBridge, ARNs no SSM, falha clara quando faltar parâmetro); deploy em `dev` pela esteira e smoke test de `GET /me` com token real
- [ ] 6.6 frontend: Amplify Hosting com rewrite SPA e deploy em `dev`; criar o primeiro admin via CLI e documentar em `docs/`; o usuário repete o roteiro da 5.8 em `dev`

## 7. Fatia 2 — Categorias

- [ ] 7.1 categories-service: domínio (grupos válidos, padrão imutável, nome único por grupo ignorando caixa, visibilidade por usuário) com testes cobrindo os cenários da spec `categorias`
- [ ] 7.2 categories-service: store DynamoDB (`PK=SYSTEM|USER#id`, `SK=CAT#grupo#id`), seed idempotente das categorias padrão, handler (ListarCategorias, ObterCategoria, CriarCategoria, ExcluirCategoria), consumidor de `UserDeleted` e `cmd/local`; testes com DynamoDB Local
- [ ] 7.3 BFF: `/categorias` (GET, POST, DELETE) e `/telas/gasto-form`; testes com clients mockados
- [ ] 7.4 frontend: feature `categorias` (lista por grupo, criar, excluir personalizada com confirmação); testes com MSW
- [ ] 7.5 **Validação local pelo usuário:** listar padrão e personalizadas, criar categoria, tentar duplicar o nome, excluir. O usuário aprova
- [ ] 7.6 Deploy em `dev`: CDK do categories-service no padrão da 6.5, BFF e frontend pela esteira; smoke test

## 8. Fatia 3 — Lançamento de gastos

- [ ] 8.1 expenses-service: domínio (validações de valor, data, descrição e recorrência; grupo desnormalizado) com testes cobrindo os cenários da spec `lancamento-gastos`
- [ ] 8.2 expenses-service: store DynamoDB (`PK=USER#id`, `SK=EXP#AAAA-MM-DD#ulid`) com CRUD, listagem paginada, filtros por grupo e categoria, `ExisteGastoNaCategoria`, consumidor de `UserDeleted` (lotes, idempotente) e `cmd/local`; testes com DynamoDB Local, incluindo o isolamento entre usuários (404)
- [ ] 8.3 BFF: `/gastos` (POST, PUT, DELETE) com validação de categoria, `/telas/gastos`, e bloqueio de exclusão de categoria em uso; testes cobrindo os cenários das specs
- [ ] 8.4 frontend: feature `lancamento-gastos` (formulário com máscara de moeda, lista com filtros e paginação, edição, exclusão com confirmação, invalidação de queries); testes com MSW
- [ ] 8.5 **Validação local pelo usuário:** lançar gasto de cartão e fixo, filtrar, paginar, editar, excluir e tentar excluir uma categoria em uso. O usuário aprova
- [ ] 8.6 Deploy em `dev`: CDK do expenses-service no padrão da 6.5, BFF e frontend pela esteira; smoke test

## 9. Fatia 4 — Painel

- [ ] 9.1 expenses-service: `ResumoMensal` (totais por grupo e por categoria, 5 mais recentes); testes com somas em centavos e mês vazio
- [ ] 9.2 BFF: `/telas/dashboard` com chamadas paralelas e percentuais pelo maior resto; testes cobrindo a soma de 100% e o mês vazio
- [ ] 9.3 frontend: aceitar o ADR 0005 e criar a feature `dashboard` (seletor de mês, totais, pizza com alternância todas/CARTAO/FIXO, últimos 5, estado vazio); testes com MSW
- [ ] 9.4 **Validação local pelo usuário:** conferir totais e pizza contra os gastos lançados, trocar de mês e ver o mês vazio. O usuário aprova
- [ ] 9.5 Deploy em `dev` do expenses-service, BFF e frontend; smoke test

## 10. Fatia 5 — Administração

- [ ] 10.1 users-service: regras do último admin e de não bloquear a si mesmo, operações no Cognito (disable/enable + global sign-out, troca de grupo, delete) atrás de uma interface, com adaptador fake para o que o cognito-local não suportar, e publicação de `UserDeleted`; testes com client Cognito mockado
- [ ] 10.2 BFF: `/admin/telas/usuarios` e `/admin/usuarios/{id}` (PATCH, DELETE); testes cobrindo 403 para cliente e os cenários da spec `gestao-usuarios`
- [ ] 10.3 frontend: feature `admin` (lista paginada, bloquear/reativar, alterar papel, excluir), com rota visível só para admin; testes cobrindo o cliente sem acesso
- [ ] 10.4 **Validação local pelo usuário:** bloquear → login recusado → reativar → login aceito; promover a admin; excluir usuário e ver os dados dele removidos. O usuário aprova
- [ ] 10.5 Deploy em `dev` com as permissões Cognito Admin restritas ao pool; repetir o roteiro da 10.4 em `dev`, já que o Cognito real substitui o adaptador fake

## 11. PWA

- [ ] 11.1 frontend: manifest, ícones, service worker (shell offline, aviso de offline, atualização de versão) e headers de cache; verificar localmente com `expo export -p web` servido localmente, Lighthouse PWA e acesso pelo celular na rede local
- [ ] 11.2 **Validação local pelo usuário:** instalar pelo celular na rede local, abrir offline e ver o aviso. O usuário aprova
- [ ] 11.3 Deploy em `dev` e instalação real em Android e iOS

## 12. Integração ponta a ponta e produção

- [ ] 12.1 E2E Playwright rodando no ambiente local e em `dev`: cadastro → confirmação → login → lançar gasto → ver no painel e na pizza → editar → excluir; o teste passa na esteira do frontend
- [ ] 12.2 E2E admin em `dev`: admin bloqueia usuário → login recusado → reativa → login aceito; admin exclui usuário → dados removidos; o teste passa
- [ ] 12.3 Promover todos os repositórios para `prod` via tag `v1.0.0` com aprovação; verificar o smoke test em prod e o canary concluído
- [ ] 12.4 Após 7 dias, conferir no Cost Explorer que o custo está dentro do orçamento; registrar o resultado em `docs/`

## Workflow follow-up

- Aceitar o ADR 0005 (biblioteca de gráficos) e movê-lo para o repositório do frontend antes da tarefa 9.3.
- Registrar a decisão de execução local e entrega em fatias num ADR 0008 e atualizar `docs/plano.md` (fases) e `docs/diagramas.md` (CI/CD).
- Arquivar a change com `/opsx:archive` após a revisão, sincronizando as specs em `openspec/specs/`.
