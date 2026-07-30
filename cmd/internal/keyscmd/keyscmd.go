// Package keyscmd implements `pair keys` — the in-session keybinding help that
// Alt+h pages (#132).
package keyscmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/keyhelp"
)

// Run renders the keybindings. `--center <cols>` centres the block in that many
// terminal columns, which is what bin/pair-help asks for.
//
// It ALWAYS returns 0, even when a source read fails, printing a one-line
// diagnostic as the body instead. bin/pair-help runs under `set -euo pipefail`, so a
// non-zero exit would kill the floating pane before `less` ever opened — replacing
// #132's useless help key with a dead one. A visible diagnostic is strictly better
// than a pane that flashes and closes.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWithSources(args, keyhelp.DefaultSources(), stdout, stderr)
}

// RunWithSources is Run with the source reader injected, matching how every sibling
// in the dispatcher table takes its seam as a parameter (contextcmd.Run takes an Env,
// agentcmd.RunRestart takes a Runtime). Keeps the always-exit-0 contract testable
// without a mutable package-level var, so tests stay parallel-safe.
func RunWithSources(args []string, src keyhelp.SourceReader, stdout, stderr io.Writer) int {
	cols := 0
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--center":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					cols = n
				}
				i++
			}
		case strings.HasPrefix(arg, "--center="):
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "--center=")); err == nil && n > 0 {
				cols = n
			}
		case arg == "-h", arg == "--help":
			_, _ = fmt.Fprintln(stdout, "usage: pair keys [--center <cols>]")
			return 0
		default:
			// Named rather than ignored: a typo silently printing the help reads as
			// success, and this command is invoked from a shim nobody watches.
			_, _ = fmt.Fprintf(stderr, "pair keys: unknown argument %q\n", arg)
			_, _ = fmt.Fprintln(stdout, "usage: pair keys [--center <cols>]")
			return 2
		}
	}

	sections, err := keyhelp.Sections(src)
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
