package couchcore

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// #287: a cold resume starts a new zellij server, and zellij 0.45.1 panics when
// a connection it accepted before the first client initialized the session
// closes. The registration poll's PairSession runs `list-sessions` every 10 ms,
// which connects to every socket, and at that cadence every cold launch it
// overlapped died (probes/zellijbirthrace -hammer 10ms: 10/10). So no session
// probe may run until the thread's pane has been born, meaning its sidecar
// changed since a baseline taken while the helper was still blocked.
//
// The previous launch's sidecars sit on disk throughout: ours, which Pair's
// create path clears after the ack and the new pane rewrites, and a twin
// another agent left. Neither the stale files nor the clear may pass for a
// birth.
func TestColdResumeMakesNoSessionProbeBeforeThePaneIsBorn(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPaneSidecar(address, "claude")
	env.Artifacts.SetPaneSidecar(address, "codex")

	baselines := -1 // pane observations made before the helper was released
	env.Runner.AfterAcknowledge = func(string) error {
		baselines = env.Artifacts.PaneQueries()
		env.Artifacts.ClearPaneSidecar(address, "claude") // Pair's create path
		return nil
	}
	waits, born := 0, false
	env.Artifacts.PaneSidecarsHook = func(ThreadAddress) error {
		if baselines < 0 {
			return nil
		}
		waits++
		if waits == 3 { // the new pane writes its sidecar: the session is up
			born = true
			env.Artifacts.SetPaneSidecar(address, "claude")
			env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), true)
		}
		return nil
	}
	env.Artifacts.BeforePairSession = func(ThreadAddress) error {
		if baselines >= 0 && !born {
			t.Errorf("a session probe (list-sessions) ran after the ack and before the pane was born (wait %d)", waits)
		}
		return nil
	}

	if _, _, err := env.Couch.Resume(address); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if baselines != 1 {
		t.Fatalf("pane observations before the helper was released = %d, want 1 (the baseline)", baselines)
	}
	if !born {
		t.Fatalf("registration finished without waiting for the pane (waits = %d)", waits)
	}
}

// A warm reattach attaches to a live session, and its pane was born long ago, so
// there is nothing to wait for and no baseline to take.
func TestWarmReattachSkipsTheBirthWait(t *testing.T) {
	env := newTestEnv(t, "/repo")
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.Reservation = false
	record.LatestLaunchProfile = &profile
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(created.Address, "pair-"+string(created.Address.Tag))
	env.Artifacts.SetPairSession(created.Address, "pair-"+string(created.Address.Tag), true)

	if _, _, err := env.Couch.Resume(created.Address); err != nil {
		t.Fatalf("warm reattach: %v", err)
	}
	if n := env.Artifacts.PaneQueries(); n != 0 {
		t.Fatalf("warm reattach observed pane sidecars %d times, want 0", n)
	}
}

// A pane that is never born (the server died at birth, or Pair never got that
// far) ends in the ordinary registration failure and its diagnosis, not in a
// probe of a server that may still be coming up.
func TestColdResumeTimesOutWhenThePaneIsNeverBorn(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.resumeRegistrationTimeout = 150 * time.Millisecond
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	released := false
	env.Runner.AfterAcknowledge = func(string) error {
		released = true
		return nil
	}
	var paneQueriesAtProbe []int
	env.Artifacts.BeforePairSession = func(ThreadAddress) error {
		if released { // the pre-release liveness check is not part of the wait
			paneQueriesAtProbe = append(paneQueriesAtProbe, env.Artifacts.PaneQueries())
		}
		return nil
	}

	_, _, err := env.Couch.Resume(parked.Address)
	if err == nil || !strings.Contains(err.Error(), "await Pair registration") {
		t.Fatalf("Resume error = %v, want the registration failure", err)
	}
	// Probes do follow: the failure's diagnosis and the start cleanup's presence
	// check both ask once the deadline has passed. None may interleave with the
	// wait, so no pane observation can come after the first probe.
	waited := env.Artifacts.PaneQueries()
	if waited < 3 {
		t.Fatalf("pane observations = %d; the wait did not poll", waited)
	}
	for _, at := range paneQueriesAtProbe {
		if at != waited {
			t.Fatalf("a session probe ran at pane observation %d of %d, inside the birth wait", at, waited)
		}
	}
}

