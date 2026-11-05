# Spec Delta

## Purpose

Classifica os gastos em grupos (cartão e fixo) e categorias, combinando um conjunto padrão do sistema com categorias personalizadas por usuário.

## ADDED Requirements

### Requirement: Grupos de gasto
Toda categoria MUST pertencer a exatamente um grupo: `CARTAO` (gastos feitos no cartão) ou `FIXO` (gastos fixos recorrentes).

#### Scenario: Categoria sem grupo
- **WHEN** alguém tenta criar uma categoria sem grupo ou com grupo diferente de `CARTAO` e `FIXO`
- **THEN** o sistema recusa a criação com erro de validação

### Requirement: Categorias padrão
O sistema SHALL disponibilizar a todos os usuários as categorias padrão: no grupo `CARTAO`, Alimentação, Transporte, Lazer, Saúde, Compras e Outros; no grupo `FIXO`, Aluguel, Luz, Água, Internet, Telefone e Condomínio.

#### Scenario: Usuário novo
- **WHEN** um usuário recém-cadastrado abre o formulário de gasto
- **THEN** todas as categorias padrão aparecem, agrupadas por `CARTAO` e `FIXO`

#### Scenario: Categoria padrão é imutável para o usuário
- **WHEN** um usuário tenta alterar ou excluir uma categoria padrão
- **THEN** o sistema recusa a operação

### Requirement: Categorias personalizadas
O sistema SHALL permitir que o usuário crie categorias próprias, informando nome, grupo e cor. Essas categorias MUST ser visíveis apenas para o próprio usuário.

#### Scenario: Criar categoria
- **WHEN** o usuário cria a categoria "Pet" no grupo `CARTAO`
- **THEN** ela aparece junto às categorias padrão do grupo `CARTAO` somente para esse usuário

#### Scenario: Nome duplicado no grupo
- **WHEN** o usuário tenta criar uma categoria com o mesmo nome, ignorando maiúsculas e minúsculas, de outra visível no mesmo grupo
- **THEN** o sistema recusa a criação

#### Scenario: Isolamento entre usuários
- **WHEN** o usuário A lista categorias
- **THEN** as categorias personalizadas do usuário B não aparecem

### Requirement: Exclusão de categoria personalizada em uso
O sistema MUST NOT excluir uma categoria personalizada que tenha gastos associados.

#### Scenario: Categoria com gastos
- **WHEN** o usuário tenta excluir uma categoria personalizada usada por pelo menos um gasto
- **THEN** o sistema recusa a exclusão e informa que há gastos vinculados

#### Scenario: Categoria sem gastos
- **WHEN** o usuário exclui uma categoria personalizada sem gastos
- **THEN** ela deixa de ser listada
