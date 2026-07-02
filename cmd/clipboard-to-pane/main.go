// clipboard-to-pane — pull the OS clipboard and inject it into the nvim draft
// pane (triggers PairPasteQuote). Thin entry over clipcmd.RunClipboardToPaneCLI;
// logic in cmd/internal/clipcmd (#93 M4, ported from bin/clipboard-to-pane.sh).
// Reached via the bin/clipboard-to-pane.sh shim, which copy-on-select execs.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
)

func main() {
	os.Exit(clipcmd.RunClipboardToPaneCLI(os.Getenv, os.Stderr))
}
