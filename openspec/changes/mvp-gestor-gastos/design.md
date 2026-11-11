# Design

## Context

Projeto greenfield (ver proposal.md, seção Why). As decisões de arquitetura já estão registradas e esta change as segue sem reabri-las:

- `docs/plano.md`: arquitetura, modelo de dados, endpoints do BFF, telas e fases.
- `docs/diagramas.md`: arquitetura, sequências de cada fluxo, ciclo de vida, navegação, ER e CI/CD.
- ADRs 0001 (Go), 0002 (DynamoDB), 0003 (microserviços + BFF), 0004 (Lambda Invoke), 0005 (gráficos, proposta), 0006 (repositórios separados), 0007 (CI/CD).
- Front segue a skill `app-multiplataforma-guardrails` (Expo + RN + Expo Router, TanStack Query, Zod, `app → features → shared`).

Esta change mora no `gestor-gastos-infra-base`, mas o trabalho se espalha pelos sete repositórios da org Rossete-Inc. Cada grupo de tarefas indica o repositório.

## Goals / Non-Goals

**Goals:**
- Atender a todas as specs desta change com custo AWS próximo de zero.
- Cada repositório implantável de forma independente, pela esteira, desde a primeira tarefa.
- Contrato do BFF (`openapi.yaml`) definido antes do front consumi-lo.

**Non-Goals:**
- Geração automática mensal de gastos fixos recorrentes. No MVP o flag `recorrente` é só informativo; a geração fica para uma change futura com EventBridge Scheduler.
- Apps nativos nas lojas (Maestro, builds EAS).
- Importação de fatura de cartão, metas, orçamento por categoria, multi-moeda e compartilhamento entre usuários.
- Contas AWS separadas por ambiente.

## Decisions

### Mapeamento capability → componentes

| Capability | Componentes |
|---|---|
| `autenticacao` | Cognito (infra-base), authorizer JWT da HTTP API, `features/auth` no front |
| `gestao-usuarios` | users-service, rotas `/me` e `/admin/*` do BFF, `features/admin` |
| `categorias` | categories-service, `/telas/gasto-form` e `/categorias` |
| `lancamento-gastos` | expenses-service, `/gastos` e `/telas/gastos` |
| `painel-gastos` | operação `ResumoMensal` do expenses-service + `/telas/dashboard` do BFF + `features/dashboard` |
| `aplicativo-pwa` | frontend (manifest, service worker, Amplify Hosting) |
| `entrega-continua` | stack `cicd` + workflows reutilizáveis no infra-base, chamados por cada repositório |
| `controle-custos` | stack `budget` no infra-base, retenção de logs em cada stack |

### Ativação da conta por evento
O trigger *Post Confirmation* (infra-base) adiciona o usuário ao grupo `cliente` e publica `UserConfirmed` no EventBridge. O users-service consome o evento e cria o perfil. Assim o infra-base não depende de um serviço implantado depois dele. A alternativa, chamar o users-service direto, cria dependência circular no primeiro deploy.

### Status da conta: Cognito é a fonte da verdade do acesso
Login e bloqueio valem no Cognito (`AdminDisableUser`, `AdminUserGlobalSignOut`). A tabela do users-service espelha o status para listagem e regras, como "último admin". Não há checagem de status a cada requisição. O bloqueio atua pela revogação dos refresh tokens, e o access token restante expira em até 15 minutos (validade configurada curta para isso). Alternativa descartada: consultar o users-service em toda requisição, o que dobraria a latência e o custo.

### Regras que atravessam serviços ficam no BFF
- **Excluir categoria em uso:** o BFF consulta o expenses-service (`ExisteGastoNaCategoria`) antes de pedir a exclusão ao categories-service. A corrida entre consultar e excluir é aceita no MVP, porque um gasto órfão continua exibido com a categoria "Removida".
- **Lançar gasto:** o BFF valida a categoria no categories-service e envia o `grupo` ao expenses-service, que o grava desnormalizado. Isso permite filtrar e agregar sem join.
- **Isolamento:** o BFF sempre repassa o `userId` extraído do token (claim `sub`) e nunca o recebe do corpo da requisição. Cada serviço filtra pela partition key `USER#<userId>`, então um gasto alheio simplesmente não é encontrado (404).

