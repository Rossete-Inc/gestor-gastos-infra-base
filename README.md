# gestor-gastos-infra-base

Infraestrutura compartilhada e documentação transversal do **Gestor de Gastos**: uma plataforma para lançar gastos de cartão e gastos fixos, com gráficos por categoria, login com permissões e admin.

Todos os repositórios do sistema ficam na organização GitHub [Rossete-Inc](https://github.com/orgs/Rossete-Inc/repositories).

## O que este repositório contém

- `docs/plano.md`: plano de desenvolvimento.
- `docs/diagramas.md`: diagramas Mermaid (arquitetura, fluxos, telas, dados, deploy).
- `docs/adr/`: decisões de arquitetura transversais.
- `docs/local.md` e `local/`: ambiente local (DynamoDB Local, dynamodb-admin, cognito-local e seed). Para subir: `task local:up` e depois `task local:seed`.
- `.github/workflows/`: workflows reutilizáveis da esteira (`go-service.yml`, `frontend.yml`), usados por todos os repositórios. Como chamar e o que configurar: `docs/workflows.md`.
- `infra/`: AWS CDK (Go) com stack `cicd` (OIDC + roles de deploy), Cognito (+ trigger), API Gateway HTTP API + authorizer JWT, barramento EventBridge e AWS Budget. Os IDs são publicados no SSM Parameter Store em `/gestor-gastos/<env>/...`.

## Repositórios do sistema

O sistema **não usa monorepo**: são repositórios separados (ver [ADR 0006](docs/adr/0006-repositorios-separados.md)).

| Repositório | Papel |
|---|---|
| [`gestor-gastos-infra-base`](https://github.com/Rossete-Inc/gestor-gastos-infra-base) | Este: infra compartilhada e docs |
| [`gestor-gastos-go-commons`](https://github.com/Rossete-Inc/gestor-gastos-go-commons) | Biblioteca Go compartilhada (auth, erros, logging, client de invoke) |
| [`gestor-gastos-users-service`](https://github.com/Rossete-Inc/gestor-gastos-users-service) | Microserviço de usuários |
| [`gestor-gastos-categories-service`](https://github.com/Rossete-Inc/gestor-gastos-categories-service) | Microserviço de categorias |
| [`gestor-gastos-expenses-service`](https://github.com/Rossete-Inc/gestor-gastos-expenses-service) | Microserviço de gastos |
| [`gestor-gastos-bff`](https://github.com/Rossete-Inc/gestor-gastos-bff) | BFF: único ponto público, endpoints por tela, dono do `openapi.yaml` |
| [`gestor-gastos-frontend`](https://github.com/Rossete-Inc/gestor-gastos-frontend) | App Expo/React Native (PWA) |

**Ordem de deploy:** infra-base → serviços → BFF → frontend.

## CI/CD

Todo deploy na AWS é automático via GitHub Actions + AWS CDK, autenticado por OIDC, sem access keys ([ADR 0007](docs/adr/0007-esteira-ci-cd.md)):

- **Pull request:** validações e `cdk diff`.
- **Merge na `main`:** deploy automático em **dev**.
- **Tag `vX.Y.Z`:** deploy em **prod**, com aprovação manual e canary nas Lambdas.

Este repositório guarda os workflows reutilizáveis e a stack `cicd`, que cria o OIDC provider e as roles de deploy.
