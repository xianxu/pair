package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #96/#104: `pair wrap` and `pair scribe` route through cmd/pair-go's streaming
// seam to wrapcmd.Run / scribecmd.Run. We drive each proxy's argument-error path
// — which returns before any PTY/terminal op — and assert the route reaches the
// runner (its own usage), not the dispatcher's "unknown command" / streaming-
// guard error and not a launched session. (The standalone bin/pair-wrap /
// bin/pair-scribe are gone since #104 M3, so there's nothing to compare against.)

func TestPairGoWrapRouteReachesRunner(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	// No command arg → wrapcmd usage on stderr, exit 1, before any tty op.
	r := runCommand(t, os.Environ(), pairGo, "wrap")
	if r.code != 1 {
		t.Fatalf("pair wrap route exit = %d, want 1\nstderr=%q", r.code, r.stderr)
	}
	if r.stderr == "" || strings.Contains(r.stderr, "streaming subcommand") || strings.Contains(r.stderr, "unknown command") {
		t.Fatalf("pair wrap route did not reach wrapcmd usage: %q", r.stderr)
	}
}

func TestPairGoScribeRouteReachesRunner(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	// No -log / no cmd → scribecmd usage on stderr, exit 2, before any tty op.
	r := runCommand(t, os.Environ(), pairGo, "scribe")
	if r.code != 2 {
		t.Fatalf("pair scribe route exit = %d, want 2\nstderr=%q", r.code, r.stderr)
	}
	if r.stderr == "" || strings.Contains(r.stderr, "streaming subcommand") || strings.Contains(r.stderr, "unknown command") {
		t.Fatalf("pair scribe route did not reach scribecmd usage: %q", r.stderr)
	}
}
