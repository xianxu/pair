package launcher

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// buildPickRows is pure: detached sessions first (green), then historical "no
// live session" rows (age grey + queued badge), then "+ new"; a historical tag
// that is still live is deduped out.
func TestBuildPickRows(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := SessionSnapshot{
		BaseTag: "work",
		Sessions: []Session{
			{Name: "pair-a", State: SessionDetached},
			{Name: "pair-b", State: SessionAttached}, // attached → not a pick row
		},
		Historical: []HistoricalTag{
			{Tag: "old", MTime: now, QueueCount: 2},
			{Tag: "a", MTime: now}, // live (pair-a) → deduped
		},
	}
	display, byPlain := buildPickRows(snap, "work", now.Unix())

	if len(display) != 3 {
		t.Fatalf("display rows = %d (%q), want 3 (detached + 1 historical + new)", len(display), display)
	}
	// Order: detached first, then historical, then "+ new".
	if display[0] != ansiGreen+"pair-a"+ansiReset {
		t.Fatalf("row 0 = %q, want green pair-a", display[0])
	}
	if display[2] != "+ new work session" {
		t.Fatalf("row 2 = %q, want the + new label", display[2])
	}

	wantPlain := map[string]pickSelection{
		"pair-a": {tag: "a", sessionName: "pair-a"},
		"old  (today, no live session)   [⏎ 2 queued]": {tag: "old"},
		"+ new work session":                           {isNew: true},
	}
	if len(byPlain) != len(wantPlain) {
		t.Fatalf("byPlain = %#v, want %#v", byPlain, wantPlain)
	}
	for plain, sel := range wantPlain {
		if got, ok := byPlain[plain]; !ok || got != sel {
			t.Fatalf("byPlain[%q] = %#v (ok=%v), want %#v", plain, got, ok, sel)
		}
	}
}

// A historical row with no queued prompts carries no badge.
func TestBuildPickRowsNoBadge(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := SessionSnapshot{Historical: []HistoricalTag{{Tag: "solo", MTime: now}}}
	_, byPlain := buildPickRows(snap, "work", now.Unix())
	if _, ok := byPlain["solo  (today, no live session)"]; !ok {
		t.Fatalf("byPlain = %#v, want an unbadged historical row", byPlain)
	}
}

func TestBuildPickRowsAnnotatesRepoAndAgent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := SessionSnapshot{
		Sessions: []Session{{
			Name:     "📁pair-work",
			Tag:      "work",
			RepoName: "pair",
			Agent:    "codex",
			State:    SessionDetached,
		}},
		Historical: []HistoricalTag{{
			Tag:        "old",
			MTime:      now,
			RepoName:   "pair",
			Agent:      "claude",
			QueueCount: 1,
		}},
	}
	display, byPlain := buildPickRows(snap, "work", now.Unix())

	if len(display) != 3 {
		t.Fatalf("display rows = %d (%q), want live + historical + new", len(display), display)
	}
	if _, ok := byPlain["pair/work  codex  (detached)"]; !ok {
		t.Fatalf("byPlain = %#v, want annotated live row", byPlain)
	}
	if _, ok := byPlain["pair/old  claude  (today, no live session)   [⏎ 1 queued]"]; !ok {
		t.Fatalf("byPlain = %#v, want annotated historical row", byPlain)
	}
}

func TestBuildPickRowsExplicitAgentClassifiesLiveRows(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := SessionSnapshot{
		Sessions: []Session{
			{Name: "📁pair-cx", Tag: "cx", RepoName: "pair", Agent: "codex", State: SessionAttached},
			{Name: "📁pair-cl", Tag: "cl", RepoName: "pair", Agent: "claude", State: SessionDetached},
		},
		Historical: []HistoricalTag{
			{Tag: "old-cx", MTime: now, RepoName: "pair", Agent: "codex"},
			{Tag: "old-cl", MTime: now, RepoName: "pair", Agent: "claude"},
		},
	}
	display, byPlain := buildPickRowsWithPolicy(snap, "pair", now.Unix(), PickPolicy{RequestedAgent: "codex"})

	var plains []string
	for _, row := range display {
		plains = append(plains, stripANSI(row))
	}
	for _, want := range []string{
		"pair/cx  codex  (attached)",
		"pair/cl  claude  (detached, unavailable for codex)",
		"pair/old-cx  codex  (today, no live session)",
		"pair/old-cl  claude  (today, no live session)",
		"+ new pair session",
	} {
		if !containsString(plains, want) {
			t.Fatalf("display rows = %q, want %q", plains, want)
		}
	}
	if sel, ok := byPlain["pair/cx  codex  (attached)"]; !ok || sel.tag != "cx" || sel.sessionName != "📁pair-cx" {
		t.Fatalf("same-agent live selection = %#v ok=%v, want attachable cx", sel, ok)
	}
	if sel, ok := byPlain["pair/cl  claude  (detached, unavailable for codex)"]; !ok || !sel.disabled {
		t.Fatalf("different-agent live selection = %#v ok=%v, want disabled row", sel, ok)
	}
	if sel, ok := byPlain["pair/old-cl  claude  (today, no live session)"]; !ok || !sel.needsContinuation || sel.tag != "old-cl" {
		t.Fatalf("different-agent historical selection = %#v ok=%v, want continuation selection", sel, ok)
	}
}

