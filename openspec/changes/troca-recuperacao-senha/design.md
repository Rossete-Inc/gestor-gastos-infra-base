# Design

## Context

- **Como funciona hoje.** O front fala direto com o Cognito por `fetch` em `src/features/auth/api/cognito.ts`, função `chamar()`, que envia `X-Amz-Target` e `ClientId`. O BFF não tem rotas de autenticação.
- **O User Pool (`infra-base/infra/auth.go`) já está pronto para esta change:**
  - `AccountRecovery_EMAIL_ONLY` está ligado.
  - O app client é público, sem segredo, com `PreventUserExistenceErrors` e revogação de token ligados. O access token dura 15 minutos e o refresh token 30 dias.
  - O trigger Post Confirmation já ignora `PostConfirmation_ConfirmForgotPassword` (`internal/postconfirmation/handler.go`).
- **Como a sessão é guardada.** No front (`sessao.ts`), o access token fica em memória. O refresh token fica no secure storage, mas só no app nativo; na web ele não é persistido.

Decisões do usuário (2026-11-16):
- A troca de senha fica no Perfil.
- A recuperação fica na tela de login.
- Trocar ou recuperar a senha encerra as outras sessões.

## Goals / Non-Goals

**Goals:**
- Trocar e recuperar a senha sem back-end novo e sem custo adicional.
- Encerrar as sessões dos outros dispositivos após a troca, sem deslogar o dispositivo atual.

**Non-Goals:**
- Troca de e-mail.
- Autenticação em dois fatores.
- Envio de e-mail pelo SES.
- Redefinição de senha pelo admin.

## Decisions

### Chamadas diretas ao Cognito, sem BFF
`ChangePassword`, `GlobalSignOut`, `ForgotPassword` e `ConfirmForgotPassword` são operações públicas do Cognito. As duas primeiras usam o access token, e as duas últimas só o `ClientId`. Seguem o mesmo padrão de `cadastrar` e `confirmar`.

Alternativa descartada: rotas no BFF com `Admin*`. Elas exigiriam permissões IAM maiores e uma rota pública sem autenticação no API Gateway, sem nenhum ganho.

### Encerrar as outras sessões mantendo a atual
O Cognito não tem a operação "encerrar todas as sessões menos esta". A sequência usada é:

1. **Troca de senha:** `ChangePassword(atual, nova)` → `GlobalSignOut(accessToken)` → login com a nova senha (SRP; `USER_PASSWORD_AUTH` no ambiente local) → a sessão local guarda os novos tokens.
   - O `GlobalSignOut` revoga todos os refresh tokens. Os outros dispositivos caem na próxima renovação, em até 15 minutos.
   - Se o login depois do `GlobalSignOut` falhar, o app encerra a sessão local e leva ao login, informando que a senha foi alterada.
2. **Recuperação:** `ForgotPassword(email)` → `ConfirmForgotPassword(email, código, nova)` → login → `GlobalSignOut` → volta ao login com o e-mail preenchido e a mensagem `?senhaRedefinida=1`.
   - O login intermediário existe só para obter um access token e revogar as sessões antigas.
   - Esse login é silencioso, e uma falha nele não impede o sucesso da redefinição. Ela fica registrada como aviso no console.
   - A conta bloqueada falha nesse login, o que é o esperado.

### Mensagens e privacidade
- O `ForgotPassword` sempre leva à tela de código com a mesma mensagem. Com `PreventUserExistenceErrors`, o Cognito já responde com sucesso para um e-mail inexistente.
- Erros mapeados em `model/erros.ts`, por operação (`trocaSenha`, `esqueciSenha` e `redefinirSenha`):
  - `NotAuthorizedException`, ou `InvalidPasswordException` no cognito-local: "senha atual não confere".
  - `InvalidPasswordException` no Cognito real: política de senha.
  - `CodeMismatchException` e `ExpiredCodeException`: código inválido ou expirado.
  - `LimitExceededException` e `TooManyRequestsException`: tente mais tarde.
- A validação de força e a de confirmação reaproveitam `model/senha.ts` e os schemas Zod de `model/formularios.ts`. A nova senha igual à atual é recusada no próprio formulário.

### Rotas e telas
- `(auth)/esqueci-senha.tsx`: campo de e-mail, já preenchido se vier do login.
- `(auth)/redefinir-senha.tsx`: código, nova senha e confirmação, além do botão "Reenviar código", que chama `ForgotPassword` de novo.
- `(app)/alterar-senha.tsx`: senha atual, nova senha e confirmação, com header de voltar. É aberta por um novo `MenuItem` "Alterar senha" no `PerfilView`, acima de "Sair".
- `LoginForm`: link "Esqueci minha senha" abaixo do campo de senha, e aviso de sucesso quando `senhaRedefinida=1`.

### E-mail em pt-BR
No `auth.go`, a mensagem de verificação do pool já está em pt-BR (`UserVerification`). Esperamos que o Cognito também use esse template no código de recuperação. Por isso, o texto será revisto para servir aos dois casos, por exemplo: "Seu código do Gestor de Gastos é {####}".

A tarefa 1.1 confere em `dev` qual texto chega de fato. Se a recuperação vier com o texto padrão em inglês, a alternativa é um trigger *Custom Message*: uma Lambda pequena que fica no free tier. Ela entra nesta change só se o usuário aprovar; caso contrário, o texto em inglês é aceito.

### Ambiente local
O cognito-local 3.23.2 deve suportar `ForgotPassword`, `ConfirmForgotPassword`, `ChangePassword` e `GlobalSignOut`, e o código aparece no log como no cadastro. Isso será conferido na tarefa 1.2. Se alguma operação faltar, o comportamento fica documentado em `docs/local.md`, e os testes unitários (MSW) continuam cobrindo o fluxo.

## Risks / Trade-offs

- **Outros dispositivos continuam com acesso por até 15 minutos.** É o prazo do access token já emitido. Mesmo trade-off aceito para o bloqueio no MVP.
- **Login extra após a troca.** Se ele falhar por rede, o usuário é levado ao login; a senha já foi trocada.
- **Limite de cerca de 50 e-mails por dia do Cognito.** Aceitável. Migrar para o SES continua como Open Question do MVP.
- **E2E não lê o e-mail real.** O E2E cobre a troca completa e só o pedido de código da recuperação. A redefinição com código fica na validação manual em `dev`.

## Migration Plan

- **Ordem do deploy:**
  1. Merge no infra-base (texto do e-mail).
  2. Merge no frontend.
- Não há migração de dados.
- **Rollback:** reverter o PR do frontend.

## Open Questions

Nenhuma.
