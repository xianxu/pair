// pair-wrap — transparent PTY proxy around a TUI coding agent. Thin shim over
// wrapcmd.Run; the logic + docs live in cmd/internal/wrapcmd (shared with the
// `pair wrap` dispatcher route, #96). Installed at bin/pair-wrap and invoked by
// zellij/layouts/main.kdl on pair startup via a PATH lookup.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/wrapcmd"
)

func main() {
	os.Exit(wrapcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
