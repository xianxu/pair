// pair-changelog-open — open the session's distilled change log in a read-only
// nvim viewer, building/refreshing it with a detached distiller (Alt+l). Thin
// entry over opener.RunChangelogCLI; logic in cmd/internal/opener (#93 M2, ported
// from the bin/pair-changelog-open shell script). Invoked by zellij/config.kdl.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/opener"
)

func main() {
	os.Exit(opener.RunChangelogCLI(os.Args[1:], os.Getenv, os.Stderr))
}
