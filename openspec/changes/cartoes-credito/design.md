# Design

## Context

- **Grupos de gasto.** Hoje `CARTAO` e `FIXO` são grupos de categoria. O BFF grava o grupo desnormalizado no gasto (`bff/internal/api/gastos.go`, em `categoriaDoGasto` e `salvarGasto`).
- **Tabela do expenses-service.** Usa `PK=USER#<userId>` e `SK=EXP#<AAAA-MM-DD>#<ulid>`, sem GSI.
  - A listagem e o `ResumoMensal` consultam por `begins_with(SK, "EXP#AAAA-MM-")`.
  - A exclusão de usuário (`ExcluirDoUsuario`) já apaga a partição inteira.
  - O `ID` do gasto contém a data (`IDDe`).
- **Painel.** O BFF monta o painel (`internal/api/dashboard.go`) com percentuais calculados pelo maior resto.

Decisões do usuário (2026-11-16):
- **Cadastro do cartão:** apelido, cor, fechamento, vencimento e limite.
- **Mês do gasto:** o mês da fatura, nomeada pelo mês do vencimento.
- **Obrigatoriedade:** o cartão é obrigatório em `CARTAO` e opcional em `FIXO`.
- **Gastos antigos:** ficam como "Sem cartão".
- **Limite usado:** desconta as faturas não vencidas e as parcelas futuras.
- **Navegação:** nova aba "Cartões".
- **Onde fica:** os cartões ficam no expenses-service.

## Goals / Non-Goals

**Goals:**
- Visão por cartão e por fatura, sem perder a visão total.
- Nenhum recurso AWS novo e nenhuma consulta extra cara: continua uma Query por mês.

**Non-Goals:**
- Controle de pagamento de fatura, juros e pagamento parcial.
- Importar fatura (OFX/CSV).
- Cartão de débito ou conta corrente.
- Cartões adicionais compartilhados.
- Parcelamentos e recorrências, que vêm na change `parcelamentos-recorrencias`.

## Decisions

### Cartões no expenses-service (ADR 0009)
O cartão vive na mesma tabela, com `PK=USER#<userId>` e `SK=CARD#<ulid>`. Os atributos são:
- `apelido`, `apelidoNorm` (minúsculas e NFC, para a unicidade);
- `cor`, `diaFechamento`, `diaVencimento`, `limiteCentavos`;
- `arquivado`, `criadoEm` e `atualizadoEm`.

Motivos:
- A regra "não excluir cartão com gastos" e o cálculo do mês de referência ficam no mesmo serviço do gasto, sem corrida entre serviços.
- Não há repositório, esteira, role OIDC nem Lambda novos.
- A exclusão do usuário já cobre os cartões.

Alternativa descartada: um `cards-service` próprio, que seguiria à risca o ADR 0003. Custaria um repositório, uma esteira, uma tabela, uma Lambda e uma chamada a mais em cada lançamento, sem ganho de isolamento relevante, porque cartão e gasto mudam juntos. O ADR 0009 registra a exceção.

### Mês de referência e nova sort key
- **Atributo novo no gasto.** O gasto ganha `mesReferencia` (`AAAA-MM`) e `cartaoId` (opcional).
- **Nova sort key:** `EXP#<mesReferencia>#<data>#<ulid>`.
  - A Query do mês passa a ser `begins_with(SK, "EXP#AAAA-MM#")`.
  - A ordem dentro do mês continua pela data da compra.
  - O `ID` público continua opaco e passa a codificar `mesReferencia|data|ulid`.
- **Função pura `MesDaFatura(data, diaFechamento, diaVencimento) string`** em `internal/cartao`:
  1. `fech = min(diaFechamento, últimoDia(mês(data)))`.
  2. Se `dia(data) <= fech`, o mês de fechamento é o mês da data; senão, é o mês seguinte.
  3. Se `diaVencimento > diaFechamento`, o vencimento cai no mês de fechamento; senão, no mês seguinte.
  4. O mês do vencimento é o `mesReferencia`.
