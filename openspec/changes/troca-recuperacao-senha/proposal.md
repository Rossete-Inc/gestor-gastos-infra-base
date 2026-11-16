# Proposal

## Why

Hoje o cliente não consegue trocar a própria senha nem recuperá-la quando a esquece. A única saída é pedir a um admin, que também não tem essa operação. Isso bloqueia o uso do app por quem perdeu a senha e impede quem suspeita de vazamento de trocá-la.

## What Changes

- **Alterar senha:** novo item "Alterar senha" no Perfil. O cliente informa a senha atual e a nova senha, que deve seguir a mesma política do cadastro. Ao concluir, as sessões em outros aparelhos são encerradas, e o aparelho atual continua logado.
- **Recuperar senha:** novo link "Esqueci minha senha" na tela de login. O cliente informa o e-mail e recebe um código. Com o código, define uma nova senha e volta ao login. As sessões abertas antes da recuperação são encerradas.
- **Privacidade:** a recuperação nunca revela se um e-mail tem conta.
- **E-mail:** a mensagem de recuperação passa a ser enviada em pt-BR.

## Capabilities

### New Capabilities

Nenhuma.

### Modified Capabilities

- `autenticacao`: ganha os requisitos "Alteração de senha" e "Recuperação de senha".

## Impact

- **`gestor-gastos-frontend`:** feature `auth` com novas operações do Cognito, telas `esqueci-senha` e `redefinir-senha` no grupo `(auth)` e `alterar-senha` no grupo `(app)`, item no `PerfilView` e testes unitários e E2E.
- **`gestor-gastos-infra-base`:** texto pt-BR do e-mail de recuperação no User Pool (`infra/auth.go`) e nota sobre o cognito-local em `docs/local.md`.
- **Sem mudanças** no BFF, nos microserviços e no contrato `openapi.yaml`.
- **Custo:** nenhum recurso novo. Os e-mails usam o envio padrão do Cognito, limitado a cerca de 50 por dia, o que basta para o volume atual.
