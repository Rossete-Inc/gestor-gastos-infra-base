# Spec Delta

## ADDED Requirements

### Requirement: Parcelamento pelo valor total
O sistema SHALL permitir lançar uma compra parcelada informando o valor total e o número de parcelas, de 2 a 72. Cada parcela MUST receber o valor total dividido pelo número de parcelas, em centavos, e o resto da divisão MUST ir para a 1ª parcela. A soma das parcelas MUST ser igual ao valor total.

#### Scenario: Divisão exata
- **WHEN** o usuário lança R$ 1.200,00 em 3 parcelas no cartão "Nubank", com compra em 2026-11-10, fechamento dia 25 e vencimento dia 5
- **THEN** são criadas 3 parcelas de R$ 400,00, nas faturas de dezembro de 2026, janeiro de 2027 e fevereiro de 2027

#### Scenario: Divisão com resto
- **WHEN** o usuário lança R$ 100,00 em 3 parcelas
- **THEN** as parcelas são R$ 33,34, R$ 33,33 e R$ 33,33

#### Scenario: Número de parcelas inválido
- **WHEN** o usuário informa 1 ou 73 parcelas
- **THEN** o sistema recusa com erro de validação no campo parcelas

### Requirement: Parcelamento pelo valor da parcela
O sistema SHALL permitir lançar um parcelamento já em andamento, informando o valor da parcela, o número da parcela atual e o total de parcelas, ou o mês final. O sistema MUST criar as parcelas da atual até a última, a partir do mês de referência da data informada. A parcela atual MUST ser menor ou igual ao total.

#### Scenario: Compra em andamento
- **WHEN** o usuário lança uma parcela de R$ 199,90, parcela atual 3 de 10, com data em 2026-11-10 no cartão "Nubank", que fecha dia 25 e vence dia 5
- **THEN** são criadas as parcelas 3/10 a 10/10, de R$ 199,90 cada, da fatura de dezembro de 2026 à de julho de 2027

#### Scenario: Informar mês final
- **WHEN** o usuário informa a parcela atual 1 e o mês final julho de 2027 para uma compra que entra na fatura de dezembro de 2026
- **THEN** o sistema calcula 8 parcelas e cria 1/8 a 8/8

#### Scenario: Parcela atual maior que o total
- **WHEN** o usuário informa a parcela atual 11 de 10
- **THEN** o sistema recusa com erro de validação

### Requirement: Distribuição das parcelas
Cada parcela MUST ser um gasto próprio, com a mesma categoria, cartão e descrição. As parcelas MUST ocupar meses de referência consecutivos: faturas seguidas quando houver cartão, e meses seguidos da data quando não houver. Cada parcela MUST exibir sua posição no formato "n/N".

#### Scenario: Parcelamento sem cartão
- **WHEN** o usuário lança R$ 600,00 em 3 parcelas na categoria Condomínio, sem cartão, com data em 2026-11-15
- **THEN** são criadas parcelas em novembro e dezembro de 2026 e em janeiro de 2027, no dia 15

#### Scenario: Identificação da parcela
- **WHEN** o usuário vê a lista de janeiro de 2027 com a 2ª parcela da "Geladeira" de 3
- **THEN** o item aparece como "Geladeira 2/3"

### Requirement: Recorrência mensal
O sistema SHALL permitir marcar um gasto como recorrente, com mês final opcional. A recorrência MUST gerar uma ocorrência por mês, no mesmo dia, com os mesmos valor, categoria, cartão e descrição, a partir do mês do lançamento e até o mês final ou até ser encerrada. Um dia que não exista no mês MUST ser tratado como o último dia daquele mês. As ocorrências MUST aparecer na lista, nos totais e no painel de cada mês sem ação do usuário.

#### Scenario: Ocorrência no mês seguinte
- **WHEN** o usuário lança o aluguel de R$ 1.500,00 em 2026-11-05 como recorrente e abre dezembro de 2026
- **THEN** existe um gasto de aluguel de R$ 1.500,00 em 2026-12-05

#### Scenario: Dia 31
- **WHEN** a recorrência foi criada no dia 31 e o usuário abre fevereiro de 2027
- **THEN** a ocorrência de fevereiro tem a data 2027-02-28

#### Scenario: Mês final
- **WHEN** a recorrência tem mês final em março de 2027 e o usuário abre abril de 2027
- **THEN** não há ocorrência em abril

#### Scenario: Painel sem abrir a lista
- **WHEN** o usuário abre direto o painel de um mês ainda não visitado
- **THEN** os totais e a pizza já incluem as ocorrências recorrentes daquele mês

### Requirement: Alcance da edição e da exclusão em séries
Ao editar ou excluir um gasto que pertence a um parcelamento ou a uma recorrência, o sistema SHALL pedir o alcance: "só esta", "esta e as próximas" ou "todas".
- **"Só esta"** MUST alterar apenas o gasto escolhido, que MUST manter o ajuste mesmo quando a série for alterada depois.
- **"Esta e as próximas"** MUST alterar o gasto escolhido e os seguintes, incluindo as ocorrências futuras da recorrência.
- **"Todas"** MUST alterar toda a série.

#### Scenario: Ajustar a conta de luz de um mês
- **WHEN** o usuário edita a ocorrência de dezembro da recorrência "Luz" para R$ 230,00 com o alcance "só esta"
- **THEN** só dezembro passa a R$ 230,00, e os demais meses mantêm o valor da recorrência

#### Scenario: Reajuste a partir de um mês
- **WHEN** o usuário edita a ocorrência de março do "Aluguel" para R$ 1.600,00 com o alcance "esta e as próximas"
- **THEN** março e os meses seguintes passam a R$ 1.600,00, e os anteriores continuam R$ 1.500,00

#### Scenario: Encerrar recorrência
- **WHEN** o usuário exclui a ocorrência de abril do "Streaming" com o alcance "esta e as próximas"
- **THEN** não há mais ocorrências a partir de abril, e as anteriores continuam

#### Scenario: Quitar parcelamento antecipado
- **WHEN** o usuário exclui a parcela 6/10 com o alcance "esta e as próximas"
- **THEN** as parcelas 6/10 a 10/10 deixam de existir, e as parcelas 1/10 a 5/10 continuam

#### Scenario: Excluir série inteira
- **WHEN** o usuário exclui qualquer parcela com o alcance "todas"
- **THEN** todas as parcelas da compra deixam de existir

#### Scenario: Ajuste preservado
- **WHEN** a ocorrência de dezembro foi ajustada com "só esta" e depois o usuário altera "todas" para R$ 200,00
- **THEN** a ocorrência de dezembro mantém o valor ajustado

### Requirement: Séries no limite do cartão
No limite usado do cartão, as parcelas MUST contar em todas as faturas ainda não vencidas, inclusive as futuras. As ocorrências recorrentes MUST contar somente até a fatura do mês corrente.

#### Scenario: Parcelas futuras no limite
- **WHEN** o usuário lança R$ 1.200,00 em 3 parcelas no cartão com limite de R$ 5.000,00 e sem outros gastos
- **THEN** o limite usado é R$ 1.200,00 e o disponível é R$ 3.800,00

#### Scenario: Recorrência no limite
- **WHEN** o cartão tem uma assinatura recorrente de R$ 50,00
- **THEN** o limite usado inclui R$ 50,00 da fatura corrente, e não as ocorrências dos meses seguintes