func TestBuildPickRowsLabelsLegacyUnscopedManualImport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snap := SessionSnapshot{Historical: []HistoricalTag{{
		Tag:            "pair-old",
		MTime:          now,
		LegacyUnscoped: true,
	}}}
	display, byPlain := buildPickRows(snap, "pair", now.Unix())
	plain := "legacy unscoped pair-old  (manual import)"
	if len(display) != 2 {
		t.Fatalf("display = %q, want legacy row + new", display)
	}
	if _, ok := byPlain[plain]; !ok {
		t.Fatalf("byPlain = %#v, want %q", byPlain, plain)
	}
	if sel := byPlain[plain]; sel.tag != "pair-old" || !sel.legacy {
		t.Fatalf("selection = %#v, want legacy pair-old", sel)
	}
}

func TestRunLaunchPickLegacyImportsFlatFiles(t *testing.T) {
	rt := newFakeRuntime()
	rt.historical = []HistoricalTag{{Tag: "pair-old", MTime: time.Unix(1_700_000_000, 0), LegacyUnscoped: true}}
	rt.files["/global/draft-pair-old.md"] = "legacy draft"
	rt.files["/global/queue-pair-old/000001.md"] = "queued prompt"
	rt.pickFunc = func(header string, options []string) string {
		return "legacy unscoped pair-old  (manual import)"
	}
	opts := baseOpts(LaunchArgs{Agent: "claude"})
	opts.GlobalDataDir = "/global"

	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if got := rt.files["/data/draft-pair-old.md"]; got != "legacy draft" {
		t.Fatalf("imported draft = %q", got)
	}
	if got := rt.files["/global/draft-pair-old.md"]; got != "legacy draft" {
		t.Fatalf("legacy source should be preserved, got %q", got)
	}
	if got := rt.files["/data/queue-pair-old/000001.md"]; got != "queued prompt" {
		t.Fatalf("imported queue prompt = %q", got)
	}
	if got := rt.files["/global/queue-pair-old/000001.md"]; got != "queued prompt" {
		t.Fatalf("legacy queue source should be preserved, got %q", got)
	}
	if entries := rt.ledger["pair-old"]; len(entries) == 0 || !entries[0].LegacyImport {
		t.Fatalf("ledger entries = %#v, want legacy import marker", entries)
	}
}

// Picking a live detached session attaches it — and the agent is inferred from
// the picked tag (resume-by-name), NOT the bare-`pair` claude default, so a
// detached codex session attaches as codex.
func TestRunLaunchPickAttachInfersAgent(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessions = []Session{{Name: "📁work-svc", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-svc",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "svc",
	}}}
	rt.inferAgent = map[string]string{"svc": "codex"}
	rt.attachCode = 0
	rt.pickFunc = func(header string, options []string) string {
		for _, option := range options {
			plain := stripANSI(option)
			if plain == "work/svc  ?  (detached)" {
				return plain
			}
		}
		t.Fatalf("picker options = %q, want scoped svc row", options)
		return ""
	}

	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.attached) != 1 || rt.attached[0] != "📁work-svc" {
		t.Fatalf("attached = %v, want [📁work-svc]", rt.attached)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "svc|codex" {
		t.Fatalf("pollers = %v, want [svc|codex] (agent inferred from the picked tag)", rt.pollers)
	}
}

