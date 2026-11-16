# Spec Delta

## MODIFIED Requirements

### Requirement: Lançar gasto
O sistema SHALL permitir que o usuário registre um gasto com valor, data, categoria, descrição opcional e cartão, como gasto à vista, parcelado ou recorrente.
- O grupo do gasto MUST ser o grupo da categoria escolhida.
- O gasto de categoria do grupo `CARTAO` MUST informar um cartão ativo do usuário.
- O gasto de categoria do grupo `FIXO` MAY informar um cartão.
- O gasto com cartão MUST receber o mês de referência da fatura do cartão.
- O gasto sem cartão MUST receber como mês de referência o mês da data.

#### Scenario: Gasto de cartão
- **WHEN** o usuário lança R$ 45,90 em 2026-11-03 na categoria Alimentação, com descrição "Almoço", no cartão "Nubank", que fecha dia 25 e vence dia 5
- **THEN** o gasto é salvo no grupo `CARTAO`, vinculado ao "Nubank", e aparece na lista de dezembro de 2026

#### Scenario: Gasto de cartão sem cartão
- **WHEN** o usuário tenta lançar um gasto na categoria Alimentação sem informar o cartão
- **THEN** o sistema recusa com erro de validação no campo cartão

#### Scenario: Gasto fixo sem cartão
- **WHEN** o usuário lança R$ 1.500,00 em 2026-11-05 na categoria Aluguel, sem cartão
- **THEN** o gasto é salvo no grupo `FIXO` e aparece na lista de novembro de 2026

#### Scenario: Gasto fixo no cartão
- **WHEN** o usuário lança R$ 39,90 em 2026-11-10 na categoria Internet, no cartão "Nubank", que fecha dia 25 e vence dia 5
- **THEN** o gasto é salvo no grupo `FIXO`, vinculado ao "Nubank", e aparece na lista de dezembro de 2026

#### Scenario: Gasto fixo recorrente
- **WHEN** o usuário lança R$ 1.500,00 em 2026-11-05 na categoria Aluguel marcado como recorrente
- **THEN** o gasto é salvo no grupo `FIXO` e se repete nos meses seguintes, conforme a capability `parcelamentos-recorrencias`

#### Scenario: Gasto parcelado
- **WHEN** o usuário lança uma compra de R$ 1.200,00 em 3 parcelas no cartão "Nubank"
- **THEN** são criados 3 gastos de R$ 400,00 em faturas consecutivas, conforme a capability `parcelamentos-recorrencias`

#### Scenario: Recorrente antigo
- **WHEN** existe um gasto lançado antes desta mudança com o flag recorrente
- **THEN** ele continua exibido como recorrente e não gera ocorrências em outros meses

### Requirement: Editar gasto
O sistema SHALL permitir que o usuário altere o valor, a data, a categoria, a descrição e o cartão de um gasto próprio, com as mesmas validações do lançamento. O mês de referência MUST ser recalculado com os dados atuais do cartão. Um gasto antigo de categoria `CARTAO` sem cartão MUST poder ser editado para receber um cartão. Ao editar um gasto de parcelamento ou de recorrência, o usuário MUST escolher o alcance da alteração.

#### Scenario: Troca de categoria entre grupos
- **WHEN** o usuário altera um gasto de Alimentação para Internet
- **THEN** o gasto passa ao grupo `FIXO` e os totais refletem a mudança

#### Scenario: Atribuir cartão a gasto antigo
- **WHEN** o usuário edita um gasto antigo "Sem cartão" de Alimentação, de 2026-11-10, e escolhe o "Nubank", que fecha dia 25 e vence dia 5
- **THEN** o gasto passa a ser do "Nubank" e muda para a lista de dezembro de 2026

#### Scenario: Trocar de cartão
- **WHEN** o usuário troca o cartão de um gasto para outro com fechamento diferente
- **THEN** o mês de referência é recalculado pelo novo cartão

#### Scenario: Editar item de série
- **WHEN** o usuário edita uma parcela ou ocorrência recorrente
- **THEN** o app pede o alcance "só esta", "esta e as próximas" ou "todas" antes de salvar

### Requirement: Excluir gasto
O sistema SHALL permitir que o usuário exclua um gasto próprio, após confirmação no app. Ao excluir um gasto de parcelamento ou de recorrência, o usuário MUST escolher o alcance da exclusão.

#### Scenario: Exclusão confirmada
- **WHEN** o usuário confirma a exclusão de um gasto à vista
- **THEN** o gasto deixa de aparecer na lista e nos totais

#### Scenario: Excluir item de série
- **WHEN** o usuário exclui uma parcela ou ocorrência recorrente
- **THEN** o app pede o alcance "só esta", "esta e as próximas" ou "todas", e exclui conforme a escolha
