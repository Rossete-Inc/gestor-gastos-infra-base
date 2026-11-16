# 0009 - Cartões de crédito no expenses-service

- **Status:** Proposta
- **Data:** 2026-11-16

## Contexto

A change `cartoes-credito` cria a entidade cartão: apelido, cor, dias de fechamento e vencimento e limite. O cartão define o mês de referência de cada gasto (a fatura), entra no cálculo do limite usado e não pode ser excluído enquanto tiver gastos.

O ADR 0003 separa os microserviços por domínio: users, categories e expenses. Seguido à risca, ele levaria a um `cards-service` novo.

## Decisão

Os cartões ficam no **expenses-service**, na mesma tabela dos gastos: `PK=USER#<userId>`, `SK=CARD#<ulid>`. O contrato do serviço ganha as operações de cartão, e o BFF expõe `/cartoes`.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Cartões no expenses-service (escolhida) | Mês de referência e "não excluir com gastos" resolvidos no mesmo serviço, sem corrida; nenhum repositório, esteira, role OIDC, tabela ou Lambda novos; a exclusão do usuário já apaga a partição | O expenses-service cresce e passa a ter dois agregados |
| `cards-service` próprio | Segue o ADR 0003 à risca; deploy independente | Mais um repositório, esteira e tabela; uma chamada extra a cada lançamento para obter os dias do cartão; regra de exclusão atravessando serviços, como nas categorias |
| Cartão como categoria personalizada | Nenhuma entidade nova | Não guarda fechamento, vencimento nem limite; mistura dois eixos de classificação |

## Trade-offs

- **Ganhamos:** consistência entre o cartão e os gastos, nenhum custo novo e menos peças para manter.
- **Perdemos:** a regra "um domínio por serviço" tem uma exceção, e o expenses-service fica maior.
- **Quando reconsiderar:**
  - se os cartões ganharem regras próprias, como integração bancária, importação de fatura ou compartilhamento;
  - se outro serviço precisar ler cartões sem passar pelos gastos.

## Consequências

- O `contract` do expenses-service ganha as operações `ListarCartoes`, `CriarCartao`, `AtualizarCartao`, `ArquivarCartao`, `ExcluirCartao`, `ExisteGastoNoCartao` e `ResumoCartoes`.
- Leituras que varrem a partição do usuário passam a filtrar o prefixo `EXP#`.
- O ADR 0003 continua valendo para novos domínios. Esta é uma exceção registrada.
