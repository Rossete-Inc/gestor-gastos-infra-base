# Spec Delta

## ADDED Requirements

### Requirement: Cadastro de cartão
O sistema SHALL permitir que o usuário cadastre cartões de crédito, informando:
- apelido, obrigatório, até 40 caracteres e único entre os cartões ativos do usuário, sem diferenciar maiúsculas de minúsculas;
- cor da paleta do app;
- dia de fechamento, de 1 a 31;
- dia de vencimento, de 1 a 31;
- limite em reais, maior que zero e até R$ 1.000.000,00.

O usuário MUST poder ter no máximo 20 cartões ativos.

#### Scenario: Cadastro válido
- **WHEN** o usuário cadastra "Nubank" com fechamento dia 25, vencimento dia 5 e limite R$ 5.000,00
- **THEN** o cartão é salvo e aparece na lista de cartões e no formulário de gasto

#### Scenario: Apelido duplicado
- **WHEN** o usuário tenta cadastrar "nubank" tendo um cartão ativo "Nubank"
- **THEN** o sistema recusa com erro de validação no campo apelido

#### Scenario: Dia inválido
- **WHEN** o usuário informa o fechamento no dia 0 ou o vencimento no dia 32
- **THEN** o sistema recusa com erro de validação no campo correspondente

#### Scenario: Limite de cartões
- **WHEN** o usuário com 20 cartões ativos tenta cadastrar outro
- **THEN** o sistema recusa e informa o limite

### Requirement: Edição de cartão
O sistema SHALL permitir que o usuário altere o apelido, a cor, os dias de fechamento e vencimento e o limite de um cartão próprio, com as mesmas validações do cadastro. A mudança dos dias MUST valer apenas para gastos lançados ou editados depois dela.

#### Scenario: Mudança de fechamento
- **WHEN** o usuário muda o fechamento de 25 para 20 tendo gastos já lançados
- **THEN** os gastos existentes continuam no mesmo mês de referência, e os novos seguem o dia 20

### Requirement: Arquivar e excluir cartão
- O sistema SHALL permitir arquivar um cartão. O cartão arquivado MUST NOT aparecer no formulário de gasto, e seus gastos MUST continuar nos totais e no histórico.
- O sistema SHALL permitir reativar um cartão arquivado.
- O sistema MUST NOT excluir um cartão com gastos associados.

#### Scenario: Excluir cartão sem gastos
- **WHEN** o usuário exclui um cartão sem gastos
- **THEN** o cartão deixa de ser listado

#### Scenario: Excluir cartão com gastos
- **WHEN** o usuário tenta excluir um cartão usado por ao menos um gasto
- **THEN** o sistema recusa, informa que há gastos vinculados e oferece arquivar

#### Scenario: Cartão arquivado
- **WHEN** o usuário arquiva um cartão
- **THEN** ele some do formulário de gasto, e os gastos já lançados continuam aparecendo com o nome do cartão

### Requirement: Mês de referência da fatura
Um gasto em cartão MUST pertencer ao mês da fatura em que é cobrado, identificado pelo mês do vencimento. A compra feita até o dia de fechamento, inclusive, entra na fatura que fecha naquele mês; depois dele, na que fecha no mês seguinte.

#### Scenario: Compra antes do fechamento
- **WHEN** o cartão fecha dia 25 e vence dia 5, e a compra é feita em 2026-11-10
- **THEN** o gasto entra na fatura de dezembro de 2026

#### Scenario: Compra depois do fechamento
- **WHEN** o cartão fecha dia 25 e vence dia 5, e a compra é feita em 2026-11-28
- **THEN** o gasto entra na fatura de janeiro de 2027

### Requirement: Mês de vencimento e dias inexistentes
A fatura MUST vencer no mês do fechamento quando o dia de vencimento for maior que o de fechamento, e no mês seguinte quando não for. Um dia de fechamento ou de vencimento que não exista no mês MUST ser tratado como o último dia daquele mês.

#### Scenario: Vencimento no mesmo mês do fechamento
- **WHEN** o cartão fecha dia 3 e vence dia 10, e a compra é feita em 2026-11-02
- **THEN** o gasto entra na fatura de novembro de 2026

#### Scenario: Fechamento no dia 31
- **WHEN** o cartão fecha dia 31 e vence dia 10, e a compra é feita em 2027-02-28
- **THEN** o fechamento de fevereiro é considerado dia 28 e o gasto entra na fatura de março de 2027

### Requirement: Limite disponível
O sistema SHALL exibir para cada cartão o limite usado e o limite disponível:
- o limite usado é a soma dos gastos do cartão em faturas cujo vencimento ainda não passou, mais todas as parcelas futuras;
- o disponível é o limite menos o usado, e pode ficar negativo.

O sistema não controla o pagamento de faturas: uma fatura é considerada paga quando a data de vencimento passa.

#### Scenario: Limite com fatura aberta
- **WHEN** o cartão tem limite R$ 5.000,00 e R$ 1.200,00 em gastos numa fatura que vence daqui a 10 dias
- **THEN** o app exibe usado R$ 1.200,00 e disponível R$ 3.800,00

#### Scenario: Fatura vencida
- **WHEN** a data de vencimento de uma fatura passa
- **THEN** os gastos dessa fatura deixam de contar no limite usado

### Requirement: Visão por cartão
O sistema SHALL exibir uma tela de cartões com, para o mês selecionado, o valor da fatura de cada cartão ativo, o limite usado e o disponível. Ao abrir um cartão, o sistema SHALL listar os gastos daquela fatura.

#### Scenario: Lista de cartões
- **WHEN** o usuário abre a aba Cartões em dezembro de 2026
- **THEN** cada cartão mostra o total da fatura de dezembro e o limite usado e disponível

#### Scenario: Detalhe do cartão
- **WHEN** o usuário toca no cartão "Nubank"
- **THEN** o app lista os gastos da fatura do mês selecionado desse cartão, com navegação entre meses

#### Scenario: Usuário sem cartões
- **WHEN** o usuário sem cartões abre a aba Cartões
- **THEN** o app exibe um estado vazio com atalho para cadastrar um cartão

### Requirement: Isolamento de cartões por usuário
Toda leitura e escrita de cartões MUST ser restrita ao usuário autenticado.

#### Scenario: Cartão alheio
- **WHEN** o usuário A tenta ler, editar, excluir ou lançar um gasto num cartão do usuário B
- **THEN** a API responde 404 ou erro de validação, como se o cartão não existisse, e nada é alterado
