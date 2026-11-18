# Spec Delta

## ADDED Requirements

### Requirement: Excluir categoria remove a meta
Ao excluir uma categoria personalizada, o sistema SHALL remover também a meta dessa categoria, se houver. A exclusão continua sujeita à regra de não haver gastos na categoria.

#### Scenario: Categoria com meta e sem gastos
- **WHEN** o usuário exclui a categoria "Pet", que tem meta de R$ 300,00 e nenhum gasto
- **THEN** a categoria deixa de ser listada e a meta de "Pet" deixa de existir

#### Scenario: Categoria com meta e com gastos
- **WHEN** o usuário tenta excluir uma categoria com meta e com gastos
- **THEN** o sistema recusa a exclusão e a meta continua valendo
