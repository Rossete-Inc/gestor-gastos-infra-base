# Spec Delta

## ADDED Requirements

### Requirement: Ativar notificações no aparelho
O app SHALL permitir que cada pessoa ative e desative as notificações push no aparelho em uso, pelo Perfil. Ativar MUST pedir a permissão do navegador. A escolha vale só para aquele aparelho e só para aquela pessoa, qualquer que seja a conta ativa.

#### Scenario: Ativar notificações
- **WHEN** o usuário liga "Notificações" no Perfil e permite no navegador
- **THEN** o aparelho passa a receber as notificações das contas de que ele participa

#### Scenario: Permissão negada
- **WHEN** o usuário liga "Notificações" e nega a permissão no navegador
- **THEN** a opção volta para desligada e o app explica como liberar a permissão nas configurações do navegador

#### Scenario: Desativar notificações
- **WHEN** o usuário desliga "Notificações" no Perfil
- **THEN** aquele aparelho deixa de receber notificações, e os outros aparelhos dele continuam como estavam

#### Scenario: iOS sem o app instalado
- **WHEN** o usuário abre o app no Safari do iOS sem tê-lo instalado na tela inicial
- **THEN** a opção de notificações explica que é preciso instalar o app para recebê-las

### Requirement: Receber notificação com o app fechado
O aparelho com notificações ativadas SHALL exibir a notificação mesmo com o app fechado. Tocar nela MUST abrir o app no painel.

#### Scenario: Toque na notificação
- **WHEN** o usuário toca numa notificação de meta com o app fechado
- **THEN** o app abre no painel

#### Scenario: Aparelho que não aceita mais notificações
- **WHEN** o navegador de um aparelho revoga a assinatura de notificações
- **THEN** o sistema para de enviar para aquele aparelho, sem afetar os demais
