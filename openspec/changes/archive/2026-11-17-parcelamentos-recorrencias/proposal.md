# Proposal

## Why

Grande parte dos gastos se repete: aluguel, luz e assinaturas todo mês, e compras parceladas no cartão por vários meses. Hoje o cliente precisa lançar cada mês à mão, e o flag `recorrente` é só um rótulo. Com isso, os meses futuros e o limite do cartão ficam incompletos.

## What Changes

- **Parcelamento.** Há dois modos:
  - **Pelo total:** o cliente informa o valor total e o número de parcelas (de 2 a 72). O sistema divide o valor e lança uma parcela por fatura (ou por mês, sem cartão). Os centavos que sobram na divisão vão para a 1ª parcela.
  - **Pela parcela:** o cliente informa o valor da parcela, a parcela atual e o mês final. É o caso de uma compra já em andamento, por exemplo na 3ª de 10. O sistema lança da parcela atual até a última.
- **Identificação.** Cada parcela aparece como "Descrição 3/10".
- **Recorrência.** O gasto marcado como recorrente se repete todo mês, no mesmo dia, até o cliente encerrar ou até um mês final opcional.
  - As ocorrências aparecem sozinhas em cada mês.
  - O valor de um mês pode ser ajustado sem mudar os demais (ex.: a conta de luz).
- **Alcance das alterações.** Ao editar ou excluir um item de um parcelamento ou de uma recorrência, o cliente escolhe o alcance: **só esta**, **esta e as próximas** ou **todas**. Encerrar uma recorrência é excluir "esta e as próximas".
- **Limite do cartão.** Todas as parcelas futuras contam no limite. As recorrências contam só até a fatura atual.
- **BREAKING (comportamento):** o switch "Gasto recorrente" deixa de ser só informativo. Os gastos antigos com `recorrente=true` continuam como rótulo e não geram ocorrências.

## Capabilities

### New Capabilities

- `parcelamentos-recorrencias`: parcelamento nos dois modos, recorrência mensal, ajuste de uma ocorrência e alcance de edição e exclusão.

### Modified Capabilities

- `lancamento-gastos`: o lançamento passa a aceitar parcelamento e recorrência; edição e exclusão ganham o alcance para itens de série.

## Impact

- **`gestor-gastos-expenses-service`:** entidade série (`SERIE#`), parcelas criadas na hora, ocorrências recorrentes criadas na leitura do mês e operações com alcance.
- **`gestor-gastos-bff`:** `GastoIn` ganha `parcelamento` e `recorrencia`; `PUT` e `DELETE /gastos/{id}` ganham o parâmetro `escopo`; `openapi.yaml` atualizado.
- **`gestor-gastos-frontend`:** modo "À vista / Parcelado / Recorrente" no formulário, prévia das parcelas, diálogo de alcance e selo "3/10"/"recorrente" na lista.
- **Custo:** sem recursos AWS novos. A recorrência não usa agendador (ver design).
- **Depende de:** change `cartoes-credito` (mês de referência por fatura).
