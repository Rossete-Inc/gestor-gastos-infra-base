# Spec Delta

## Purpose

Mantém o perfil de cada usuário e permite que administradores gerenciem os clientes (status, papel e exclusão) com permissões adequadas.

## ADDED Requirements

### Requirement: Papéis de acesso
Todo usuário ativo MUST ter exatamente um papel: `cliente` ou `admin`. Operações administrativas MUST ser permitidas apenas para o papel `admin`.

#### Scenario: Cliente tenta acessar área admin
- **WHEN** um usuário com papel `cliente` chama uma operação administrativa
- **THEN** a API responde 403 e nada é alterado

#### Scenario: Navegação por papel
- **WHEN** um usuário com papel `cliente` usa o app
- **THEN** a área de administração não é exibida

### Requirement: Primeiro administrador
O sistema SHALL ter ao menos um administrador, criado fora do fluxo público de cadastro. O cadastro público MUST NOT conceder o papel `admin`.

#### Scenario: Cadastro público
- **WHEN** um visitante conclui o cadastro e a confirmação
- **THEN** a conta recebe somente o papel `cliente`

### Requirement: Consulta do próprio perfil
O sistema SHALL permitir que o usuário autenticado consulte o próprio nome, e-mail, papel e data de criação.

#### Scenario: Ver perfil
- **WHEN** o usuário autenticado abre a tela de perfil
- **THEN** o sistema exibe seu nome, e-mail, papel e data de criação

### Requirement: Listagem de usuários pelo admin
O sistema SHALL permitir ao admin listar usuários de forma paginada, exibindo nome, e-mail, papel, status e data de criação.

#### Scenario: Admin lista usuários
- **WHEN** o admin abre a tela de usuários
- **THEN** o sistema exibe a primeira página de usuários e permite carregar as seguintes

### Requirement: Bloqueio e reativação
O sistema SHALL permitir ao admin bloquear e reativar usuários. O bloqueio MUST impedir novos logins e MUST invalidar as sessões ativas do usuário.

#### Scenario: Admin bloqueia usuário
- **WHEN** o admin bloqueia um usuário ativo
- **THEN** o status passa a `BLOQUEADO`, as sessões do usuário são invalidadas e novos logins são recusados

#### Scenario: Admin reativa usuário
- **WHEN** o admin reativa um usuário bloqueado
- **THEN** o status volta a `ATIVO` e o usuário consegue fazer login novamente

#### Scenario: Admin tenta bloquear a si mesmo
- **WHEN** o admin tenta bloquear a própria conta
- **THEN** o sistema recusa a operação

### Requirement: Alteração de papel
O sistema SHALL permitir ao admin alterar o papel de outro usuário entre `cliente` e `admin`. O sistema MUST NOT permitir remover o papel `admin` do último administrador ativo.

#### Scenario: Promover a admin
- **WHEN** o admin altera o papel de um cliente para `admin`
- **THEN** no próximo login esse usuário tem acesso à área de administração

#### Scenario: Último admin
- **WHEN** o único admin ativo tenta rebaixar o próprio papel para `cliente`
- **THEN** o sistema recusa a operação

### Requirement: Exclusão de usuário
O sistema SHALL permitir ao admin excluir um usuário. A exclusão MUST remover o acesso imediatamente e MUST remover os gastos e categorias personalizadas desse usuário.

#### Scenario: Admin exclui usuário
- **WHEN** o admin exclui um usuário
- **THEN** o usuário não consegue mais fazer login e, após o processamento, seus gastos e categorias personalizadas não existem mais

### Requirement: Ciclo de vida da conta
Uma conta MUST estar em um dos estados `PENDENTE_CONFIRMACAO`, `ATIVO`, `BLOQUEADO` ou `EXCLUIDO`, com transições apenas: pendente→ativo (confirmação), ativo↔bloqueado (admin), ativo ou bloqueado→excluído (admin).

#### Scenario: Transição inválida
- **WHEN** alguém tenta reativar uma conta `EXCLUIDO`
- **THEN** o sistema recusa a operação
