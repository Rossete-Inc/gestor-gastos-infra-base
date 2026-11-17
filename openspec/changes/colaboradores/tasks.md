# Tasks

> Cada grupo indica o repositório da org Rossete-Inc. Começa depois de `parcelamentos-recorrencias` estar em `dev` (decisão do usuário). Fluxo: PR → esteira verde → merge → deploy automático em `dev` → validação do usuário em `dev`.

## 1. infra-base

- [x] 1.1 Aceitar o ADR 0010 (conta compartilhada resolvida no BFF) e atualizar `docs/plano.md` (users-svc: itens `CONTA#`, GSI2) e `docs/diagramas.md` (sequência do acesso a conta alheia e modelo de dados) e liberar o cabeçalho `X-Conta` no CORS da HTTP API (`infra/api.go`, com `cdk deploy` manual da stack da API); verificar a renderização no PR e os testes do CDK

## 2. users-service

- [ ] 2.1 Criar o domínio `internal/colaboracao`: papel, convite, membro, normalização de e-mail, limite de 5 e regras (não convidar a si mesmo, não repetir); verificar com testes de tabela
- [ ] 2.2 Implementar o store: itens `CONTA#`/`MEMBRO#`/`CONVITE#`, GSI2 (`MEMBRO#`, `EMAIL#`), aceite transacional e limpeza no `UserDeleted` (dono e colaborador), adicionando o GSI2 no CDK; verificar com testes de integração no DynamoDB Local e `cdk diff` só com o índice novo
- [ ] 2.3 Implementar app, contrato e handler das operações do design (`ObterAcesso` … `NomesDe`); verificar com testes cobrindo todos os cenários da spec `colaboracao` que dependem do serviço
- [ ] 2.4 Fazer o merge e o deploy em `dev`; verificar o smoke test verde

## 3. expenses-service

- [ ] 3.1 Gravar `criadoPor` e `atualizadoPor` (`autorId` no contrato) e aceitar o filtro `autorId` na listagem, tratando o gasto sem autor como do dono; verificar com testes de app e de integração
- [ ] 3.2 Fazer o merge e o deploy em `dev`; verificar o smoke test verde

## 4. BFF

- [ ] 4.1 Atualizar o contrato: `openapi.yaml` (cabeçalho `X-Conta`, rotas `/contas` e `/colaboradores`, `lancadoPor`/`editadoPor`, `autorId`, `/me` com contas e convites) e `Rotas`; verificar com o lint e com `TestRotasIguaisAoContrato`
- [ ] 4.2 Implementar o middleware de conta ativa (404 sem acesso, 403 `somente_leitura` em escrita, cache de 10 s) e a troca do `userId` do envelope pelo dono em gastos, cartões e categorias; verificar com testes de cada rota de escrita para Leitor e de leitura para não membro
- [ ] 4.3 Implementar as rotas de colaboração e a autoria (`NomesDe` em lote); verificar com testes cobrindo os cenários das specs `colaboracao`, `lancamento-gastos`, `categorias` e `cartoes-credito`
- [ ] 4.4 Fazer o merge e o deploy em `dev`; verificar o smoke test verde

## 5. frontend

- [ ] 5.1 Regenerar o cliente; criar o store da conta ativa e o envio do `X-Conta` no `shared/api/http.ts`, de modo que a troca de conta limpe o cache e o 404 de conta volte para a própria; verificar com testes MSW
- [ ] 5.2 Criar a feature `colaboracao`: seletor de conta, tela de colaboradores (convidar, papel, remover, cancelar), convites recebidos (aceitar, recusar, aviso no painel) e sair da conta; verificar com testes MSW dos cenários da spec `colaboracao`
- [ ] 5.3 Implementar o modo Leitor (`usePapel`, sem ações de escrita) e "lançado por" no `ExpenseItem`, com o filtro "Pessoa" no `GastosView`; verificar com testes e `npm run verify`
- [ ] 5.4 E2E `e2e/colaboradores.spec.ts`: A convida B como Editor → B aceita e lança na conta de A → A vê "lançado por B" → A muda B para Leitor → B não lança → B sai e perde o acesso; verificar que passa em `dev` na esteira
- [ ] 5.5 Fazer o merge e o deploy em `dev`; verificar a esteira verde

## 6. Validação

- [ ] 6.1 **Validação em `dev` pelo usuário** com duas contas: convidar, aceitar, alternar contas, lançar como colaborador, ver a autoria e filtrar, mudar o papel para Leitor, remover, sair, e convidar um e-mail sem conta que depois se cadastra. O usuário aprova

## Workflow follow-up

- Arquivar com `/opsx:archive colaboradores` após a validação 6.1, depois de `cartoes-credito` e `parcelamentos-recorrencias` estarem arquivadas (o delta de `cartoes-credito` exige a spec principal).
