// copy-on-select — zellij's copy_command (fires on every selection finalize).
// Thin entry over clipcmd.RunCopyOnSelectCLI; logic in cmd/internal/clipcmd
// (#93 M4, ported from bin/copy-on-select.sh; the shim is retired #94 M2, so
// zellij's `copy_command "copy-on-select"` resolves this binary directly).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
)

func main() {
	os.Exit(clipcmd.RunCopyOnSelectCLI(os.Args[1:], os.Stdin, os.Getenv, os.Stderr))
}
