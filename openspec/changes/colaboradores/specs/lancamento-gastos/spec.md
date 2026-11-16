# Spec Delta

## MODIFIED Requirements

### Requirement: Isolamento de dados por usuário
Toda leitura e escrita de gastos MUST ser restrita à conta ativa, que é a do próprio usuário ou uma conta em que ele colabora, independentemente de outros identificadores enviados na requisição. A escrita MUST exigir que o usuário seja o dono ou um colaborador `EDITOR` da conta ativa.

#### Scenario: Acesso a gasto alheio
- **WHEN** o usuário A tenta ler, editar ou excluir um gasto da conta de B, da qual A não participa, informando o identificador dele
- **THEN** a API responde 404 e o gasto de B não é alterado

#### Scenario: Colaborador acessa a conta do dono
- **WHEN** o usuário A, colaborador da conta de B, lista os gastos com a conta de B ativa
- **THEN** a API devolve os gastos da conta de B

#### Scenario: Leitor tenta escrever
- **WHEN** o usuário A, colaborador `LEITOR` da conta de B, tenta lançar, editar ou excluir um gasto nela
- **THEN** a API responde 403 e nada muda
