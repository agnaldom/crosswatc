# Webhooks rejeitados (CrosswatcGitHubWebhookFailures / CrosswatcArgoNotificationFailures)

- `401` → segredo divergente: `CROSSWATC_GITHUB_WEBHOOK_SECRET` vs. GitHub App; `CROSSWATC_ARGOCD_WEBHOOK_TOKEN` vs. argocd-notifications.
- `400` → payload/headers inválidos: confira o "Recent Deliveries" do GitHub App.
- `413` → payload acima do limite.
- Webhooks são gatilho: o resync periódico (`CROSSWATC_RESYNC_INTERVAL`) cobre entregas perdidas, mas a UI fica atrasada.
