# Spec Delta

## Purpose

Mantém o custo de infraestrutura AWS sob controle, alertando os responsáveis antes que gastos inesperados se acumulem.

## ADDED Requirements

### Requirement: Alerta de orçamento
A conta AWS SHALL ter um orçamento mensal de US$ 5,00 que envia alerta por e-mail aos responsáveis quando o custo real atingir 80% ou a previsão do mês ultrapassar 100%.

#### Scenario: Custo real próximo do limite
- **WHEN** o custo acumulado do mês chega a US$ 4,00
- **THEN** os responsáveis recebem um e-mail de alerta

#### Scenario: Previsão acima do orçamento
- **WHEN** a previsão de custo do mês passa de US$ 5,00
- **THEN** os responsáveis recebem um e-mail de alerta

### Requirement: Retenção de logs limitada
Os logs das funções MUST ser retidos por no máximo 7 dias em `dev` e 30 dias em `prod`.

#### Scenario: Log antigo
- **WHEN** um log de uma função em `dev` completa 8 dias
- **THEN** ele não está mais armazenado
