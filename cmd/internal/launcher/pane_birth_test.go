package launcher

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/panebirth"
)

// #287: the title poller makes no zellij call until this launch's agent pane
// has written its sidecar (zellij 0.45.1 panics when a connection accepted
// before the first client initialized the session closes). The poller's
// evidence is only fresh because the create path clears the previous launch's
// sidecar first -- before the poller, the session watcher or zellij can start.
// A stale file surviving to the spawn would pass the gate at once and put the
// poller's first list-sessions back in the new server's birth window.
func TestCreateClearsTheAgentPaneSidecarBeforeSpawningAnything(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINTED-1"}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"})
	paths, err := artifactpath.ResolveScoped(opts.Env.DataDir, "bugfix")
	if err != nil {
		t.Fatal(err)
	}
	// What the poller this create spawns will wait for: the file that must be
	// gone when it starts.
	stale, err := panebirth.Evidence(opts.Env.DataDir, "bugfix", "claude")
	if err != nil {
		t.Fatal(err)
	}
	twin := paths.Pane("codex")
	rt.files[stale] = `{"pane_id":"3","cwd":"/old"}` + "\n"
	rt.files[twin] = `{"pane_id":"3","cwd":"/old"}` + "\n"

	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("create = %d, %v", code, err)
	}
	if len(rt.filesAtPollerSpawn) != 1 || len(rt.filesAtLaunch) != 1 {
		t.Fatalf("spawns = %d, launches = %d, want 1 each", len(rt.filesAtPollerSpawn), len(rt.filesAtLaunch))
	}
	if _, ok := rt.filesAtPollerSpawn[0][stale]; ok {
		t.Fatalf("the previous launch's %s survived to the poller's spawn", stale)
	}
	if _, ok := rt.filesAtLaunch[0][stale]; ok {
		t.Fatalf("the previous launch's %s survived to LaunchSession", stale)
	}
	// Only this agent's sidecar is the poller's evidence; a twin left by
	// another agent is not the create path's to remove.
	if _, ok := rt.filesAtPollerSpawn[0][twin]; !ok {
		t.Fatalf("the create path removed another agent's sidecar %s", twin)
	}
}

// Attach never clears: the session is live, its pane wrote the sidecar at its
// own birth and will not write it again, so the attach-path poller's gate is
// that file. Removing it would leave the poller waiting for a pane that already
// ran, and the session without titles.
func TestAttachLeavesTheLivePaneSidecar(t *testing.T) {
	rt := newFakeRuntime()
	opts := baseOpts(LaunchArgs{})
	paths, err := artifactpath.ResolveScoped(opts.Env.DataDir, "live")
	if err != nil {
		t.Fatal(err)
	}
	live := paths.Pane("claude")
	rt.files[live] = `{"pane_id":"1","cwd":"/home/u/work"}` + "\n"

	if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "📁work-live", "claude"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(rt.removed, live) {
		t.Fatalf("attach removed the live pane's sidecar %s", live)
	}
	if len(rt.filesAtPollerSpawn) != 1 {
		t.Fatalf("title pollers spawned = %d, want 1", len(rt.filesAtPollerSpawn))
	}
	if _, ok := rt.filesAtPollerSpawn[0][live]; !ok {
		t.Fatalf("the attach-path poller would find no pane sidecar to pass its gate")
	}
}

// #288: while LaunchSession blocks, the create path watches for its pane's
// birth. A client hung on a server that died at birth never returns on its
// own, so the watch ends it -- but only on proof: no pane within the bound
// AND zellij listing no live session. Each test drives the production watch
// through the fake's model of zellij: a pane command that writes (or
// withholds) the sidecar, a client that stays up until released or
// cancelled, and scripted liveness answers.

const watchedSession = "📁work-bugfix"

func watchedCreate(t *testing.T, rt *fakeRuntime) (LaunchOptions, string) {
	t.Helper()
	rt.uuids = []string{"MINTED-1"}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"})
	opts.BirthBound = 30 * time.Millisecond
	evidence, err := panebirth.Evidence(opts.Env.DataDir, "bugfix", "claude")
	if err != nil {
		t.Fatal(err)
	}
	return opts, evidence
}

func TestCreateWhosePaneNeverComesAndWhoseSessionIsGoneFails(t *testing.T) {
	r := &retainedRuntime{fakeRuntime: newFakeRuntime()}
	opts, _ := watchedCreate(t, r.fakeRuntime)
	r.launchBlock = true
	r.livenessScript = []livenessAnswer{{}} // zellij's empty inventory
	// A quit marker the normal handoff path would consume: the dead path must
	// not reach the quit cleanup, because no session ever existed.
	r.quitMarkers[watchedSession] = true

	var stderr bytes.Buffer
	start := time.Now()
	code, err := RunLaunch(opts, r, &stderr)
	if err != nil || code != 1 {
		t.Fatalf("launch = %d, %v; want 1 (stderr: %s)", code, err, stderr.String())
	}
	if !r.launchCancelled {
		t.Fatal("the hung client was not ended")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("the dead launch took %s to end", elapsed)
	}
	for _, want := range []string{"never came up", watchedSession, "zellij.log"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr lacks %q:\n%s", want, stderr.String())
		}
	}
	if _, ok := r.files[layoutRecordPath(opts.Env.DataDir, "bugfix")]; ok {
		t.Fatal("the dead launch kept the layout record it wrote")
	}
	if !r.quitMarkers[watchedSession] {
		t.Fatal("the dead launch ran the quit cleanup of a session that never existed")
	}
	if r.launchCount != 1 {
		t.Fatalf("launches = %d; a dead birth must not re-enter the loop", r.launchCount)
	}
	if last := r.events[len(r.events)-1]; last != "finish:false" {
		t.Fatalf("retention ended with %q, want finish:false (events %v)", last, r.events)
	}
}

