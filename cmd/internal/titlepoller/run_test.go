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
	sessionAliveSeq   []bool // consumed per call; falls back to sessionAliveDflt
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
func (f *fakeRuntime) SessionAlive(string) bool {
	f.sessionAliveCalls++
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

// Single-instance guard: a live poller for this tag already recorded in the
// pidfile → Run returns immediately without ever probing the session.
func TestRunDefersToLiveInstance(t *testing.T) {
	rt := newFake()
	rt.files["/dd/title-pid-T"] = "4242\n"
	rt.alive["4242"] = true
	rt.commands["4242"] = "/x/bin/pair-title T claude"
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
