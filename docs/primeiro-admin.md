# Primeiro administrador em dev

O cadastro público só concede o papel `cliente` (spec `gestao-usuarios`, requisito "Primeiro administrador"). O primeiro `admin` é criado por um admin da conta AWS, pelo CLI, fora do app. Este roteiro é feito **uma vez por ambiente**, depois do deploy das stacks `auth` e `api` do infra-base, do users-service e do BFF (tarefas 6.4 e 6.5).

## Como o papel `admin` é decidido

| Onde | O que vale |
|---|---|
| Cognito | O grupo `admin` do User Pool. O access token traz os grupos na claim `cognito:groups` |
| BFF | `/admin/*` só aceita token com `admin` em `cognito:groups` (403 sem ele) |
| users-service | O perfil guarda o `papel`. O `GET /me` do BFF cria o perfil sob demanda quando ele não existe, com `papel = admin` se o token estiver no grupo `admin` e `cliente` nos outros casos. Perfil existente não muda de papel |

Consequências para o roteiro:

1. **Não use o cadastro pelo app para o admin.** Ao confirmar o e-mail, o trigger Post Confirmation põe a conta no grupo `cliente` e publica `UserConfirmed`, e o users-service cria o perfil como `cliente`. Promover depois exige a área de admin (fatia 5).
2. **Crie a conta com `admin-create-user`.** A senha definida por `admin-set-user-password --permanent` não dispara o Post Confirmation: a conta fica sem grupo e sem perfil.
3. **Ponha no grupo `admin` antes do primeiro login.** O token só traz os grupos do momento em que foi emitido, e o primeiro `GET /me` grava o papel de vez.
4. **Só o grupo `admin`.** Todo usuário tem exatamente um papel; o grupo `cliente` não é necessário para o admin.

Use um e-mail diferente do que você vai usar no roteiro de cliente (por exemplo, `voce+admin@gmail.com`, se o seu provedor aceita o `+`). O e-mail precisa existir só para recuperação de senha: com `--message-action SUPPRESS` nada é enviado agora, o que também poupa a cota de ~50 e-mails/dia do Cognito.

## Comandos

Em um terminal bash (Git Bash no Windows), com credenciais de admin da conta `363218498905`:

```sh
export AWS_REGION=us-east-1
aws sts get-caller-identity --query Account --output text   # 363218498905

POOL_ID=$(aws ssm get-parameter --name /gestor-gastos/dev/auth/user-pool-id --query Parameter.Value --output text)
ADMIN_EMAIL='voce+admin@exemplo.com'   # troque
ADMIN_NOME='Seu Nome'                  # troque; o perfil exige nome

# 1. Conta já com e-mail verificado e sem convite por e-mail
aws cognito-idp admin-create-user --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" \
  --user-attributes Name=email,Value="$ADMIN_EMAIL" Name=email_verified,Value=true Name=name,Value="$ADMIN_NOME" \
  --message-action SUPPRESS

# 2. Senha definitiva (8+ caracteres, com maiúscula, minúscula e número).
#    O read -s evita que a senha fique no histórico do shell.
read -rs -p 'Senha do admin: ' ADMIN_SENHA; echo
aws cognito-idp admin-set-user-password --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" \
  --password "$ADMIN_SENHA" --permanent
unset ADMIN_SENHA

# 3. Grupo admin (antes do primeiro login)
aws cognito-idp admin-add-user-to-group --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" --group-name admin
```

## Verificação

```sh
aws cognito-idp admin-get-user --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" \
  --query "[UserStatus, Enabled]" --output text          # CONFIRMED  True
aws cognito-idp admin-list-groups-for-user --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" \
  --query "Groups[].GroupName" --output text             # admin
```

Depois, no site do Amplify (`aws ssm get-parameter --name /gestor-gastos/dev/frontend/url --query Parameter.Value --output text`), entre com o e-mail e a senha do admin e abra o **Perfil**: o papel deve ser **admin**. Esse primeiro acesso cria o perfil no users-service.

Para conferir no banco:

```sh
TABELA=$(aws ssm get-parameter --name /gestor-gastos/dev/users/table-name --query Parameter.Value --output text)
SUB=$(aws cognito-idp admin-get-user --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" \
  --query "UserAttributes[?Name=='sub'].Value | [0]" --output text)
aws dynamodb get-item --table-name "$TABELA" \
  --key "{\"PK\":{\"S\":\"USER#$SUB\"},\"SK\":{\"S\":\"PROFILE\"}}" \
  --query "Item.[papel.S, status.S]" --output text       # admin  ATIVO
```

## Se o perfil nasceu como `cliente`

Acontece se a conta entrou no app antes do passo 3, ou se foi criada pelo cadastro público. Enquanto a área de admin não existe (fatia 5), corrija à mão:

```sh
# Garante o grupo e tira o cliente, se houver
aws cognito-idp admin-add-user-to-group --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" --group-name admin
aws cognito-idp admin-remove-user-from-group --user-pool-id "$POOL_ID" --username "$ADMIN_EMAIL" --group-name cliente

# Apaga o perfil; o próximo GET /me o recria como admin
aws dynamodb delete-item --table-name "$TABELA" \
  --key "{\"PK\":{\"S\":\"USER#$SUB\"},\"SK\":{\"S\":\"PROFILE\"}}"
```

Depois, **saia e entre de novo** no app: só um token novo traz o grupo `admin`.

## Por que não automatizar

Criar o admin pelo CDK ou pela esteira poria o e-mail (dado pessoal) no código público ou em variáveis do GitHub, e a senha em algum lugar além da cabeça de quem a escolheu. Como é uma operação única por ambiente, o roteiro manual é mais simples e mais seguro.
