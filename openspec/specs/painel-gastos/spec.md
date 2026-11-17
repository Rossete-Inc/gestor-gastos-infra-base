# painel-gastos Specification

## Purpose

Mostra ao usuário, de forma visual, para onde vai o dinheiro em cada mês: totais, divisão entre cartão e fixos e participação de cada categoria.

## Requirements

### Requirement: Seleção de mês
O painel SHALL exibir os dados de um mês selecionado pelo usuário, abrindo por padrão no mês corrente.

#### Scenario: Abertura do painel
- **WHEN** o usuário abre o painel em 2026-11-04
- **THEN** os dados exibidos são de novembro de 2026

#### Scenario: Navegar entre meses
- **WHEN** o usuário seleciona outubro de 2026
- **THEN** todos os blocos do painel passam a refletir outubro de 2026

### Requirement: Totais do mês
O painel SHALL exibir, para o mês de referência selecionado:
- o total gasto no mês;
- o total de cada grupo (`CARTAO` e `FIXO`), cuja soma MUST ser igual ao total do mês;
- o total de cada cartão com gastos no mês, mais o total "Sem cartão", cuja soma MUST ser igual ao total do mês.

#### Scenario: Totais por grupo
- **WHEN** o mês tem R$ 300,00 em gastos de cartão e R$ 1.700,00 em gastos fixos
- **THEN** o painel exibe total R$ 2.000,00, cartão R$ 300,00 e fixo R$ 1.700,00

#### Scenario: Totais por cartão
- **WHEN** a fatura de dezembro tem R$ 800,00 no "Nubank", R$ 200,00 no "Inter" e há R$ 1.500,00 sem cartão em dezembro
- **THEN** o painel de dezembro exibe "Nubank" R$ 800,00, "Inter" R$ 200,00, "Sem cartão" R$ 1.500,00 e total R$ 2.500,00

#### Scenario: Compra após o fechamento
- **WHEN** o usuário compra R$ 100,00 em 2026-11-28 num cartão que fecha dia 25 e vence dia 5
- **THEN** o valor não entra nos totais de novembro nem de dezembro, e sim nos de janeiro de 2027

### Requirement: Gráfico de pizza por categoria
O painel SHALL exibir um gráfico de pizza com a participação de cada categoria no total do mês, com valor e percentual por fatia, usando a cor da categoria. O usuário MUST poder alternar entre ver todas as categorias, apenas `CARTAO`, apenas `FIXO`, por grupo ou por cartão.

#### Scenario: Participação por categoria
- **WHEN** o mês tem R$ 750,00 em Alimentação e R$ 250,00 em Transporte, e o usuário vê apenas `CARTAO`
- **THEN** o gráfico mostra Alimentação com 75% (R$ 750,00) e Transporte com 25% (R$ 250,00)

#### Scenario: Percentuais somam 100%
- **WHEN** os percentuais calculados têm casas decimais
- **THEN** os percentuais exibidos somam 100%

#### Scenario: Agrupamento por grupo
- **WHEN** o usuário escolhe ver a pizza por grupo
- **THEN** o gráfico tem duas fatias, `CARTAO` e `FIXO`

#### Scenario: Agrupamento por cartão
- **WHEN** o usuário escolhe ver a pizza por cartão
- **THEN** o gráfico tem uma fatia por cartão com gastos no mês, na cor do cartão, mais a fatia "Sem cartão" quando houver

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

### Requirement: Detalhe da fatia do gráfico
O painel SHALL exibir o detalhe de uma fatia do gráfico de pizza quando o usuário passar o mouse sobre ela, no navegador, ou tocar nela ou no item correspondente da legenda. O detalhe MUST mostrar:
- o nome da fatia;
- o valor e o percentual;
- o grupo (`CARTAO` ou `FIXO`) quando a fatia for uma categoria;
- a divisão do valor por cartão, incluindo "Sem cartão".

#### Scenario: Passar o mouse numa categoria
- **WHEN** no navegador o usuário passa o mouse sobre a fatia Alimentação, que soma R$ 420,00 (R$ 300,00 no "Nubank" e R$ 120,00 no "Inter") e representa 35% do mês
- **THEN** a fatia Alimentação salta para fora do anel, as outras ficam esmaecidas e um balão sai dela com "Alimentação · Cartão · R$ 420,00 · 35%" e as linhas "Nubank R$ 300,00" e "Inter R$ 120,00"

#### Scenario: Tocar numa fatia no celular
- **WHEN** no celular o usuário toca na fatia Aluguel
- **THEN** o balão com o detalhe sai da fatia Aluguel, que fica em destaque, e um novo toque fora do gráfico o fecha

#### Scenario: Visão por cartão
- **WHEN** a pizza está na visão por cartão e o usuário seleciona a fatia "Nubank"
- **THEN** o detalhe mostra o nome do cartão, o valor e o percentual da fatura do mês, e a divisão entre os grupos `CARTAO` e `FIXO`

### Requirement: Balão e destaque da fatia selecionada
O detalhe da fatia SHALL aparecer num balão que sai da borda da fatia selecionada, com uma seta apontando para ela, sem sair da largura do card. A fatia selecionada SHALL saltar para fora do anel e as demais SHALL ficar esmaecidas até a seleção acabar.

#### Scenario: Fatia na metade de cima
- **WHEN** o usuário seleciona uma fatia na metade de cima do gráfico
- **THEN** o balão aparece acima da fatia, com a seta apontando para baixo, e as outras fatias ficam esmaecidas

#### Scenario: Fatia perto da borda da tela
- **WHEN** no celular o usuário seleciona uma fatia do lado direito do gráfico
- **THEN** o balão fica inteiro dentro do card, e a seta continua apontando para a fatia

#### Scenario: Fim da seleção
- **WHEN** o usuário toca fora do gráfico ou, no navegador, tira o mouse dele
- **THEN** o balão some e todas as fatias voltam ao tamanho e à cor normais
