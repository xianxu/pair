package titlepoller

import (
	"fmt"
	"testing"
	"time"
)

// fakeRuntime is a scriptable Runtime for the loop + frame-meter tests. Unset
// fields behave benignly (no-op writes, empty reads, dead processes).
type fakeRuntime struct {
	now               time.Time
	nowAdvance        time.Duration // advance per Now() call (0 = fixed clock)
	sleeps            int
	pid               string
	alive             map[string]bool
	commands          map[string]string
	namedSessions     map[string]bool // per-session-name override (e.g. a foreign cmux owner)
	sessionAliveSeq   []bool          // consumed per call for names NOT in namedSessions
	sessionAliveIdx   int
	sessionAliveCalls int // total SessionAlive calls (probe accounting)
	sessionAliveDflt  bool
	renamed           []string // "session|paneID|title"
	cmuxAvail         bool
	cmuxRenamed       []string
	files             map[string]string
	wrote             map[string]string
	removed           []string
	mtimes            map[string]time.Time
	panes             []PaneInfo
	counts            map[string]string // agent → context count
	transcripts       map[string]string // agent → transcript path
}

func newFake() *fakeRuntime {
	return &fakeRuntime{
		alive: map[string]bool{}, commands: map[string]string{},
		files: map[string]string{}, wrote: map[string]string{},
		mtimes: map[string]time.Time{}, counts: map[string]string{},
		transcripts: map[string]string{}, now: time.Unix(1_700_000_000, 0),
	}
}

func (f *fakeRuntime) Now() time.Time {
	t := f.now
	f.now = f.now.Add(f.nowAdvance)
	return t
}
func (f *fakeRuntime) Sleep(time.Duration)            { f.sleeps++ }
func (f *fakeRuntime) Getpid() string                 { return f.pid }
func (f *fakeRuntime) ProcessAlive(p string) bool     { return f.alive[p] }
func (f *fakeRuntime) ProcessCommand(p string) string { return f.commands[p] }
func (f *fakeRuntime) SessionAlive(name string) bool {
	f.sessionAliveCalls++
	if v, ok := f.namedSessions[name]; ok {
		return v // foreign-owner probes resolve by name, not by call order
	}
	if f.sessionAliveIdx < len(f.sessionAliveSeq) {
		v := f.sessionAliveSeq[f.sessionAliveIdx]
		f.sessionAliveIdx++
		return v
	}
	return f.sessionAliveDflt
}
func (f *fakeRuntime) RenamePane(s, id, t string) error {
	f.renamed = append(f.renamed, s+"|"+id+"|"+t)
	return nil
}
func (f *fakeRuntime) CmuxAvailable() bool { return f.cmuxAvail }
func (f *fakeRuntime) CmuxRenameWorkspace(t string) error {
	f.cmuxRenamed = append(f.cmuxRenamed, t)
	return nil
}
func (f *fakeRuntime) ReadFile(p string) (string, error) {
	if v, ok := f.files[p]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no such file: %s", p)
}
func (f *fakeRuntime) WriteFile(p, d string) error { f.wrote[p] = d; return nil }
func (f *fakeRuntime) Remove(p string)             { f.removed = append(f.removed, p) }
func (f *fakeRuntime) ModTime(p string) (time.Time, bool) {
	m, ok := f.mtimes[p]
	return m, ok
}
func (f *fakeRuntime) PaneFiles(string, string) []PaneInfo   { return f.panes }
func (f *fakeRuntime) ContextCount(_, agent string) string   { return f.counts[agent] }
func (f *fakeRuntime) TranscriptPath(_, agent string) string { return f.transcripts[agent] }

func fixtureOpts() Options {
	return Options{Tag: "T", Agent: "claude", DataDir: "/dd", Home: "/Users/x"}
}

