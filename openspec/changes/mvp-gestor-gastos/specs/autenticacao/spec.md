# Spec Delta

## Purpose

Permite que uma pessoa crie sua conta, confirme o e-mail e acesse o sistema com sessão autenticada, base para todas as demais funcionalidades.

## ADDED Requirements

### Requirement: Cadastro de usuário
O sistema SHALL permitir que um visitante crie uma conta informando nome, e-mail e senha. A conta MUST ficar pendente até a confirmação do e-mail.

#### Scenario: Cadastro com dados válidos
- **WHEN** o visitante envia nome, e-mail ainda não cadastrado e senha válida
- **THEN** o sistema cria a conta como pendente de confirmação e envia um código de verificação para o e-mail

#### Scenario: E-mail já cadastrado
- **WHEN** o visitante tenta se cadastrar com um e-mail que já possui conta
- **THEN** o sistema recusa o cadastro e informa que o e-mail já está em uso

### Requirement: Política de senha
O sistema MUST exigir senha com no mínimo 8 caracteres, contendo letra maiúscula, letra minúscula e número.

#### Scenario: Senha fraca
- **WHEN** o visitante tenta se cadastrar com a senha "abc123"
- **THEN** o sistema recusa o cadastro e descreve a regra de senha não atendida

### Requirement: Confirmação de e-mail
O sistema SHALL ativar a conta somente após o usuário informar o código de verificação recebido por e-mail. Ao ativar, a conta MUST receber o papel `cliente`.

#### Scenario: Código correto
- **WHEN** o usuário informa o código de verificação correto dentro da validade
- **THEN** a conta passa a ativa, com papel `cliente`, e o usuário é direcionado ao login

#### Scenario: Código incorreto ou expirado
- **WHEN** o usuário informa um código incorreto ou expirado
- **THEN** o sistema recusa a confirmação e permite solicitar o reenvio do código

### Requirement: Login
O sistema SHALL autenticar usuários ativos por e-mail e senha e estabelecer uma sessão autenticada.

#### Scenario: Credenciais válidas
- **WHEN** um usuário ativo informa e-mail e senha corretos
- **THEN** o sistema abre a sessão e exibe o painel do usuário

#### Scenario: Credenciais inválidas
- **WHEN** alguém informa e-mail ou senha incorretos
- **THEN** o sistema recusa o acesso com mensagem genérica, sem revelar se o e-mail existe

#### Scenario: Conta não confirmada
- **WHEN** um usuário com conta pendente de confirmação tenta fazer login
- **THEN** o sistema recusa o acesso e direciona para a tela de confirmação de e-mail

#### Scenario: Conta bloqueada
- **WHEN** um usuário bloqueado pelo admin tenta fazer login
- **THEN** o sistema recusa o acesso

### Requirement: Proteção de recursos autenticados
Toda operação de dados MUST exigir uma sessão autenticada e válida. Requisições sem sessão válida MUST ser recusadas.

#### Scenario: Requisição sem sessão
- **WHEN** uma requisição à API chega sem credencial ou com credencial expirada ou adulterada
- **THEN** a API responde 401 e nenhum dado é retornado

### Requirement: Renovação de sessão
O sistema SHALL renovar a sessão de forma transparente enquanto a credencial de renovação for válida.

#### Scenario: Sessão de acesso expirada
- **WHEN** a credencial de acesso expira e a credencial de renovação ainda é válida
- **THEN** o app obtém uma nova credencial sem pedir login novamente

#### Scenario: Renovação expirada
- **WHEN** a credencial de renovação também expirou
- **THEN** o app encerra a sessão local e exibe a tela de login

### Requirement: Logout
O sistema SHALL permitir que o usuário encerre a sessão, removendo as credenciais armazenadas no dispositivo.

#### Scenario: Usuário sai do app
- **WHEN** o usuário aciona "Sair"
- **THEN** as credenciais locais são removidas e a tela de login é exibida
