// pair-changelog — distill a pair session's TTY into an append-mostly change
// log (#53). Thin shim over changelogcmd.Run; the logic lives in
// cmd/internal/changelogcmd (shared with the `pair changelog` dispatcher
// route). Streams live per-batch progress to stderr for the Alt+l spinner.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/changelogcmd"
)

func main() { os.Exit(changelogcmd.Run(os.Args[1:], os.Stderr)) }
