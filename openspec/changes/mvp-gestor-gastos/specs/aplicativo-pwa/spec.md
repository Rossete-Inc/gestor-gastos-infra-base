# Spec Delta

## Purpose

Disponibiliza o sistema como aplicativo web progressivo (PWA), instalável em Android e iOS a partir do navegador, sem depender das lojas de aplicativos.

## ADDED Requirements

### Requirement: Instalação como app
O app SHALL ser instalável na tela inicial de Android (Chrome) e iOS (Safari, "Adicionar à Tela de Início"), abrindo em tela cheia com nome e ícone próprios.

#### Scenario: Instalação no Android
- **WHEN** o usuário acessa o app no Chrome do Android e escolhe instalar
- **THEN** o ícone "Gestor de Gastos" aparece na tela inicial e abre o app sem a barra do navegador

#### Scenario: Instalação no iOS
- **WHEN** o usuário adiciona o app à tela de início pelo Safari no iOS
- **THEN** o app abre em modo standalone com o ícone e o nome definidos

### Requirement: Layout responsivo mobile-first
O app MUST ser utilizável em telas a partir de 360 px de largura sem rolagem horizontal, e também em desktop.

#### Scenario: Tela pequena
- **WHEN** o app é aberto em uma tela de 360 px de largura
- **THEN** todas as telas são utilizáveis sem rolagem horizontal

### Requirement: Comportamento sem conexão
Sem conexão, o app MUST abrir sua interface e informar que está offline, sem permitir gravações.

#### Scenario: App aberto offline
- **WHEN** o usuário abre o app instalado sem conexão
- **THEN** a interface carrega e exibe um aviso de que não há conexão

#### Scenario: Tentativa de salvar offline
- **WHEN** o usuário tenta salvar um gasto sem conexão
- **THEN** o app informa que é preciso estar conectado e mantém os dados preenchidos no formulário

### Requirement: Atualização de versão
Quando uma nova versão for publicada, o app SHALL passar a usá-la sem exigir reinstalação.

#### Scenario: Nova versão publicada
- **WHEN** uma nova versão é publicada e o usuário reabre o app
- **THEN** o app carrega a nova versão ou oferece recarregar para aplicá-la

### Requirement: Acessibilidade básica
Botões apenas com ícone MUST ter rótulo acessível, e áreas de toque MUST ter no mínimo 44×44 px.

#### Scenario: Leitor de tela
- **WHEN** um leitor de tela foca o botão de adicionar gasto
- **THEN** ele anuncia um rótulo descritivo, como "Adicionar gasto"
