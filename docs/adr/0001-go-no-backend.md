# 0001 - Go como linguagem do back-end

- **Status:** Aceita
- **Data:** 2026-11-04

## Contexto

O back-end roda em AWS Lambda e o objetivo é o menor custo possível. O custo de Lambda é cobrado por GB-segundo, e o cold start afeta a experiência do PWA. As opções avaliadas foram Go e Kotlin.

## Decisão

Usar Go (runtime `provided.al2023`, arm64) no BFF e em todos os microserviços.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Go (escolhida) | Cold start ~100 ms; roda bem com 128–256 MB; binário único; arm64 nativo | Linguagem nova para o time; menos açúcar sintático |
| Kotlin (JVM) | Time já domina; ecossistema rico | Cold start de 1–3 s sem SnapStart; precisa de 512 MB+; custo maior por invocação |
| Kotlin/Native ou GraalVM | Cold start baixo | Build complexo, bibliotecas limitadas |

## Trade-offs

- **Ganhamos:** menor custo e latência, deploy simples.
- **Perdemos:** familiaridade imediata da equipe com Kotlin.
- **Quando reconsiderar:** se o back sair de Lambda para containers de longa duração (ex.: ECS/Fargate), onde o cold start deixa de importar.

## Consequências

Cada serviço e o BFF são módulos Go em repositórios próprios (ADR 0006). O código compartilhado fica no módulo `gestor-gastos-go-commons`, versionado por tag semver. O CI de cada repositório roda `go test`, `go vet` e `golangci-lint`.
