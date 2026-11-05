# Spec Delta

## Purpose

Permite registrar, corrigir, remover e consultar os gastos de cartão e fixos do usuário, garantindo que cada pessoa acesse apenas os próprios lançamentos.

## ADDED Requirements

### Requirement: Lançar gasto
O sistema SHALL permitir que o usuário registre um gasto com valor, data, categoria e descrição opcional. O grupo do gasto MUST ser o grupo da categoria escolhida.

#### Scenario: Gasto de cartão
- **WHEN** o usuário lança R$ 45,90 em 2026-11-03 na categoria Alimentação com descrição "Almoço"
- **THEN** o gasto é salvo no grupo `CARTAO` e aparece na lista de novembro de 2026

#### Scenario: Gasto fixo recorrente
- **WHEN** o usuário lança R$ 1.500,00 na categoria Aluguel marcado como recorrente
- **THEN** o gasto é salvo no grupo `FIXO`, sinalizado como recorrente

### Requirement: Validação do lançamento
O sistema MUST recusar lançamentos com valor menor ou igual a zero, valor acima de R$ 1.000.000,00, data inválida, descrição acima de 140 caracteres ou categoria inexistente ou não visível ao usuário.

#### Scenario: Valor inválido
- **WHEN** o usuário tenta lançar um gasto com valor R$ 0,00
- **THEN** o sistema recusa com erro de validação no campo valor

#### Scenario: Categoria de outro usuário
- **WHEN** o usuário tenta lançar um gasto usando a categoria personalizada de outro usuário
- **THEN** o sistema recusa com erro de validação, como se a categoria não existisse

### Requirement: Precisão monetária
Valores monetários MUST ser armazenados e somados sem perda de precisão, em centavos de real (BRL).

#### Scenario: Soma de centavos
- **WHEN** o usuário lança R$ 0,10 e R$ 0,20 no mesmo mês
- **THEN** o total do mês exibido é exatamente R$ 0,30

### Requirement: Editar gasto
O sistema SHALL permitir que o usuário altere valor, data, categoria, descrição e recorrência de um gasto próprio, com as mesmas validações do lançamento.

#### Scenario: Troca de categoria entre grupos
- **WHEN** o usuário altera um gasto de Alimentação para Internet
- **THEN** o gasto passa ao grupo `FIXO` e os totais refletem a mudança

### Requirement: Excluir gasto
O sistema SHALL permitir que o usuário exclua um gasto próprio, após confirmação no app.

#### Scenario: Exclusão confirmada
- **WHEN** o usuário confirma a exclusão de um gasto
- **THEN** o gasto deixa de aparecer na lista e nos totais

### Requirement: Isolamento de dados por usuário
Toda leitura e escrita de gastos MUST ser restrita ao usuário autenticado, independentemente de identificadores enviados na requisição.

#### Scenario: Acesso a gasto alheio
- **WHEN** o usuário A tenta ler, editar ou excluir um gasto do usuário B informando o identificador dele
- **THEN** a API responde 404 e o gasto de B não é alterado

### Requirement: Listagem de gastos
O sistema SHALL listar os gastos do usuário em um mês, do mais recente para o mais antigo, com paginação e filtros opcionais por grupo e categoria. Cada item MUST trazer o nome e a cor da categoria.

#### Scenario: Filtro por grupo
- **WHEN** o usuário lista novembro de 2026 filtrando o grupo `FIXO`
- **THEN** somente gastos de categorias do grupo `FIXO` daquele mês são exibidos

#### Scenario: Paginação
- **WHEN** o mês tem mais gastos do que cabem em uma página
- **THEN** o sistema retorna a primeira página e um cursor para a próxima

#### Scenario: Mês sem gastos
- **WHEN** o usuário lista um mês sem lançamentos
- **THEN** o app exibe um estado vazio com atalho para lançar um gasto