- **Quem calcula.** O serviço calcula o `mesReferencia` no `CriarGasto` e no `AtualizarGasto`, com o cartão lido na mesma chamada, porque mora na mesma partição. Gasto sem cartão usa o mês da data.
- **Validação do cartão.** O expenses-service valida se o cartão existe e está ativo. O BFF valida se a categoria `CARTAO` veio com cartão, porque conhece o grupo.
- **Mudar o dia de fechamento** não recalcula os gastos existentes, conforme a spec.

### Migração dos gastos existentes
O script `scripts/migrar-mes-referencia` (Go, idempotente) percorre a tabela por Scan paginado. Para cada item `EXP#` no formato antigo, faz uma `TransactWriteItems` que:
1. apaga o item antigo;
2. grava o item novo com `mesReferencia` igual ao mês da data e `cartaoId` vazio.

Ordem de implantação:
1. O código novo lê os dois formatos durante a transição. A Query do mês usa os dois prefixos (`EXP#AAAA-MM-` e `EXP#AAAA-MM#`) até a migração ser concluída.
2. Roda-se o script em `dev`.
3. Um PR seguinte remove a leitura do formato antigo.

Os volumes atuais são de teste, de dezenas de itens.

O `ID` antigo, que o front pode ter em cache de rota, deixa de existir após a migração. O efeito é aceitável: um 404 que se resolve com a lista recarregada.

### Consultas que varrem a partição
`ExisteNaCategoria` e outras leituras da partição passam a filtrar `begins_with(SK, "EXP#")`, para ignorar `CARD#` e, na próxima change, `SERIE#`.

Nova operação `ExisteGastoNoCartao(cartaoId)`, usada pela exclusão do cartão.

### Limite usado
A operação `ResumoCartoes(mes, hoje)`:
1. Lê os cartões do usuário.
2. Faz uma Query `SK BETWEEN "EXP#<mesMin>" AND "EXP#~"`.
   - `mesMin` é o mês de referência mais antigo cuja fatura ainda pode estar aberta, calculado a partir de `hoje`.
   - A Query retorna tudo do mês atual em diante, o que inclui parcelas futuras.
3. Soma por cartão, em memória:
   - `faturaDoMes`: os gastos do mês pedido;
   - `limiteUsado`: os gastos cujo vencimento (dia de vencimento no `mesReferencia`) é igual ou posterior a `hoje`.

O volume pessoal cabe numa Query. `hoje` vem do BFF, na data local do Brasil (America/Sao_Paulo).

### Contrato do serviço
- **`GastoIn` e `Gasto`** ganham `cartaoId` (opcional) e `mesReferencia` (só na saída).
- **`ListarGastos`** ganha o filtro `cartaoId`, que aceita o valor especial `SEM_CARTAO`.
- **`ResumoMensal`** ganha `totaisPorCartao` (com a chave vazia para "Sem cartão").
- **Novas operações:** `ListarCartoes`, `CriarCartao`, `AtualizarCartao`, `ArquivarCartao` (arquivar e reativar), `ExcluirCartao`, `ExisteGastoNoCartao` e `ResumoCartoes`.

### BFF
- **Lançamento.** `POST /gastos` e `PUT /gastos/{id}` aceitam `cartaoId`. Se a categoria for `CARTAO` e não houver `cartaoId`, a resposta é 422 com o campo `cartaoId`.
- **Rotas de cartão:**
  - `GET /cartoes` e `POST /cartoes`;
  - `PUT /cartoes/{id}`;
  - `PATCH /cartoes/{id}` com `{arquivado}`;
  - `DELETE /cartoes/{id}`, que devolve 409 quando há gastos.
- **`GET /telas/cartoes?mes`**: lista os cartões com a fatura do mês, o limite usado e o disponível.
- **`/telas/gastos`** ganha o parâmetro `cartaoId`. Cada item ganha `cartao{id, apelido, cor}`.
- **`/telas/gasto-form`** ganha a lista de cartões ativos.
- **`/telas/dashboard`** ganha `totaisPorCartao[]` e a visão de pizza `cartoes`, com o mesmo cálculo de percentuais.
- **Contrato.** `openapi.yaml` e `Rotas` (`infra/config.go`) mudam juntos.

