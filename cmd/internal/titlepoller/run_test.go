package titlepoller

import (
	"fmt"
	"slices"
	"strings"
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
	counts            map[string]string    // agent → context count
	activities        map[string]time.Time // agent → established activity
	// events is the ordered log of zellij calls and pane-sidecar stats, so a
	// test can assert what happened BEFORE the pane was born (#287).
	events []string
	// sleepHook runs after each Sleep with the running count: the seam a test
	// uses to make the pane appear mid-wait.
	sleepHook func(sleeps int)
}

// fixturePane is the agent pane sidecar fixtureOpts' poller gates on.
const fixturePane = "/dd/pane-T-claude.json"

// withLivePane seeds the sidecar a live session's pane has already written --
// the attach shape, and the precondition every steady-loop test assumes.
func withLivePane(f *fakeRuntime) *fakeRuntime {
	f.mtimes[fixturePane] = f.now
	return f
}

func newFake() *fakeRuntime {
	return &fakeRuntime{
		alive: map[string]bool{}, commands: map[string]string{},
		files: map[string]string{}, wrote: map[string]string{},
		mtimes: map[string]time.Time{}, counts: map[string]string{},
		activities: map[string]time.Time{}, now: time.Unix(1_700_000_000, 0),
	}
}

func (f *fakeRuntime) Now() time.Time {
	t := f.now
	f.now = f.now.Add(f.nowAdvance)
	return t
}
func (f *fakeRuntime) Sleep(time.Duration) {
	f.sleeps++
	if f.sleepHook != nil {
		f.sleepHook(f.sleeps)
	}
}
func (f *fakeRuntime) Getpid() string                 { return f.pid }
func (f *fakeRuntime) ProcessAlive(p string) bool     { return f.alive[p] }
func (f *fakeRuntime) ProcessCommand(p string) string { return f.commands[p] }
func (f *fakeRuntime) SessionAlive(name string) bool {
	f.sessionAliveCalls++
	f.events = append(f.events, "session-alive")
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
	f.events = append(f.events, "rename")
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
	if p == fixturePane {
		f.events = append(f.events, fmt.Sprintf("stat-pane:%t", ok))
	}
	return m, ok
}
func (f *fakeRuntime) PaneFiles(string, string) []PaneInfo { return f.panes }
func (f *fakeRuntime) ContextCount(_, agent string) string { return f.counts[agent] }
func (f *fakeRuntime) SessionActivity(_, agent string) (time.Time, bool) {
	value, ok := f.activities[agent]
	return value, ok
}

func fixtureOpts() Options {
	return Options{Tag: "T", Agent: "claude", DataDir: "/dd"}
}

// Shell harness case 6: one tick with a count → "<agent> (<count>)".
func TestUpdateFrameTitlesWithCount(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	rt.counts["claude"] = "970k"
	updateFrameTitles(fixtureOpts(), rt, frameCache{}, "pair-T")
	want := "pair-T|7|claude (970k)"
	if len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("renamed = %v, want [%q]", rt.renamed, want)
	}
}

// Shell harness case 7: no count → bare "<agent>" (no parens).
func TestUpdateFrameTitlesNoCount(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	updateFrameTitles(fixtureOpts(), rt, frameCache{}, "pair-T")
	want := "pair-T|7|claude"
	if len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("renamed = %v, want [%q]", rt.renamed, want)
	}
}

// Shell harness case 8: two ticks, same state → exactly ONE rename (skip guard).
func TestUpdateFrameTitlesUnchangedSkip(t *testing.T) {
	rt := newFake()
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
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
		{Agent: "claude", PaneID: "0"},
		{Agent: "codex", PaneID: "0"},
	}
	rt.counts["claude"] = "970k"
	rt.counts["codex"] = "512k"
	cache := frameCache{}
	updateFrameTitles(fixtureOpts(), rt, cache, "pair-T")
	updateFrameTitles(fixtureOpts(), rt, cache, "pair-T") // second tick: must not flip-flop
	want := "pair-T|0|claude (970k)"
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
// respawn — the poller claims the pidfile and proceeds. Here the pane never
// appears within the grace window, so Run exits 0 after writing the pidfile.
func TestRunReclaimsStalePidfileThenGraceTimeout(t *testing.T) {
	rt := newFake()
	rt.files["/dd/title-pid-T"] = "4242\n"
	rt.alive["4242"] = true
	rt.commands["4242"] = "/usr/sbin/cupsd" // recycled PID, not our poller
	rt.pid = "9001"
	rt.sessionAliveDflt = false // and no pane is ever born
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
	if rt.sessionAliveCalls != 0 {
		t.Fatalf("an unborn session must never be probed, SessionAlive called %d times", rt.sessionAliveCalls)
	}
}

