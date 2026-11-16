# Spec Delta

## MODIFIED Requirements

### Requirement: Categorias personalizadas
O sistema SHALL permitir que o usuário crie categorias próprias, informando nome, grupo e cor. Essas categorias pertencem à conta ativa e MUST ser visíveis apenas para o dono e para os colaboradores dela. Criar e excluir categorias MUST exigir que o usuário seja o dono ou um colaborador `EDITOR` da conta.

#### Scenario: Criar categoria
- **WHEN** o usuário cria a categoria "Pet" no grupo `CARTAO`
- **THEN** ela aparece junto às categorias padrão do grupo `CARTAO` somente para a conta ativa

#### Scenario: Nome duplicado no grupo
- **WHEN** o usuário tenta criar uma categoria com o mesmo nome, ignorando maiúsculas e minúsculas, de outra visível no mesmo grupo
- **THEN** o sistema recusa a criação

#### Scenario: Isolamento entre usuários
- **WHEN** o usuário A lista categorias
- **THEN** as categorias personalizadas do usuário B não aparecem, a menos que A esteja com a conta de B ativa como colaborador

#### Scenario: Categoria criada por colaborador
- **WHEN** um colaborador `EDITOR` cria a categoria "Escola" com a conta da Ana ativa
- **THEN** a categoria passa a aparecer para a Ana e para os demais colaboradores da conta dela
