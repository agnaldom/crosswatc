// Command worker runs discovery, adapters and the state reconciler.
package main

import (
	"fmt"

	"github.com/agnaldom/crosswatc/internal/version"
)

func main() {
	fmt.Printf("crosswatc-worker %s (%s)\n", version.Version, version.Commit)
}
