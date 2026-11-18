# Tasks

> Cada grupo indica o repositório da org Rossete-Inc. Um PR por repositório por etapa, sem PRs empilhados. Fluxo: PR → esteira verde → merge → deploy automático em `dev` → validação do usuário em `dev`.

## 1. infra-base

- [x] 1.1 Escrever e aceitar o ADR 0011 (Web Push com VAPID, assinaturas no users-service, evento `MetasAtingidas` publicado pelo expenses-service), atualizar `docs/plano.md` (itens `META#` e `PUSH#`, evento, parâmetros SSM da VAPID) e `docs/diagramas.md` (sequência lançamento → evento → push e modelo de dados); verificar a renderização no PR e o `openspec validate metas-gastos --strict`
- [ ] 1.2 Gerar o par VAPID e criar à mão `/gestor-gastos/dev/push/vapid-privada` (SecureString) e `/gestor-gastos/dev/push/vapid-publica` (String), documentando o comando no plano; verificar com `aws ssm get-parameter` (sem `--with-decryption` para a privada)

## 2. go-commons

- [ ] 2.1 Criar o tipo `MetasAtingidasDetail{donoId, autorId, alertas[]}` e o `DetailType` em `events/events.go` e publicar uma tag nova; verificar com teste de serialização e a tag disponível no `go get`

## 3. expenses-service

- [ ] 3.1 Criar o domínio `internal/meta`: vigências por mês (valor no mês, definir, remover), validação de valor > 0, estado (dentro, aviso ≥ 80%, estourada ≥ 100%) e `limiaresCruzados`; verificar com testes de tabela cobrindo os cenários de "Estados da meta", "Alterar e remover meta" e "Novo mês"
- [ ] 3.2 Implementar o store do item `META#<categoriaId>`: ler, gravar vigência, apagar e marcar limiar com update condicional, nas implementações Dynamo e memstore; verificar com testes de integração no DynamoDB Local, incluindo dois updates concorrentes do mesmo limiar
- [ ] 3.3 Expor `ListarMetas`, `DefinirMeta`, `ExcluirMeta` no contrato e no handler, e incluir as metas do mês em `ResumoMensal`; verificar com testes de app e de handler
- [ ] 3.4 Detectar cruzamentos nos pontos de escrita (lançar com parcelas e recorrência, editar em qualquer escopo, inclusive troca de categoria e de mês), devolver `alertasDeMeta` em lançar e editar e não alertar na exclusão; verificar com testes cobrindo os cenários de "Progresso da meta no mês" e "Aviso ao salvar um gasto"
- [ ] 3.5 Publicar um `MetasAtingidas` por escrita com alertas, com falha só registrada em log, e dar `events:PutEvents` no barramento pelo CDK; verificar com teste do publicador falso e teste do CDK
- [ ] 3.6 Fazer o merge e o deploy em `dev`; verificar o smoke test verde

## 4. users-service

- [ ] 4.1 Implementar as assinaturas push: item `PUSH#<sha256(endpoint)>`, `RegistrarAssinaturaPush` idempotente e `RemoverAssinaturaPush`; verificar com testes de app e de integração no DynamoDB Local
- [ ] 4.2 Consumir `MetasAtingidas`: destinatários = dono + colaboradores, texto único por evento (citando o mês quando não for o corrente), envio com `webpush-go` e a chave do SSM, e remoção da assinatura em 404/410; verificar com testes usando um servidor push falso (`httptest`) para os cenários de "Notificação push da meta"
- [ ] 4.3 Declarar no CDK a regra de `MetasAtingidas` (retentativas, `MaxEventAge` de 2 h, DLQ) e a leitura do parâmetro SSM da VAPID; verificar com os testes do CDK
- [ ] 4.4 Fazer o merge e o deploy em `dev`; verificar o smoke test verde

## 5. BFF

- [ ] 5.1 Atualizar o contrato: `openapi.yaml` (`/metas`, `/metas/{categoriaId}`, `/notificacoes/assinaturas`, `metas[]` no dashboard, `alertasDeMeta` em criar e editar gasto) e `Rotas` em `infra/config.go`; verificar com o lint e com `TestRotasIguaisAoContrato`
- [ ] 5.2 Implementar as rotas `/metas` com `naConta` (validação da categoria, `mesAtual` em `fusoBrasilia`, Leitor 403) e `/notificacoes/assinaturas` para a pessoa logada; verificar com testes de cada rota, incluindo Leitor e conta alheia
- [ ] 5.3 Incluir `metas[]` ordenadas por estado no `/telas/dashboard`, repassar `alertasDeMeta` em `POST/PUT /gastos` e chamar `ExcluirMeta` ao excluir categoria; verificar com testes cobrindo os cenários de "Metas do mês no painel" e "Excluir categoria remove a meta"
- [ ] 5.4 Fazer o merge, o deploy em `dev` e o `cdk deploy` manual da stack `api` do infra-base com as rotas novas; verificar o smoke test verde

## 6. frontend

- [ ] 6.1 Regenerar o cliente a partir do `openapi.yaml` e ler `EXPO_PUBLIC_VAPID_PUBLIC_KEY` do SSM na esteira; verificar com `npm run verify`
- [ ] 6.2 Criar a feature `metas`: tela `/metas` (definir, alterar, remover, ações escondidas para o Leitor) e `MetasDoMes` no Painel, com acesso pelo Painel e pelo Perfil; verificar com testes MSW dos cenários de "Definir meta mensal por categoria", "Permissão para gerenciar metas" e "Metas do mês no painel"
- [ ] 6.3 Mostrar o aviso após salvar gasto a partir de `alertasDeMeta`, com o callback no `onSuccess` do hook; verificar com teste MSW que simula a tela desmontando
- [ ] 6.4 Criar a feature `notificacoes` (switch no Perfil, permissão, `pushManager.subscribe`, registro e remoção no BFF, orientação no iOS sem instalar) e os handlers `push` e `notificationclick` em `scripts/gerar-sw.js`; verificar com testes cobrindo os cenários de "Ativar notificações no aparelho" (Push API simulada) e um teste do service worker gerado
- [ ] 6.5 E2E `e2e/metas.spec.ts`: definir a meta de uma categoria → lançar até 80% e ver o aviso → estourar e ver "estourada" no Painel → Leitor não vê as ações; verificar que passa em `dev` na esteira
- [ ] 6.6 Fazer o merge e o deploy em `dev`; verificar a esteira verde

## 7. Validação

- [ ] 7.1 **Validação em `dev` pelo usuário:**
  - criar, alterar e remover metas;
  - ver os avisos de 80% e 100% ao salvar e no Painel;
  - ativar notificações e receber o push no Android e no desktop, e no iOS com a PWA instalada;
  - tocar na notificação e abrir o Painel;
  - colaborador Editor gerencia metas e recebe o push; o Leitor só vê;
  - excluir uma categoria com meta.

  O usuário aprova.

## Workflow follow-up

- Arquivar com `/opsx:archive metas-gastos` após a validação 7.1, num PR do infra-base.
