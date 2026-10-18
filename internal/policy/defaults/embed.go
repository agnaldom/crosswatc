// Package defaults embeds the built-in Rego policies.
package defaults

import "embed"

//go:embed *.rego
var FS embed.FS
