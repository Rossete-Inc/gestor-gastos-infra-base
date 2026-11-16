# Spec Delta

## ADDED Requirements

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
