# Design

## Context

- **Isolamento atual.** O BFF tira o `userId` da claim `sub` do token (middleware `autenticado`, `bff/internal/api/api.go`) e o repassa no envelope do invoke. Cada serviço filtra por `PK=USER#<userId>`: gastos e cartões no expenses-service, categorias personalizadas no categories-service.
- **users-service.** Guarda o perfil em `PK=USER#<id>`, `SK=PROFILE`, com o GSI1 para a listagem do admin.
- **Exclusão de usuário.** Publica `UserDeleted`, que os serviços consomem para apagar a partição.

Decisões do usuário (2026-11-16):
- o colaborador vê a conta inteira do dono;
- papéis `EDITOR` e `LEITOR`;
- convite por e-mail, aceito no app, sem notificação;
- seletor de conta;
- "lançado por", com filtro por pessoa;
- até 5 colaboradores, contando convites;
- o colaborador pode sair;
- a exclusão do dono encerra tudo.

## Goals / Non-Goals

**Goals:**
- Compartilhar a conta sem migrar dados e sem mudar o isolamento dos serviços de dados.
- Revogar acesso e mudar papel com efeito na requisição seguinte, com no máximo alguns segundos de cache.

**Non-Goals:**
- Notificação por e-mail ou push.
- Transferir a posse da conta.
- Permissões por cartão ou por categoria.
- Histórico de alterações além do último editor.
- O admin gerenciar colaborações.

## Decisions

### A conta é a partição do dono, resolvida no BFF (ADR 0010)

1. **Conta ativa.** O front manda a conta ativa no cabeçalho `X-Conta: <donoId>`. Sem o cabeçalho, ou com o próprio `sub`, vale a própria conta.
2. **Resolução no BFF.** Um middleware, logo depois de `autenticado`, resolve a conta. Se ela não for a do próprio usuário, chama `ObterAcesso(donoId, sub)` no users-service.
   - **Sem acesso:** 404 `conta_nao_encontrada`. Não revela se a conta existe.
   - **Com acesso:** põe no contexto `Conta{DonoID, Papel, AutorID: sub}`.
3. **Papel.** Rotas de escrita de gastos, cartões e categorias exigem `DONO` ou `EDITOR`; um Leitor recebe 403 `somente_leitura`. As rotas de gestão (`/colaboradores`) exigem a própria conta.
4. **Envelope.** As chamadas ao expenses-service e ao categories-service usam `userId = DonoID`. Os serviços não mudam: continuam vendo uma partição só.

**Cache.** Fica em memória na Lambda do BFF: 10 s por (sub, dono), só para respostas positivas. A remoção vale em até 10 s, o que é aceitável. Quem não é membro sempre consulta.

**Alternativa descartada:** chave `CONTA#<id>` em todas as tabelas, com o membro gravado em cada serviço. Exigiria migrar gastos, cartões e categorias e espalhar a regra de acesso por três serviços.

### Membros e convites no users-service

| Item | PK | SK | GSI2PK | Atributos |
|---|---|---|---|---|
| membro | `CONTA#<donoId>` | `MEMBRO#<userId>` | `MEMBRO#<userId>` | `papel`, `nome` (cópia), `desde` |
| convite | `CONTA#<donoId>` | `CONVITE#<email>` | `EMAIL#<email>` | `papel`, `criadoEm`, `nomeDono` |

O GSI2 (`GSI2PK`, `GSI2SK=CONTA#<donoId>`, projeção ALL) responde a "de quais contas participo" e a "quais convites recebi" sem Scan.

**Operações do contrato:**
- `ObterAcesso`
- `ListarColaboradores` (membros e convites)
- `Convidar`, `CancelarConvite`
- `ListarConvitesRecebidos`, `AceitarConvite`, `RecusarConvite`
- `MudarPapel`, `RemoverColaborador`, `SairDaConta`
- `ListarMinhasContas`
- `NomesDe(ids)`

**Regras:**
- **Limite.** Membros mais convites somam até 5. A contagem é feita pela Query da partição `CONTA#`; a corrida entre dois convites simultâneos é aceita.
- **Aceite.** Usa o e-mail **verificado do token**, obtido no BFF via `GetUser`, como já faz o `/me`, e nunca um e-mail do corpo. É uma transação: apaga o convite e grava o membro, com condição de existência.
- **Recusa.** Pode quem recebeu o convite; cancelar pode o dono. Os dois apagam o convite.
- **`UserDeleted`.** Apaga a partição `CONTA#<id>` (membros e convites da conta) e as participações via GSI2 (`MEMBRO#<id>`). O consumidor é idempotente.

### Autoria nos gastos