// Shell harness case 6: one tick with a count → "<agent> (<count>) [<cwd>]".
func TestUpdateFrameTitlesWithCount(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", CwdDisplay: "~/repo"}}
	rt.counts["claude"] = "970k"
	updateFrameTitles(fixtureOpts(), rt, frameCache{}, "pair-T")
	want := "pair-T|7|claude (970k) [~/repo]"
	if len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("renamed = %v, want [%q]", rt.renamed, want)
	}
}

// Shell harness case 7: no count → "<agent> [<cwd>]" (no parens).
func TestUpdateFrameTitlesNoCount(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", CwdDisplay: "~/repo"}}
	updateFrameTitles(fixtureOpts(), rt, frameCache{}, "pair-T")
	want := "pair-T|7|claude [~/repo]"
	if len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("renamed = %v, want [%q]", rt.renamed, want)
	}
}

// cwd_display empty → falls back to abbrevCwd(cwd, home).
func TestUpdateFrameTitlesCwdFallback(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", Cwd: "/Users/x/repo"}}
	rt.counts["claude"] = "5k"
	updateFrameTitles(fixtureOpts(), rt, frameCache{}, "pair-T")
	want := "pair-T|7|claude (5k) [~/repo]"
	if len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("renamed = %v, want [%q]", rt.renamed, want)
	}
}

// Shell harness case 8: two ticks, same state → exactly ONE rename (skip guard).
func TestUpdateFrameTitlesUnchangedSkip(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", CwdDisplay: "~/repo"}}
	rt.counts["claude"] = "970k"
	cache := frameCache{}
	updateFrameTitles(fixtureOpts(), rt, cache, "pair-T")
	updateFrameTitles(fixtureOpts(), rt, cache, "pair-T")
	if len(rt.renamed) != 1 {
		t.Fatalf("expected 1 rename across two identical ticks, got %d: %v", len(rt.renamed), rt.renamed)
	}
}

// #97 regression: a stale pane-<tag>-<other>.json twin sharing the live pane_id
// must NOT hijack the frame. The glob (PaneFiles) can return both the active
// agent's file and a stale twin from a prior session on the same pane_id; before
// the fix, the pane_id-keyed frameCache rendered a different title for the same
// pane per file → alphabetical last-wins (codex > claude) + a per-tick flip-flop.
// With opts.Agent="claude" the active agent, exactly one rename to claude must
// fire, and it must stay stable across a second identical tick (no flip-flop).
//
// NOTE: this relies on the frame updater filtering pane.Agent==opts.Agent. If a
// future change reorders PaneFiles or drops that filter, the assertion below
// (single rename, to claude) is what catches the regression.
func TestUpdateFrameTitlesIgnoresStaleAgentTwin(t *testing.T) {
	rt := newFake()
	// Same pane_id "0" for both; alphabetical order would pick codex without the
	// active-agent filter. claude is the active agent (fixtureOpts).
	rt.panes = []PaneInfo{
		{Agent: "claude", PaneID: "0", CwdDisplay: "~/repo"},
		{Agent: "codex", PaneID: "0", CwdDisplay: "~/repo"},
	}
	rt.counts["claude"] = "970k"
	rt.counts["codex"] = "512k"
	cache := frameCache{}
	updateFrameTitles(fixtureOpts(), rt, cache, "pair-T")
	updateFrameTitles(fixtureOpts(), rt, cache, "pair-T") // second tick: must not flip-flop
	want := "pair-T|0|claude (970k) [~/repo]"
	if len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("renamed = %v, want exactly [%q] (active agent, no stale-twin hijack)", rt.renamed, want)
	}
}

// Single-instance guard: a live poller for this tag already recorded in the
// pidfile → Run returns immediately without ever probing the session.
func TestRunDefersToLiveInstance(t *testing.T) {
	rt := newFake()
	rt.files["/dd/title-pid-T"] = "4242\n"
	rt.alive["4242"] = true
	rt.commands["4242"] = "/x/bin/pair title T claude"
	rt.sessionAliveDflt = true // would loop forever if the guard failed to short-circuit
	code := Run(fixtureOpts(), rt)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if rt.sessionAliveCalls != 0 || len(rt.wrote) != 0 {
		t.Fatalf("guard should short-circuit before the session probe/pidfile write (probes=%d writes=%d)", rt.sessionAliveCalls, len(rt.wrote))
	}
}

