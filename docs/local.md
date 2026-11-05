# Ambiente local

Todo o Gestor de Gastos roda na sua máquina sem conta nem credencial da AWS. O DynamoDB e o Cognito são emulados em Docker, e cada serviço, o BFF e o app rodam direto do código. Cada fatia é validada aqui antes de subir para `dev`. O porquê está no design da change `mvp-gestor-gastos`, seção "Execução local e entrega em fatias".

## Pré-requisitos

| Ferramenta | Versão testada | Para quê |
| --- | --- | --- |
| Docker Desktop | 29.1 | DynamoDB Local, dynamodb-admin e cognito-local |
| Go | 1.27 | seed, serviços e BFF |
| Node.js | 24 | frontend (Expo) |
| go-task | 3.54 | comandos `task local:*` |
| AWS CLI v2 (opcional) | 2.37 | inspecionar o ambiente local pela linha de comando |

Para instalar o go-task, use `winget install Task.Task` ou `go install github.com/go-task/task/v3/cmd/task@latest`. A segunda opção coloca o binário em `%USERPROFILE%\go\bin`, que precisa estar no `PATH`.

## Subir o ambiente

Na raiz do `gestor-gastos-infra-base`:

```sh
task local:up     # sobe os 3 containers e cria User Pool, grupos e app client
task local:seed   # cria tabelas, categorias padrão e usuários de teste
```

Os dois comandos são idempotentes e podem ser rodados de novo a qualquer momento. Os dados sobrevivem a `task local:down` e só são apagados por `task local:reset`.

| Comando | O que faz |
| --- | --- |
| `task local:up` | Sobe os containers, espera ficarem saudáveis e garante o User Pool |
| `task local:seed` | Garante o User Pool, as 3 tabelas, as 12 categorias padrão e os 2 usuários de teste |
| `task local:reset` | Apaga todos os dados (`down -v`), sobe de novo e refaz o seed |
| `task local:down` | Para os containers e mantém os dados |
| `task local:logs` | Segue o log do cognito-local, onde aparece o código de confirmação |
| `task local:status` | Mostra o estado dos containers |

## Portas

| Serviço | Endereço |
| --- | --- |
| DynamoDB Local | http://localhost:8000 |
| dynamodb-admin (navegador) | http://localhost:8001 |
| cognito-local | http://localhost:9229 |
| BFF (`cmd/local`) | http://localhost:8080 (a confirmar na tarefa 5.4) |
| Expo web | http://localhost:8081 |

## IDs do Cognito local

O cognito-local gera um ID novo de User Pool e de app client a cada `task local:reset`. O seed grava os valores atuais em `local/.env.cognito`, que fica fora do Git:

```sh
COGNITO_ENDPOINT=http://localhost:9229
COGNITO_USER_POOL_ID=local_xxxxxxxx
COGNITO_CLIENT_ID=xxxxxxxxxxxxxxxxxxxxxxxxx
COGNITO_JWKS_URL=http://localhost:9229/local_xxxxxxxx/.well-known/jwks.json
COGNITO_ISSUER=http://localhost:9229/local_xxxxxxxx
```

O BFF e o `.env.local` do frontend leem esses valores. Depois de um reset, reinicie o BFF e o Expo para eles pegarem os IDs novos.

## Usuários de teste

| E-mail | Senha | Grupo |
| --- | --- | --- |
| `admin@local.test` | `Local1234` | `admin` |
| `cliente@local.test` | `Local1234` | `cliente` |

Os dois já vêm confirmados. A senha só existe no ambiente local.

Para testar o login pela linha de comando:

```sh
source local/.env.cognito
AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local AWS_REGION=us-east-1 \
aws cognito-idp initiate-auth --endpoint-url http://localhost:9229 \
  --client-id "$COGNITO_CLIENT_ID" --auth-flow USER_PASSWORD_AUTH \
  --auth-parameters USERNAME=admin@local.test,PASSWORD=Local1234
```