- O BFF envia `autorId` em `CriarGasto` e `AtualizarGasto`. O expenses-service grava `criadoPor` (só na criação) e `atualizadoPor`.
- Gastos antigos, sem `criadoPor`, são exibidos como do dono.
- **Filtro.** `ListarGastos` aceita `autorId` (`FilterExpression criadoPor = :autor`). Para o autor igual ao dono, a expressão é `criadoPor = :autor OR attribute_not_exists(criadoPor)`.
- **Nomes.** O BFF resolve os nomes com `NomesDe`, em lote, no users-service. O nome é o do perfil, que continua existindo para ex-colaboradores. Se o autor foi excluído, aparece "Ex-colaborador".

### Rotas do BFF

| Rota | Quem | Função |
|---|---|---|
| `GET /contas` | qualquer usuário | Minha conta, contas em que colaboro (dono e papel) e convites recebidos |
| `POST /contas/convites/{donoId}/aceitar` | convidado | Aceita |
| `POST /contas/convites/{donoId}/recusar` | convidado | Recusa |
| `DELETE /contas/{donoId}/participacao` | colaborador | Sai da conta |
| `GET /colaboradores` | dono | Membros e convites |
| `POST /colaboradores` | dono | Convida `{email, papel}` |
| `PATCH /colaboradores/{userId}` | dono | Muda papel |
| `DELETE /colaboradores/{userId}` | dono | Remove |
| `DELETE /colaboradores/convites/{email}` | dono | Cancela convite |

Nas telas existentes:
- `Gasto` ganha `lancadoPor{id, nome}` e `editadoPor`;
- `/telas/gastos` ganha `autorId` e a lista de pessoas da conta, para montar o filtro;
- `/me` passa a trazer `contas` e `convitesPendentes`, para o aviso no painel.
  - Decisão de implementação (2026-11-17): o `/me` continua só com o perfil. As contas e os convites vêm de `GET /contas`, que o seletor e o aviso de convites já consultam; assim o `/me` não depende do users-service para a colaboração.

### Frontend

- **Conta ativa.** Fica num store Zustand (`features/colaboracao/model/contaAtiva.ts`), persistido no armazenamento local. O mutator `shared/api/http.ts` lê a conta por um provedor configurável, no mesmo padrão de `configurarAutenticacao`, e envia o `X-Conta`.
  - Decisão de implementação (2026-11-17): sem Zustand, que o projeto não usa. É um store com assinantes, como o da sessão, lido por `useSyncExternalStore` e persistido por `shared/platform/preferencias`.
- **Troca de conta.** Faz `queryClient.clear()` e leva ao painel. Um 404 `conta_nao_encontrada` volta para a própria conta e mostra um aviso.
  - Decisão de implementação (2026-11-17): usa `resetQueries()`, que recarrega as telas abertas em vez de deixá-las vazias. Como a tela que pediu a troca pode desmontar (vira "Carregando"), a navegação depois de aceitar convite ou sair da conta fica no `onSuccess` do hook, e não no `mutate`, cujos callbacks o TanStack Query descarta ao desmontar (frontend #32).
- **Telas:**
  - seletor no `Header`, que só aparece se houver mais de uma conta;
  - Perfil → "Colaboradores", para o dono;
  - "Convites", com um aviso no painel quando houver pendentes;
  - "Sair desta conta", no Perfil, quando a conta ativa é alheia.
- **Modo Leitor.** O hook `usePapel()` esconde o FAB, as ações de editar e excluir e os formulários de cartão e categoria.
- **Autoria.** O `ExpenseItem` mostra "por Bruno" quando o autor não é o usuário; o `GastosView` ganha o filtro "Pessoa".

## Risks / Trade-offs

- **Acesso removido continua por até 10 s.** É o cache do BFF. → Prazo curto e aceito; um TTL zero custaria uma chamada por requisição.
- **Latência extra em conta alheia.** Uma invocação ao users-service por requisição sem cache. → Cache curto e chamada em paralelo com o resto quando possível.
- **O BFF passa a ser a única barreira de acesso entre contas.** Os serviços confiam no envelope. → É o mesmo modelo de hoje (ADR 0004); os testes do BFF cobrem 404, 403 e escrita do Leitor em cada rota.
- **E-mail do convite digitado errado.** O convite fica pendente para ninguém. → O dono vê e cancela; a vaga conta no limite.

## Migration Plan

1. users-service: tabela com o GSI2 (mudança aditiva no CDK) e operações novas → deploy.
2. expenses-service: autoria e filtro `autorId` → deploy. Os gastos antigos ficam sem `criadoPor`.
3. BFF: middleware, rotas e contrato → deploy.
4. frontend → deploy. Mesclar só depois do `cdk deploy` do BFF.

**Rollback:** reverter os PRs. Os itens `CONTA#` ficam órfãos e inofensivos.

## Open Questions

Nenhuma.
