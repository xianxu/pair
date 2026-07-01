package main

import (
	"os"
	"path/filepath"
	"testing"
)

// #96: `pair wrap` and `pair scribe` route through cmd/pair-go's streaming seam
// to the SAME wrapcmd.Run / scribecmd.Run as the standalone bin/pair-wrap and
// bin/pair-scribe shims. We drive each proxy's argument-error path — which
// returns before any PTY/terminal op — and assert the dispatch route matches
// the standalone binary byte-for-byte on exit code + stdout + stderr. That
// proves the route reaches the runner (a mis-wire would launch a session or
// return the "unknown command"/streaming-guard error instead).

func TestPairGoWrapRouteMatchesStandaloneWrap(t *testing.T) {
	bin := t.TempDir()
	pairWrap := filepath.Join(bin, "pair-wrap")
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairWrap, "../pair-wrap")
	buildCommand(t, pairGo, ".")

	env := os.Environ()
	// No command arg → "pair-wrap: usage: …" on stderr, exit 1, before any tty op.
	legacy := runCommand(t, env, pairWrap)
	dispatch := runCommand(t, env, pairGo, "wrap")
	if legacy.code != 1 {
		t.Fatalf("standalone pair-wrap usage exit = %d, want 1\nstderr=%q", legacy.code, legacy.stderr)
	}
	if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
		t.Fatalf("pair-go wrap route mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
			legacy.code, legacy.stdout, legacy.stderr,
			dispatch.code, dispatch.stdout, dispatch.stderr)
	}
}

func TestPairGoScribeRouteMatchesStandaloneScribe(t *testing.T) {
	bin := t.TempDir()
	pairScribe := filepath.Join(bin, "pair-scribe")
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairScribe, "../pair-scribe")
	buildCommand(t, pairGo, ".")

	env := os.Environ()
	// No -log / no cmd → usage on stderr, exit 2, before any tty op.
	legacy := runCommand(t, env, pairScribe)
	dispatch := runCommand(t, env, pairGo, "scribe")
	if legacy.code != 2 {
		t.Fatalf("standalone pair-scribe usage exit = %d, want 2\nstderr=%q", legacy.code, legacy.stderr)
	}
	if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
		t.Fatalf("pair-go scribe route mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
			legacy.code, legacy.stdout, legacy.stderr,
			dispatch.code, dispatch.stdout, dispatch.stderr)
	}
}
