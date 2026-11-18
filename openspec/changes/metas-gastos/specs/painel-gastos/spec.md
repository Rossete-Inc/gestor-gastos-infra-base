# Spec Delta

## ADDED Requirements

### Requirement: Metas do mês no painel
Quando a conta tiver metas, o painel SHALL exibir a seção "Metas do mês" para o mês selecionado, com uma linha por meta: nome e cor da categoria, total gasto, valor da meta, percentual e estado (dentro da meta, aviso ou estourada). As metas estouradas MUST aparecer primeiro, depois as em aviso, e por fim as demais. Sem metas, a seção MUST mostrar um atalho para criar uma, salvo para o colaborador `LEITOR`.

#### Scenario: Metas com estados diferentes
- **WHEN** o mês tem a meta de "Lazer" estourada (R$ 1.050,00 de R$ 1.000,00) e a de "Combustível" em 40% (R$ 360,00 de R$ 900,00)
- **THEN** a seção mostra "Lazer" primeiro, como estourada em 105%, e "Combustível" depois, dentro da meta em 40%

#### Scenario: Conta sem metas
- **WHEN** a conta não tem nenhuma meta
- **THEN** a seção mostra um atalho para criar a primeira meta

#### Scenario: Leitor sem metas
- **WHEN** um colaborador `LEITOR` abre o painel de uma conta sem metas
- **THEN** a seção não mostra o atalho para criar meta

#### Scenario: Outro mês
- **WHEN** o usuário navega para o mês anterior no painel
- **THEN** a seção mostra o progresso das metas daquele mês, com o valor de meta que valia nele
