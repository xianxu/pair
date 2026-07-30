// Package keyscmd implements `pair keys` — the in-session keybinding help that
// Alt+h pages (#132).
package keyscmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/keyhelp"
)

// sources is a seam so the always-exit-0 contract below is testable. That contract
// is load-bearing, not cosmetic: bin/pair-help runs under `set -euo pipefail`, so a
// non-zero exit kills the floating pane before less opens.
var sources = func() keyhelp.SourceReader { return keyhelp.DefaultSources() }

// Run renders the keybindings. `--center <cols>` centres the block in that many
// terminal columns, which is what bin/pair-help asks for.
//
// It ALWAYS returns 0, even when a source read fails, printing a one-line
// diagnostic as the body instead. bin/pair-help runs under `set -euo pipefail`, so a
// non-zero exit would kill the floating pane before `less` ever opened — replacing
// #132's useless help key with a dead one. A visible diagnostic is strictly better
// than a pane that flashes and closes.
func Run(args []string, stdout, stderr io.Writer) int {
	cols := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--center":
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err == nil && n > 0 {
					cols = n
				}
				i++
			}
		case "-h", "--help":
			_, _ = fmt.Fprintln(stdout, "usage: pair keys [--center <cols>]")
			return 0
		}
	}

	sections, err := keyhelp.Sections(sources())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pair keys: %v\n", err)
		_, _ = fmt.Fprintf(stdout, "keybind help unavailable: %v\n", err)
		return 0
	}

	out := keyhelp.Render(sections)
	if cols > 0 {
		out = keyhelp.Center(out, cols)
	}
	_, _ = fmt.Fprint(stdout, out)
	return 0
}
