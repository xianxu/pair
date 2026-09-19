package launcher

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

func TestCompactionDecision(t *testing.T) {
	cases := []struct {
		name              string
		force             bool
		inPaneOrFake      bool
		tag, session, own string
		want              bool
	}{
		{"force wins", true, false, "", "", "", true},
		{"legacy tag match in pane", false, true, "demo", "pair-demo", "", true},
		{"own scoped public session match in pane", false, true, "demo", "📁work-demo", "📁work-demo", true},
		{"other repo scoped public session does not match", false, true, "demo", "📁work-demo", "", false},
		{"tag mismatch", false, true, "demo", "pair-other", "", false},
		{"not in pane", false, false, "demo", "pair-demo", "", false},
		{"empty tag", false, true, "", "pair-", "", false},
	}
	for _, c := range cases {
		if got := compactionDecision(c.force, c.inPaneOrFake, c.tag, c.session, c.own); got != c.want {
			t.Errorf("%s: compactionDecision = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSerializeRestartMarkerRoundTrip(t *testing.T) {
	m := RestartMarker{Tag: "demo", Agent: "codex", NewSession: true, Continue: "demo-slug"}
	got := parseRestartMarker(serializeRestartMarker(m))
	if !sameRestartMarker(got, m) {
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
	o.ContinueCheckpoint, _ = checkpoint.New("/repo/workshop/continuation/demo.md", "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nContinue demo.\n")
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
	if len(rt.touchedQuit) != 1 {
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

func TestRunLaunchCompactionUsesScopedPublicSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.parkOK = true
	opts := compactOpts(false, true, "📁work-demo")
	opts.PairSession = "📁work-demo"
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	m, ok := rt.writtenMarkers["📁work-demo"]
	if !ok {
		t.Fatalf("restart markers = %#v, want scoped public session", rt.writtenMarkers)
	}
	if m.Tag != "demo" || m.Agent != "claude" || !m.NewSession || m.Continue != "demo" {
		t.Fatalf("restart marker = %+v", m)
	}
	if len(rt.touchedQuit) != 1 {
		t.Fatalf("quit marker = %v", rt.touchedQuit)
	}
	if len(rt.killed) != 1 || rt.killed[0] != "📁work-demo" {
		t.Fatalf("killed = %v", rt.killed)
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

// In-session compaction is the other restart-marker writer that kills a live
// session (#284). In a session Couch presents but did not create it has no
// thread address to hand Couch, so it refuses before parking, marking or
// killing anything.
func TestRunLaunchCompactionRefusesACouchPresentedSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.parkOK = true
	rt.RecordOuterTTY("demo", true)
	code, err := run(t, compactOpts(true, false, ""), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v, want the refusal", code, err)
	}
	if len(rt.writtenMarkers) != 0 || len(rt.touchedQuit) != 0 || len(rt.killed) != 0 || len(rt.parked) != 0 {
		t.Fatalf("mutated before refusing: markers=%v quit=%v killed=%v parked=%v", rt.writtenMarkers, rt.touchedQuit, rt.killed, rt.parked)
	}
}

// The refusal must name a route that RUNS from the state it leaves behind
// (#284 close review, BR-1). The first attempt named `pair continue --retry`,
// which needs a retained restart marker — and this arm returns before writing
// one, so the advice was dead. What survives the refusal is the checkpoint doc
// itself, so `--checkpoint <path>` is the route, and this pins all three facts:
// no marker was written, the named route parses, and the path it names still
// resolves and validates.
func TestCompactionRefusalNamesARouteThatRuns(t *testing.T) {
	rt := newFakeRuntime()
	rt.parkOK = true
	rt.RecordOuterTTY("demo", true)
	opts := compactOpts(true, false, "")
	source := opts.ContinueCheckpoint.SourcePath
	rt.files[source] = "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nContinue demo.\n"

	var stderr bytes.Buffer
	if code, err := RunLaunch(opts, rt, &stderr); code != 1 || err != nil {
		t.Fatalf("code=%d err=%v, want the refusal", code, err)
	}
	if len(rt.writtenMarkers) != 0 {
		t.Fatalf("a marker was written after all (%v) — then --retry would be the route", rt.writtenMarkers)
	}
	route := regexp.MustCompile("pair continue --checkpoint ([^\\s`]+)").FindStringSubmatch(stderr.String())
	if route == nil {
		t.Fatalf("refusal names no --checkpoint route: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "--retry") {
		t.Fatalf("refusal still names --retry, which cannot run without a marker: %q", stderr.String())
	}
	if route[1] != source {
		t.Fatalf("route names %q, want the retained checkpoint %q", route[1], source)
	}
	args, err := ParseArgs([]string{"continue", "--checkpoint", route[1]})
	if err != nil || args.Command != "continue" || args.ContinueCheckpoint != source {
		t.Fatalf("named route does not parse: %+v err=%v", args, err)
	}
	c, err := rt.ReadCheckpoint(args.ContinueCheckpoint)
	if err != nil {
		t.Fatalf("the route's checkpoint does not resolve: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("the route's checkpoint does not validate: %v", err)
	}
}
