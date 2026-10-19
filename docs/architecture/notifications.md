# Notificações (spec §60)

| Tipo | Evento canônico que dispara |
|---|---|
| `WAITING_APPROVAL` | `GITOPS.CHECK.SUCCEEDED` (CI verde, falta aprovação) |
| `FAILED` | `INFRA.RECONCILE.FAILED`, `GITOPS.SYNC.FAILED`, `GITOPS.PR.FAILED` |
| `OPERATIONAL` | `LIFECYCLE.OPERATIONAL` |
| `DRIFT_DETECTED` | `LIFECYCLE.DRIFT.DETECTED` |

## Fluxo

1. Um trigger em `tracker_event` (migration 00004) grava a notificação na tabela `notification` (outbox) — vale para eventos inseridos pela API e pelo worker, e é idempotente por `event_id`.
2. O worker drena a outbox (`FOR UPDATE SKIP LOCKED`), resolve os destinatários pela squad dona e envia para Teams e/ou e-mail.
3. Falha em todos os canais → nova tentativa com backoff quadrático (30 s, 2 min, 4,5 min…; 6 tentativas). Falha só em um canal não reenvia o outro (evita duplicatas). Sem destinatário → `skipped`.

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