// Loop integration (claim path): one active tick through Run renders BOTH the
// zellij frame title and the cmux workspace title, wiring activityMTime → age →
// updateFrameTitles + updateWorkspaceTitle. Then the session goes missing and
// the loop exits.
func TestRunRendersFrameAndCmuxTitles(t *testing.T) {
	rt := withLivePane(newFake())
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	rt.counts["claude"] = "970k"
	rt.mtimes["/dd/draft-T.md"] = rt.now // fresh activity ⇒ age ≈ 0 < 2*poll
	rt.cmuxAvail = true
	// first tick live, then gone.
	rt.sessionAliveSeq = []bool{true}
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.CmuxWorkspaceID = "WS1"
	opts.MissThreshold = 2
	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if want := "pair-T|7|claude (970k)"; len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("frame renamed = %v, want [%q]", rt.renamed, want)
	}
	if want := cmuxWorkspaceTitle(prefixHot+" ", "pair-T"); len(rt.cmuxRenamed) != 1 || rt.cmuxRenamed[0] != want {
		t.Fatalf("cmux renamed = %v, want [%q]", rt.cmuxRenamed, want)
	}
	if rt.wrote["/dd/cmux-owner-WS1"] != "T\tpair-T\n" {
		t.Fatalf("owner file = %q, want claimed by T", rt.wrote["/dd/cmux-owner-WS1"])
	}
}

func TestRunUsesScopedPublicSessionName(t *testing.T) {
	rt := withLivePane(newFake())
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	rt.counts["claude"] = "970k"
	rt.mtimes["/dd/draft-T.md"] = rt.now
	rt.sessionAliveSeq = []bool{true}
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.SessionName = "📁work-T"
	opts.MissThreshold = 1

	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if want := "📁work-T|7|claude (970k)"; len(rt.renamed) != 1 || rt.renamed[0] != want {
		t.Fatalf("frame renamed = %v, want [%q]", rt.renamed, want)
	}
}