// A stale pidfile (recycled PID whose argv isn't our poller) must NOT wedge the
// respawn — the poller claims the pidfile and proceeds. Here the session never
// appears within the grace window, so Run exits 0 after writing the pidfile.
func TestRunReclaimsStalePidfileThenGraceTimeout(t *testing.T) {
	rt := newFake()
	rt.files["/dd/title-pid-T"] = "4242\n"
	rt.alive["4242"] = true
	rt.commands["4242"] = "/usr/sbin/cupsd" // recycled PID, not our poller
	rt.pid = "9001"
	rt.sessionAliveDflt = false // session never shows up
	opts := fixtureOpts()
	opts.StartupGrace = 0 // → default 30s, but Now() never advances, so...
	// Make the grace loop terminate: advance Now past the deadline on the 2nd check.
	rt.nowAdvance = 40 * time.Second
	code := Run(opts, rt)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if rt.wrote["/dd/title-pid-T"] != "9001\n" {
		t.Fatalf("expected pidfile reclaimed with our pid, wrote = %v", rt.wrote)
	}
}

// Loop integration (claim path): one active tick through Run renders BOTH the
// zellij frame title and the cmux workspace title, wiring activityMTime → age →
// updateFrameTitles + updateWorkspaceTitle. Then the session goes missing and
// the loop exits.
func TestRunRendersFrameAndCmuxTitles(t *testing.T) {
	rt := newFake()
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", CwdDisplay: "~/repo"}}
	rt.counts["claude"] = "970k"
	rt.mtimes["/dd/draft-T.md"] = rt.now // fresh activity ⇒ age ≈ 0 < 2*poll
	rt.cmuxAvail = true
	// grace probe true, first tick true, then gone.
	rt.sessionAliveSeq = []bool{true, true}
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.CmuxWorkspaceID = "WS1"
	opts.MissThreshold = 2
	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if want := "pair-T|7|claude (970k) [~/repo]"; len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("frame renamed = %v, want [%q]", rt.renamed, want)
	}
	if want := cmuxWorkspaceTitle(prefixHot+" ", "pair-T"); len(rt.cmuxRenamed) != 1 || rt.cmuxRenamed[0] != want {
		t.Fatalf("cmux renamed = %v, want [%q]", rt.cmuxRenamed, want)
	}
	if rt.wrote["/dd/cmux-owner-WS1"] != "T\n" {
		t.Fatalf("owner file = %q, want claimed by T", rt.wrote["/dd/cmux-owner-WS1"])
	}
}

func TestRunUsesScopedPublicSessionName(t *testing.T) {
	rt := newFake()
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", CwdDisplay: "~/repo"}}
	rt.counts["claude"] = "970k"
	rt.mtimes["/dd/draft-T.md"] = rt.now
	rt.sessionAliveSeq = []bool{true, true}
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.SessionName = "pair-work-T"
	opts.MissThreshold = 1

	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if want := "pair-work-T|7|claude (970k) [~/repo]"; len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("frame renamed = %v, want [%q]", rt.renamed, want)
	}
}

