# autenticacao Specification

## Purpose

Permite que uma pessoa crie sua conta, confirme o e-mail e acesse o sistema com sessão autenticada, base para todas as demais funcionalidades.

## Requirements

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

### Requirement: Alteração de senha
O sistema SHALL permitir que o usuário autenticado altere a própria senha, informando a senha atual e a nova senha. A nova senha MUST atender à política de senha e MUST ser diferente da atual. Ao concluir, as sessões do usuário em outros dispositivos MUST ser encerradas, e a sessão do dispositivo atual MUST continuar ativa.

#### Scenario: Troca com dados válidos
- **WHEN** o usuário autenticado informa a senha atual correta e uma nova senha válida, confirmada corretamente
- **THEN** a senha é alterada, o app informa o sucesso e o usuário continua logado neste dispositivo

#### Scenario: Senha atual incorreta
- **WHEN** o usuário informa uma senha atual incorreta
- **THEN** o sistema recusa a troca, informa que a senha atual não confere e a senha não muda

#### Scenario: Nova senha fraca
- **WHEN** o usuário informa como nova senha "abc123"
- **THEN** o sistema recusa a troca e descreve a regra de senha não atendida

#### Scenario: Confirmação diferente
- **WHEN** a confirmação não é igual à nova senha
- **THEN** o app não envia a troca e indica o erro no campo de confirmação

#### Scenario: Outras sessões encerradas
- **WHEN** o usuário troca a senha estando logado também em outro dispositivo
- **THEN** o outro dispositivo perde a sessão ao tentar renovar a credencial e exibe a tela de login

#### Scenario: Acesso pelo perfil
- **WHEN** o usuário abre a tela de perfil
- **THEN** existe a opção "Alterar senha", que leva ao formulário de troca

### Requirement: Recuperação de senha
O sistema SHALL permitir que um visitante redefina a senha de uma conta ativa a partir do login, informando o e-mail, o código de verificação recebido por e-mail e uma nova senha que atenda à política de senha. O sistema MUST NOT revelar se o e-mail informado possui conta. Após a redefinição, as sessões abertas anteriormente MUST ser encerradas.

#### Scenario: Solicitar código
- **WHEN** o visitante aciona "Esqueci minha senha" na tela de login e informa um e-mail
- **THEN** o app exibe a mensagem "Se houver uma conta com este e-mail, enviamos um código" e abre o formulário de nova senha

#### Scenario: E-mail sem conta
- **WHEN** o visitante solicita o código para um e-mail sem conta
- **THEN** o app exibe a mesma mensagem e o mesmo formulário do e-mail com conta, sem indicar a diferença

#### Scenario: Redefinição com código correto
- **WHEN** o visitante informa o código correto dentro da validade e uma nova senha válida
- **THEN** a senha é redefinida, as sessões anteriores da conta são encerradas e o app exibe o login com o e-mail preenchido e um aviso de sucesso

#### Scenario: Código incorreto ou expirado
- **WHEN** o visitante informa um código incorreto ou expirado
- **THEN** o sistema recusa a redefinição, a senha não muda e o app permite solicitar um novo código

#### Scenario: Muitas tentativas
- **WHEN** o limite de tentativas ou de envios de código é atingido
- **THEN** o sistema recusa a operação e o app pede para tentar novamente mais tarde

#### Scenario: Nova senha fraca na recuperação
- **WHEN** o visitante informa um código correto e a nova senha "abc123"
- **THEN** o sistema recusa a redefinição e descreve a regra de senha não atendida

#### Scenario: Conta bloqueada
- **WHEN** o visitante redefine a senha de uma conta bloqueada pelo admin
- **THEN** o login com a nova senha continua recusado enquanto a conta estiver bloqueada
