# Proposal

## Why

Hoje não existe um lugar simples para registrar e acompanhar gastos de cartão e gastos fixos (aluguel, luz, internet…) com visão por categoria. O MVP entrega esse fluxo mínimo: criar conta, lançar gastos e ver para onde o dinheiro vai, com um admin que gerencia os clientes. Tudo isso a custo de infraestrutura próximo de zero.

## What Changes

- Cadastro de usuário com confirmação de e-mail, login, sessão e logout.
- Papéis `cliente` e `admin`. O admin lista, bloqueia, reativa, altera o papel e exclui usuários.
- Categorias padrão do sistema, divididas em dois grupos: `CARTAO` (alimentação, transporte, lazer, saúde, compras, outros) e `FIXO` (aluguel, luz, água, internet, telefone, condomínio). O usuário também pode criar categorias personalizadas.
- Lançamento, edição, exclusão e listagem de gastos com filtros por mês, grupo e categoria. Cada usuário só acessa os próprios dados.
- Painel mensal com total, total por grupo, gráfico de pizza por categoria e últimos lançamentos.
- App PWA instalável em Android e iOS.
- Ambiente local completo (Docker) para executar e validar cada funcionalidade na máquina do desenvolvedor antes de implantá-la na AWS. A entrega é feita em fatias verticais (Autenticação e perfil → Categorias → Gastos → Painel → Admin), e cada fatia só sobe para `dev` depois de validada localmente.
- Esteira de entrega contínua: todo deploy na AWS é automático, sem credenciais fixas, com aprovação e rollback em produção.
- Alerta de custo da conta AWS.

## Capabilities

### New Capabilities

- `autenticacao`: cadastro, confirmação de e-mail, login, renovação de sessão e logout.
- `gestao-usuarios`: perfil do usuário e administração de usuários pelo admin (listar, bloquear, reativar, alterar papel, excluir) e ciclo de vida da conta.
- `categorias`: categorias padrão por grupo e categorias personalizadas do usuário.
- `lancamento-gastos`: CRUD e listagem filtrada de gastos, com isolamento por usuário.
- `painel-gastos`: resumo mensal e agrupamentos para gráficos.
- `aplicativo-pwa`: instalação e execução do app como PWA em Android, iOS e navegador.
- `entrega-continua`: execução e validação local antes da nuvem e comportamento da esteira de CI/CD (validações, deploy automático por ambiente, aprovação, rollback, autenticação sem segredo fixo).
- `controle-custos`: alerta de orçamento da conta AWS.

### Modified Capabilities

Nenhuma. É a primeira change do projeto.

## Impact

- **Repositórios novos na org Rossete-Inc:** `gestor-gastos-go-commons`, `-users-service`, `-categories-service`, `-expenses-service`, `-bff`, `-frontend`. Este repositório (`gestor-gastos-infra-base`) recebe a infra compartilhada e os workflows reutilizáveis.
- **AWS:** Cognito, API Gateway HTTP API, Lambda, DynamoDB, EventBridge, SSM Parameter Store, CodeDeploy, Amplify Hosting, AWS Budgets e IAM (OIDC do GitHub).
- **Ambiente local:** Docker Compose no `gestor-gastos-infra-base` (DynamoDB Local, dynamodb-admin, cognito-local), seed de dados e Taskfile. Cada serviço Go e o BFF ganham um `cmd/local` (servidor HTTP), e o front ganha um `.env.local` apontando para o BFF local.
- **Contrato público:** API do BFF descrita em `openapi.yaml` (repositório `gestor-gastos-bff`).
- **Design:** arquivo novo no Figma com design system e telas.
- **Referências:** `docs/plano.md`, `docs/diagramas.md` e ADRs 0001–0007.
