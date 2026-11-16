# Proposal

## Why

Gastos de casa costumam ser de mais de uma pessoa: um casal ou uma família lança e acompanha os mesmos gastos. Hoje cada conta é isolada, então a única forma de dividir o controle é compartilhar a senha. Isso é inseguro e não mostra quem lançou o quê.

## What Changes

- **Convite por e-mail.** O dono de uma conta convida outra pessoa como **Editor** (lança, edita e exclui; mexe em cartões e categorias) ou como **Leitor** (só consulta). Quem já tem conta vê o convite no app e o aceita ou recusa. Quem ainda não tem conta vê o convite depois de se cadastrar com aquele e-mail. Nenhum e-mail de notificação é enviado.
- **Conta compartilhada inteira.** O colaborador vê e usa os gastos, os cartões, as categorias personalizadas e o painel do dono. Ele continua com a própria conta e alterna entre as duas por um **seletor de conta**.
- **Autoria.** Cada gasto registra e exibe quem lançou e quem editou por último. A lista de gastos ganha um filtro por pessoa.
- **Gestão pelo dono.** O dono lista colaboradores e convites, muda papéis, remove colaboradores e cancela convites. Cada conta aceita até **5 colaboradores**, e os convites pendentes contam no limite.
- **Sair e exclusão.** O colaborador pode sair da conta quando quiser. Se a conta do dono for excluída, as colaborações terminam junto com os dados.
- **BREAKING (comportamento):** o isolamento dos dados deixa de ser "só o próprio usuário" e passa a ser "a conta à qual o usuário tem acesso".

## Capabilities

### New Capabilities

- `colaboracao`: convites, aceite e recusa, papéis, limite, sair e remover, seletor de conta e autoria dos lançamentos.

### Modified Capabilities

- `lancamento-gastos`: o isolamento passa a ser por conta acessível; o Leitor não escreve; os gastos ganham autoria e o filtro por pessoa.
- `categorias`: as categorias personalizadas passam a ser visíveis para a conta, e não só para o usuário.
- `cartoes-credito`: o isolamento dos cartões passa a ser por conta.
- `gestao-usuarios`: excluir uma conta encerra as colaborações dela e as participações do usuário em outras contas.

## Impact

- **`gestor-gastos-users-service`:** itens de membro e convite, GSI2 na tabela e operações de colaboração. O consumidor de `UserDeleted` passa a limpar as colaborações.
- **`gestor-gastos-bff`:**
  - middleware de conta ativa (cabeçalho `X-Conta`), checagem de papel e rotas `/contas` e `/colaboradores`;
  - autoria em `/gastos` e `/telas/gastos`;
  - `openapi.yaml` atualizado.
- **`gestor-gastos-expenses-service`:** `criadoPor`/`atualizadoPor` nos gastos e filtro `autorId` na listagem.
- **`gestor-gastos-categories-service`:** sem mudança. O BFF repassa o dono da conta como usuário.
- **`gestor-gastos-frontend`:**
  - seletor de conta e o cabeçalho `X-Conta` no client HTTP;
  - feature `colaboracao` (colaboradores, convites recebidos, sair);
  - modo Leitor e "lançado por" na lista de gastos.
- **`gestor-gastos-infra-base`:** ADR 0010 (conta compartilhada resolvida no BFF), plano e diagramas.
- **Custo:** sem recursos novos além de um GSI na tabela do users-service, cobrado por demanda. Cada requisição a uma conta alheia faz uma consulta a mais ao users-service, com cache curto.
- **Depende de:** `cartoes-credito` (para o MODIFIED de `cartoes-credito`) e `parcelamentos-recorrencias` (ordem de entrega).
