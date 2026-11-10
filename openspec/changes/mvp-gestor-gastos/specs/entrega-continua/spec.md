# Spec Delta

## Purpose

Garante que toda mudança chegue à AWS por uma esteira automática, validada, auditável e reversível, sem deploys manuais nem credenciais de longa duração.

## ADDED Requirements

### Requirement: Execução local sem AWS
Toda funcionalidade MUST poder ser executada de ponta a ponta na máquina do desenvolvedor, com os serviços, o BFF, o app e as dependências emuladas, sem credenciais nem acesso à AWS.

#### Scenario: Subir o ambiente local
- **WHEN** o desenvolvedor sobe o ambiente local e inicia os serviços da fatia
- **THEN** ele consegue se cadastrar, entrar e usar a funcionalidade no app sem nenhuma credencial AWS configurada

#### Scenario: Mesmo código local e na nuvem
- **WHEN** uma regra de negócio é validada no ambiente local
- **THEN** o mesmo código de handler e de domínio é o que roda na AWS, mudando só o transporte (HTTP local × Lambda) e os endpoints das dependências

### Requirement: Validação local antes da nuvem
Uma funcionalidade MUST ser validada localmente pelo responsável antes de ser integrada na `main`, que é o que dispara o deploy em `dev`.

#### Scenario: Pull request sem validação local
- **WHEN** um pull request de funcionalidade não tem a validação local confirmada no checklist
- **THEN** a esteira reprova o pull request e nada é implantado

#### Scenario: Fatia validada
- **WHEN** o responsável confirma a validação local de uma fatia e o pull request é integrado
- **THEN** a fatia é implantada em `dev`

### Requirement: Validação em pull request
Todo pull request em qualquer repositório do sistema MUST executar lint, testes, build e a prévia das mudanças de infraestrutura, e MUST NOT fazer deploy.

#### Scenario: Teste quebrado
- **WHEN** um pull request contém um teste falhando
- **THEN** a esteira marca o pull request como reprovado e nada é implantado

#### Scenario: Prévia de infraestrutura
- **WHEN** um pull request altera a infraestrutura
- **THEN** a diferença de recursos fica visível no próprio pull request

### Requirement: Deploy automático em dev
Todo merge na branch `main` de um repositório implantável SHALL implantar automaticamente no ambiente `dev` e executar um smoke test.

#### Scenario: Merge na main
- **WHEN** um pull request aprovado é integrado na `main` do `gestor-gastos-expenses-service`
- **THEN** a nova versão é implantada em `dev` sem ação manual e o smoke test é executado

### Requirement: Ambiente único na nuvem
A AWS SHALL ter um único ambiente, `dev`, implantado somente a partir da `main`. Tags de versão MUST NOT disparar deploy.

#### Scenario: Tag criada
- **WHEN** a tag `v1.0.0` é criada em um repositório implantável
- **THEN** a esteira não executa nenhum deploy

#### Scenario: Deploy a partir de outra branch
- **WHEN** alguém tenta implantar em `dev` a partir de uma branch diferente da `main`
- **THEN** a esteira não executa o deploy

### Requirement: Rollback automático das funções
Em `dev`, novas versões de funções MUST receber tráfego gradualmente e MUST voltar automaticamente à versão anterior se a taxa de erros ultrapassar o limite definido.

#### Scenario: Versão com erros
- **WHEN** uma nova versão em `dev` gera erros acima do limite durante a liberação gradual
- **THEN** o tráfego volta integralmente à versão anterior sem intervenção manual

### Requirement: Sem credenciais de longa duração
A esteira MUST se autenticar na AWS com credenciais temporárias, restritas ao repositório e à finalidade: deploy em `dev` só no Environment `dev`, e prévia só de leitura no pull request. Chaves de acesso fixas MUST NOT ser usadas.

#### Scenario: Repositório não autorizado
- **WHEN** um workflow de um repositório fora da organização Rossete-Inc, ou de um job fora do Environment `dev`, tenta assumir a permissão de deploy
- **THEN** a AWS recusa a autenticação

#### Scenario: Prévia no pull request
- **WHEN** o job de prévia de infraestrutura de um pull request se autentica na AWS
- **THEN** ele recebe uma permissão só de leitura, que não consegue criar nem alterar recursos

### Requirement: Deploys não concorrentes
Dois deploys do mesmo repositório no mesmo ambiente MUST NOT executar ao mesmo tempo.

#### Scenario: Merges seguidos
- **WHEN** dois merges na `main` ocorrem em sequência rápida
- **THEN** os deploys em `dev` são executados um após o outro

### Requirement: Pré-requisitos de deploy
O deploy MUST falhar com mensagem clara quando um parâmetro exigido de outro repositório ainda não existir no ambiente.

#### Scenario: BFF antes dos serviços
- **WHEN** o BFF é implantado em um ambiente onde o expenses-service ainda não foi implantado
- **THEN** a esteira falha informando qual parâmetro está ausente, sem criar recursos parciais
