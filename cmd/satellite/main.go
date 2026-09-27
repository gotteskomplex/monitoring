// Command satellite runs the on-premises check executor.
// Enrollment, connection, scheduler and checks are implemented in phase 2.
package main

import (
	"fmt"
	"os"

	"github.com/gotteskomplex/monitoring/internal/platform/version"
)

func main() {
	cmd := "help"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "version":
		fmt.Printf("satellite %s (commit %s, built %s, protocol %d)\n",
			version.Version, version.Commit, version.Date, version.ProtocolVersion)
	default:
		fmt.Fprintln(os.Stderr, "usage: satellite version\n(enroll/run follow in phase 2)")
		if cmd != "help" {
			os.Exit(2)
		}
	}
}
