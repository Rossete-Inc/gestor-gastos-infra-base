# 0008 - Execução local e entrega em fatias verticais

- **Status:** Aceita (atualizada em 2026-11-11 e 2026-11-12, ver a última seção)
- **Data:** 2026-11-05

## Contexto

O plano original (`docs/plano.md`) entregava o MVP **por camada**: toda a infraestrutura, depois todos os serviços Go, depois o BFF, depois o front e, no fim, o E2E. Nesse arranjo, nada fica de pé de ponta a ponta até a última fase. Um erro de contrato entre BFF e front, ou entre serviço e evento, só aparece no final, quando corrigir custa mais.

Havia também uma restrição de custo e de acesso: a conta AWS é de estudo e tem um ambiente só (ADR 0007). Desenvolver contra a nuvem exigiria credencial na máquina e um deploy a cada tentativa.

## Decisão

1. **Fatias verticais.** A entrega segue uma funcionalidade por vez, de ponta a ponta: serviço + rotas do BFF + telas do front. A ordem é Autenticação e perfil → Categorias → Gastos → Painel → Administração, seguida de PWA e E2E.
2. **Ambiente local completo, sem AWS.** O infra-base tem `local/` com DynamoDB Local e cognito-local em Docker, e um Taskfile com `task local:up`, `local:seed`, `local:reset` e `local:down`. Cada serviço e o BFF têm um `cmd/local` que serve o **mesmo** handler da Lambda por HTTP. O que muda entre a nuvem e o local está no design da change `mvp-gestor-gastos`, seção "Execução local e entrega em fatias".
3. **Base na nuvem uma vez só,** depois que a fatia 1 ficou pronta: bootstrap, OIDC, Budget, Cognito, API e barramento. As fatias seguintes só acrescentam o serviço e as rotas delas.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Fatias verticais com ambiente local (escolhida) | Cada fatia funciona de ponta a ponta cedo; contrato testado na hora; desenvolvimento sem credencial nem custo | Manter o ambiente local e as diferenças dele em relação à nuvem |
| Entrega por camada (plano original) | Cada repositório termina de uma vez | Integração só no fim; erros de contrato tardios |
| Fatias direto na nuvem, sem ambiente local | Nenhuma diferença entre o teste e a nuvem | Credencial na máquina; um deploy por tentativa; custo e espera |
| LocalStack | Emula mais serviços | Cognito e outros só na versão paga |

## Trade-offs

- **Ganhamos:** integração contínua entre as camadas, desde a fatia 1. Desenvolvimento e testes de integração sem AWS. Feedback do usuário a cada fatia, e não só no fim.
- **Perdemos:** o ambiente local não é idêntico à nuvem. O login local usa `USER_PASSWORD_AUTH` (o cognito-local não tem SRP), o trigger Post Confirmation não existe localmente e os eventos vão por HTTP direto. Esses pontos só são exercitados de verdade em `dev`.
- **Quando reconsiderar:** se o projeto ganhar um segundo ambiente (ADR 0007), o portão de validação pode voltar a ficar antes da nuvem; se o ambiente local deixar de ser usado até para testes de integração, ele pode ser aposentado.

## Consequências

- `tasks.md` da change `mvp-gestor-gastos` é organizado por fatia, e cada fatia termina com uma tarefa "Validação em `dev` pelo usuário".
- `docs/local.md` documenta como subir o ambiente local; cada repositório documenta o próprio `cmd/local` no README.
- As fases de `docs/plano.md` e o diagrama de CI/CD de `docs/diagramas.md` refletem esse fluxo.

## Atualizações de 2026-11-11 e 2026-11-12: validação em `dev`

Na decisão original, cada fatia era **validada localmente pelo usuário antes de subir para a AWS**. O template de PR tinha o item "Validado localmente", e um job da esteira reprovava o PR de funcionalidade sem ele marcado.

- **2026-11-11:** a validação do usuário passou para `dev`, depois do merge e do deploy pela esteira. O job e o item do template saíram (infra-base #20). O ambiente local continua sendo onde se desenvolve e se testa, mas deixou de ser portão para a nuvem.
- **2026-11-12:** a verificação antes do merge passou a ser só testes (unitários e de integração), typecheck, lint e a esteira do PR, sem subir o app localmente. A prova de ponta a ponta é automática e roda em `dev`: o smoke test de cada repositório, o canary nas Lambdas e, no frontend, os E2E Playwright das tarefas 12.1 e 12.2, que rodam depois de cada deploy.

O fluxo de uma fatia ficou: **PR** (testes e `cdk diff`) → **merge** → **deploy em `dev`** (canary, smoke e, no frontend, E2E) → **validação do usuário em `dev`**.

- **Ganhamos:** menos atrito para o usuário, que valida no ambiente de verdade, com SRP, trigger e eventos reais.
- **Perdemos:** um defeito que o roteiro manual pegaria localmente chega a `dev` antes. Com um ambiente só e sem usuários reais, o risco é baixo, e o rollback é o próprio canary ou um novo merge.
- **Ordem entre repositórios:** quando um PR depende de permissão nova na stack `cicd`, que tem deploy manual, a ordem é merge do PR do infra-base → `cdk deploy` da `cicd` concluído → merge do PR dependente. A esteira do dependente roda logo após o merge e falha se a permissão ainda não estiver aplicada.
