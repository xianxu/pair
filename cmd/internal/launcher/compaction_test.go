package launcher

import (
	"strings"
	"testing"
)

func TestCompactionDecision(t *testing.T) {
	cases := []struct {
		name         string
		force        bool
		inPaneOrFake bool
		tag, session string
		want         bool
	}{
		{"force wins", true, false, "", "", true},
		{"tag match in pane", false, true, "demo", "pair-demo", true},
		{"tag mismatch", false, true, "demo", "pair-other", false},
		{"not in pane", false, false, "demo", "pair-demo", false},
		{"empty tag", false, true, "", "pair-", false},
	}
	for _, c := range cases {
		if got := compactionDecision(c.force, c.inPaneOrFake, c.tag, c.session); got != c.want {
			t.Errorf("%s: compactionDecision = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSerializeRestartMarkerRoundTrip(t *testing.T) {
	m := RestartMarker{Tag: "demo", Agent: "codex", NewSession: true, Continue: "demo-slug"}
	got := parseRestartMarker(serializeRestartMarker(m))
	if got != m {
		t.Fatalf("round-trip = %+v, want %+v", got, m)
	}
	// The compaction shape (shell 1052-1057): tag, agent, new_session=1, continue.
	s := serializeRestartMarker(m)
	for _, want := range []string{"tag=demo\n", "agent=codex\n", "new_session=1\n", "continue=demo-slug\n"} {
		if !strings.Contains(s, want) {
			t.Fatalf("serialized %q missing %q", s, want)
		}
	}
	// A rename_to marker emits rename_to and omits absent fields.
	if rt := serializeRestartMarker(RestartMarker{Tag: "a", Agent: "claude", RenameTo: "b"}); strings.Contains(rt, "new_session") || !strings.Contains(rt, "rename_to=b\n") {
		t.Fatalf("rename marker = %q", rt)
	}
}

// compactOpts builds the LaunchOptions for an in-pane `continue <slug>` compaction.
func compactOpts(force, fake bool, session string) LaunchOptions {
	o := baseOpts(LaunchArgs{Agent: "claude"})
	o.ContinueSlug = "demo"
	o.PairTag = "demo"
	o.PairAgent = "claude"
	o.ForceInSession = force
	o.FakeInZellij = fake
	o.ZellijSession = session
	return o
}

func TestRunLaunchCompactionForced(t *testing.T) {
	rt := newFakeRuntime()
	rt.parkOK = true
	code, err := run(t, compactOpts(true, false, ""), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	m, ok := rt.writtenMarkers["pair-demo"]
	if !ok || m.Continue != "demo" || !m.NewSession || m.Tag != "demo" || m.Agent != "claude" {
		t.Fatalf("restart marker = %+v (ok=%v)", m, ok)
	}
	if len(rt.touchedQuit) != 1 || rt.touchedQuit[0] != "pair-demo" {
		t.Fatalf("quit marker = %v", rt.touchedQuit)
	}
	if len(rt.killed) != 1 || rt.killed[0] != "pair-demo" {
		t.Fatalf("killed = %v", rt.killed)
	}
	if len(rt.parked) != 1 || rt.parked[0] != "demo|claude|false" { // copy, not move
		t.Fatalf("parked = %v, want copy-mode park", rt.parked)
	}
	if rt.launched != "" { // compaction is terminal — no create handoff
		t.Fatalf("compaction must not create a session, launched=%q", rt.launched)
	}
}

func TestRunLaunchCompactionTagMatch(t *testing.T) {
	rt := newFakeRuntime()
	code, err := run(t, compactOpts(false, true, "pair-demo"), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, ok := rt.writtenMarkers["pair-demo"]; !ok {
		t.Fatal("tag-match should compact")
	}
}

// A tag MISMATCH does not compact; if the process is really in a pane it is
// rejected (shell 1064), not launched — no marker, no kill.
func TestRunLaunchCompactionTagMismatch(t *testing.T) {
	rt := newFakeRuntime()
	rt.inPane = true // the real ancestry guard fires after compaction declines
	code, err := run(t, compactOpts(false, true, "pair-other"), rt)
	if err != nil {
		t.Fatalf("mismatch should be handled natively, err=%v", err)
	}
	if code != 1 {
		t.Fatalf("mismatch in a pane should exit 1, got %d", code)
	}
	if len(rt.writtenMarkers) != 0 || len(rt.killed) != 0 {
		t.Fatalf("mismatch must not compact: markers=%v killed=%v", rt.writtenMarkers, rt.killed)
	}
}
