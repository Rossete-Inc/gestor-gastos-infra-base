# 0005 - react-native-gifted-charts para gráficos

- **Status:** Proposta

> Este ADR será movido para o repositório `gestor-gastos-frontend` quando ele for criado (ADR 0006).
- **Data:** 2026-11-04

## Contexto

O dashboard precisa de gráfico de pizza por grupo/categoria. Os guardrails exigem ADR para bibliotecas novas e que elas funcionem em web e nativo.

## Decisão

Usar `react-native-gifted-charts` (renderiza com `react-native-svg`), encapsulado por um componente em `shared/ui`.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| react-native-gifted-charts (escolhida) | Web e nativo via SVG; pizza/donut prontos; leve | API menos flexível que D3 |
| victory-native (Skia) | Animações ricas | Na web exige CanvasKit (bundle pesado) |
| SVG próprio com react-native-svg | Zero dependência extra | Mais código para manter (legenda, interação) |

## Trade-offs

- **Ganhamos:** gráfico pronto e consistente nas duas plataformas.
- **Perdemos:** controle fino de animações.
- **Quando reconsiderar:** se precisarmos de gráficos complexos ou a lib deixar de ser mantida.

## Consequências

Componente `PieChart` em `shared/ui`, recebendo dados já agregados pelo BFF. Validar a manutenção da lib antes de instalar.
