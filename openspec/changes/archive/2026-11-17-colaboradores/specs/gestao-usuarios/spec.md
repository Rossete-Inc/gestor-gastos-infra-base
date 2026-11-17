# Spec Delta

## MODIFIED Requirements

### Requirement: Exclusão de usuário
O sistema SHALL permitir ao admin excluir um usuário. A exclusão MUST remover o acesso imediatamente e MUST remover os gastos, os cartões e as categorias personalizadas da conta desse usuário. A exclusão MUST também encerrar as colaborações da conta dele, incluindo convites pendentes, e as participações dele em outras contas.

#### Scenario: Admin exclui usuário
- **WHEN** o admin exclui um usuário
- **THEN** o usuário não consegue mais fazer login e, após o processamento, seus gastos, cartões e categorias personalizadas não existem mais

#### Scenario: Dono com colaboradores é excluído
- **WHEN** o admin exclui a Ana, que tem o Bruno como colaborador e um convite pendente para a Carla
- **THEN** a conta da Ana some do seletor do Bruno, o convite da Carla deixa de existir e nenhum dos dois acessa os dados dela

#### Scenario: Colaborador é excluído
- **WHEN** o admin exclui o Bruno, colaborador da conta da Ana
- **THEN** o Bruno deixa de constar entre os colaboradores da Ana, e os gastos que ele lançou continuam na conta dela