// A session that is live but ATTACHED fails the detached proof, so it reaches
// the cold path. Pair refuses that resume (it isn't a create boundary) and no
// pane is ever written. Waiting for one would run out the registration
// deadline, and the cold-resume cleanup owns the session: it would delete a
// live agent. So a session that is live before the helper is released means
// no birth is coming, and there is nothing to wait for (#287 close review).
func TestColdResumeAgainstALiveSessionNeitherWaitsNorDeletesIt(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.resumeRegistrationTimeout = 150 * time.Millisecond
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), true) // live, not detached

	if _, _, err := env.Couch.Resume(address); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if containsAddress(env.Artifacts.Quiesces(), address) {
		t.Fatalf("a cold resume deleted the live session of %+v", address)
	}
	if n := env.Artifacts.PaneQueries(); n != 0 {
		t.Fatalf("pane observations = %d; a live session has no birth to wait for", n)
	}
}

// Whether the session is live decides whether a birth is coming, so a check
// that cannot be answered fails the start before Pair is released. Guessing
// "not live" means a wait that may time out into a deletion; guessing "live"
// means probing during a birth.
func TestColdResumeRefusesToReleasePairWhenLivenessIsUnknown(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("zellij unreachable") }
	released := false
	env.Runner.AfterAcknowledge = func(string) error {
		released = true
		return nil
	}

	_, _, err := env.Couch.Resume(parked.Address)
	if err == nil || !strings.Contains(err.Error(), "observe session before cold resume") {
		t.Fatalf("Resume error = %v, want the liveness failure", err)
	}
	if released {
		t.Fatalf("the helper was released without knowing whether a birth is coming")
	}
}

// A failed observation mid-wait is "not yet": the wait goes on. Ending it would
// fail registration, and the cleanup would then delete the session this launch
// had just created.
func TestColdResumeWaitsThroughAFailedPaneObservation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	released := false
	env.Runner.AfterAcknowledge = func(string) error {
		released = true
		return nil
	}
	waits := 0
	env.Artifacts.PaneSidecarsHook = func(ThreadAddress) error {
		if !released {
			return nil
		}
		waits++
		switch waits {
		case 1:
			return errors.New("transient stat failure")
		case 2:
			env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), true) // up: pane born
		}
		return nil
	}

	if _, _, err := env.Couch.Resume(address); err != nil {
		t.Fatalf("Resume after a transient observation failure: %v", err)
	}
	if containsAddress(env.Artifacts.Quiesces(), address) {
		t.Fatalf("the session this launch created was quiesced")
	}
}

// A baseline that cannot be observed fails the start BEFORE the helper is
// released. Without a baseline nothing could be recognised as a birth, and the
// alternative, probing anyway, is what killed the session.
func TestColdResumeRefusesToReleasePairWithoutAPaneBaseline(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.PaneSidecarsHook = func(ThreadAddress) error { return errors.New("scope unreadable") }
	released := false
	env.Runner.AfterAcknowledge = func(string) error {
		released = true
		return nil
	}

	_, _, err := env.Couch.Resume(parked.Address)
	if err == nil || !strings.Contains(err.Error(), "observe pane sidecars") {
		t.Fatalf("Resume error = %v, want the baseline failure", err)
	}
	if released {
		t.Fatalf("the helper was released without a pane baseline")
	}
}

// The real observer: the thread's own scope directory, glob and stat, no zellij.
func TestScopedPaneSidecarsObservesTheThreadsScope(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)

	marks, err := checker.PaneSidecars(address)
	if err != nil || len(marks) != 0 {
		t.Fatalf("no scope dir yet: marks = %v, err = %v; want empty, nil", marks, err)
	}

	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	want := PaneMarks{}
	for _, agent := range []string{"claude", "codex"} {
		path := paths.Pane(agent)
		if err := os.WriteFile(path, []byte(`{"pane_id":"1"}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want[path] = info.ModTime()
	}
	// Other families in the same directory are not panes.
	if err := os.WriteFile(filepath.Join(paths.ScopeDir(), "draft-"+string(address.Tag)+".md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	marks, err = checker.PaneSidecars(address)
	if err != nil {
		t.Fatal(err)
	}
	if !maps.EqualFunc(marks, want, time.Time.Equal) {
		t.Fatalf("marks = %v, want %v", slices.Sorted(maps.Keys(marks)), slices.Sorted(maps.Keys(want)))
	}
}