// Picking "+ new" creates a fresh free-slot session with the name prompt, using
// the default agent (a brand-new session, not resume-by-name).
func TestRunLaunchPickNewCreates(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessions = []Session{{Name: "pair-work", State: SessionDetached}}
	rt.promptValue = "work-2"
	rt.pickFunc = func(header string, options []string) string { return "+ new work session" }

	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-2" {
		t.Fatalf("launched = %q, want 📁work-2 (prompted free-slot create)", rt.launched)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "work-2|claude" {
		t.Fatalf("pollers = %v, want [work-2|claude]", rt.pollers)
	}
}

// Picking a historical (no-live-session) tag re-creates by name — resume-by-name,
// no prompt — with the agent inferred from that tag.
func TestRunLaunchPickHistoricalCreatesByName(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessions = []Session{{Name: "pair-live", State: SessionDetached}}
	rt.historical = []HistoricalTag{{Tag: "gone", MTime: time.Unix(1_700_000_000, 0)}}
	rt.inferAgent = map[string]string{"gone": "codex"}
	rt.pickFunc = func(header string, options []string) string {
		return "gone  (today, no live session)"
	}

	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-gone" {
		t.Fatalf("launched = %q, want 📁work-gone", rt.launched)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "gone|codex" {
		t.Fatalf("pollers = %v, want [gone|codex] (agent inferred)", rt.pollers)
	}
	if rt.family != nil { // no name prompt for a resume-by-name create
		t.Fatalf("family prompt shown = %v, want none for a historical re-create", rt.family)
	}
}

func TestRunLaunchExplicitAgentDifferentHistoricalUsesContinuation(t *testing.T) {
	rt := newFakeRuntime()
	rt.historical = []HistoricalTag{{Tag: "old", MTime: time.Unix(1_700_000_000, 0), RepoName: "work", Agent: "claude"}}
	rt.inferAgent = map[string]string{"old": "claude"}
	rt.continuationDocs = map[string][2]string{"old": {"/continuations/20260816-old.md", "claude"}}
	rt.pickFunc = func(header string, options []string) string {
		return "work/old  claude  (today, no live session)"
	}

	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-old" {
		t.Fatalf("launched = %q, want 📁work-old", rt.launched)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "old|codex" {
		t.Fatalf("pollers = %v, want [old|codex]", rt.pollers)
	}
	if got := rt.files["/data/draft-old.md"]; !strings.Contains(got, "Read workshop/continuation/20260816-old.md") {
		t.Fatalf("draft-old = %q, want continuation seed", got)
	}
}

func TestRunLaunchExplicitAgentDifferentHistoricalSynthesizesContinuationWhenDocMissing(t *testing.T) {
	rt := newFakeRuntime()
	rt.historical = []HistoricalTag{{Tag: "old", MTime: time.Unix(1_700_000_000, 0), RepoName: "work", Agent: "claude"}}
	rt.files["/data/draft-old.md"] = "existing WIP"
	rt.pickFunc = func(header string, options []string) string {
		return "work/old  claude  (today, no live session)"
	}
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true})
	var stderr bytes.Buffer

	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%q", code, err, stderr.String())
	}
	if rt.launched != "📁work-old" {
		t.Fatalf("launched = %q, want 📁work-old", rt.launched)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "old|codex" {
		t.Fatalf("pollers = %v, want [old|codex]", rt.pollers)
	}
	draft := rt.files["/data/draft-old.md"]
	for _, want := range []string{
		"Continue Pair tag old with codex.",
		"The previous driver was claude.",
		"No continuation doc was found.",
		"parked-old and parked-scrollback-old-*.raw/events.jsonl if present",
		"existing WIP",
	} {
		if !strings.Contains(draft, want) {
			t.Fatalf("draft-old = %q, want %q", draft, want)
		}
	}
	if strings.Contains(draft, ".claude/projects") || strings.Contains(draft, "native claude transcript") {
		t.Fatalf("draft-old = %q, should not reference agent-native transcript", draft)
	}
}

// Dismissing the picker (fzf ESC → empty) exits 0 without any handoff.
func TestRunLaunchPickAbort(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessions = []Session{{Name: "📁work-a", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-a",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "a",
	}}}
	rt.pickFunc = func(header string, options []string) string { return "" }

	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" || len(rt.attached) != 0 {
		t.Fatalf("pick abort should not hand off: launched=%q attached=%v", rt.launched, rt.attached)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
