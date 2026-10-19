// Package version exposes build metadata injected via -ldflags.
package version

// Set at build time: -ldflags "-X github.com/agnaldom/crosswatc/internal/version.Version=v0.1.0".
var (
	Version = "dev"
	Commit  = "none"
)
