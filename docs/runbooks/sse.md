# Stream SSE (CrosswatcSSEErrors)

- Assinantes lentos são derrubados e reconectam com `Last-Event-ID`; muitos drops → proxy/ingress com buffering ligado (precisa `X-Accel-Buffering: no`) ou timeout curto.
- 5xx em `/api/v1/stream` → `LISTEN/NOTIFY` sem conexão: verifique o PostgreSQL.
- A UI cai para polling de 15s automaticamente.
