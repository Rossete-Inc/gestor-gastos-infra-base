# 0007 - Esteira de CI/CD com deploy automático na AWS

- **Status:** Aceita (atualizada em 2026-11-10: ambiente único, ver a última seção)
- **Data:** 2026-11-04

## Contexto

O sistema tem sete repositórios separados (ADR 0006). Fazer deploy manual de cada um seria lento, propenso a erro e dependeria de credenciais na máquina de alguém. Toda a infraestrutura e o código devem chegar à AWS de forma automática, repetível e auditável.

## Decisão

Cada repositório tem uma esteira no **GitHub Actions**, que sobe automaticamente na AWS tudo o que ele possui, via **AWS CDK**:

- **Autenticação:** OIDC do GitHub na AWS, com role específica por repositório e ambiente. Não há access keys guardadas.
- **Pull request:** validações e `cdk diff`, sem deploy.
- **Merge na `main`:** deploy automático em **dev**, seguido de smoke test.
- **Tag `vX.Y.Z`:** deploy em **prod**, com aprovação manual no GitHub Environment `prod`.
- **Lambdas em prod:** publicadas por alias com **CodeDeploy canary** e rollback automático por alarme.

O único passo manual é o bootstrap da conta, feito uma vez (ver "Consequências").

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| GitHub Actions + OIDC + CDK (escolhida) | Já está onde o código vive; free tier generoso; sem segredos de longa duração; o mesmo CDK roda local e no CI | Os minutos do Actions podem acabar em repositórios privados com muito uso |
| AWS CodePipeline + CodeBuild | Tudo dentro da AWS | Custo por pipeline ativo; mais infraestrutura por repositório; experiência de PR pior |
| Deploy manual com `cdk deploy` | Zero configuração | Não repetível, depende de credenciais locais, sem trilha de auditoria |
| Access keys de IAM em secrets do GitHub | Simples | Chave de longa duração; risco de vazamento; rotação manual |

## Trade-offs

- **Ganhamos:** deploy automático e auditável em cada merge, sem credenciais fixas. Rollback automático em prod.
- **Perdemos:** um bootstrap inicial manual e a necessidade de manter workflows em sete repositórios. Para isso, criamos workflows reutilizáveis.
- **Quando reconsiderar:** se os minutos do GitHub Actions estourarem, ou se a organização exigir que tudo rode dentro da AWS.

## Consequências

- **Bootstrap manual, uma única vez por conta, feito por um admin:** rodar `cdk bootstrap` e o primeiro deploy da stack `cicd` do `gestor-gastos-infra-base`. Essa stack cria o OIDC provider do GitHub e as roles `gh-deploy-<repo>-<env>`.
- **Escopo das roles:** a trust policy de cada role aceita só `repo:Rossete-Inc/<repo>:environment:<env>`. A role só pode assumir as roles de bootstrap do CDK.
- **Ambientes:** começamos com **dev** e **prod** na mesma conta, separados por prefixo nos nomes dos recursos e por `/gestor-gastos/<env>/` no SSM. Contas separadas ficam para quando houver usuários reais.
- **Concorrência:** um `concurrency` group por repositório e ambiente evita dois deploys simultâneos.
- **Workflows compartilhados:** os workflows reutilizáveis (Go + CDK, frontend) ficam no `gestor-gastos-infra-base` em `.github/workflows/` e são chamados pelos outros repositórios com `uses: Rossete-Inc/gestor-gastos-infra-base/.github/workflows/<nome>.yml@<tag>`.
- **`go-commons`:** não tem deploy. A esteira só testa e publica a tag.
- **Frontend:** `expo export -p web` e deploy no Amplify Hosting pela API (`aws amplify create-deployment`). O fluxo é o mesmo dos demais repositórios: dev no merge, prod na tag.
- **Pré-requisitos no SSM:** se um parâmetro exigido no SSM não existir (ex.: o BFF antes dos serviços), a esteira falha com uma mensagem clara. Isso garante a ordem do primeiro deploy.

## Atualização de 2026-11-10: ambiente único e role só leitura para o diff

**Motivo:** é um projeto de estudo e precisa ficar barato. Manter `dev` e `prod` dobra os recursos com custo fixo e o trabalho de cada deploy, sem usuários reais que justifiquem a separação.

O que muda em relação à decisão acima:

- **Um ambiente só:** a AWS (us-east-1) tem apenas **dev**. Não existe mais `prod`, nem o GitHub Environment `prod`.
- **Deploy:** o merge na `main` implanta em `dev` (Environment `dev`, restrito à branch `main`, sem aprovação manual). Tags `vX.Y.Z` não fazem deploy. A `go-commons` continua publicando tags semver, porque é uma biblioteca.
- **Rollback:** o alias `live` e o canary do CodeDeploy com rollback automático passam a valer em `dev`.
- **Roles:** a de deploy é `gh-deploy-<repo>-dev`, com trust `repo:Rossete-Inc/<repo>:environment:dev`.
- **`cdk diff` no PR:** deixa de usar o Environment `dev` e a role de deploy. A stack `cicd` cria, para cada repositório com CDK (infra-base, users-service, categories-service, expenses-service e BFF), a role só leitura `gh-diff-<repo>`, com trust `repo:Rossete-Inc/<repo>:pull_request` (o `sub` de um job de `pull_request` sem `environment:`). Ela só assume a lookup role do bootstrap do CDK e lê templates do CloudFormation e o SSM `/gestor-gastos/*`. O diff roda com `--method=template`, porque criar change set exigiria a role de deploy. Assim, um PR de branch do próprio repositório não alcança permissão de escrita na conta.
- **Retenção de logs:** 7 dias, a regra de `dev`.

Um segundo ambiente fica para uma decisão futura, com um ADR próprio, se o projeto passar a ter usuários reais.
