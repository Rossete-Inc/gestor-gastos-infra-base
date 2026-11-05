# Spec Delta

## Purpose

Mostra ao usuário, de forma visual, para onde vai o dinheiro em cada mês: totais, divisão entre cartão e fixos e participação de cada categoria.

## ADDED Requirements

### Requirement: Seleção de mês
O painel SHALL exibir os dados de um mês selecionado pelo usuário, abrindo por padrão no mês corrente.

#### Scenario: Abertura do painel
- **WHEN** o usuário abre o painel em 2026-11-04
- **THEN** os dados exibidos são de novembro de 2026

#### Scenario: Navegar entre meses
- **WHEN** o usuário seleciona outubro de 2026
- **THEN** todos os blocos do painel passam a refletir outubro de 2026

### Requirement: Totais do mês
O painel SHALL exibir o total gasto no mês e o total de cada grupo (`CARTAO` e `FIXO`), cuja soma MUST ser igual ao total do mês.

#### Scenario: Totais por grupo
- **WHEN** o mês tem R$ 300,00 em gastos de cartão e R$ 1.700,00 em gastos fixos
- **THEN** o painel exibe total R$ 2.000,00, cartão R$ 300,00 e fixo R$ 1.700,00

### Requirement: Gráfico de pizza por categoria
O painel SHALL exibir um gráfico de pizza com a participação de cada categoria no total do mês, com valor e percentual por fatia, usando a cor da categoria. O usuário MUST poder alternar entre ver todas as categorias, apenas `CARTAO` ou apenas `FIXO`.

#### Scenario: Participação por categoria
- **WHEN** o mês tem R$ 750,00 em Alimentação e R$ 250,00 em Transporte, e o usuário vê apenas `CARTAO`
- **THEN** o gráfico mostra Alimentação com 75% (R$ 750,00) e Transporte com 25% (R$ 250,00)

#### Scenario: Percentuais somam 100%
- **WHEN** os percentuais calculados têm casas decimais
- **THEN** os percentuais exibidos somam 100%

#### Scenario: Agrupamento por grupo
- **WHEN** o usuário escolhe ver a pizza por grupo
- **THEN** o gráfico tem duas fatias, `CARTAO` e `FIXO`

### Requirement: Últimos lançamentos
O painel SHALL exibir os 5 gastos mais recentes do mês selecionado, com acesso à lista completa.

#### Scenario: Lista resumida
- **WHEN** o mês tem 12 gastos
- **THEN** o painel exibe os 5 mais recentes e um atalho "Ver todos"

### Requirement: Painel sem dados
Quando o mês não tiver gastos, o painel MUST exibir totais zerados e um estado vazio no lugar do gráfico.

#### Scenario: Mês vazio
- **WHEN** o usuário abre um mês sem lançamentos
- **THEN** os totais aparecem como R$ 0,00 e o gráfico é substituído por uma mensagem com atalho para lançar um gasto

### Requirement: Atualização após alterações
O painel MUST refletir lançamentos, edições e exclusões assim que o usuário retornar a ele após a operação.

#### Scenario: Novo gasto refletido
- **WHEN** o usuário lança um gasto e volta ao painel
- **THEN** totais, gráfico e últimos lançamentos já incluem o novo gasto
