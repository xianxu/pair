// clipboard-to-pane — pull the OS clipboard and inject it into the nvim draft
// pane (triggers PairPasteQuote). Thin entry over clipcmd.RunClipboardToPaneCLI;
// logic in cmd/internal/clipcmd (#93 M4, ported from bin/clipboard-to-pane.sh).
// copy-on-select execs it directly (the shim is retired #94 M2).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
)

func main() {
	os.Exit(clipcmd.RunClipboardToPaneCLI(os.Getenv, os.Stderr))
}
