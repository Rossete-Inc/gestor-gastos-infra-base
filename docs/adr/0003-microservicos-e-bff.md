# 0003 - Microserviços com BFF como único ponto público

- **Status:** Aceita
- **Data:** 2026-11-04

## Contexto

O requisito pede microserviços e um BFF que forneça os dados de tela. O front não deve conhecer a divisão interna dos serviços.

## Decisão

Três microserviços (`users`, `categories`, `expenses`), cada um dono da sua tabela. Um BFF exposto via API Gateway HTTP API com authorizer JWT do Cognito é o único ponto público; ele expõe endpoints orientados a tela e também delega as escritas.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| BFF único ponto público (escolhida) | Um só contrato para o front; menos superfície de ataque; agregação em um lugar | BFF pode virar gargalo de mudanças |
| Front chama cada serviço direto | Menos uma camada | Front acoplado aos serviços; várias chamadas por tela |
| Monolito modular em uma Lambda | Mais simples e barato | Não atende ao requisito de microserviços |

## Trade-offs

- **Ganhamos:** contrato estável (OpenAPI do BFF) e serviços que evoluem de forma independente.
- **Perdemos:** um salto a mais de latência por requisição.
- **Quando reconsiderar:** se surgir um segundo cliente com necessidades diferentes (ex.: app de admin separado), avaliar um segundo BFF.

## Consequências

O `openapi.yaml` do repositório `gestor-gastos-bff` é a fonte da verdade do contrato. Ele é publicado em cada release, e o front gera tipos com `orval` a partir de uma versão fixada (ADR 0006). Autorização em duas camadas: o BFF checa o grupo `admin` e cada serviço filtra sempre pelo `userId` recebido.
