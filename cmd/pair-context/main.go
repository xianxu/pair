// pair-context — print one agent pane's current context size (humanized
// token count), or nothing. Invoked as `pair-context <tag> <agent>` by the
// pair-title poller. Tolerant: any failure prints nothing and exits 0, so a
// hiccup never garbles the pane title.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
)

func main() {
	os.Exit(contextcmd.Run(os.Args[1:], contextcmd.EnvFromOS(), os.Stdout))
}
