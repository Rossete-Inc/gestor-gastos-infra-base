# 0011 - Alertas de meta com Web Push

- **Status:** Aceita (2026-11-18)
- **Data:** 2026-11-18

## Contexto

A change `metas-gastos` cria metas mensais por categoria e avisa quando uma meta cruza 80% ou 100%. O aviso deve aparecer no app e chegar como notificação push, inclusive com o app fechado. O app é uma PWA (Expo web no Amplify), sem versão de loja. Hoje não há notificação de nenhum tipo, nem SES, SNS ou scheduler na conta, e o orçamento é de US$ 5 por mês.

## Decisão

1. **Detecção na escrita.**
   - As metas ficam no expenses-service, em `PK=USER#<dono>`, `SK=META#<categoriaId>`.
   - Ao gravar um gasto, o próprio serviço recalcula o total da categoria nos meses afetados.
   - Ao cruzar um limiar pela primeira vez no mês, ele marca o limiar com update condicional, para deduplicar.
   - Ele publica **um** evento `MetasAtingidas` por escrita no `gestor-gastos-bus`.
2. **Envio pelo users-service.**
   - Ele consome `MetasAtingidas` e calcula os destinatários: o dono e os colaboradores (ADR 0010).
   - Lê as assinaturas Web Push de cada destinatário, guardadas em `PK=USER#<id>`, `SK=PUSH#<sha256(endpoint)>`.
   - Envia pelo protocolo Web Push com VAPID.
   - Uma assinatura que responde 404 ou 410 é apagada.
3. **Chaves VAPID no SSM.**
   - A privada fica em `/gestor-gastos/<env>/push/vapid-privada` (SecureString), lida pelo users-service.
   - A pública fica em `/gestor-gastos/<env>/push/vapid-publica` (String), lida pela esteira do frontend.
   - O par é gerado uma vez, à mão.

## Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| Web Push com VAPID, envio no users-service (escolhida) | Funciona na PWA sem loja; gratuito; o users-service já conhece os membros da conta | Exige assinatura por aparelho; no iOS só com a PWA instalada (16.4+) |
| SNS Mobile Push (FCM/APNs) | Serviço gerenciado | Exige app nativo e contas de desenvolvedor; não atende a PWA |
| E-mail (SES ou Cognito) | Funciona em qualquer aparelho | Fora da decisão do usuário; o Cognito limita a ~50 e-mails por dia e o SES exige saída do sandbox |
| Job agendado que confere as metas | Pegaria também as recorrências criadas na leitura | Varre todas as contas; o projeto evitou scheduler (change `parcelamentos-recorrencias`) |
| Enviar o push no BFF, logo após o lançamento | Sem evento | Lançamento mais lento; segredo VAPID e lógica de membros no BFF |

## Trade-offs

- **Ganhamos:** notificação real com o app fechado, sem custo e sem app de loja. A detecção é síncrona e deduplicada.
- **Perdemos:**
  - ocorrências recorrentes criadas na leitura do mês não notificam;
  - se a publicação do evento falhar depois de gravar o gasto, o push daquele cruzamento se perde (o aviso no app continua);
  - a reentrega do evento pode duplicar um push.
- **Trocar a chave VAPID** invalida todas as assinaturas: só fazer com motivo, sabendo que as pessoas terão de reativar no Perfil.
- **Quando reconsiderar:** app nativo nas lojas (FCM/APNs), necessidade de entrega garantida (outbox) ou de notificações agendadas.

## Consequências

- **expenses-service:**
  - metas, detecção e publicação de evento, a primeira vez que o serviço publica;
  - permissão `events:PutEvents`.
- **users-service:**
  - assinaturas push e consumidor de `MetasAtingidas`;
  - dependência `github.com/SherClockHolmes/webpush-go`;
  - leitura do SSM da chave privada.
- **go-commons:** tipo e `DetailType` do evento `MetasAtingidas`.
- **BFF:** rotas `/metas` e `/notificacoes/assinaturas`.
- **Front:** feature de notificações e handlers `push` e `notificationclick` no service worker.
