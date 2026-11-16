# 0010 - Conta compartilhada resolvida no BFF

- **Status:** Proposta
- **Data:** 2026-11-16

## Contexto

A change `colaboradores` permite que o dono de uma conta convide outras pessoas, como Editor ou Leitor, para ver e lançar gastos na conta dele. Hoje o isolamento é por usuário em todos os serviços: o BFF repassa o `sub` do token como `userId`, e cada tabela usa `PK=USER#<userId>` (ADRs 0002, 0003 e 0004).

## Decisão

A "conta" é a partição já existente do dono. O BFF lê a conta ativa no cabeçalho `X-Conta` e confere o acesso e o papel no users-service, que guarda membros e convites (`PK=CONTA#<donoId>`, GSI2 para as consultas inversas). Depois, repassa aos serviços de dados `userId = donoId`. O autor real segue em `autorId` só onde importa (gastos).

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Partição do dono + checagem no BFF (escolhida) | Nenhuma migração de dados; expenses e categories não mudam o isolamento; regra de acesso num lugar só | O BFF vira a única barreira entre contas; uma chamada a mais por requisição a conta alheia (cache de 10 s) |
| Chave `CONTA#<id>` em todas as tabelas | Modelo explícito de conta em todos os serviços | Migrar gastos, cartões e categorias; regra de membro replicada em três serviços |
| Copiar os gastos para cada colaborador | Leitura simples | Duplicação, inconsistência e custo |

## Trade-offs

- **Ganhamos:** compartilhamento sem migração e com pouca mudança nos serviços de dados.
- **Perdemos:**
  - os serviços passam a confiar no BFF também para o acesso entre contas (já confiavam para o `userId`);
  - a remoção de um colaborador leva até 10 s para valer.
- **Quando reconsiderar:** se surgirem contas com muitos membros, permissões por recurso (por cartão ou categoria) ou acesso direto aos serviços sem passar pelo BFF.

## Consequências

- **users-service:** ganha o domínio de colaboração e o GSI2.
- **BFF:** ganha o middleware de conta ativa e testes de 403 e 404 para cada rota.
- **expenses-service:** grava a autoria.
- **Front:** envia `X-Conta` em todas as chamadas ao BFF.
