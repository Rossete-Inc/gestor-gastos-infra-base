# Spec Delta

## Purpose

Permite que a conta defina um teto mensal de gastos por categoria, acompanhe o progresso de cada meta e seja avisada, no app e por notificação push, ao se aproximar do teto ou estourá-lo.

## ADDED Requirements

### Requirement: Definir meta mensal por categoria
O sistema SHALL permitir definir, para cada categoria visível na conta ativa, padrão ou personalizada, uma meta mensal com um valor em reais maior que zero. Cada categoria tem no máximo uma meta. A meta MUST valer todo mês, a partir do mês corrente, até ser alterada ou removida.

#### Scenario: Criar meta
- **WHEN** o usuário define uma meta de R$ 1.000,00 para a categoria "Lazer" em novembro de 2026
- **THEN** a meta de "Lazer" passa a valer R$ 1.000,00 em novembro de 2026 e nos meses seguintes

#### Scenario: Valor inválido
- **WHEN** o usuário tenta definir uma meta de R$ 0,00 ou de valor negativo
- **THEN** o sistema recusa e informa que o valor deve ser maior que zero

#### Scenario: Categoria inexistente
- **WHEN** o usuário tenta definir uma meta para uma categoria que não é visível na conta ativa
- **THEN** o sistema recusa e informa que a categoria não foi encontrada

#### Scenario: Segunda meta para a mesma categoria
- **WHEN** a categoria "Lazer" já tem meta e o usuário define outra
- **THEN** o novo valor substitui o anterior, sem criar uma segunda meta

### Requirement: Alterar e remover meta
O sistema SHALL permitir alterar o valor de uma meta e removê-la. Alterar MUST valer a partir do mês corrente, sem mudar a meta registrada para meses anteriores. Remover MUST tirar a meta do mês corrente em diante.

#### Scenario: Alterar o valor
- **WHEN** a meta de "Combustível" é R$ 900,00 desde outubro de 2026 e o usuário a altera para R$ 1.100,00 em novembro de 2026
- **THEN** novembro e os meses seguintes usam R$ 1.100,00, e outubro continua com R$ 900,00

#### Scenario: Remover a meta
- **WHEN** o usuário remove a meta de "Lazer" em novembro de 2026
- **THEN** "Lazer" deixa de ter meta em novembro de 2026 e nos meses seguintes

### Requirement: Permissão para gerenciar metas
Criar, alterar e remover metas MUST exigir que o usuário seja o dono ou um colaborador `EDITOR` da conta ativa. Um colaborador `LEITOR` SHALL ver as metas e o progresso, mas MUST NOT alterá-los.

#### Scenario: Editor define meta
- **WHEN** um colaborador `EDITOR` define uma meta com a conta da Ana ativa
- **THEN** a meta passa a valer para a conta da Ana e aparece para todas as pessoas da conta

#### Scenario: Leitor tenta alterar meta
- **WHEN** um colaborador `LEITOR` tenta criar, alterar ou remover uma meta
- **THEN** o sistema recusa com a indicação de acesso somente leitura

### Requirement: Progresso da meta no mês
O progresso de uma meta num mês SHALL ser o total dos gastos da categoria naquele mês, incluindo os lançados por qualquer pessoa da conta, dividido pelo valor da meta. O mês de cada gasto MUST ser o mesmo mês de referência usado pelo painel: o mês da fatura para gastos de cartão. Parcelas e recorrências MUST contar no mês de cada ocorrência.

#### Scenario: Gastos de várias pessoas
- **WHEN** a meta de "Lazer" é R$ 1.000,00, a Ana lança R$ 300,00 e o Bruno, colaborador, lança R$ 200,00 em "Lazer" no mesmo mês
- **THEN** o progresso da meta é R$ 500,00 de R$ 1.000,00 (50%)

#### Scenario: Compra no cartão após o fechamento
- **WHEN** uma compra de R$ 100,00 em "Lazer" cai, pelo fechamento do cartão, na fatura de dezembro
- **THEN** ela conta para a meta de "Lazer" de dezembro, e não para a de novembro

