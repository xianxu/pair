// pair-session-watch — async codex/agy session-id discovery. Thin shim over
// sessionwatch.RunCLI; the logic lives in cmd/internal/sessionwatch (shared
// with the `pair session-watch` dispatcher route). The launcher spawns
// bin/pair-session-watch directly (the .sh re-exec shim was retired #94 M2).
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

func main() { os.Exit(sessionwatch.RunCLI(os.Args[1:], os.Getenv, os.Stderr)) }
