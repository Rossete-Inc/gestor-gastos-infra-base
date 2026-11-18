# Design

## Context

Motivação e escopo: ver `proposal.md`. Requisitos: specs `metas-gastos`, `painel-gastos`, `categorias` e `aplicativo-pwa` desta change.

Estado atual relevante:

- **expenses-service.**
  - Guarda gastos em `PK=USER#<dono>`, `SK=EXP#<mesReferencia>#<data>#<ulid>`.
  - O `ResumoMensal` soma o mês na leitura (`store.DoMes` + `gasto.Resumir`), sem agregação materializada.
  - Parcelas são gravadas todas no lançamento; recorrências são criadas na leitura do mês (`gerarRecorrencias`).
  - Pontos de escrita: `Lancar`, `AtualizarGastoNoEscopo`, `ExcluirGastoNoEscopo` (`internal/app/series.go`), além de `CriarGasto`, `AtualizarGasto` e `ExcluirGasto` (`internal/app/app.go`).
  - Hoje só **consome** eventos (`UserDeleted`), não publica nenhum.
- **users-service.** Conhece os membros de cada conta (`ListarColaboradores(donoID)`, `internal/app/colaboracao.go`) e já publica `UserDeleted` (`internal/app/admin.go`).
- **BFF.**
  - Resolve a conta ativa e o papel (`naConta`, `internal/api/conta.go`; ADR 0010) e usa o fuso de Brasília (`fusoBrasilia`, `internal/api/cartoes.go`).
  - Monta o painel em `internal/api/dashboard.go`.
- **Notificações e agendamento.** O app não tem nenhum mecanismo de notificação, e a conta AWS não tem SES, SNS nem scheduler.

## Goals / Non-Goals

**Goals:**
- Detectar o cruzamento de 80%/100% no momento da escrita, sem job agendado e sem varrer contas.
- Notificar uma vez por (meta, mês, limiar), mesmo com reentrega de eventos.
- Nenhum recurso pago novo.

**Non-Goals:**
- Limiares configuráveis, metas por grupo ou meta total do mês.
- Notificação por e-mail.
- Push nativo (APNs/FCM em app de loja); só Web Push na PWA.
- Push para ocorrências recorrentes criadas na leitura do mês (o app mostra o estado, mas não notifica).
- Histórico de notificações dentro do app.

## Decisions

### 1. Metas no expenses-service, com vigências por mês

- **Item:** `PK=USER#<dono>`, `SK=META#<categoriaId>`, com os atributos:
  - `nomeCategoria`, `grupo`;
  - `vigencias`: lista ordenada de `{desde: "AAAA-MM", valorCentavos | null}`;
  - `alertas`: mapa `mes → [80, 100]` dos limiares já notificados.
- **Valor no mês M:** a última vigência com `desde ≤ M`; `null` significa sem meta.
  - **Definir no mês corrente C:** substitui a vigência `desde = C` ou acrescenta uma nova.
  - **Remover:** acrescenta `{desde: C, null}`. Assim, os meses anteriores continuam mostrando a meta que valia neles.
- **O mês corrente vem do BFF** (`mesAtual` no payload, calculado com `fusoBrasilia`), para o serviço não depender de fuso.
- **Por que no expenses-service:** é ele que soma o mês e grava os gastos. A detecção fica na mesma requisição da escrita, sem chamada entre serviços. Mesmo raciocínio do ADR 0009 para cartões.
  - *Alternativa descartada:* metas no categories-service. Cada escrita de gasto exigiria consultar outro serviço para saber a meta.
- **Nome da categoria gravado na meta:** categoria não se renomeia, e o push precisa do nome sem consultar o categories-service.
- **Exclusão do usuário:** `UserDeleted` já apaga a partição inteira, então as metas saem junto.
- **Exclusão da categoria:** o BFF chama `ExcluirMeta` depois de excluir a categoria. Esse `ExcluirMeta` apaga o item inteiro, e não acrescenta uma vigência `null`.

### 2. Detecção do cruzamento na escrita

- **Meses afetados:** depois de gravar, cada ponto de escrita junta os pares (categoria, mês) que mudaram, antes e depois da escrita:
  - lançar, inclusive parcelas e recorrência inicial;
  - editar, inclusive troca de categoria ou de mês e edição em escopo;
  - excluir.
