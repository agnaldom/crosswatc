# GET /resources/{id} lento (CrosswatcGetResourceSlow)

SLO: p95 < 400 ms (spec §53).

1. Traces (`resource.id`, `correlation.id`) mostram onde o tempo vai: banco ou kube (catalog).
2. Banco: `EXPLAIN ANALYZE` em `tracked_resource`/`resource_projection`; confira bloat e índices de §37.
3. Rode `make load-test` contra um ambiente de staging para reproduzir.