// Loop integration (defer path): a live FOREIGN owner of the cmux workspace →
// the frame title still renders, but the workspace title is left alone.
func TestRunDefersCmuxToLiveForeignOwner(t *testing.T) {
	rt := newFake()
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7", CwdDisplay: "~/repo"}}
	rt.counts["claude"] = "12k"
	rt.mtimes["/dd/draft-T.md"] = rt.now
	rt.cmuxAvail = true
	rt.files["/dd/cmux-owner-WS1"] = "99\n"             // owned by tag 99…
	rt.namedSessions = map[string]bool{"pair-99": true} // …which is still alive
	rt.sessionAliveSeq = []bool{true, true}             // pair-T: grace + tick
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.CmuxWorkspaceID = "WS1"
	opts.MissThreshold = 2
	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if len(rt.renamed) != 1 {
		t.Fatalf("frame should still render, renamed = %v", rt.renamed)
	}
	if len(rt.cmuxRenamed) != 0 {
		t.Fatalf("must defer to the live foreign owner, cmuxRenamed = %v", rt.cmuxRenamed)
	}
	if _, wrote := rt.wrote["/dd/cmux-owner-WS1"]; wrote {
		t.Fatalf("must not overwrite a live owner's file")
	}
}

// updateWorkspaceTitle reclaims a STALE owner (its pair-<owner> session is gone).
func TestUpdateWorkspaceTitleReclaimsStaleOwner(t *testing.T) {
	rt := newFake()
	rt.cmuxAvail = true
	rt.files["/dd/cmux-owner-WS1"] = "99\n"
	rt.namedSessions = map[string]bool{"pair-99": false} // stale owner
	opts := fixtureOpts()
	opts.CmuxWorkspaceID = "WS1"
	got := updateWorkspaceTitle(opts, rt, 1*time.Hour, "pair-T", "__init__")
	if got != prefixHot+" " {
		t.Fatalf("returned prefix = %q, want reclaim with hot prefix", got)
	}
	if len(rt.cmuxRenamed) != 1 {
		t.Fatalf("stale owner should be reclaimed + renamed, cmuxRenamed = %v", rt.cmuxRenamed)
	}
	if rt.wrote["/dd/cmux-owner-WS1"] != "T\n" {
		t.Fatalf("owner file should be reclaimed by T, got %q", rt.wrote["/dd/cmux-owner-WS1"])
	}
}

// updateWorkspaceTitle is a no-op when the heat bucket is unchanged.
func TestUpdateWorkspaceTitleSkipsUnchangedBucket(t *testing.T) {
	rt := newFake()
	rt.cmuxAvail = true
	got := updateWorkspaceTitle(fixtureOpts(), rt, 1*time.Hour, "pair-T", prefixHot+" ")
	if got != prefixHot+" " || len(rt.cmuxRenamed) != 0 {
		t.Fatalf("unchanged bucket must be a no-op: prefix=%q renamed=%v", got, rt.cmuxRenamed)
	}
}

// activityMTime picks the most recent mtime across the draft and the transcript.
func TestActivityMTimePicksLatest(t *testing.T) {
	rt := newFake()
	base := rt.now
	rt.mtimes["/dd/draft-T.md"] = base.Add(-time.Hour)
	rt.transcripts["claude"] = "/x/transcript.jsonl"
	rt.mtimes["/x/transcript.jsonl"] = base // newer
	if got := activityMTime(fixtureOpts(), rt); !got.Equal(base) {
		t.Fatalf("activityMTime = %v, want the newer transcript mtime %v", got, base)
	}
	// No sources resolve ⇒ zero time.
	empty := newFake()
	if got := activityMTime(fixtureOpts(), empty); !got.IsZero() {
		t.Fatalf("activityMTime with no sources = %v, want zero", got)
	}
}

// The loop self-terminates after MissThreshold consecutive session misses.
func TestRunExitsOnSessionMissThreshold(t *testing.T) {
	rt := newFake()
	rt.pid = "9001"
	// grace: session appears immediately; then it's gone for every poll.
	rt.sessionAliveSeq = []bool{true}
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.MissThreshold = 3
	code := Run(opts, rt)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	// 1 grace probe + 3 miss probes = 4; exits on the 3rd miss.
	if rt.sessionAliveCalls != 4 {
		t.Fatalf("SessionAlive called %d times, want 4 (1 grace + 3 misses)", rt.sessionAliveCalls)
	}
}
