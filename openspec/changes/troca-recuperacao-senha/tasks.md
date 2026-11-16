# Tasks

> Cada grupo indica o repositório da org Rossete-Inc. Fluxo: PR → esteira verde → merge → deploy automático em `dev` → validação do usuário em `dev`.

## 1. infra-base

- [ ] 1.1 Revisar o texto pt-BR do e-mail de verificação/recuperação em `infra/auth.go` para servir aos dois fluxos; verificar com `go test ./...` e `cdk diff` sem outras mudanças no pool
- [ ] 1.2 Conferir no cognito-local (`task local:up`) o suporte a `ForgotPassword`, `ConfirmForgotPassword`, `ChangePassword` e `GlobalSignOut` com chamadas `curl`, e registrar o resultado e onde ver o código em `docs/local.md`; verificar que o documento descreve os comandos usados

## 2. frontend — operações e modelo

- [ ] 2.1 Adicionar `alterarSenha`, `sairDeTodos`, `esqueciSenha` e `redefinirSenha` em `src/features/auth/api/cognito.ts`; verificar com testes em `cognito.test.ts` que conferem o `X-Amz-Target` e o corpo de cada chamada
- [ ] 2.2 Adicionar as operações `trocaSenha`, `esqueciSenha` e `redefinirSenha` em `model/erros.ts`, com os mapeamentos do design.md, e os schemas Zod de troca e redefinição em `model/formularios.ts` (confirmação igual e nova senha diferente da atual); verificar com `model.test.ts`
- [ ] 2.3 Criar as mutações `useAlterarSenha`, `useEsqueciSenha` e `useRedefinirSenha` em `api/mutacoes.ts`, com a sequência de encerrar as outras sessões e relogar do design.md, e exportá-las no `index.ts`; verificar com testes MSW cobrindo o sucesso, a falha no relogin e o login silencioso que falha na recuperação

## 3. frontend — telas

- [ ] 3.1 Criar `(auth)/esqueci-senha.tsx` e `(auth)/redefinir-senha.tsx` com os formulários, o reenvio do código e a mensagem neutra, e adicionar em `LoginForm.tsx` o link "Esqueci minha senha" e o aviso `senhaRedefinida`; verificar com testes de componente cobrindo os cenários da spec "Recuperação de senha"
- [ ] 3.2 Criar `(app)/alterar-senha.tsx` e o `MenuItem` "Alterar senha" no `PerfilView.tsx`, ajustando o mock de `@/features/auth` em `PerfilView.test.tsx`; verificar com testes cobrindo os cenários da spec "Alteração de senha"
- [ ] 3.3 Rodar `npm run verify` (typecheck, lint, format, boundaries e Jest) sem erros

## 4. E2E e entrega

- [ ] 4.1 E2E Playwright `e2e/senha.spec.ts`: criar a conta via `aws.ts` → entrar → alterar a senha pelo perfil → sair → entrar com a nova senha (e recusar a antiga) → "Esqueci minha senha" mostra a mensagem neutra; verificar que o teste passa em `dev` na esteira
- [ ] 4.2 Merge e deploy em `dev` (infra-base antes do frontend); verificar a esteira verde e o smoke test
- [ ] 4.3 **Validação em `dev` pelo usuário:** trocar a senha pelo perfil, ver outra sessão (outro navegador) cair, recuperar a senha pela tela de login com o código do e-mail e entrar com a nova. O usuário aprova

## Workflow follow-up

- Arquivar a change com `/opsx:archive troca-recuperacao-senha` após a validação 4.3.