- **Avaliação de cada par com meta vigente:**
  1. Soma o total da categoria no mês, reaproveitando `store.DoMes`.
  2. Calcula os limiares cruzados: `limiaresCruzados(total, meta)` devolve os limiares atingidos ainda não marcados em `alertas[mes]`.
  3. Grava os novos limiares com `UpdateItem` condicional (`NOT contains(alertas.#mes, :limiar)`).
     - Se a condição falhar, outra escrita concorrente já notificou, e aquele limiar sai da lista.
     - Isso deduplica e dá idempotência.
- **Excluir gasto nunca gera alerta,** e não desmarca alertas: o cenário "Limiar já notificado no mês" da spec vale.
- **Retorno:** a saída de lançar e editar passa a incluir `alertasDeMeta[]` com `{categoriaId, nomeCategoria, mes, limiar, totalCentavos, metaCentavos}`.
  - Quando 80% e 100% cruzam juntos no mesmo mês, só o de 100% vai para o aviso e o push. Os dois são marcados em `alertas`.
- **Custo:** uma Query do mês por par afetado. Uma compra em 12× tem até 12 Queries, e só quando a categoria tem meta.

### 3. Evento `MetasAtingidas`, um por escrita

- Com `alertasDeMeta` não vazio, o expenses-service publica **um** evento `MetasAtingidas{donoId, autorId, alertas[]}` no `gestor-gastos-bus`.
- O tipo vem do go-commons (`events`), e o publicador é o `events.Publisher` já usado pelo users-service.
- **Um evento por escrita**, e não um por alerta, para que um parcelamento que estoura vários meses gere **uma** notificação (cenário "Um lançamento cruza limiares em vários meses").
- **Falha na publicação:** fica registrada em log e não desfaz o gasto. O aviso no app já foi devolvido na resposta, e o push daquele cruzamento se perde (ver riscos).
- O CDK do expenses-service ganha `events:PutEvents` no barramento.

### 4. Push pelo users-service, com Web Push (VAPID) — ADR 0011

- **Assinaturas:** um item por aparelho, `PK=USER#<id>`, `SK=PUSH#<sha256(endpoint)>`, com `endpoint`, `p256dh`, `auth` e `criadoEm`.
  - O hash do endpoint torna o registro idempotente.
  - Operações: `RegistrarAssinaturaPush` e `RemoverAssinaturaPush` (pelo endpoint).
- **Consumidor de `MetasAtingidas`:**
  1. destinatários = dono + `ListarColaboradores(donoId)`, Editores e Leitores;
  2. lê as assinaturas de cada destinatário;
  3. monta um texto curto, por exemplo "Meta de Lazer: 80% usados (R$ 820,00 de R$ 1.000,00)", ou "3 metas atingidas: …" quando há várias, citando o mês quando não for o corrente;
  4. envia com `github.com/SherClockHolmes/webpush-go`.
  - Uma resposta `404`/`410` apaga a assinatura.
  - A regra no EventBridge segue o padrão do `UserDeleted`: retentativas, `MaxEventAge` de 2 h e DLQ.
- **Reentrega do evento** pode repetir o push. Isso é aceito: é raro, e o pior caso é uma notificação duplicada.
- **Chaves VAPID:**
  - par gerado uma vez, à mão;
  - a privada fica no SSM SecureString `/gestor-gastos/dev/push/vapid-privada` (parâmetro standard, chave KMS gerenciada pela AWS, sem custo) e é lida na inicialização da Lambda;
  - a pública fica no SSM String `/gestor-gastos/dev/push/vapid-publica`, que a esteira do front injeta como `EXPO_PUBLIC_VAPID_PUBLIC_KEY`, igual aos outros parâmetros que o front já lê do SSM.
- **Alternativas descartadas:**
  - SNS Mobile Push: exige FCM/APNs e app nativo.
  - E-mail: fora da decisão do usuário.
  - Enviar o push no próprio BFF: aumentaria a latência do lançamento e espalharia segredo e lógica de membros pelo BFF.

### 5. BFF

