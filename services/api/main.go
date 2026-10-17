// Command api serves the Crosswatc REST + SSE API.
package main

import (
	"fmt"

	"github.com/agnaldom/crosswatc/internal/version"
)

func main() {
	fmt.Printf("crosswatc-api %s (%s)\n", version.Version, version.Commit)
}
