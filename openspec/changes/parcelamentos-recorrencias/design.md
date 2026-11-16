# Design

## Context

- **Depende da change `cartoes-credito`.** De lá vêm o gasto com `cartaoId` e `mesReferencia`, a SK `EXP#<mesRef>#<data>#<ulid>`, a função `MesDaFatura` e o `ResumoCartoes` com limite usado.
- **O MVP adiou a geração de recorrentes.** O design do MVP previa EventBridge Scheduler para isso, e hoje o flag `recorrente` é só informativo. Não existe nenhum job agendado na conta.

Decisões do usuário (2026-11-16):
- Parcelamento em dois modos: pelo total, ou pela parcela com a parcela atual e o mês final.
- Recorrência automática até ser encerrada.
- Alcance esta / esta e as próximas / todas.

## Goals / Non-Goals

**Goals:**
- Parcelas e recorrências corretas na lista, nos totais, no painel e no limite, sem custo AWS novo.
- Operações idempotentes: repetir uma leitura ou uma requisição não duplica gastos.

**Non-Goals:**
- Recorrências com periodicidade diferente de mensal (semanal, anual).
- Juros, antecipação com desconto e estorno.
- Notificações de vencimento.
- Converter os gastos antigos com `recorrente=true` em séries.

## Decisions

### Entidade série
Item `PK=USER#<userId>`, `SK=SERIE#<ulid>`, com:
- `tipo`: `PARCELAMENTO` ou `RECORRENCIA`;
- `valorCentavos`, que é o valor da parcela ou da recorrência e muda com "esta e as próximas" ou "todas";
- `categoriaId`, `grupo`, `cartaoId` e `descricao`;
- `dia` e `mesInicial`;
- `mesFinal`, opcional na recorrência;
- `totalParcelas`;
- `versao`, para controle otimista.

Cada gasto da série leva `serieId`, `numero` (a parcela, ou o índice do mês na recorrência) e `ajustado`, que vale `true` quando o gasto foi editado com o alcance "só esta".

### Parcelamento criado na hora
O `CriarParcelamento` calcula todas as parcelas e grava a série junto com as N parcelas.
- A gravação usa `TransactWriteItems`, com até 100 itens. O teto de 72 parcelas cabe numa transação só.
- A data de cada parcela é a data da compra somada de k meses, com o dia limitado ao último dia do mês. O `mesReferencia` sai de `MesDaFatura`, de modo que parcelas consecutivas caem em faturas consecutivas.
- **Modo pela parcela.** As parcelas vão de `parcelaAtual` a N, a partir da data informada. O mês final, quando informado, define N.
- **Divisão pelo total.** O valor de cada parcela é `total / N`, e a 1ª parcela recebe também o resto `total % N`. A conta é toda em centavos `int64`.
- **Idempotência.** O BFF repassa o cabeçalho `Idempotency-Key` gerado pelo front. O ID da série é derivado dessa chave, e a transação usa `attribute_not_exists`. Um reenvio, portanto, não duplica a série.

### Recorrência criada na leitura, sem agendador
Ao ler um mês M (`ListarGastos`, `ResumoMensal` e `ResumoCartoes`), o serviço segue estes passos:
1. Lê as séries `RECORRENCIA` ativas do usuário, com uma Query em `begins_with(SK, "SERIE#")`. O volume é pequeno.
2. Para cada série com `mesInicial <= M <= mesFinal` (ou sem mês final), monta a ocorrência esperada de M.
3. Grava com `PutItem` condicional (`attribute_not_exists`), numa SK determinística: `EXP#<mesRef>#<data>#S<serieId>-<M>`.
   - Uma leitura repetida ou concorrente não duplica a ocorrência.
   - Uma ocorrência excluída com "só esta" vira um item com `excluido=true`, ignorado nas somas e na lista, para não reaparecer.
4. Só então executa a Query do mês.

No `ResumoCartoes`, que olha meses futuros, as recorrências são criadas apenas até o mês corrente. As ocorrências de meses futuros que já existam porque o usuário as visitou são ignoradas no limite, conforme a spec.

Alternativas descartadas:
- **EventBridge Scheduler mensal:** varreria todos os usuários e exigiria uma Lambda de job e um índice de séries por usuário ativo. Além disso, o mês só ficaria completo depois de o job rodar, o que falha ao navegar para um mês futuro.
- **Ocorrências virtuais, nunca gravadas:** complicariam a edição "só esta", a paginação e o cursor assinado.

