# Diagramas — Gestor de Gastos

Diagramas em [Mermaid](https://mermaid.js.org/), renderizados direto pelo GitHub. Decisões de fundo: [plano](plano.md) e [ADRs](adr/).

1. [Arquitetura](#1-arquitetura)
2. [Repositórios e dependências](#2-repositórios-e-dependências)
3. [Cadastro e confirmação de e-mail](#3-cadastro-e-confirmação-de-e-mail)
4. [Login e chamada autenticada](#4-login-e-chamada-autenticada)
5. [Lançar gasto](#5-lançar-gasto)
6. [Dashboard (dados de tela)](#6-dashboard-dados-de-tela)
7. [Admin bloqueia usuário](#7-admin-bloqueia-usuário)
8. [Exclusão de usuário (evento)](#8-exclusão-de-usuário-evento)
9. [Ciclo de vida do usuário](#9-ciclo-de-vida-do-usuário)
10. [Navegação entre telas](#10-navegação-entre-telas)
11. [Modelo de dados](#11-modelo-de-dados)
12. [CI/CD e ordem de deploy](#12-cicd-e-ordem-de-deploy) (inclui deploy por OIDC)

---

## 1. Arquitetura

Só o BFF é público. Os microserviços são invocados via IAM (ADR 0004) e cada um é dono da sua tabela (ADR 0002).

```mermaid
flowchart TB
    cliente(["Cliente"])
    admin(["Admin"])

    subgraph front["gestor-gastos-frontend"]
        pwa["PWA Expo<br/>Amplify Hosting"]
    end

    subgraph base["gestor-gastos-infra-base"]
        cognito["Cognito User Pool<br/>grupos: admin, cliente"]
        trigger["Lambda Post Confirmation"]
        apigw["API Gateway HTTP API<br/>authorizer JWT"]
        bus[["EventBridge<br/>gestor-gastos-bus"]]
        ssm[("SSM Parameter Store")]
    end

    subgraph bffrepo["gestor-gastos-bff"]
        bff["BFF Lambda Go"]
    end

    subgraph svcs["Microserviços"]
        users["users-service"]
        cats["categories-service"]
        exps["expenses-service"]
        ddbU[("DynamoDB users")]
        ddbC[("DynamoDB categories")]
        ddbE[("DynamoDB expenses")]
    end

    cliente --> pwa
    admin --> pwa
    pwa -- "login SRP" --> cognito
    pwa -- "HTTPS + JWT" --> apigw
    apigw --> bff
    bff -- "Lambda Invoke (IAM)" --> users
    bff -- "Lambda Invoke (IAM)" --> cats
    bff -- "Lambda Invoke (IAM)" --> exps
    users --> ddbU
    cats --> ddbC
    exps --> ddbE
    cognito --> trigger
    trigger -- "UserConfirmed" --> bus
    users -- "UserDeleted" --> bus
    bus -- "UserConfirmed" --> users
    bus -- "UserDeleted" --> exps
    bus -- "UserDeleted" --> cats
    users -- "Admin API" --> cognito
    bff -. "lê ARNs" .-> ssm
```

## 2. Repositórios e dependências

Os repositórios não dependem de código uns dos outros, só de parâmetros (SSM), do contrato (OpenAPI) e da lib versionada (ADR 0006).

```mermaid
flowchart LR
    commons["gestor-gastos-go-commons<br/>github.com/Rossete-Inc/gestor-gastos-go-commons<br/>tag semver"]
    base["gestor-gastos-infra-base"]
    users["gestor-gastos-users-service"]
    cats["gestor-gastos-categories-service"]
    exps["gestor-gastos-expenses-service"]
    bff["gestor-gastos-bff"]
    front["gestor-gastos-frontend"]
    ssm[("SSM Parameter Store")]

    commons -- "go get" --> users
    commons -- "go get" --> cats
    commons -- "go get" --> exps
    commons -- "go get" --> bff

    base -- "pool id, API id, bus" --> ssm
    users -- "ARN" --> ssm
    cats -- "ARN" --> ssm
    exps -- "ARN" --> ssm
    ssm -- "ARNs, API id" --> bff
    ssm -- "pool id, client id" --> front

    bff -- "openapi.yaml na release" --> front
```

## 3. Cadastro e confirmação de e-mail

O trigger fica no `infra-base`, que sobe antes dos serviços. Por isso ele não chama o `users-service` direto: ele publica um evento.

```mermaid
sequenceDiagram
    autonumber
    actor U as Usuário
    participant P as PWA
    participant C as Cognito
    participant T as Trigger Post Confirmation
    participant B as EventBridge
    participant US as users-service
    participant DB as DynamoDB users

    U->>P: Preenche nome, e-mail e senha
    P->>C: SignUp
    C-->>U: E-mail com código
    U->>P: Informa o código
    P->>C: ConfirmSignUp
    C->>T: Dispara trigger
    T->>C: AdminAddUserToGroup cliente
    T->>B: PutEvents UserConfirmed
    T-->>C: OK
    C-->>P: Conta confirmada
    B->>US: UserConfirmed
    US->>DB: PutItem perfil status ATIVO
    P-->>U: Redireciona para login
```

## 4. Login e chamada autenticada

```mermaid
sequenceDiagram
    autonumber
    actor U as Usuário
    participant P as PWA
    participant C as Cognito
    participant G as API Gateway
    participant F as BFF

    U->>P: E-mail e senha
    P->>C: InitiateAuth SRP
    C-->>P: idToken, accessToken, refreshToken
    Note over P: Tokens guardados pelo Amplify Auth<br/>e injetados no cliente HTTP
    P->>G: GET /me com Bearer accessToken
    G->>G: Valida assinatura, expiração e audience
    alt token inválido
        G-->>P: 401
        P->>C: Refresh com refreshToken
    else token válido
        G->>F: Requisição + claims sub e cognito:groups
        F-->>P: 200 perfil e papel
        P-->>U: Dashboard ou área admin
    end
```

## 5. Lançar gasto

O BFF valida se a categoria existe e pertence ao usuário ou ao sistema. O `expenses-service` sempre grava sob o `userId` vindo do token, nunca do corpo da requisição.

```mermaid
sequenceDiagram
    autonumber
    actor U as Usuário
    participant P as PWA
    participant F as BFF
    participant CS as categories-service
    participant ES as expenses-service
    participant DB as DynamoDB expenses

    U->>P: Abre Novo gasto
    P->>F: GET /telas/gasto-form
    F->>CS: ListarCategorias userId
    CS-->>F: Categorias do sistema + do usuário
    F-->>P: Categorias agrupadas CARTAO e FIXO
    U->>P: Valor, data, categoria, descrição
    P->>P: Validação Zod
    P->>F: POST /gastos
    F->>CS: ObterCategoria userId, categoriaId
    alt categoria inexistente ou de outro usuário
        CS-->>F: NOT_FOUND
        F-->>P: 422
    else categoria válida
        CS-->>F: Categoria com grupo
        F->>ES: CriarGasto userId, dados, grupo
        ES->>DB: PutItem valorCentavos
        ES-->>F: Gasto criado
        F-->>P: 201
        P->>P: Invalida queries de dashboard e lista
    end
```

## 6. Dashboard (dados de tela)

O BFF chama os serviços em paralelo e devolve a tela pronta: o front não junta dados.

```mermaid
sequenceDiagram
    autonumber
    participant P as PWA
    participant F as BFF
    participant ES as expenses-service
    participant CS as categories-service

    P->>F: GET /telas/dashboard?mes=2026-11
    par em paralelo
        F->>ES: ResumoMensal userId, mes
        ES-->>F: Totais por categoriaId e grupo, últimos 5
    and
        F->>CS: ListarCategorias userId
        CS-->>F: Nomes e cores
    end
    F->>F: Junta nome e cor, calcula percentuais
    F-->>P: total, totalPorGrupo, pizza, ultimos
```

## 7. Admin bloqueia usuário

Autorização em duas camadas: o BFF exige o grupo `admin`, e o `users-service` revalida o papel do chamador.

```mermaid
sequenceDiagram
    autonumber
    actor A as Admin
    participant P as PWA
    participant F as BFF
    participant US as users-service
    participant C as Cognito
    participant DB as DynamoDB users

    A->>P: Bloquear usuário
    P->>F: PATCH /admin/usuarios/id status BLOQUEADO
    alt chamador sem grupo admin
        F-->>P: 403
    else chamador admin
        F->>US: AlterarStatus solicitante, alvo
        US->>C: AdminDisableUser
        US->>C: AdminUserGlobalSignOut
        US->>DB: UpdateItem status BLOQUEADO
        US-->>F: OK
        F-->>P: 200
    end
```

## 8. Exclusão de usuário (evento)

```mermaid
sequenceDiagram
    autonumber
    participant F as BFF
    participant US as users-service
    participant C as Cognito
    participant B as EventBridge
    participant ES as expenses-service
    participant CS as categories-service

    F->>US: ExcluirUsuario alvo
    US->>C: AdminDeleteUser
    US->>US: Marca perfil EXCLUIDO
    US->>B: PutEvents UserDeleted
    US-->>F: OK
    par consumidores independentes
        B->>ES: UserDeleted
        ES->>ES: Apaga gastos do usuário em lotes
    and
        B->>CS: UserDeleted
        CS->>CS: Apaga categorias personalizadas
    end
    Note over B,CS: Falhas vão para DLQ e são reprocessadas
```

## 9. Ciclo de vida do usuário

```mermaid
stateDiagram-v2
    [*] --> PendenteConfirmacao: SignUp
    PendenteConfirmacao --> Ativo: ConfirmSignUp
    PendenteConfirmacao --> [*]: código expirado
    Ativo --> Bloqueado: admin bloqueia
    Bloqueado --> Ativo: admin reativa
    Ativo --> Excluido: admin exclui
    Bloqueado --> Excluido: admin exclui
    Excluido --> [*]
```

## 10. Navegação entre telas

```mermaid
flowchart TD
    start(["Abre o app"]) --> sessao{"Sessão válida?"}
    sessao -- "não" --> login["Login"]
    login --> cadastro["Cadastro"]
    cadastro --> confirmar["Confirmar e-mail"]
    confirmar --> login
    login -- "sucesso" --> papel{"Grupo admin?"}
    sessao -- "sim" --> papel

    papel -- "cliente" --> dash
    papel -- "admin" --> dash

    subgraph app["Área do cliente (tabs)"]
        dash["Dashboard<br/>totais + pizza"]
        lista["Gastos<br/>filtros mês, grupo, categoria"]
        form["Novo / Editar gasto"]
        cats["Categorias"]
        perfil["Perfil"]
    end

    subgraph adm["Área admin"]
        usuarios["Usuários<br/>bloquear, reativar, papel"]
    end

    dash --> lista
    dash --> form
    lista --> form
    form -- "salvar" --> lista
    dash --> cats
    dash --> perfil
    perfil -- "somente admin" --> usuarios
    perfil -- "sair" --> login
```

## 11. Modelo de dados

Visão lógica. Cada entidade fica numa tabela de um serviço diferente, então as relações são **por ID**, sem chave estrangeira. As chaves físicas estão no [plano](plano.md#microserviços-e-dados-dynamodb-single-table-por-serviço).

```mermaid
erDiagram
    USUARIO ||--o{ GASTO : "lança"
    USUARIO ||--o{ CATEGORIA : "cria personalizadas"
    CATEGORIA ||--o{ GASTO : "classifica"

    USUARIO {
        string userId PK "sub do Cognito"
        string nome
        string email
        string papel "admin ou cliente"
        string status "ATIVO, BLOQUEADO, EXCLUIDO"
        string criadoEm
    }
    CATEGORIA {
        string categoriaId PK
        string dono "SYSTEM ou userId"
        string grupo "CARTAO ou FIXO"
        string nome
        string cor
    }
    GASTO {
        string gastoId PK "ULID"
        string userId FK
        string categoriaId FK
        string grupo "CARTAO ou FIXO"
        int valorCentavos
        string data "AAAA-MM-DD"
        string descricao
        boolean recorrente
    }
```

## 12. CI/CD e ordem de deploy

Todo deploy na AWS é automático (ADR 0007). A nuvem tem um ambiente só, `dev`, e tags não fazem deploy. A ordem só importa no primeiro deploy, por causa dos parâmetros no SSM.

```mermaid
flowchart LR
    subgraph pipeline["Esteira de cada repositório (GitHub Actions)"]
        direction LR
        pr["Pull request"] --> checks["lint + testes + build<br/>cdk synth + cdk diff"]
        checks --> merge["merge na main"]
        merge --> dev["cdk deploy dev<br/>canary nas Lambdas"]
        dev --> alarme{"alarme de erros?"}
        alarme -- "sim" --> rollback["rollback automático"]
        alarme -- "não" --> ok["100% do tráfego"]
        ok --> smoke["smoke test dev"]
    end

    subgraph ordem["Primeiro deploy em dev"]
        direction LR
        o0["0. bootstrap manual<br/>cdk bootstrap + stack cicd"] --> o1["1. infra-base"]
        o1 --> o2["2. users, categories,<br/>expenses services"]
        o2 --> o3["3. bff"]
        o3 --> o4["4. frontend"]
    end
```

### Deploy autenticado por OIDC

```mermaid
sequenceDiagram
    autonumber
    participant GH as GitHub Actions
    participant STS as AWS STS
    participant CFN as CloudFormation via CDK
    participant SSM as SSM Parameter Store
    participant CD as CodeDeploy

    GH->>GH: Gera token OIDC do job
    GH->>STS: AssumeRoleWithWebIdentity gh-deploy-repo-dev
    STS->>STS: Valida repo Rossete-Inc e environment dev
    STS-->>GH: Credenciais temporárias de 1h
    GH->>SSM: Lê parâmetros exigidos
    alt parâmetro ausente
        SSM-->>GH: ParameterNotFound
        GH->>GH: Falha com mensagem clara
    else parâmetros ok
        GH->>CFN: cdk deploy
        CFN->>CD: Publica nova versão da Lambda no alias live
        CD->>CD: Canary 10% por 5 minutos
        CD-->>CFN: Sucesso ou rollback
        CFN-->>GH: Status do deploy
        GH->>SSM: Publica ARNs e IDs atualizados
    end
```
