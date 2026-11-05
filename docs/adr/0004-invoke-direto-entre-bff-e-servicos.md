# 0004 - BFF invoca os microserviços via Lambda Invoke

- **Status:** Aceita
- **Data:** 2026-11-04

## Contexto

O BFF precisa chamar os microserviços. Expor cada serviço por HTTP teria custo e superfície pública adicionais.

## Decisão

O BFF chama os serviços com `lambda:InvokeFunction` (síncrono, autorizado por IAM), por meio de um client interno na `gestor-gastos-go-commons` que esconde o transporte. O BFF descobre os ARNs dos serviços no SSM Parameter Store.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Lambda Invoke via IAM (escolhida) | Sem custo de API Gateway; sem endpoint público; auth por IAM | Acopla ao transporte Lambda |
| API Gateway privado por serviço | Padrão HTTP | Custo extra por requisição; mais infraestrutura |
| Service mesh / VPC Lattice | Recursos avançados | Custo e complexidade desproporcionais ao MVP |

## Trade-offs

- **Ganhamos:** menor custo e menos infraestrutura.
- **Perdemos:** portabilidade do transporte.
- **Quando reconsiderar:** se algum serviço sair de Lambda; nesse caso basta trocar a implementação do client interno.

## Consequências

Os serviços recebem um envelope de requisição próprio (operação + payload + `userId`), definido na `gestor-gastos-go-commons`. Eventos assíncronos (ex.: `UserDeleted`) trafegam pelo EventBridge.