### Agregação do painel
`ResumoMensal` faz uma única Query por `PK=USER#id` e `SK begins_with EXP#AAAA-MM` e soma os valores em memória, devolvendo totais por grupo, totais por categoria e os 5 mais recentes. O BFF junta nome e cor das categorias e calcula os percentuais pelo método do maior resto, para que somem 100%. O volume pessoal (dezenas a centenas de itens por mês) cabe numa Query. O gatilho para materializar os totais via Streams está no ADR 0002.

### Paginação
A listagem usa cursor opaco (o `LastEvaluatedKey` codificado em base64url e assinado com HMAC, para o cliente não forjar chaves), com 20 itens por página.

### Contrato e tipos
O `openapi.yaml` (BFF) é escrito primeiro e validado com lint no CI. O front gera tipos com `orval` a partir de uma release fixada, e as respostas são validadas com Zod. Os erros seguem `application/problem+json` (RFC 9457), com `errors[]` por campo nos 422.

### Valores e datas
- **Valor:** `valorCentavos` int64 em toda a cadeia. O front converte na borda, na máscara do campo e na formatação `Intl` pt-BR.
- **Datas:** `AAAA-MM-DD` sem fuso. O mês corrente é calculado no fuso do dispositivo.

### Repositórios públicos e região
- **Repositórios:** os sete repositórios da Rossete-Inc são **públicos**. A org está no plano GitHub Free, que em repositório privado não oferece proteção de branch nem Environments, e a aprovação obrigatória (required reviewers) em repositório privado só existe no Enterprise. Como públicos, eles têm proteção da `main`, Environments e trust OIDC por environment sem custo. Também dispensam o `GOPRIVATE` e o token de leitura para baixar a `go-commons`, e os minutos do Actions são ilimitados. Nenhum segredo fica no código: o deploy usa OIDC, e a configuração fica no SSM. Decisão tomada em 2026-11-05.
- **Ambiente único:** a AWS tem um só ambiente, `dev`, implantado a cada merge na `main` (Environment `dev`, restrito à `main`, sem aprovação manual). Tags `vX.Y.Z` não fazem deploy; a `go-commons` continua publicando tags semver como biblioteca. O motivo é custo: é um projeto de estudo. Um segundo ambiente fica para uma decisão futura (ADR 0007). Decisão tomada em 2026-11-10.
- **Região:** toda a infraestrutura fica em **us-east-1**. É a região mais barata e com todos os serviços. A latência a partir do Brasil, de cerca de 120 ms, é aceitável para o MVP. `sa-east-1` custaria de 30% a 50% mais.

### Execução local e entrega em fatias
Cada funcionalidade roda e é validada localmente pelo responsável antes de subir para a AWS. A entrega é organizada em **fatias verticais**. Cada fatia junta serviço + rotas do BFF + telas do front e só vai para `dev` depois da validação local.

> **Atualização de 2026-11-11 (decisão do usuário):** a validação do usuário passa a ser feita em `dev`, depois do merge e do deploy pela esteira. O ambiente local continua sendo onde se desenvolve e se testa, mas deixou de ser portão para a nuvem. Saiu da esteira o job "Validado localmente" e, do template de PR, o item correspondente. Trade-off: um defeito que o roteiro do usuário pegaria localmente agora chega a `dev` antes; com um ambiente só e sem usuários reais, o risco é baixo, e o rollback é o próprio canary das Lambdas ou um novo merge.

| Na nuvem | Localmente |
|---|---|
| DynamoDB | DynamoDB Local (Docker), com o dynamodb-admin para inspecionar |
| Cognito (User Pool, grupos, JWT) | cognito-local (Docker), com a mesma política de senha, os mesmos grupos e o mesmo app client |
| Lambda + API Gateway | `cmd/local` em cada serviço e no BFF: servidor HTTP que chama o **mesmo** handler |
| Lambda Invoke (BFF → serviço) | Transporte HTTP do pacote `invoke` da `go-commons`, escolhido por `RUNTIME=local` |
| EventBridge | Publisher local da `go-commons`, que faz POST direto aos consumidores listados em `LOCAL_EVENT_TARGETS` |
| Trigger Post Confirmation | Não existe localmente. O seed coloca os usuários de teste nos grupos, e o `/me` cria o perfil sob demanda |
| Amplify Hosting | `expo start --web`, com `.env.local` apontando para o BFF e o cognito-local |

