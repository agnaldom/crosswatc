# Observabilidade (spec §52–54)

## Métricas

`crosswatc-api` expõe `/metrics` em `:9090` (`CROSSWATC_METRICS_ADDR`, fora do Service público); `crosswatc-worker` em `:8081/metrics`.

| Métrica | Tipo | Origem |
|---|---|---|
| `crosswatc_resources_total{state}` | gauge | worker, a cada resync |
| `crosswatc_operations_total{type}` / `crosswatc_operations_failed_total{type,reason}` | counter | API (UPDATE) e worker (CREATE, falhas) |
| `crosswatc_operations_stuck_total` | counter | worker |
| `crosswatc_revision_duration_seconds{kind,state}` | histogram | início → estado terminal |
| `crosswatc_stage_duration_seconds{state}` | histogram | tempo em cada estado da revision |
| `crosswatc_event_ingestion_lag_seconds{source}` | histogram | webhook recebido → processado |
| `crosswatc_reconcile_duration_seconds`, `crosswatc_resync_duration_seconds`, `crosswatc_reconcile_errors_total` | | worker |
| `crosswatc_sse_connections`, `crosswatc_sse_dropped_total` | | API |
| `crosswatc_drift_total`, `crosswatc_drift_active` | | worker |
| `crosswatc_informer_restarts_total{watch}` | counter | worker |
| `crosswatc_webhook_failures_total{source,reason}` | counter | API |
| `crosswatc_http_request_duration_seconds{method,route,code}` | histogram | API |

## Tracing

OpenTelemetry com exportador OTLP/HTTP habilitado por `OTEL_EXPORTER_OTLP_ENDPOINT` (Helm: `observability.otlpEndpoint`). Spans: requests HTTP (otelgin), `reconciler.resync`, `reconciler.resource` com `resource.id`, `correlation.id`, `owner`, `environment`, `provider`, `kind`.

## SLOs e alertas

Helm (`observability.prometheusRule.enabled`) cria recording rules para disponibilidade, p95 de `GET /resources/{id}`, lag de ingestão e de reconciliação, e os alertas do §54 com `runbook_url` para `docs/runbooks/`. Dashboard Grafana em `deploy/helm/crosswatc/dashboards/crosswatc.json` (`observability.grafanaDashboard.enabled`).
