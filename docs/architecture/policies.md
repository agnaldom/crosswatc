# Policy Validation (spec §40, §58)

Políticas em **Rego (OPA embutido)**, avaliadas em dois lugares com as mesmas regras:

1. **Portal** — `POST /resources/{id}/changes/preview` (`phase: "preview"`) e `POST /changes` (`phase: "submit"`), depois da validação de schema. Violações aparecem em `policy.violations` e bloqueiam o PR.
2. **CI do repositório de claims** — `crosswatc-policy` compara o manifesto antes/depois de um PR feito fora do portal.

## Regras padrão (`internal/policy/defaults/crosswatc.rego`)

| Regra | Bloqueia |
|---|---|
| `naming/resource-name` | nome fora do padrão DNS (minúsculas, números, `-`, 3–63) |
| `prd/no-replacement` | mudança com impacto `REPLACEMENT` em `prd` |
| `prd/justification` | justificativa < 15 caracteres em `prd` (só no submit) |
| `tags/required` | esvaziar `tags.cost-center` ou `tags.owner` |
| `tags/key-format` | chave de tag fora de `[a-z0-9-]` |
| `security/no-public-access` | habilitar `publicNetworkAccess`/`publicAccess`/`publiclyAccessible`/`allowBlobPublicAccess` |

## Input

```json
{
  "phase": "preview | submit | ci",
  "resource": {"name": "...", "owner": "...", "environment": "...", "provider": "...", "kind": "...", "state": "..."},
  "changes": [{"field": "vmSize", "path": "spec.parameters.vmSize", "from": "D2", "to": "D4", "impact": "IN_PLACE"}],
  "current": {"...": "spec.parameters atuais, valores sensíveis como [REDACTED]"},
  "justification": "..."
}
```

## Políticas da organização

Arquivos `*.rego` com `package crosswatc` adicionam regras `deny contains {"rule": ..., "message": ..., "field": ...}` (ou uma string). Na API: `CROSSWATC_POLICY_DIR`; no Helm: `policies.configMap`.

## CI do repositório de claims

```yaml
- run: go install github.com/agnaldom/crosswatc/services/policy@latest
- run: |
    for f in $(git diff --name-only origin/main...HEAD -- 'claims/**.yaml'); do
      env=$(echo "$f" | cut -d/ -f2)
      git show origin/main:"$f" > /tmp/base.yaml 2>/dev/null || : > /tmp/base.yaml
      name=$(yq '.metadata.name' "$f")
      policy -head "$f" -base /tmp/base.yaml -name "$name" -environment "$env" -policies policies/
    done
```

Saída `0` sem violações, `1` com violações, `2` erro de uso. Fora do portal o impacto é `UNKNOWN` (sem o schema do XRD).
