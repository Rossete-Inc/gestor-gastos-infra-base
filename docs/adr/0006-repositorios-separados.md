# 0006 - Repositórios separados, sem monorepo

- **Status:** Aceita
- **Data:** 2026-11-04

## Contexto

O plano inicial previa um único repositório com `infra/`, `backend/` (via `go.work`) e `frontend/`. A decisão é não usar monorepo: cada parte do sistema deve evoluir, versionar e ser implantada de forma independente, coerente com a escolha por microserviços (ADR 0003).

## Decisão

Um repositório por deployable, mais uma biblioteca Go compartilhada, todos criados na organização GitHub [Rossete-Inc](https://github.com/orgs/Rossete-Inc/repositories):

- `gestor-gastos-infra-base`: este repositório. Guarda a documentação transversal e a infra compartilhada (Cognito, HTTP API, EventBridge, Budget).
- `gestor-gastos-go-commons`
- `gestor-gastos-users-service`
- `gestor-gastos-categories-service`
- `gestor-gastos-expenses-service`
- `gestor-gastos-bff`
- `gestor-gastos-frontend`

A integração entre repositórios acontece só por:

1. **SSM Parameter Store** (`/gestor-gastos/<env>/...`), para IDs e ARNs.
2. **Contrato OpenAPI versionado**, publicado nas releases do BFF.
3. **Tags semver** da `gestor-gastos-go-commons`.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Um repositório por deployable (escolhida) | Deploy, versão, CI e permissões independentes; fronteiras de serviço explícitas | Mudanças que atravessam serviços exigem PRs em vários repositórios; contrato e lib precisam de versionamento |
| Monorepo único | Mudanças atômicas; um só CI | Acopla ciclos de release; contraria a independência dos microserviços; descartado pelo usuário |
| Por camada (docs, frontend, backend, infra) | Menos repositórios | O backend vira um mini-monorepo com os mesmos problemas |

## Trade-offs

- **Ganhamos:** autonomia de deploy por serviço e falhas de CI isoladas. Cada serviço também é dono da própria infra (tabela, Lambda, IAM).
- **Perdemos:** a facilidade de refatorações transversais. Agora é preciso coordenar versões: bump da `go-commons`, nova release do `openapi.yaml`.
- **Quando reconsiderar:** se a maioria das mudanças passar a exigir alterar três ou mais repositórios ao mesmo tempo.

## Consequências

- **Parâmetros:** nada de CloudFormation exports entre stacks de repositórios diferentes, porque eles travam deploys independentes. Toda troca de parâmetros passa pelo SSM.
- **Ordem de deploy:** infra-base → serviços → BFF → frontend.
- **Módulos Go:** usam o path `github.com/Rossete-Inc/<repo>`, por exemplo `github.com/Rossete-Inc/gestor-gastos-go-commons`. Os repositórios são públicos (decisão de 2026-11-05, ver design.md da change `mvp-gestor-gastos`), então o CI baixa a `go-commons` sem `GOPRIVATE` nem token.
- **ADRs:** os específicos de um repositório vivem nele; os transversais ficam aqui.
- **Mudança no contrato do BFF:** gera uma nova release, e o frontend atualiza a versão fixada antes de regenerar os tipos.
