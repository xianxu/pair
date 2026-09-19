// Package keyscmd implements `pair keys` — the in-session keybinding help that
// Alt+h pages (#132). When Couch presents the client, Couch's keys lead (#282).
package keyscmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/launcher"
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
	return RunWith(args, Deps{
		Sources: keyhelp.DefaultSources(),
		Getenv:  os.Getenv,
		CouchPresents: func() (bool, error) {
			dataDir, tag := os.Getenv("PAIR_DATA_DIR"), os.Getenv("PAIR_TAG")
			if dataDir == "" || tag == "" {
				return false, nil // outside a Pair session: nothing presents it
			}
			return launcher.ReadOuterPresenter(dataDir, tag)
		},
	}, stdout, stderr)
}

// Deps is every seam `pair keys` reads through, injected the way every sibling
// in the dispatcher table takes its seam as a parameter (contextcmd.Run takes an
// Env, agentcmd.RunRestart takes a Runtime). It keeps the always-exit-0 contract
// testable without a mutable package-level var, and keeps tests hermetic even
// when the repo is tested inside a Couch thread.
type Deps struct {
	Sources keyhelp.SourceReader
	Getenv  func(string) string
	// CouchPresents reports whether the client attached to this session was
	// launched by Couch -- read from the attach record, not the session env,
	// which names whoever created the session (#282).
	CouchPresents func() (bool, error)
}

// RunWith is Run with its seams injected.
func RunWith(args []string, deps Deps, stdout, stderr io.Writer) int {
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

	couch, perr := deps.CouchPresents()
	if perr != nil {
		// An untrusted record never conjures Couch's section.
		_, _ = fmt.Fprintf(stderr, "pair keys: %v\n", perr)
		couch = false
	}
	build := keyhelp.Sections
	if launcher.CouchOwnsRestart(launcher.CouchHostedEnv(deps.Getenv), couch) {
		// The rule `pair restart` refuses by (#284), so the hosted wording is
		// the true one exactly when Pair's Alt+n cannot reload (#282).
		build = keyhelp.HostedSections
	}
	sections, err := build(deps.Sources)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pair keys: %v\n", err)
		_, _ = fmt.Fprintf(stdout, "keybind help unavailable: %v\n", err)
		return 0
	}

	if couch {
		// Couch presents this client: its keys lead, and any chord it takes
		// from every pane replaces Pair's row for it.
		bs := couchkeys.Bindings()
		sections = keyhelp.Layer(couchkeys.HelpSections(bs), couchkeys.Claimed(bs), sections)
	}
	out := keyhelp.Render(sections)
	if cols > 0 {
		out = keyhelp.Center(out, cols)
	}
	_, _ = fmt.Fprint(stdout, out)
	return 0
}
