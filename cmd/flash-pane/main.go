// flash-pane — flash a zellij pane's background as a brief visual cue. Thin
// entry over clipcmd.RunFlashPaneCLI; logic in cmd/internal/clipcmd (#93 M4,
// ported from bin/flash-pane.sh). Reached via the bin/flash-pane.sh shim, which
// copy-on-select execs with the source pane id.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
)

func main() {
	os.Exit(clipcmd.RunFlashPaneCLI(os.Args[1:], os.Getenv, os.Stderr))
}
