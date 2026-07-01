// pair-scrollback-open — render the agent pane's captured scrollback and open it
// in a read-only ANSI-aware nvim viewer (Alt+/). Thin entry over
// opener.RunScrollbackCLI; logic in cmd/internal/opener (#93 M2, ported from the
// bin/pair-scrollback-open shell script). Invoked by zellij/config.kdl.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/opener"
)

func main() {
	os.Exit(opener.RunScrollbackCLI(os.Args[1:], os.Getenv, os.Stderr))
}