### Frontend
- **Feature `cartoes`:**
  - `api/queries.ts`, com invalidação de `telas/*` após as mutações;
  - `model/cartao.ts` (Zod);
  - componentes `CartoesView`, `CartaoForm` e `CartaoDetalhe`.
- **Rotas:**
  - `(app)/(tabs)/cartoes.tsx`, a nova aba, entre Gastos e Categorias;
  - `(app)/cartoes/novo.tsx` e `(app)/cartoes/[id].tsx` (detalhe e edição).
- **Componentes reaproveitados:**
  - `MoneyField` para o limite;
  - a paleta de cores das categorias;
  - `Select` para os dias de fechamento e vencimento;
  - `MonthSelector` e `SummaryCard`;
  - uma barra de limite simples.
- **`GastoForm`:**
  - `Select` de cartão, que só aparece e é obrigatório quando a categoria é `CARTAO`;
  - para `FIXO`, "Pagar no cartão" é opcional;
  - texto auxiliar "Entra na fatura de dez/2026", calculado no front pela mesma regra (`shared/lib/fatura.ts`, com testes de tabela iguais aos do Go).
- **`GastosView`:** filtro de cartão. **`PainelView`:** bloco "Por cartão" e a opção "Cartões" no seletor da pizza.

### Detalhe da fatia (pedido do usuário em 2026-11-16)
- **Dados (BFF).** Cada `Fatia` do `/telas/dashboard` ganha:
  - `grupo`, nas visões por categoria;
  - `detalhe[]`, com `{rotulo, valorCentavos}`. Nas visões por categoria, os itens são a divisão por cartão, incluindo "Sem cartão". Na visão por cartão, a divisão por grupo.
- **Cálculo da divisão.** O `ResumoMensal` passa a devolver os totais por (categoria, cartão), somados na mesma passada em memória, sem nenhuma Query extra.
- **Interação (front).** O `PieChart` (`shared/ui`) recebe `fatiaSelecionada`/`onSelecionar`:
  - **Toque:** usa o `onPress`/`focusOnPress` do `react-native-gifted-charts`.
  - **Mouse, no navegador:** `onHoverIn`/`onHoverOut` do `Pressable`, suportado pelo React Native Web, na legenda e numa camada sobre o gráfico.
  - **Exibição:** o detalhe aparece num cartão abaixo do gráfico, e não num tooltip flutuante. Assim funciona igual no toque e no mouse, e não é cortado no celular.

## Risks / Trade-offs

- **Mudança de semântica do mês.** Totais de meses anteriores mudam para quem atribuir cartão a gastos antigos. → Esperado, já que reflete a fatura. A mudança é comunicada no roteiro de validação.
- **Migração da sort key.** → Script idempotente, leitura dupla durante a transição e tabela com `RETAIN`. Ambiente só `dev`.
- **Regra de fatura simplificada.** Não considera fins de semana e feriados (bancos movem o vencimento). → Aceito: o mês de referência não muda com isso.
- **Exceção ao ADR 0003.** → Registrada no ADR 0009, com gatilho para extrair o serviço se os cartões ganharem regras próprias (ex.: integração bancária).

## Migration Plan

1. Merge do ADR 0009 e destes artefatos no infra-base.
2. expenses-service com leitura dupla, deploy e smoke test. Depois, rodar `scripts/migrar-mes-referencia` em `dev` (`AWS_PROFILE=rossete.inc`) e conferir a contagem de itens.
3. BFF (contrato novo, campos opcionais) → deploy.
4. frontend → deploy, mesclado só depois do `cdk deploy` do BFF terminar.
5. PR no expenses-service removendo a leitura do formato antigo.

Rollback: reverter os PRs. Os itens migrados continuam legíveis pela versão com leitura dupla.

## Open Questions

Nenhuma.
