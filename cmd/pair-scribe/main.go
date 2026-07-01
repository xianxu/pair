// pair-scribe — a script(1) replacement with SIGUSR1/SIGUSR2 pause/resume of
// the typescript. Thin shim over scribecmd.Run; the logic + docs live in
// cmd/internal/scribecmd (shared with the `pair scribe` dispatcher route, #96).
//
// This binary is NOT part of pair's runtime — it's user shell tooling that
// swaps for script(1) at the top of the zsh session. It stays installed at
// ~/.local/bin/pair-scribe by `make install` so the user's ~/.zshrc `exec`
// line keeps working. See README.md for the full why + the zshrc snippet.
package main

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/scribecmd"
)

func main() {
	os.Exit(scribecmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
