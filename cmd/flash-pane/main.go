// flash-pane — flash a zellij pane's background as a brief visual cue. Thin
// entry over clipcmd.RunFlashPaneCLI; logic in cmd/internal/clipcmd (#93 M4,
// ported from bin/flash-pane.sh). copy-on-select execs it directly with the
// source pane id (the shim is retired #94 M2).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
)

func main() {
	os.Exit(clipcmd.RunFlashPaneCLI(os.Args[1:], os.Getenv, os.Stderr))
}
