package launcher

import (
	"reflect"
	"strings"
	"testing"
)

// pair#228 sites 4-6, counted at the fake-Runtime seam. `pair resume <tag>` --
// couch's reattach -- reads only liveness, so it must take NO full snapshot
// (each asks every live session for its clients, ~250 ms apiece) and must not
// re-probe a name zellij has already accepted by running a session under it.
func TestForcedResumeTakesLivenessNotAFullSnapshot(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessions = []Session{{Name: "📁work-live", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-live", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "live",
	}}}
	rt.blocksReuse["📁work-live"] = true
	rt.inferAgent["live"] = "codex"
	opts := baseOpts(LaunchArgs{ForcedTag: "live"})
	opts.SkipConfigPicker = true

	if _, err := run(t, opts, rt); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rt.attached, []string{"📁work-live"}) {
		t.Fatalf("attached = %v, want the live session", rt.attached)
	}
	if rt.sessionsCalls != 0 || rt.livenessCalls != 2 {
		t.Fatalf("full snapshots = %d, liveness snapshots = %d; want 0 and 2 (the sweep and runOnce)", rt.sessionsCalls, rt.livenessCalls)
	}
	if rt.probeCount != 0 {
		t.Fatalf("name probes = %d, want 0: a live session under the name proves zellij accepts it", rt.probeCount)
	}
}

// The other half: bare `pair` DOES read attach state (a detached session sends
// it to the picker), so it still takes a full snapshot.
func TestBarePairStillTakesAFullSnapshotForThePicker(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessions = []Session{{Name: "📁work-live", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-live", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "live",
	}}}
	picked := false
	rt.pickFunc = func(string, []string) string { picked = true; return "" }

	if _, err := run(t, baseOpts(LaunchArgs{}), rt); err != nil {
		t.Fatal(err)
	}
	if !picked {
		t.Fatal("bare pair with a detached session did not reach the picker")
	}
	if rt.sessionsCalls < 1 {
		t.Fatalf("full snapshots = %d, want at least 1: the picker needs attach state", rt.sessionsCalls)
	}
}

// Site 6's rule, in the function that owns it. A non-exited session under the
// name proves zellij accepts its length, so it is not probed; an exited one and
// a brand-new candidate still are (#215's acceptance behaviour).
func TestAssignSessionNameAcceptsALiveNameWithoutProbing(t *testing.T) {
	scope := mustScope(t, "/home/u/work")
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-live", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "live",
	}}}
	probes := 0
	accepts := func(string) bool { probes++; return true }

	for _, state := range []SessionState{SessionLive, SessionDetached, SessionAttached} {
		probes = 0
		name, _, err := AssignSessionName(index, []Session{{Name: "📁work-live", State: state}}, scope, "live", accepts)
		if err != nil || name != "📁work-live" || probes != 0 {
			t.Fatalf("%s prior name: got (%q, %v) with %d probes, want 📁work-live with 0", state, name, err, probes)
		}
	}

	probes = 0
	if _, _, err := AssignSessionName(index, []Session{{Name: "📁work-live", State: SessionExited}}, scope, "live", accepts); err != nil {
		t.Fatal(err)
	}
	if probes != 1 {
		t.Fatalf("an EXITED session proved acceptance: %d probes, want 1", probes)
	}

	probes = 0
	if _, _, err := AssignSessionName(index, []Session{{Name: "📁work-live", State: SessionLive}}, scope, "fresh", accepts); err != nil {
		t.Fatal(err)
	}
	if probes == 0 {
		t.Fatal("a new tag's candidate name was accepted without a probe")
	}
}

// Every reader of attached-versus-detached refuses a liveness snapshot, through
// one rule (pair#228 close review: the guard had reached one reader of the
// class, not all three). This is the picker's member; DecideLaunch's is
// TestDecideLaunchRefusesAttachStateItWasNotGiven and couchcore's is
// TestProjectDetachedSessionsRefusesAttachStateItWasNotGiven.
func TestThePickerRefusesAttachStateItWasNotGiven(t *testing.T) {
	rt := newFakeRuntime()
	rt.pickFunc = func(string, []string) string {
		t.Fatal("the picker opened on sessions whose attach state was never asked")
		return ""
	}
	var stderr strings.Builder
	snap := SessionSnapshot{BaseTag: "work", Sessions: []Session{{Name: "📁work-live", State: SessionLive}}}
	_, aborted, code := resolvePickWithPolicy(rt, snap, "work", 0, PickPolicy{}, &stderr)
	if !aborted || code != 1 || !strings.Contains(stderr.String(), "attach state") {
		t.Fatalf("aborted=%v code=%d stderr=%q, want an abort naming the missing attach state", aborted, code, stderr.String())
	}
}