A troca em relação ao design do MVP fica registrada aqui. Não precisa de ADR, porque nenhum ADR fixava o Scheduler.

### Alcance da edição e da exclusão
`AtualizarGasto` e `ExcluirGasto` ganham o parâmetro `escopo` (`ESTA`, `PROXIMAS` ou `TODAS`), obrigatório quando o gasto tem `serieId`.

- **`ESTA`:** altera só o item e marca `ajustado=true`. Na exclusão de uma ocorrência recorrente, grava o marcador `excluido=true`.
- **`PROXIMAS` numa recorrência:** encerra a série original com `mesFinal = M-1` e cria uma nova série a partir de M com os dados novos. As ocorrências já gravadas de M em diante que não estejam `ajustado` são reescritas (edição) ou apagadas (exclusão).
- **`PROXIMAS` num parcelamento:** reescreve ou apaga as parcelas de `numero >= n` que não estejam ajustadas.
- **`TODAS`:** atualiza a série e todos os itens não ajustados. Na exclusão, apaga a série e todos os itens.
- **Mudança de data, cartão ou categoria com `PROXIMAS` ou `TODAS`:** recalcula a data e o `mesReferencia` de cada item afetado. Como a SK muda, cada item é apagado e regravado.
- **Transações.** As operações rodam em `TransactWriteItems`, em lotes de até 100 itens. Uma recorrência com muitas ocorrências futuras visitadas é rara, e acima de 100 itens a operação segue em lotes idempotentes.

### Contrato e BFF
- **`GastoIn`** ganha `parcelamento{modo: TOTAL|PARCELA, totalCentavos?, parcelaCentavos?, parcelas?, parcelaAtual?, mesFinal?}` e `recorrencia{mesFinal?}`, mutuamente exclusivos.
  - O campo `recorrente` (booleano) continua só na saída, para os gastos antigos.
  - O switch do front passa a enviar `recorrencia`.
- **`Gasto`** ganha `serie{id, tipo, numero, total?}`, para exibir "3/10" ou "recorrente".
- **Rotas:** `PUT /gastos/{id}?escopo=` e `DELETE /gastos/{id}?escopo=`. Sem escopo num item de série, a resposta é 422.
- **Validações no BFF:** de 2 a 72 parcelas; `parcelaAtual <= parcelas`; o mês final não pode ser anterior ao mês de referência da compra.
- **Mês corrente:** calculado em America/Sao_Paulo, como em `cartoes-credito`.

### Frontend
- **`GastoForm`:**
  - `SegmentedControl` "À vista / Parcelado / Recorrente", que substitui o switch.
  - Modo **Parcelado:** alterna entre "Valor total" e "Valor da parcela". O modo "Valor da parcela" pede a parcela atual e o total ou o mês final.
  - Prévia "3× de R$ 400,00 · dez/2026 a fev/2027", calculada por `shared/lib/parcelas.ts`, que espelha a regra do Go e é testada com os mesmos casos.
  - Modo **Recorrente:** mês final opcional.
- **Alcance:** diálogo `EscopoSerieDialog` ao salvar ou excluir um item de série.
- **Lista:** selo "3/10" ou "recorrente" no `ExpenseItem`.
- **Idempotência:** `Idempotency-Key` gerada por envio do formulário.

## Risks / Trade-offs

- **Leitura com escrita:** a primeira leitura de um mês pode gravar até R itens, sendo R o número de recorrências ativas. → O volume pessoal é pequeno e o `PutItem` condicional é barato; as leituras seguintes não escrevem.
- **Ocorrências não existem antes da visita.** Exportações e relatórios de meses não abertos não as veriam. → Toda leitura de mês passa pela criação; não há outra forma de leitura.
- **Edição em série com muitos itens.** → Lotes idempotentes e testes com 72 parcelas.
- **Concorrência entre a criação na leitura e uma edição "esta e as próximas".** → `versao` na série e escrita condicional; em caso de conflito, a operação é refeita.

## Migration Plan

1. O expenses-service, com a série e a criação de ocorrências na leitura, vai para o deploy. Não há migração de dados: os gastos antigos não têm `serieId`.
2. O BFF entra com os campos novos, todos opcionais, e com `escopo`.
3. O frontend entra por último, mesclado só depois do `cdk deploy` do BFF.

**Rollback:** reverter os PRs. As séries criadas continuam na tabela e são ignoradas pela versão anterior; as ocorrências já gravadas continuam como gastos comuns.

## Open Questions

Nenhuma.