#### Scenario: Gasto parcelado
- **WHEN** uma compra de R$ 600,00 em 3 parcelas é lançada em "Compras", com meta de R$ 500,00
- **THEN** cada um dos três meses soma R$ 200,00 à meta de "Compras"

#### Scenario: Novo mês
- **WHEN** a meta de "Lazer" estourou em novembro e começa dezembro
- **THEN** o progresso de dezembro começa em R$ 0,00, somando só os gastos de dezembro

### Requirement: Estados da meta
Cada meta SHALL ter, no mês, um destes estados: **dentro da meta** (abaixo de 80% do valor), **aviso** (de 80% até menos de 100%) e **estourada** (100% ou mais). Os limiares de 80% e 100% são fixos e iguais para todas as metas.

#### Scenario: Abaixo de 80%
- **WHEN** a meta é R$ 1.000,00 e o total do mês é R$ 799,99
- **THEN** a meta está dentro da meta

#### Scenario: Exatamente 80%
- **WHEN** a meta é R$ 1.000,00 e o total do mês é R$ 800,00
- **THEN** a meta está em aviso

#### Scenario: Exatamente 100%
- **WHEN** a meta é R$ 1.000,00 e o total do mês é R$ 1.000,00
- **THEN** a meta está estourada

### Requirement: Aviso ao salvar um gasto
Ao criar ou editar um gasto, o sistema SHALL informar ao usuário, na resposta, cada meta cujo progresso cruzou 80% ou 100% por causa daquele lançamento, com o nome da categoria, o total e o valor da meta. O app SHALL exibir esse aviso logo depois de salvar.

#### Scenario: Cruzar 80%
- **WHEN** a meta de "Lazer" é R$ 1.000,00, o total do mês é R$ 700,00 e o usuário lança R$ 120,00 em "Lazer"
- **THEN** o app avisa que a meta de "Lazer" chegou a 80%: R$ 820,00 de R$ 1.000,00

#### Scenario: Cruzar 80% e 100% de uma vez
- **WHEN** a meta de "Lazer" é R$ 1.000,00, o total do mês é R$ 700,00 e o usuário lança R$ 400,00 em "Lazer"
- **THEN** o app avisa apenas que a meta de "Lazer" estourou: R$ 1.100,00 de R$ 1.000,00

#### Scenario: Lançamento que não cruza limiar
- **WHEN** a meta de "Lazer" já está em aviso e o usuário lança mais um gasto que não chega a 100%
- **THEN** o app não mostra aviso de meta

### Requirement: Notificação push da meta
Quando um gasto criado ou editado faz uma meta cruzar 80% ou 100% pela primeira vez no mês, o sistema SHALL enviar uma notificação push a todos os aparelhos com notificações ativadas do dono e dos colaboradores da conta. Cada limiar MUST notificar no máximo uma vez por meta e por mês.

#### Scenario: Notificação ao chegar a 80%
- **WHEN** o Bruno, colaborador, lança um gasto que leva a meta de "Lazer" da Ana a 80% em novembro
- **THEN** os aparelhos ativados da Ana e do Bruno recebem a notificação "Meta de Lazer: 80% usados (R$ 820,00 de R$ 1.000,00)"

#### Scenario: Limiar já notificado no mês
- **WHEN** a meta de "Lazer" já notificou 80% em novembro, um gasto é excluído (o total cai para 70%) e outro gasto a leva de novo a 85%
- **THEN** nenhuma nova notificação de 80% é enviada em novembro

#### Scenario: Um lançamento cruza limiares em vários meses
- **WHEN** uma compra parcelada faz a meta de "Compras" estourar em dezembro e em janeiro
- **THEN** cada aparelho ativado recebe uma única notificação, que cita os dois meses

#### Scenario: Nenhum aparelho ativado
- **WHEN** uma meta cruza um limiar e ninguém da conta ativou notificações
- **THEN** nenhuma notificação é enviada e o aviso continua aparecendo no app

#### Scenario: Recorrência gerada ao abrir o mês
- **WHEN** uma ocorrência recorrente passa a existir apenas porque o mês foi aberto, sem nenhum lançamento novo
- **THEN** o estado da meta reflete a ocorrência, mas nenhuma notificação push é enviada por ela