As credenciais `local`/`local` são fictícias. O emulador não as valida, mas o AWS CLI exige alguma.

## Código de confirmação de e-mail

O cognito-local não envia e-mail. Ao se cadastrar pelo app, rode `task local:logs` e procure o quadro **Confirmation Code Delivery**:

```
│   Username:    novo@local.test   │
│   Code:        053701            │
```

## Tabelas

Os nomes e as chaves abaixo são o contrato com o CDK de cada serviço. Se mudarem num lado, precisam mudar no outro, que é o `local/seed/dynamo.go`.

| Tabela | PK | SK | Índices |
| --- | --- | --- | --- |
| `gestor-gastos-users` | `USER#<id>` | `PROFILE` | `GSI1` (`GSI1PK=USERS`, `GSI1SK=<criadoEm>#<id>`) para a listagem do admin |
| `gestor-gastos-categories` | `SYSTEM` ou `USER#<id>` | `CAT#<grupo>#<id>` | — |
| `gestor-gastos-expenses` | `USER#<id>` | `EXP#<AAAA-MM-DD>#<ulid>` | — |

As categorias padrão ficam com `PK=SYSTEM` e IDs estáveis (`alimentacao`, `aluguel` etc.), definidos em `local/seed/dados.go`. Para inspecionar, abra o dynamodb-admin em http://localhost:8001.

## Como subir cada repositório

Cada repositório documenta o seu próprio comando no README, à medida que as fatias são feitas:

| Repositório | Comando local | Tarefa |
| --- | --- | --- |
| `users-service`, `categories-service`, `expenses-service` | `go run ./cmd/local` com `RUNTIME=local` | 5.2, 7.2, 8.2 |
| `bff` | `go run ./cmd/local` com `RUNTIME=local` e os endpoints dos serviços | 5.4 |
| `frontend` | `npx expo start --web` com `.env.local` | 5.5 |

## Diferenças entre local e nuvem

| Ponto | Na nuvem | Localmente | Consequência |
| --- | --- | --- | --- |
| Fluxo de login | SRP | `USER_PASSWORD_AUTH` (o cognito-local não implementa SRP) | O front escolhe o fluxo por variável de ambiente |
| Política de senha | Aplicada pelo Cognito | **Não aplicada** pelo cognito-local: aceita `abc123` | A regra da spec também é validada no front (Zod). O cenário "senha fraca" só vale de verdade em `dev` |
| Login inválido | `NotAuthorizedException` | `InvalidPasswordException` para senha errada | O front trata os dois como "e-mail ou senha incorretos" |
| Grupo após cadastro | O trigger Post Confirmation põe o usuário em `cliente` e publica `UserConfirmed` | Não há trigger: quem se cadastra fica **sem grupo** | O BFF só exige grupo para `admin`, então um usuário sem grupo funciona como cliente. O perfil nasce sob demanda no `/me` |
| E-mail de confirmação | Enviado pelo Cognito | Código no log (`task local:logs`) | — |
| Validade do access token | 15 min | 15 min (o emulador respeita) | — |
| Operações de admin no Cognito | Todas | Algumas não existem no emulador | Adaptador fake atrás da mesma interface (tarefa 10.1) |
| EventBridge | Barramento real com DLQ | POST direto aos consumidores em `LOCAL_EVENT_TARGETS` | Sem retentativa nem DLQ localmente |
| IDs do User Pool | Fixos no SSM | Mudam a cada reset | Releia `local/.env.cognito` após um reset |

## Problemas comuns

- **`task: command not found`:** o go-task não está no `PATH`. Veja os pré-requisitos.
- **Porta ocupada:** outro processo usa a 8000, 8001 ou 9229. Pare-o ou rode `task local:down` em outro clone.
- **Login falha depois de um reset:** o BFF ou o Expo ainda usam os IDs antigos do User Pool. Reinicie-os.
