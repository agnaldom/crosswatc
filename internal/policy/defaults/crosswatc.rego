# Default Crosswatc policies (spec §31 stage 3, §40). Evaluated on every preview and submit.
# Each rule adds {"rule": ..., "message": ..., "field": ...} to deny.
#
# input:
#   phase:        "preview" | "submit"
#   resource:     {name, owner, environment, provider, kind, state}
#   changes:      [{field, path, from, to, impact}]
#   current:      spec.parameters (sensitive values redacted)
#   justification string (submit only)
package crosswatc

import rego.v1

name_pattern := `^[a-z][a-z0-9-]{1,61}[a-z0-9]$`

required_tags := {"cost-center", "owner"}

# Keep in sync with httpapi.DefaultProductionEnvironments.
production := {"prd", "prod", "production"}

deny contains v if {
	not regex.match(name_pattern, input.resource.name)
	v := {"rule": "naming/resource-name", "message": sprintf("Nome %q fora do padrão DNS (minúsculas, números e '-', 3–63 chars)", [input.resource.name])}
}

deny contains v if {
	input.resource.environment in production
	some c in input.changes
	c.impact == "REPLACEMENT"
	v := {"rule": "prd/no-replacement", "field": c.field,
		"message": "Em produção, mudanças que recriam o recurso exigem janela com a Plataforma; abra um chamado"}
}

deny contains v if {
	input.phase == "submit"
	input.resource.environment in production
	count(trim_space(object.get(input, "justification", ""))) < 15
	v := {"rule": "prd/justification", "message": "Em produção a justificativa precisa ter ao menos 15 caracteres"}
}

deny contains v if {
	some c in input.changes
	startswith(c.field, "tags.")
	tag := substring(c.field, 5, -1)
	tag in required_tags
	trim_space(sprintf("%v", [c.to])) == ""
	v := {"rule": "tags/required", "field": c.field, "message": sprintf("A tag obrigatória %q não pode ficar vazia", [tag])}
}

deny contains v if {
	some c in input.changes
	startswith(c.field, "tags.")
	tag := substring(c.field, 5, -1)
	not regex.match(`^[a-z0-9][a-z0-9-]*$`, tag)
	v := {"rule": "tags/key-format", "field": c.field, "message": sprintf("Chave de tag %q deve ser minúscula (a-z, 0-9, '-')", [tag])}
}

public_fields := {"publicNetworkAccess", "publicAccess", "publiclyAccessible", "allowBlobPublicAccess"}

deny contains v if {
	some c in input.changes
	parts := split(c.field, ".")
	parts[count(parts) - 1] in public_fields
	c.to in {true, "true", "Enabled", "enabled"}
	v := {"rule": "security/no-public-access", "field": c.field,
		"message": "Exposição pública não pode ser habilitada pelo portal; use um endpoint privado"}
}
