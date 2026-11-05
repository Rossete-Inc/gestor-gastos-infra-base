# 0002 - DynamoDB on-demand como banco de dados

- **Status:** Aceita
- **Data:** 2026-11-04

## Contexto

Precisamos de persistência serverless com custo próximo de zero em baixo volume. Cada microserviço deve ser dono dos seus dados.

## Decisão

Usar DynamoDB em modo on-demand, com uma tabela por microserviço (single-table design dentro de cada serviço).

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| DynamoDB on-demand (escolhida) | Paga por requisição; free tier de 25 GB; zero administração | Sem joins nem agregações SQL; modelagem orientada a acesso |
| Aurora Serverless v2 | SQL facilita relatórios | Custo mínimo mensal (~US$ 40+) |
| RDS em instância pequena | SQL, previsível | Custo fixo, não é serverless |

## Trade-offs

- **Ganhamos:** custo ~zero e escala automática.
- **Perdemos:** consultas ad hoc; agregações (totais por categoria) são feitas no serviço.
- **Quando reconsiderar:** se os relatórios ficarem complexos ou o volume por usuário tornar a soma em memória cara. Primeiro passo: materializar totais via DynamoDB Streams.

## Consequências

Valores monetários gravados como inteiro em centavos (`valorCentavos`, int64). Chaves documentadas em `docs/plano.md`.