func TestBornCreateIsNeverProbedOrEnded(t *testing.T) {
	rt := newFakeRuntime()
	opts, evidence := watchedCreate(t, rt)
	rt.launchWrites = map[string]string{evidence: `{"pane_id":"1","cwd":"/home/u/work"}` + "\n"}
	rt.launchBlock = true
	rt.launchRelease = make(chan struct{})
	time.AfterFunc(10*opts.BirthBound, func() { close(rt.launchRelease) })

	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("launch = %d, %v", code, err)
	}
	if rt.launchCancelled || rt.launchProbes != 0 {
		t.Fatalf("cancelled = %v, probes = %d; a born session is not the watch's to question", rt.launchCancelled, rt.launchProbes)
	}
}

// Once zellij has listed the session live, the watch never acts: the session
// may be healthy with a sidecar that failed to write, and ending it later
// would skip the quit cleanup.
func TestUnbornButListedLiveCreateIsLeftAlone(t *testing.T) {
	rt := newFakeRuntime()
	opts, _ := watchedCreate(t, rt)
	rt.launchBlock = true
	rt.launchRelease = make(chan struct{})
	rt.launchProbed = make(chan struct{}, 8)
	rt.livenessScript = []livenessAnswer{{sessions: []Session{{Name: watchedSession, State: SessionDetached}}}}
	go func() {
		select {
		case <-rt.launchProbed:
			time.Sleep(5 * opts.BirthBound) // room for a wrong second probe or a cancel
			close(rt.launchRelease)
		case <-time.After(5 * time.Second):
		}
	}()

	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("launch = %d, %v", code, err)
	}
	if rt.launched != watchedSession {
		t.Fatalf("launched %q; the scripted answer names %q", rt.launched, watchedSession)
	}
	if rt.launchCancelled || rt.launchProbes != 1 {
		t.Fatalf("cancelled = %v, probes = %d; want a single probe and no teardown", rt.launchCancelled, rt.launchProbes)
	}
}

// A failed probe is not an absence: the watch asks again after another bound,
// and ends the client only on an answer.
func TestUnansweredProbeIsAskedAgainNotActedOn(t *testing.T) {
	rt := newFakeRuntime()
	opts, _ := watchedCreate(t, rt)
	rt.launchBlock = true
	timeout := livenessAnswer{err: errors.New("zellij list-sessions: context deadline exceeded")}
	rt.livenessScript = []livenessAnswer{timeout, timeout, {}}

	if code, err := run(t, opts, rt); err != nil || code != 1 {
		t.Fatalf("launch = %d, %v; want 1", code, err)
	}
	if !rt.launchCancelled || rt.probesAtCancel != 3 {
		t.Fatalf("cancelled = %v after %d probes; want the cancel only after the third, answered, probe", rt.launchCancelled, rt.probesAtCancel)
	}
}

// A client that exits on its own ends the watch with it: no probe, and no
// waiting out the bound.
func TestWatchEndsWithAClientThatExitsFirst(t *testing.T) {
	rt := newFakeRuntime()
	opts, _ := watchedCreate(t, rt)
	opts.BirthBound = 0 // the production 10 s

	start := time.Now()
	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("launch = %d, %v", code, err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("the launcher took %s to return after its client did", elapsed)
	}
	if rt.launchProbes != 0 {
		t.Fatalf("probes = %d after the client had exited", rt.launchProbes)
	}
}

// The verdict must match its cause. A session that came up without its
// sidecar and was quit cleanly while the probe found nothing is a normal end:
// it keeps the handoff path and its quit cleanup, whichever of the watch's
// stop and its verdict lands first.
func TestCleanQuitRacingTheProbeKeepsTheNormalPath(t *testing.T) {
	rt := newFakeRuntime()
	opts, _ := watchedCreate(t, rt)
	rt.launchBlock = true
	rt.launchRelease = make(chan struct{})
	rt.launchReturned = make(chan struct{})
	rt.livenessScript = []livenessAnswer{{}} // by the time it answers, nothing is listed
	rt.livenessHook = func(n int) {
		if n == 1 {
			close(rt.launchRelease) // the operator quits: the client exits 0
			<-rt.launchReturned
		}
	}
	rt.quitMarkers[watchedSession] = true

	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("launch = %d, %v; want 0 (stderr: %s)", code, err, stderr.String())
	}
	if strings.Contains(stderr.String(), "never came up") {
		t.Fatalf("a clean quit was reported as a dead birth:\n%s", stderr.String())
	}
	if rt.quitMarkers[watchedSession] {
		t.Fatal("the clean quit skipped its quit cleanup")
	}
}