// Loop integration (defer path): a live FOREIGN owner of the cmux workspace →
// the frame title still renders, but the workspace title is left alone.
func TestRunDefersCmuxToLiveForeignOwner(t *testing.T) {
	rt := withLivePane(newFake())
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	rt.counts["claude"] = "12k"
	rt.mtimes["/dd/draft-T.md"] = rt.now
	rt.cmuxAvail = true
	rt.files["/dd/cmux-owner-WS1"] = "99\n"             // owned by tag 99…
	rt.namedSessions = map[string]bool{"pair-99": true} // …which is still alive
	rt.sessionAliveSeq = []bool{true}                   // pair-T: one live tick
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

func TestRunDefersCmuxToLiveScopedForeignOwner(t *testing.T) {
	rt := withLivePane(newFake())
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	rt.counts["claude"] = "12k"
	rt.mtimes["/dd/draft-T.md"] = rt.now
	rt.cmuxAvail = true
	rt.files["/dd/cmux-owner-WS1"] = "99\tpair-pair-99\n"
	rt.namedSessions = map[string]bool{
		"pair-99":      false,
		"pair-pair-99": true,
	}
	rt.sessionAliveSeq = []bool{true} // pair-T: one live tick
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.CmuxWorkspaceID = "WS1"
	opts.MissThreshold = 2
	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if len(rt.cmuxRenamed) != 0 {
		t.Fatalf("must defer to the live scoped foreign owner, cmuxRenamed = %v", rt.cmuxRenamed)
	}
	if _, wrote := rt.wrote["/dd/cmux-owner-WS1"]; wrote {
		t.Fatalf("must not overwrite a live scoped owner's file")
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
	if rt.wrote["/dd/cmux-owner-WS1"] != "T\tpair-T\n" {
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

// activityMTime picks the most recent time across the draft and established root.
func TestActivityMTimePicksLatest(t *testing.T) {
	rt := newFake()
	base := rt.now
	rt.mtimes["/dd/draft-T.md"] = base.Add(-time.Hour)
	rt.activities["claude"] = base // newer
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
	rt := withLivePane(newFake())
	rt.pid = "9001"
	// one live tick, then the session is gone for every poll.
	rt.sessionAliveSeq = []bool{true}
	rt.sessionAliveDflt = false
	opts := fixtureOpts()
	opts.MissThreshold = 3
	code := Run(opts, rt)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	// 1 live tick + 3 miss probes = 4; exits on the 3rd miss.
	if rt.sessionAliveCalls != 4 {
		t.Fatalf("SessionAlive called %d times, want 4 (1 live tick + 3 misses)", rt.sessionAliveCalls)
	}
}

// #287: zellij 0.45.1 panics when a connection accepted before the first real
// client closes, and `list-sessions` connects to every socket. The poller is
// spawned BEFORE `zellij --new-session-with-layout`, so its first probe used to
// land in the new server's birth window and kill it (4 launches in 10, measured
// by probes/zellijbirthrace). The gate: no zellij call of any kind until this
// launch's agent pane has written its sidecar -- a pane exists only once the
// session has initialized.
func TestRunMakesNoZellijCallBeforeThePaneAppears(t *testing.T) {
	rt := newFake()
	rt.pid = "9001"
	rt.panes = []PaneInfo{{Agent: "claude", PaneID: "7"}}
	rt.mtimes["/dd/draft-T.md"] = rt.now
	rt.sessionAliveSeq = []bool{true}
	rt.sessionAliveDflt = false
	rt.sleepHook = func(sleeps int) {
		if sleeps == 3 { // the pane is born during the third wait
			rt.mtimes[fixturePane] = rt.now
		}
	}
	opts := fixtureOpts()
	opts.MissThreshold = 1
	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}

	born := slices.Index(rt.events, "stat-pane:true")
	if born < 0 {
		t.Fatalf("the gate never saw the pane: events = %v", rt.events)
	}
	for _, e := range rt.events[:born] {
		if e == "session-alive" || e == "rename" {
			t.Fatalf("zellij call %q before the pane was born: events = %v", e, rt.events)
		}
	}
	if misses := strings.Count(strings.Join(rt.events[:born], " "), "stat-pane:false"); misses != 3 {
		t.Fatalf("the gate should have waited through 3 unborn stats, saw %d: %v", misses, rt.events)
	}
	// And once born, the steady loop resumes as before: probe, render, probe.
	if want := []string{"session-alive", "rename", "session-alive"}; !slices.Equal(rt.events[born+1:], want) {
		t.Fatalf("after birth events = %v, want %v", rt.events[born+1:], want)
	}
}

// A pane that never appears (the session died at birth, or never started) ends
// the poller at the grace deadline WITHOUT one zellij call: a probe of a server
// that is still coming up is exactly what kills it.
func TestRunExitsWithoutTouchingZellijWhenThePaneNeverAppears(t *testing.T) {
	rt := newFake()
	rt.pid = "9001"
	rt.nowAdvance = 5 * time.Second // 30s grace ⇒ a handful of stats, then give up
	rt.sessionAliveDflt = true      // would render forever if the gate leaked
	if code := Run(fixtureOpts(), rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if rt.sessionAliveCalls != 0 || len(rt.renamed) != 0 {
		t.Fatalf("no zellij call may precede birth: probes=%d renames=%v events=%v", rt.sessionAliveCalls, rt.renamed, rt.events)
	}
	if rt.sleeps == 0 {
		t.Fatalf("the gate gave up without waiting")
	}
}

// The attach shape: the live session's pane wrote its sidecar long ago, so the
// gate costs one stat and the first probe follows at once. The same rule serves
// both spawn sites because only the create path clears the sidecar.
func TestRunPassesTheGateAtOnceForALivePane(t *testing.T) {
	rt := withLivePane(newFake())
	rt.pid = "9001"
	rt.sessionAliveSeq = []bool{false}
	opts := fixtureOpts()
	opts.MissThreshold = 1
	if code := Run(opts, rt); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if want := []string{"stat-pane:true", "session-alive"}; !slices.Equal(rt.events, want) {
		t.Fatalf("events = %v, want %v", rt.events, want)
	}
}
