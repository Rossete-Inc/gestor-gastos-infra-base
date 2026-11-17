# Tasks

> Cada grupo indica o repositório da org Rossete-Inc. Fluxo: PR → esteira verde → merge → deploy automático em `dev` → validação do usuário em `dev`. Mesclar o consumidor só depois de o `cdk deploy` do provedor terminar.

## 1. infra-base

- [x] 1.1 Escrever o ADR 0009 (cartões no expenses-service, exceção ao ADR 0003, gatilho para extrair) em `docs/adr/` e referenciá-lo em `docs/plano.md` e no modelo de dados de `docs/diagramas.md`; verificar a renderização no PR

## 2. expenses-service — domínio e armazenamento

- [x] 2.1 Criar `internal/cartao` (entidade, validações, normalização do apelido e `MesDaFatura`); verificar com testes de tabela cobrindo os cenários da spec "Mês de referência da fatura" (antes e depois do fechamento, vencimento no mesmo mês e dia 31 em fevereiro)
- [x] 2.2 Adicionar `cartaoId` e `mesReferencia` ao gasto, a nova SK `EXP#<mesRef>#<data>#<ulid>` e o novo `IDDe`, com leitura dupla dos dois formatos em `Listar`, `DoMes` e `Obter`; verificar com testes de store no DynamoDB Local misturando itens antigos e novos
- [x] 2.3 Store de cartões (`CARD#`): CRUD, unicidade do apelido entre os ativos, máximo de 20, arquivar e reativar, `ExisteGastoNoCartao`, e restringir `ExisteNaCategoria` a `EXP#`; incluir no `memstore` e verificar com testes de store e de isolamento entre usuários (404)
- [x] 2.4 App e contrato: `cartaoId` no `CriarGasto`/`AtualizarGasto` (valida o cartão ativo e calcula o `mesReferencia`), filtro `cartaoId`/`SEM_CARTAO` no `ListarGastos`, `totaisPorCartao` no `ResumoMensal`, operações de cartão e `ResumoCartoes(mes, hoje)`; verificar com testes de app cobrindo o limite usado (fatura vencida não conta, parcelas futuras contam) e a exclusão de cartão com gastos (conflito)
- [x] 2.5 Script `scripts/migrar-mes-referencia` (idempotente, com `--dry-run`); verificar rodando duas vezes contra o DynamoDB Local: a segunda execução não altera nada
- [x] 2.6 Atualizar o README (operações e migração), fazer o merge e o deploy em `dev`, rodar a migração em `dev` e conferir a contagem de itens antes e depois; verificar o smoke test verde

## 3. BFF

- [x] 3.1 Atualizar o `openapi.yaml`: schemas `Cartao` e `CartaoIn`; rotas `/cartoes`, `/cartoes/{id}` (PUT, PATCH, DELETE) e `/telas/cartoes`; `cartaoId` em `GastoIn`, `Gasto`, `/telas/gastos` e `/telas/gasto-form`; `totaisPorCartao` e a visão `cartoes` no `TelaDashboard`. Atualizar `Rotas` em `infra/config.go`. Verificar com o lint do OpenAPI e com `TestRotasIguaisAoContrato`
- [x] 3.2 Implementar as rotas e a regra "categoria CARTAO exige cartão" (422) em `internal/api`, além da fatia por cartão no dashboard, com o percentual pelo maior resto, e o `hoje` em America/Sao_Paulo; verificar com testes cobrindo os cenários das specs `cartoes-credito`, `lancamento-gastos` e `painel-gastos`
- [x] 3.3 Fazer o merge e o deploy em `dev`; verificar o smoke test verde

## 4. frontend

- [x] 4.1 Copiar o contrato fixado para `openapi/bff.yaml` e rodar `npm run api:gerar`; criar `shared/lib/fatura.ts` com os mesmos casos de tabela do Go; verificar com Jest
- [x] 4.2 Criar a feature `cartoes` (lista com fatura, limite usado e disponível; formulário; detalhe com `MonthSelector`; arquivar, reativar e excluir com confirmação; estado vazio) e a aba "Cartões" em `(tabs)/_layout.tsx` e `TabBar.tsx`; verificar com testes MSW cobrindo os cenários da spec `cartoes-credito`
- [x] 4.3 `GastoForm`: seletor de cartão obrigatório para CARTAO e opcional para FIXO, e o texto "Entra na fatura de …"; `GastosView`: filtro de cartão, incluindo "Sem cartão", e o apelido do cartão no item; `PainelView`: bloco "Por cartão" e a pizza "Cartões"; verificar com testes MSW e com `npm run verify`
- [x] 4.3a Detalhe da fatia: `PieChart` com seleção por toque e hover (web) e o detalhe (nome, grupo, valor, % e divisão por cartão ou grupo) num balão que sai da fatia, com a fatia saltando e as outras esmaecidas; verificar com testes cobrindo os cenários de "Detalhe da fatia do gráfico" e "Balão e destaque da fatia selecionada"
- [x] 4.4 E2E `e2e/cartoes.spec.ts`: cadastrar cartão (fechamento 25, vencimento 5) → lançar compra no dia 28 do mês corrente → vê-la na fatura do mês+2 na aba Cartões e no painel → excluir o cartão é recusado → arquivar; verificar que passa em `dev` na esteira
- [x] 4.5 Fazer o merge e o deploy em `dev`; verificar a esteira verde

## 5. Validação e limpeza

- [x] 5.1 **Validação em `dev` pelo usuário:** cadastrar dois cartões, lançar compras antes e depois do fechamento, conferir a fatura, o limite e os totais por cartão no painel, atribuir um cartão a um gasto antigo "Sem cartão" e arquivar um cartão. O usuário aprova
- [x] 5.2 expenses-service: remover a leitura do formato antigo de SK após a validação; verificar com os testes de store e com o smoke test em `dev`

## Workflow follow-up

- Arquivar a change com `/opsx:archive cartoes-credito` após a validação 5.1.
