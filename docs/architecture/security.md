# Segurança (spec §49)

## Identidade

| Camada | Mecanismo |
|---|---|
| Usuário → web | OIDC corporativo na borda (ex.: oauth2-proxy no ingress). O proxy server-side da web repassa `Authorization` ou `X-Forwarded-Access-Token` como Bearer. |
| web → API | Bearer JWT validado pela API (`go-oidc`: issuer, audience, assinatura via JWKS, expiração). |
| Grupos | Claim configurável (`CROSSWATC_OIDC_GROUPS_CLAIM`, default `groups`; aceita array ou string separada por espaço). `squad-<nome>` → squad; `CROSSWATC_SRE_GROUP` → leitura global. |
| Modo dev | Headers `X-Dev-*`; a configuração recusa `AUTH_MODE=dev` com `CROSSWATC_ENV=production`. |

## Autorização

- Leitura: `resource.owner ∈ squads` ou SRE. Recurso de outra squad responde **404** (não revela existência).
- Escrita (preview, changes, cancel): somente membros da squad dona. SRE **não** escreve.
- Webhooks (`/webhooks/*`) fora do OIDC; exigem HMAC SHA-256 (GitHub) ou Bearer (Argo CD).

## Dados sensíveis

- Secrets: só existência, via metadata API; valores nunca são lidos.
- Projeção: `applied_state.spec` passa por `redact.Map` — chaves como `password`, `secret`, `token`, `apiKey`, `privateKey`, `connectionString` viram `[REDACTED]`; referências (`*SecretRef`, `*SecretName`) são mantidas.
- `tracker_event`/`audit_log` append-only por grants; `raw_payload` expira em 90 dias.

## Superfície HTTP

- API: `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, CSP `default-src 'none'`, `Cache-Control: no-store`; body de mutação ≤ 64 KB; rate limit por usuário nas mutações (10/min, burst 5 → 429 + `Retry-After`).
- Web: CSP same-origin, HSTS, `nosniff`, `X-Frame-Options: DENY`, `Permissions-Policy`; `X-Powered-By` removido.

## Kubernetes

- ServiceAccount read-only (`get/list/watch`); única escrita é `Lease` no namespace.
- Pods: `runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, drop ALL; web sem token de ServiceAccount.
- NetworkPolicy (default ligado): API só recebe da web e do namespace do ingress; worker só expõe probes/métricas; web só do ingress.

## GitHub

- GitHub App com permissões mínimas: `contents: write`, `pull_requests: write`, `checks: read`, `statuses: read`, `metadata: read`.
- A chave privada vem de Secret montado read-only; nunca em variável de ambiente.
