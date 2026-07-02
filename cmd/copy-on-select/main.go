// copy-on-select — zellij's copy_command (fires on every selection finalize).
// Thin entry over clipcmd.RunCopyOnSelectCLI; logic in cmd/internal/clipcmd
// (#93 M4, ported from bin/copy-on-select.sh). Reached via the bin/copy-on-
// select.sh shim so zellij's `copy_command "copy-on-select.sh"` keeps resolving.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
)

func main() {
	os.Exit(clipcmd.RunCopyOnSelectCLI(os.Stdin, os.Getenv, os.Stderr))
}
