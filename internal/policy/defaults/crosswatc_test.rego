package crosswatc_test

import rego.v1

import data.crosswatc

base := {
	"phase": "preview",
	"resource": {"name": "pgsql-payments", "owner": "payments", "environment": "dev", "provider": "azure", "kind": "PostgreSQLInstance"},
	"changes": [{"field": "vmSize", "path": "spec.parameters.vmSize", "from": "D2", "to": "D4", "impact": "IN_PLACE"}],
}

rules(inp) := {v.rule | some v in crosswatc.deny with input as inp}

test_clean_change_allowed if {
	count(crosswatc.deny) == 0 with input as base
}

test_bad_name if {
	"naming/resource-name" in rules(object.union(base, {"resource": object.union(base.resource, {"name": "PG_Payments"})}))
}

test_prd_replacement_denied if {
	inp := object.union(base, {
		"resource": object.union(base.resource, {"environment": "prd"}),
		"changes": [{"field": "diskGb", "path": "spec.parameters.diskGb", "from": 64, "to": 128, "impact": "REPLACEMENT"}],
	})
	"prd/no-replacement" in rules(inp)
}

test_prd_justification_only_on_submit if {
	prd := object.union(base, {"resource": object.union(base.resource, {"environment": "prd"})})
	not "prd/justification" in rules(prd)
	"prd/justification" in rules(object.union(prd, {"phase": "submit", "justification": "curta"}))
	not "prd/justification" in rules(object.union(prd, {"phase": "submit", "justification": "Aumento de capacidade para a Black Friday"}))
}

test_tags if {
	inp := object.union(base, {"changes": [
		{"field": "tags.cost-center", "to": "", "impact": "IN_PLACE"},
		{"field": "tags.CostCenter", "to": "x", "impact": "IN_PLACE"},
	]})
	r := rules(inp)
	"tags/required" in r
	"tags/key-format" in r
}

test_public_access_denied if {
	inp := object.union(base, {"changes": [{"field": "network.publicNetworkAccess", "to": "Enabled", "impact": "IN_PLACE"}]})
	"security/no-public-access" in rules(inp)
}
