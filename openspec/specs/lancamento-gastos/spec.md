# lancamento-gastos Specification

## Purpose

Permite registrar, corrigir, remover e consultar os gastos de cartão e fixos do usuário, garantindo que cada pessoa acesse apenas os próprios lançamentos.

## Requirements

### Requirement: Lançar gasto
O sistema SHALL permitir registrar um gasto com valor, data, categoria, descrição opcional e cartão, à vista, parcelado ou recorrente.
- O grupo MUST ser o da categoria.
- Categoria `CARTAO` MUST informar um cartão ativo; `FIXO` MAY informar.
- O mês de referência MUST ser o da fatura do cartão, ou o da data sem cartão.

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

### Requirement: Validação do lançamento
O sistema MUST recusar lançamentos com:
- valor menor ou igual a zero;
- valor acima de R$ 1.000.000,00;
- data inválida;
- descrição acima de 140 caracteres;
- categoria inexistente ou não visível ao usuário;
- cartão inexistente, arquivado ou de outro usuário.

#### Scenario: Valor inválido
- **WHEN** o usuário tenta lançar um gasto com valor R$ 0,00
- **THEN** o sistema recusa com erro de validação no campo valor

#### Scenario: Categoria de outro usuário
- **WHEN** o usuário tenta lançar um gasto usando a categoria personalizada de outro usuário
- **THEN** o sistema recusa com erro de validação, como se a categoria não existisse

#### Scenario: Cartão arquivado
- **WHEN** o usuário tenta lançar um gasto num cartão arquivado
- **THEN** o sistema recusa com erro de validação no campo cartão

### Requirement: Precisão monetária
Valores monetários MUST ser armazenados e somados sem perda de precisão, em centavos de real (BRL).

#### Scenario: Soma de centavos
- **WHEN** o usuário lança R$ 0,10 e R$ 0,20 no mesmo mês
- **THEN** o total do mês exibido é exatamente R$ 0,30

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

### Requirement: Isolamento de dados por usuário
Toda leitura e escrita de gastos MUST ser restrita ao usuário autenticado, independentemente de identificadores enviados na requisição.

#### Scenario: Acesso a gasto alheio
- **WHEN** o usuário A tenta ler, editar ou excluir um gasto do usuário B informando o identificador dele
- **THEN** a API responde 404 e o gasto de B não é alterado

### Requirement: Listagem de gastos
O sistema SHALL listar os gastos do usuário pelo mês de referência, do mais recente para o mais antigo pela data, com paginação e filtros opcionais por grupo, categoria e cartão. Cada item MUST trazer o nome e a cor da categoria e, quando houver, o apelido e a cor do cartão. O filtro por cartão MUST aceitar a opção "Sem cartão".

#### Scenario: Filtro por grupo
- **WHEN** o usuário lista novembro de 2026 filtrando o grupo `FIXO`
- **THEN** somente gastos de categorias do grupo `FIXO` com mês de referência novembro de 2026 são exibidos

#### Scenario: Filtro por cartão
- **WHEN** o usuário lista dezembro de 2026 filtrando o cartão "Nubank"
- **THEN** somente os gastos da fatura de dezembro do "Nubank" são exibidos

#### Scenario: Filtro sem cartão
- **WHEN** o usuário filtra "Sem cartão"
- **THEN** somente os gastos sem cartão do mês são exibidos

#### Scenario: Paginação
- **WHEN** o mês tem mais gastos do que cabem em uma página
- **THEN** o sistema retorna a primeira página e um cursor para a próxima

#### Scenario: Mês sem gastos
- **WHEN** o usuário lista um mês sem lançamentos
- **THEN** o app exibe um estado vazio com atalho para lançar um gasto
