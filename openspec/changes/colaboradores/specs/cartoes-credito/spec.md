# Spec Delta

## MODIFIED Requirements

### Requirement: Isolamento de cartões por usuário
Toda leitura e escrita de cartões MUST ser restrita à conta ativa, que é a do próprio usuário ou uma conta em que ele colabora. A escrita MUST exigir que o usuário seja o dono ou um colaborador `EDITOR` da conta.

#### Scenario: Cartão alheio
- **WHEN** o usuário A tenta ler, editar, excluir ou lançar um gasto num cartão da conta de B, da qual A não participa
- **THEN** a API responde 404 ou erro de validação, como se o cartão não existisse, e nada é alterado

#### Scenario: Colaborador lança no cartão do dono
- **WHEN** um colaborador `EDITOR` lança um gasto no cartão "Nubank" da conta da Ana
- **THEN** o gasto entra na fatura do "Nubank" da Ana e no limite usado dele
