# Notificações (spec §60)

| Tipo | Evento canônico que dispara |
|---|---|
| `WAITING_APPROVAL` | `GITOPS.CHECK.SUCCEEDED` (CI verde, falta aprovação) |
| `FAILED` | `INFRA.RECONCILE.FAILED`, `GITOPS.SYNC.FAILED`, `GITOPS.PR.FAILED` |
| `OPERATIONAL` | `LIFECYCLE.OPERATIONAL` |
| `DRIFT_DETECTED` | `LIFECYCLE.DRIFT.DETECTED` |

## Fluxo

1. Um trigger em `tracker_event` (migration 00004) grava a notificação na tabela `notification` (outbox) — vale para eventos inseridos pela API e pelo worker, e é idempotente por `event_id`.
2. O worker reserva um lote com um único `UPDATE … SKIP LOCKED` (status `sending`, lease de 2 min) e entrega **fora de transação** — um endpoint lento não segura conexão nem locks. Se o worker cair, o lease expira e outro reprocessa.
3. Cada destinatário é um alvo (`teams:<url>`, `email:<endereço>`). Os entregues ficam em `sent_targets`; havendo falha, a nova tentativa (backoff quadrático 30 s, 2 min, 4,5 min…; 6 tentativas) só envia para os alvos que faltaram — sem duplicatas nem perdas. Sem destinatário → `skipped`.

## Roteamento (`CROSSWATC_NOTIFY_CONFIG`)

```yaml
default:                     # recebe além da squad (opcional)
  email: [plataforma@corp.com]
  kinds: [FAILED]
squads:
  payments:
    teams: ["https://prod-00.westus.logic.azure.com/workflows/..."]   # Teams Workflows / Incoming Webhook
    email: [payments@corp.com]
  cards:
    email: [cards@corp.com]
    kinds: [WAITING_APPROVAL, FAILED]
```

O arquivo contém URLs de webhook (segredos): no Helm vem de `notifications.routesSecret` (chave `routes.yaml`).

## Canais

- **Teams**: Adaptive Card 1.4 com título colorido por tipo, fatos (recurso, tipo, ambiente, squad, correlation ID) e botões *Abrir no Crosswatc* (`CROSSWATC_PORTAL_URL`) e *Pull Request*.
- **E-mail**: SMTP (`CROSSWATC_SMTP_ADDR`, `_FROM`, `_USER`, `_PASSWORD`), STARTTLS quando o servidor oferece; cabeçalhos sanitizados contra injeção.

Métrica: `crosswatc_notifications_total{channel,kind,status}`.
