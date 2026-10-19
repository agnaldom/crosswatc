# Load testing (spec §53, §58)

## Como rodar

```bash
make db-up load-seed          # migrations + 2000 recursos sintéticos (20 squads, 20 eventos cada)
CROSSWATC_AUTH_MODE=dev CROSSWATC_DATABASE_URL=$DATABASE_URL go run ./services/api
make load-test                # k6 run test/load/api.js
```

`crosswatc-dbtool seed-load` gera recursos `load-*` com projeção, topologia, revision e eventos via SQL set-based (segundos para milhares de linhas) e recusa rodar com `CROSSWATC_ENV=production`.

## Cenário (`test/load/api.js`)

Usuários virtuais (default 50, rampa de 20s, platô de 1 min) navegam como a UI: lista paginada → detalhe → state → events → topology, cada VU numa squad diferente (AuthZ real no caminho).

| Threshold | Origem |
|---|---|
| `GET /resources/{id}` p95 < 400 ms | SLO §53 |
| `GET /resources/{id}/state` p95 < 400 ms | SLO §53 (mesma tela) |
| listas p95 < 800 ms | orçamento da tela |
| `http_req_failed` < 0,5% e checks > 99,5% | disponibilidade 99,5% |

## CI

Job `load` sobe PostgreSQL, semeia 5000 recursos, inicia a API em modo dev e roda o k6; o job falha se algum threshold estourar.

## Referência local

Medição sequencial (1 cliente, 300 requests por rota, 2000 recursos) em notebook: p95 ≤ 0,8 ms em todas as rotas de leitura. Resultados sob concorrência vêm do job `load`.

## Não coberto

- Latência evento externo → UI (< 3 s): depende de GitHub/Argo/cluster reais; acompanhada por `crosswatc_event_ingestion_lag_seconds` + `crosswatc_reconcile_duration_seconds` em produção.
- Escrita (`POST /changes`): limitada por usuário e dependente do GitHub; fora do teste de carga.
