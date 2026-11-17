# Proposal

## Why

Hoje "cartão" é só um grupo de categorias (`CARTAO`). Quem tem mais de um cartão não consegue separar os gastos de cada um, conferir o valor de cada fatura nem acompanhar quanto do limite já usou. O total de gastos também fica desalinhado com o que sai do bolso no mês: uma compra feita depois do fechamento só é paga na fatura seguinte.

## What Changes

- **Cadastro de cartões.** O cliente cadastra vários cartões de crédito, informando apelido, cor, dia de fechamento, dia de vencimento e limite. Também pode editar, arquivar e excluir um cartão. A exclusão só é permitida se o cartão não tiver gastos.
- **Cartão no lançamento.**
  - Todo gasto de categoria `CARTAO` passa a exigir um cartão ativo.
  - Um gasto `FIXO` pode, opcionalmente, ser pago num cartão (ex.: streaming).
- **Mês de referência.** O gasto feito em cartão entra no mês da fatura, nomeada pelo mês do vencimento. O gasto sem cartão continua no mês da data.
  - **BREAKING (comportamento):** a lista, os totais e o painel passam a agrupar pelo mês de referência, e não mais pela data da compra.
- **Gastos antigos.** Os gastos `CARTAO` já lançados aparecem como "Sem cartão" até o cliente editar e escolher um cartão.
- **Aba "Cartões".** Nova aba com a lista de cartões, mostrando a fatura do mês e o limite usado e disponível de cada um. O detalhe de um cartão mostra os gastos dele por mês.
- **Painel e lista.** O painel ganha os totais por cartão e uma visão da pizza por cartão. A lista de gastos ganha o filtro por cartão.
- **Detalhe da fatia.** Ao passar o mouse (navegador) ou tocar (celular) numa fatia da pizza, o painel mostra o nome da fatia, o grupo, o valor, o percentual e a divisão por cartão.

## Capabilities

### New Capabilities

- `cartoes-credito`: cadastro e ciclo de vida de cartões, regra de fatura (mês de referência), limite disponível e visão por cartão.

### Modified Capabilities

- `lancamento-gastos`: o gasto ganha o cartão (obrigatório para `CARTAO`) e o mês de referência; a listagem ganha o filtro por cartão e passa a ser feita pelo mês de referência.
- `painel-gastos`: totais por cartão, a visão "cartões" na pizza, o detalhe da fatia (hover/toque) e a agregação pelo mês de referência.

## Impact

- **`gestor-gastos-expenses-service`:**
  - Entidade cartão na mesma tabela.
  - `mesReferencia` no gasto, com nova sort key e script de migração dos gastos existentes.
  - Novas operações no `contract`.
- **`gestor-gastos-bff`:** rotas `/cartoes`, `/cartoes/{id}` e `/telas/cartoes`; `cartaoId` em `/gastos`, `/telas/gastos` e `/telas/dashboard`; `openapi.yaml` atualizado.
- **`gestor-gastos-frontend`:** feature `cartoes` (nova aba), seletor de cartão no formulário de gasto, filtro na lista e bloco no painel.
- **`gestor-gastos-infra-base`:** ADR 0009 (cartões no expenses-service).
- **Custo:** sem recursos AWS novos.
