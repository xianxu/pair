package launcher

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// runRestart mirrors runCompaction's marker sequence but sourced from the live
// session + InferAgent (the pair-restart.sh path), not a continuation slug.
func TestRunRestartWritesMarkersAndKills(t *testing.T) {
	rt := newFakeRuntime()
	rt.inferAgent = map[string]string{"demo": "codex"} // the fake's InferAgent source
	var stderr bytes.Buffer
	code := runRestart(rt, LaunchArgs{Command: "restart", NewSession: true, RenameTo: "renamed"}, "pair-demo", "", &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, stderr.String())
	}
	m, ok := rt.writtenMarkers["pair-demo"]
	if !ok {
		t.Fatal("no restart marker written for pair-demo")
	}
	if m.Tag != "demo" || m.Agent != "codex" || !m.NewSession || m.RenameTo != "renamed" {
		t.Fatalf("marker = %+v", m)
	}
	if len(rt.touchedQuit) != 1 || rt.touchedQuit[0] != "pair-demo" {
		t.Fatalf("touchedQuit = %v", rt.touchedQuit)
	}
	if len(rt.killed) != 1 || rt.killed[0] != "pair-demo" {
		t.Fatalf("killed = %v", rt.killed)
	}
}

func TestRunRestartUsesEstablishedLedgerSessionID(t *testing.T) {
	rt := newFakeRuntime()
	rt.inferAgent["work"] = "codex"
	rt.ledger["work"] = []LedgerEntry{{Agent: "codex", SessionID: "SID-LIVE", Typed: true, SourceOrdinal: 1}}

	var stderr strings.Builder
	code := runRestart(rt, LaunchArgs{}, "📁work", "work", &stderr)
	if code != 0 {
		t.Fatalf("runRestart code = %d stderr=%q", code, stderr.String())
	}
	m, ok := rt.writtenMarkers["📁work"]
	if !ok {
		t.Fatalf("restart marker not written: %#v", rt.writtenMarkers)
	}
	if m.SessionID != "SID-LIVE" {
		t.Fatalf("marker session id = %q, want SID-LIVE; marker=%+v", m.SessionID, m)
	}
}

func TestRunRestartDoesNotCaptureLiveIDForNewSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.inferAgent["work"] = "codex"
	rt.ledger["work"] = []LedgerEntry{{Agent: "codex", SessionID: "SID-LIVE", Typed: true, SourceOrdinal: 1}}

	var stderr strings.Builder
	code := runRestart(rt, LaunchArgs{NewSession: true}, "📁work", "work", &stderr)
	if code != 0 {
		t.Fatalf("runRestart code = %d stderr=%q", code, stderr.String())
	}
	m := rt.writtenMarkers["📁work"]
	if m.SessionID != "" {
		t.Fatalf("fresh restart must not capture a resume id: %+v", m)
	}
}

func TestRunRestartUsesPairTagForScopedPublicSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.inferAgent = map[string]string{"bugfix": "codex"}
	var stderr bytes.Buffer
	code := runRestart(rt, LaunchArgs{Command: "restart"}, "📁work-bugfix", "bugfix", &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, stderr.String())
	}
	m, ok := rt.writtenMarkers["📁work-bugfix"]
	if !ok {
		t.Fatal("no restart marker written for scoped public session")
	}
	if m.Tag != "bugfix" || m.Agent != "codex" {
		t.Fatalf("marker = %+v, want repo-local tag bugfix/codex", m)
	}
}

func TestRunRestartRefusesUnreadableIndexWhenTagMustBeResolved(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessionIndexErr = errors.New("index unreadable")
	var stderr bytes.Buffer
	if code := runRestart(rt, LaunchArgs{}, "📁work", "", &stderr); code != 1 {
		t.Fatalf("code = %d, stderr=%q", code, stderr.String())
	}
	if len(rt.writtenMarkers) != 0 || len(rt.killed) != 0 {
		t.Fatalf("unreadable index must not restart: markers=%v killed=%v", rt.writtenMarkers, rt.killed)
	}
}

func TestRunRestartBareDefaults(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	if code := runRestart(rt, LaunchArgs{Command: "restart"}, "pair-solo", "", &stderr); code != 0 {
		t.Fatalf("exit %d", code)
	}
	m := rt.writtenMarkers["pair-solo"]
	if m.Tag != "solo" || m.NewSession || m.RenameTo != "" {
		t.Fatalf("marker = %+v, want tag=solo, no new_session/rename", m)
	}
	// No agent-<tag> record → InferAgent returns "" (faithful to the shell, which
	// writes an empty agent= line).
	if m.Agent != "" {
		t.Fatalf("Agent = %q, want empty", m.Agent)
	}
}

func TestRunQuitTouchesQuitAndKills(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	code := runQuit(rt, "pair-demo", &stderr)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if len(rt.writtenMarkers) != 0 {
		t.Fatalf("quit must not write a restart marker: %v", rt.writtenMarkers)
	}
	if len(rt.touchedQuit) != 1 || rt.touchedQuit[0] != "pair-demo" {
		t.Fatalf("touchedQuit = %v", rt.touchedQuit)
	}
	if len(rt.killed) != 1 || rt.killed[0] != "pair-demo" {
		t.Fatalf("killed = %v", rt.killed)
	}
}

func TestRunRestartMissingSession(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	if code := runRestart(rt, LaunchArgs{Command: "restart"}, "", "", &stderr); code != 1 {
		t.Fatalf("want exit 1 on empty session, got %d", code)
	}
	if len(rt.writtenMarkers) != 0 || len(rt.killed) != 0 {
		t.Fatal("must not write markers or kill when session is unset")
	}
}

func TestRunQuitMissingSession(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	if code := runQuit(rt, "", &stderr); code != 1 {
		t.Fatalf("want exit 1 on empty session, got %d", code)
	}
	if len(rt.touchedQuit) != 0 || len(rt.killed) != 0 {
		t.Fatal("must not touch or kill when session is unset")
	}
}