- **Comandos:** um Taskfile (go-task, que funciona em Windows, macOS e Linux) no infra-base oferece `task local:up`, `local:seed`, `local:reset` e `local:down`.
- **Seed:** cria as tabelas com as mesmas chaves do CDK, as 12 categorias padrão e dois usuários confirmados, `admin@local.test` e `cliente@local.test`.
- **Diferenças aceitas em relação à nuvem:**
  - O login local usa o fluxo `USER_PASSWORD_AUTH`, porque o cognito-local não implementa SRP. Na nuvem continua SRP.
  - O código de confirmação de e-mail aparece no log do cognito-local.
  - As operações de admin que o emulador não suportar usam um adaptador fake atrás da mesma interface.
- **Portão para a nuvem:**
  - ~~O template de pull request tem o item "Validado localmente". Um job da esteira reprova o PR de funcionalidade sem esse item marcado.~~ Removido em 2026-11-11 (ver a atualização acima).
  - O merge na `main` continua implantando em `dev` automaticamente.
- **Base na nuvem:** é montada uma única vez, logo depois que a fatia 1 é aprovada localmente. Fazem parte dela o bootstrap, o OIDC, o Budget, o Cognito, a API e o barramento.
- **Alternativa descartada:** LocalStack, porque Cognito e outros serviços usados só existem na versão paga, e um ambiente local ainda mais fiel não compensa o custo.

### Retenção de logs
Os logs ficam 7 dias em `dev`, o único ambiente na nuvem. A versão anterior desta change previa 30 dias para `prod`, que deixou de existir em 2026-11-10.

## Risks / Trade-offs

- [Bloqueio não é instantâneo para o access token já emitido] → validade do access token de 15 minutos, mais revogação global dos refresh tokens.
- [Sete repositórios aumentam o custo de mudanças transversais] → workflows reutilizáveis versionados, contrato versionado e tarefas ordenadas por dependência.
- [Cold start em rotas pouco usadas] → Go em arm64 com 256 MB. Provisioned concurrency só se o p95 incomodar, pois tem custo.
- [Evento `UserConfirmed` perdido deixaria usuário sem perfil] → DLQ no target do EventBridge, e o BFF cria o perfil sob demanda no `/me` se ele não existir (operação idempotente).
- [Exclusão de usuário assíncrona] → consumidores idempotentes com DLQ e alarme na DLQ.
- [O emulador local diverge do Cognito real (SRP, triggers, operações de admin)] → mesma interface com adaptadores trocáveis, smoke test em `dev` a cada fatia e E2E em `dev` ao fim do MVP.
- [Manter dois transportes (HTTP local e Lambda)] → o transporte fica isolado na `go-commons` e coberto por testes, e o handler é único.
- [Código público expõe a arquitetura e os nomes dos recursos] → nenhum segredo em código ou workflow, OIDC com trust restrita a `repo:Rossete-Inc/<repo>:environment:dev` no deploy e a `repo:Rossete-Inc/<repo>:pull_request` na role só leitura do `cdk diff`, `.env.local` no `.gitignore` e IDs de recursos lidos do SSM.

## Migration Plan

Não há dados nem sistema anterior. Primeira implantação do ambiente `dev`:

0. Ambiente local (`task local:up`) e fatia 1 validada localmente pelo responsável.
1. Bootstrap manual: `cdk bootstrap` e deploy local da stack `cicd`.
2. Merge na `main` do infra-base (auth, events, api, budget) e tag da go-commons. Em seguida vem o deploy da fatia 1: users-service → BFF → frontend.
3. Criação do primeiro admin via CLI (`AdminCreateUser` + grupo `admin`).
4. Cada fatia seguinte passa por validação local e merge, que implanta em `dev`. A ordem é categories → expenses (gastos e painel) → admin no users-service, e em cada uma o BFF e o frontend são implantados junto.

Rollback: canary com rollback automático nas Lambdas. Para infraestrutura, reverter o commit gera um novo deploy pela esteira. As tabelas DynamoDB usam `RemovalPolicy.RETAIN`.

## Open Questions

- Domínio próprio para o app e para a API, ou as URLs padrão do Amplify e da API Gateway? Não muda specs nem tarefas, só uma configuração adicional.
- E-mails de verificação pelo envio padrão do Cognito (limite de ~50/dia) ou pelo SES? Pode ficar com o padrão no MVP e migrar depois.
