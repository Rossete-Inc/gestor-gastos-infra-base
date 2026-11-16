# Tasks

> Cada grupo indica o repositório da org Rossete-Inc. Começa só depois da change `cartoes-credito` estar em `dev`. Fluxo: PR → esteira verde → merge → deploy automático em `dev` → validação do usuário em `dev`.

## 1. expenses-service — séries

- [ ] 1.1 Criar `internal/serie`:
  - entidade e validações (2–72 parcelas, parcela atual ≤ total, mês final ≥ mês inicial);
  - divisão pelo total (resto na 1ª parcela) e cálculo das datas e dos meses de referência das parcelas;
  - data da ocorrência mensal, com o dia limitado ao fim do mês.

  Verificar com testes de tabela cobrindo os cenários "Divisão exata", "Divisão com resto", "Compra em andamento", "Informar mês final" e "Dia 31".
- [ ] 1.2 Store: item `SERIE#` com `versao`; gasto com `serieId`, `numero`, `ajustado` e `excluido`; `CriarParcelamento` numa transação com `Idempotency-Key`; `PutItem` condicional das ocorrências com SK determinística. Verificar com testes no DynamoDB Local: 72 parcelas numa transação, repetir a criação não duplica e criar a ocorrência duas vezes não duplica.
- [ ] 1.3 App: criar as ocorrências recorrentes antes de `ListarGastos`, `ResumoMensal` e `ResumoCartoes` (só até o mês corrente). Ignorar itens com `excluido` nas somas e na lista. No limite, contar as parcelas futuras e deixar de fora as recorrências futuras. Verificar com testes cobrindo "Painel sem abrir a lista", "Mês final" e "Séries no limite do cartão".
- [ ] 1.4 App: `escopo` (`ESTA`, `PROXIMAS`, `TODAS`) em `AtualizarGasto` e `ExcluirGasto`, com divisão da série na recorrência, preservação de itens `ajustado` e recálculo de SK quando mudam a data ou o cartão. Verificar com testes cobrindo todos os cenários da spec "Alcance da edição e da exclusão em séries".
- [ ] 1.5 Contrato: `parcelamento`, `recorrencia` e `serie` no `GastoIn`/`Gasto`, e `escopo` nas operações. Atualizar o README. Fazer merge e deploy em `dev`. Verificar com o smoke test verde.

## 2. BFF

- [ ] 2.1 Atualizar o contrato:
  - `openapi.yaml`: `parcelamento`, `recorrencia`, `serie`, o parâmetro `escopo` e o cabeçalho `Idempotency-Key`;
  - `Rotas`, se mudarem.

  Verificar com o lint do OpenAPI e com `TestRotasIguaisAoContrato`.
- [ ] 2.2 Implementar as validações e o repasse em `internal/api/gastos.go`: 422 quando falta `escopo` num item de série, e os limites de parcelas. Verificar com testes cobrindo os cenários das specs `parcelamentos-recorrencias` e `lancamento-gastos`.
- [ ] 2.3 Merge e deploy em `dev`; verificar com o smoke test verde.

## 3. frontend

- [ ] 3.1 Atualizar o contrato fixado e rodar `npm run api:gerar`. Criar `shared/lib/parcelas.ts` com os mesmos casos de tabela do Go. Verificar com Jest.
- [ ] 3.2 `GastoForm`:
  - modo À vista / Parcelado / Recorrente;
  - campos dos dois modos de parcelamento;
  - prévia das parcelas;
  - `Idempotency-Key` por envio.

  Verificar com testes MSW cobrindo a criação nos dois modos e a recorrência.
- [ ] 3.3 Implementar o `EscopoSerieDialog` na edição e na exclusão e o selo "n/N"/"recorrente" no `ExpenseItem`. Verificar com testes MSW cobrindo "só esta", "esta e as próximas" e "todas", e com `npm run verify`.
- [ ] 3.4 E2E `e2e/parcelas.spec.ts`:
  1. parcelar R$ 300,00 em 3× num cartão;
  2. ver 1/3, 2/3 e 3/3 nos meses seguintes;
  3. excluir 2/3 com "esta e as próximas";
  4. lançar uma recorrência;
  5. abrir o mês seguinte e ver a ocorrência.

  Verificar que o teste passa em `dev` na esteira.
- [ ] 3.5 Merge e deploy em `dev`; verificar com a esteira verde.

## 4. Validação

- [ ] 4.1 **Validação em `dev` pelo usuário:**
  - lançar um parcelado pelo total e um em andamento (ex.: 3/10);
  - conferir as faturas e o limite;
  - criar o aluguel recorrente e ajustar a luz só de um mês;
  - reajustar "esta e as próximas";
  - encerrar uma recorrência e quitar um parcelamento antecipado.

  O usuário aprova.

## Workflow follow-up

- Arquivar a change com `/opsx:archive parcelamentos-recorrencias` após a validação 4.1, depois de `cartoes-credito` estar arquivada.