- **Rotas novas** (todas no `openapi.yaml` e em `infra/config.go` → `Rotas`):

  | Rota | Acesso | Ação |
  |---|---|---|
  | `GET /metas` | `naConta` | Metas vigentes no mês corrente, com nome, cor e valor |
  | `PUT /metas/{categoriaId}` | `naConta`, Dono/Editor | Define ou altera; valida a categoria no categories-service e manda `nomeCategoria`, `grupo` e `mesAtual` |
  | `DELETE /metas/{categoriaId}` | `naConta`, Dono/Editor | Remove do mês corrente em diante |
  | `POST /notificacoes/assinaturas` | `autenticado` | Registra o aparelho da pessoa logada |
  | `DELETE /notificacoes/assinaturas` | `autenticado` | Remove o aparelho, pelo `endpoint` |

  - As rotas de assinatura usam a pessoa logada e ignoram o `X-Conta`.
  - O Leitor recebe `403 somente_leitura` nas escritas pelo middleware já existente.
- **`/telas/dashboard`** ganha `metas[]`, vindas do `ResumoMensal`, que passa a devolver `{categoriaId, valorCentavos, totalCentavos}` do mês pedido.
  - O BFF acrescenta cor, percentual e estado, e ordena: estouradas, depois em aviso, depois as demais.
- **`POST /gastos` e `PUT /gastos/{id}`** repassam `alertasDeMeta`.
- **`DELETE /categorias/{id}`**, quando a exclusão dá certo, chama `ExcluirMeta`.

### 6. Frontend

- **Feature `metas`:**
  - tela `/metas`: lista as categorias com a meta vigente, define, altera e remove; ações escondidas por `usePodeEscrever`;
  - componente `MetasDoMes` para o Painel, com barra de progresso por meta e três cores de estado a partir dos tokens existentes;
  - acesso pelo Painel e pelo Perfil.
- **Aviso após salvar:** o `GastoForm` passa a entregar `alertasDeMeta` ao `onConcluido`, e a rota de Gastos mostra um `Banner`. Isso respeita a lição do frontend #32: o callback fica no `onSuccess` do hook, porque a tela pode desmontar com a invalidação do cache.
- **Feature `notificacoes`:**
  - switch "Notificações" no Perfil;
  - ao ligar: `Notification.requestPermission` → `registration.pushManager.subscribe({applicationServerKey})` → `POST /notificacoes/assinaturas`;
  - ao desligar: `unsubscribe` + `DELETE`;
  - o estado vem de `pushManager.getSubscription()`, por aparelho;
  - no iOS fora do modo standalone, a feature mostra a orientação para instalar.
- **Service worker (`scripts/gerar-sw.js`):**
  - `push` exibe `showNotification(título, {body, tag: 'metas', data: {url: '/painel'}})`;
  - `notificationclick` foca uma janela aberta ou abre `/painel`.

## Risks / Trade-offs

- **[Publicação do evento falha depois de gravar o gasto]** → O push daquele cruzamento se perde, já que o limiar ficou marcado. O aviso no app vem da resposta e o painel mostra o estado. Aceito para um app de estudo; um outbox resolveria, com complexidade desproporcional.
- **[Recorrência criada na leitura não notifica]** → Limitação registrada na spec. O estado no painel continua certo.
- **[iOS só recebe push com a PWA instalada (16.4+)]** → A UI explica; a validação do usuário inclui o iOS instalado.
- **[Chave VAPID trocada invalida todas as assinaturas]** → As assinaturas antigas devolvem 410 e são apagadas; as pessoas reativam no Perfil. O ADR registra que a chave não deve ser trocada sem motivo.
- **[Escrita mais lenta quando há meta]** → Uma Query por mês afetado, só para categorias com meta. Lançamentos comuns somam uma Query de poucos itens.
- **[Corrida entre dois lançamentos simultâneos]** → O update condicional garante que só um dos dois notifique cada limiar.

## Migration Plan

1. **infra-base:** ADR 0011, plano, diagramas e esta change. Criar à mão os dois parâmetros SSM da VAPID (gerar o par com `npx web-push generate-vapid-keys`).
2. **go-commons:** evento `MetasAtingidas` e nova tag.
3. **expenses-service:** metas, detecção, publicação e permissão `PutEvents` → deploy.
4. **users-service:** assinaturas, consumidor e envio → deploy.
5. **BFF:** rotas e contrato → deploy. O `cdk deploy` manual da stack `api` do infra-base registra as rotas novas na HTTP API, como nas changes anteriores.
6. **frontend:** metas, notificações e service worker → deploy (E2E na esteira).

Cada passo é aditivo: sem metas, nada muda no comportamento atual.

**Rollback:** reverter os PRs. Os itens `META#` e `PUSH#` ficam órfãos e inofensivos.
