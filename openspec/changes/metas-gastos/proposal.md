# Proposal

## Why

O cliente vê quanto gastou em cada categoria, mas não tem como dizer quanto **pretende** gastar nem saber, a tempo, que está perto de passar do limite. Quer, por exemplo, gastar até R$ 1.000 com Lazer e até R$ 900 com Combustível, e ser avisado antes de estourar. Sem esse aviso, ele só descobre ao conferir o Painel.

## What Changes

- **Metas mensais por categoria.** O cliente define um valor máximo por mês para uma categoria, padrão ou personalizada. A meta vale todo mês até ser alterada ou removida. O total gasto zera a cada mês, e mudar o valor vale do mês atual em diante.
- **Progresso da meta.** A meta compara o valor com o total da categoria no mês de referência (o mesmo do Painel: o mês da fatura nos gastos de cartão), incluindo parcelas e recorrências daquele mês. São três estados: dentro da meta, aviso (a partir de **80%**) e estourada (a partir de **100%**). Os limiares são fixos e iguais para todas as metas.
- **Avisos no app.**
  - O Painel ganha a seção "Metas do mês", com o progresso de cada meta.
  - Ao salvar um gasto que faz uma meta cruzar 80% ou 100%, o app mostra o aviso na hora.
- **Notificação push.**
  - Cada pessoa pode ativar as notificações no aparelho (Web Push na PWA).
  - Quando um lançamento faz uma meta cruzar 80% ou 100% pela primeira vez no mês, todos os aparelhos ativados das pessoas da conta, dono e colaboradores, recebem a notificação.
  - Cada limiar notifica uma vez por meta e por mês.
- **Permissões.** Dono e Editor criam, alteram e removem metas; o Leitor só vê. A meta soma os gastos de todas as pessoas da conta.
- **Exclusão de categoria.** Excluir uma categoria personalizada remove a meta dela.

## Capabilities

### New Capabilities

- `metas-gastos`: metas mensais por categoria, cálculo do progresso, limiares de 80% e 100%, avisos no app e disparo das notificações.

### Modified Capabilities

- `painel-gastos`: novo requisito para a seção "Metas do mês".
- `categorias`: novo requisito: excluir uma categoria personalizada remove a meta dela.
- `aplicativo-pwa`: novo requisito para ativar e desativar as notificações push no aparelho e recebê-las com o app fechado.

## Impact

- **`gestor-gastos-expenses-service`:**
  - metas na partição da conta;
  - operações de listar, definir e excluir metas;
  - progresso das metas no resumo mensal;
  - detecção do cruzamento de limiar ao gravar gastos e publicação do evento `MetasAtingidas`.
- **`gestor-gastos-users-service`:** assinaturas Web Push por aparelho, consumo de `MetasAtingidas` e envio do push para o dono e os colaboradores.
- **`gestor-gastos-bff`:**
  - rotas `/metas` e `/notificacoes/assinaturas`;
  - metas no `/telas/dashboard` e alertas na resposta de criar e editar gasto;
  - remoção da meta ao excluir categoria;
  - `openapi.yaml` atualizado.
- **`gestor-gastos-go-commons`:** evento `MetasAtingidas`, com nova tag.
- **`gestor-gastos-frontend`:**
  - feature `metas` (tela de metas e seção no Painel) e aviso após salvar gasto;
  - feature `notificacoes` (ativar no Perfil);
  - handlers `push` e `notificationclick` no service worker.
- **`gestor-gastos-infra-base`:** ADR 0011 (Web Push e evento `MetasAtingidas`), plano, diagramas e o parâmetro SSM da chave VAPID.
- **Dependência nova:** `github.com/SherClockHolmes/webpush-go` no users-service.
- **Custo:** praticamente nenhum: um parâmetro SSM SecureString standard (sem custo), alguns eventos no EventBridge e algumas invocações por alerta. O envio de Web Push pelos navegadores é gratuito.
- **Limitação:** no iOS, o push só chega com a PWA instalada na tela inicial (iOS 16.4+).
