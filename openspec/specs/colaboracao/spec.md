# colaboracao Specification

## Purpose
TBD - created by archiving change colaboradores. Update Purpose after archive.

## Requirements

### Requirement: Convidar colaborador
O dono de uma conta SHALL poder convidar outra pessoa pelo e-mail, escolhendo o papel `EDITOR` ou `LEITOR`. O e-mail MUST ser comparado sem diferenciar maiúsculas de minúsculas e sem espaços nas pontas. O sistema MUST NOT enviar e-mail de notificação.

#### Scenario: Convite válido
- **WHEN** a dona Ana convida "Bruno@x.com" como `EDITOR`
- **THEN** o convite fica pendente para "bruno@x.com" e aparece na lista de colaboradores da Ana como "Convite pendente"

#### Scenario: Convidar a si mesmo
- **WHEN** a Ana convida o próprio e-mail
- **THEN** o sistema recusa com erro de validação no campo e-mail

#### Scenario: Convite repetido
- **WHEN** a Ana convida um e-mail que já tem convite pendente ou que já é colaborador da conta
- **THEN** o sistema recusa e informa que a pessoa já foi convidada ou já colabora

### Requirement: Limite de colaboradores
Uma conta MUST ter no máximo 5 colaboradores, somando os ativos e os convites pendentes.

#### Scenario: Sexto convite
- **WHEN** a conta tem 3 colaboradores e 2 convites pendentes e a dona convida mais alguém
- **THEN** o sistema recusa e informa o limite de 5

### Requirement: Aceitar ou recusar convite
O usuário SHALL ver os convites pendentes para o e-mail da própria conta e poder aceitá-los ou recusá-los. Um convite MUST ser aceito apenas pela conta autenticada com o e-mail convidado.

#### Scenario: Aceitar
- **WHEN** o Bruno, logado com "bruno@x.com", aceita o convite da Ana
- **THEN** ele passa a colaborar na conta da Ana com o papel do convite, e a conta dela aparece no seletor de conta dele

#### Scenario: Recusar
- **WHEN** o Bruno recusa o convite
- **THEN** o convite deixa de existir, a vaga é liberada e ele não ganha acesso

#### Scenario: Convidado sem conta
- **WHEN** a Ana convida "carla@x.com", que ainda não tem conta, e a Carla depois se cadastra e confirma o e-mail
- **THEN** ao entrar, a Carla vê o convite pendente da Ana

#### Scenario: Convite de outra pessoa
- **WHEN** alguém logado com outro e-mail tenta aceitar o convite feito para "bruno@x.com"
- **THEN** o sistema recusa como se o convite não existisse

### Requirement: Papéis na conta compartilhada
O colaborador `EDITOR` SHALL poder fazer na conta do dono tudo o que o dono faz com gastos, cartões e categorias personalizadas. O colaborador `LEITOR` SHALL apenas consultar, e toda escrita dele MUST ser recusada. Convidar, remover, mudar papéis e cancelar convites MUST ser exclusivo do dono.

#### Scenario: Editor lança gasto
- **WHEN** o Bruno, editor, lança um gasto com a conta da Ana ativa
- **THEN** o gasto fica na conta da Ana e entra nos totais dela

#### Scenario: Leitor tenta lançar
- **WHEN** um colaborador leitor tenta lançar, editar ou excluir um gasto, cartão ou categoria
- **THEN** a API responde 403 e nada muda, e o app não exibe essas ações para ele

#### Scenario: Colaborador tenta convidar
- **WHEN** um colaborador, editor ou leitor, tenta convidar alguém para a conta da Ana
- **THEN** a API responde 403

### Requirement: Seletor de conta
O usuário SHALL alternar entre a própria conta e as contas em que colabora. Painel, gastos, cartões e categorias MUST refletir a conta ativa. A conta escolhida MUST continuar ativa ao reabrir o app no mesmo dispositivo, enquanto o acesso existir.

#### Scenario: Trocar de conta
- **WHEN** o Bruno troca de "Minha conta" para "Conta da Ana"
- **THEN** todas as telas passam a mostrar os dados da conta da Ana, com o nome dela no seletor

#### Scenario: Acesso perdido
- **WHEN** a conta ativa é de alguém que removeu o usuário
- **THEN** a próxima requisição é recusada, o app volta para "Minha conta" e avisa que o acesso terminou

#### Scenario: Conta alheia sem convite
- **WHEN** alguém envia uma requisição para uma conta da qual não participa
- **THEN** a API responde 404 e nenhum dado é retornado

### Requirement: Gestão pelo dono
O dono SHALL listar os colaboradores e os convites pendentes da conta, mudar o papel de um colaborador, remover um colaborador e cancelar um convite. A remoção e a mudança de papel MUST valer na próxima requisição do colaborador.

#### Scenario: Mudar para leitor
- **WHEN** a Ana muda o Bruno de `EDITOR` para `LEITOR`
- **THEN** a próxima tentativa do Bruno de lançar um gasto na conta dela é recusada

#### Scenario: Remover colaborador
- **WHEN** a Ana remove o Bruno
- **THEN** o Bruno perde o acesso, e os gastos que ele lançou continuam na conta da Ana

### Requirement: Sair de uma conta
O colaborador SHALL poder sair de uma conta em que colabora, sem precisar do dono.

#### Scenario: Colaborador sai
- **WHEN** o Bruno sai da conta da Ana
- **THEN** a conta da Ana some do seletor dele, a vaga é liberada e os gastos que ele lançou continuam na conta

### Requirement: Autoria dos lançamentos
Cada gasto MUST registrar quem o lançou e quem o editou por último. A lista de gastos SHALL exibir quem lançou e permitir filtrar por pessoa. Gastos lançados antes desta mudança MUST ser exibidos como lançados pelo dono da conta.

#### Scenario: Lançado por colaborador
- **WHEN** o Bruno lança um gasto na conta da Ana
- **THEN** a Ana vê o gasto com "lançado por Bruno"

#### Scenario: Filtro por pessoa
- **WHEN** a Ana filtra a lista de novembro pelo Bruno
- **THEN** só aparecem os gastos lançados pelo Bruno

#### Scenario: Ex-colaborador
- **WHEN** um colaborador que lançou gastos sai da conta ou é removido
- **THEN** os gastos dele continuam exibindo o nome dele como autor
