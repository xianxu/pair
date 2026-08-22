package couchcore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testEnv struct {
	Couch  *Couch
	Runner *FakeRunner
	Git    *FakeGit
	Proc   *FakeProcOps
	Dir    string
	Now    time.Time
}

// newTestEnv wires the whole composition root against fakes, with a fixed
// clock and a scripted id generator so every assertion is deterministic.
func newTestEnv(t *testing.T, trees ...string) *testEnv {
	t.Helper()
	replies := map[GitCall]string{}
	for _, tr := range trees {
		replies[GitCall{Dir: tr, Args: "rev-parse --show-toplevel"}] = tr
	}
	g := NewFakeGit(replies)
	r := NewFakeRunner()
	proc := NewFakeProcOps()
	dir := t.TempDir()
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	c, err := New(r, NewFakePathOps(nil), g, proc, NewStore(dir), FixedClock{T: now}, NewFixedIDGen("ah8d", "b2c1"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return &testEnv{Couch: c, Runner: r, Git: g, Proc: proc, Dir: dir, Now: now}
}

// spawn spawns and then marks the child live in FakeProcOps, which is what a
// real process would be. Tests that skip this are modelling a DEAD actor --
// which is a legitimate scenario, just not the default one.
func (e *testEnv) spawn(t *testing.T, args StartArgs) (ActorRecord, Handle) {
	t.Helper()
	rec, h, err := e.Couch.Spawn(args)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	e.Proc.Set(rec.PID, rec.Identity)
	return rec, h
}

func (e *testEnv) cannedTree(tree, cwd string) {
	e.Git.replies[GitCall{Dir: cwd, Args: "rev-parse --show-toplevel"}] = tree
}

func TestSpawnStartsPairAndRecordsTheActor(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, h, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if rec.ID != "couch-ah8d" {
		t.Fatalf("id = %q", rec.ID)
	}
	// couch spawns pair, not claude: pair owns zellij, the layout, and the
	// agent's resume/session-id knowledge.
	if got := env.Runner.Ops[0]; got != "start /repo: pair --layout2" {
		t.Fatalf("Ops[0] = %q", got)
	}
	if !rec.StartedAt.Equal(env.Now) {
		t.Fatalf("StartedAt = %v, want the injected clock", rec.StartedAt)
	}
	if rec.Identity == "" || rec.PID == 0 {
		t.Fatalf("liveness fields not recorded: %+v", rec)
	}
	_ = h
}

func TestSpawnStartsInASubdirectoryButRegistersTheTree(t *testing.T) {
	// The kbench/competition/arc-agi-3 case.
	env := newTestEnv(t)
	env.cannedTree("/w/kbench", "/w/kbench/competition/arc-agi-3")
	rec, _, err := env.Couch.Spawn(StartArgs{Cwd: "/w/kbench/competition/arc-agi-3"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if rec.Args.Worktree != "/w/kbench" {
		t.Fatalf("registered under %q, want the tree root", rec.Args.Worktree)
	}
	if got := env.Runner.Child(env.Runner.order[0]).Dir; got != "/w/kbench/competition/arc-agi-3" {
		t.Fatalf("child started in %q, want the requested subdirectory", got)
	}
}

func TestRefusedSpawnStartsNoProcess(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	before := len(env.Runner.Ops)
	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	var occ *TreeOccupiedError
	if !errors.As(err, &occ) {
		t.Fatalf("err = %v, want *TreeOccupiedError", err)
	}
	if len(env.Runner.Ops) != before {
		t.Fatal("a refused spawn must not fork a child")
	}
}

func TestSnapshotIsOnDiskWhileTheChildIsStillAlive(t *testing.T) {
	// `couch start` blocks for the child's lifetime, so if Save happened after
	// Wait a second shell running `couch list` would see nothing for the whole
	// session -- which is most of the time.
	env := newTestEnv(t, "/repo")
	_, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	if !h.Alive() {
		t.Fatal("child should still be running")
	}
	raw, err := os.ReadFile(filepath.Join(env.Dir, "registry.json"))
	if err != nil {
		t.Fatalf("snapshot not written before Wait: %v", err)
	}
	var snap struct {
		Actors []ActorRecord `json:"actors"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snap.Actors) != 1 {
		t.Fatalf("snapshot has %d actors", len(snap.Actors))
	}
}

func TestSpawnFailureLeavesTheTreeFree(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Runner.FailNextStart(errors.New("boom"))
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("expected a start failure")
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err != nil {
		t.Fatalf("tree still held after a failed spawn: %v", err)
	}
}

func TestIsLiveRejectsARecycledPID(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if !env.Couch.IsLive(rec) {
		t.Fatal("a running actor must read as live")
	}
	// Same PID, different process: the kernel start token differs.
	env.Proc.Set(rec.PID, "some-other-process")
	if env.Couch.IsLive(rec) {
		t.Fatal("a recycled PID must not read as the original actor")
	}
	env.Proc.Kill(rec.PID)
	if env.Couch.IsLive(rec) {
		t.Fatal("a dead PID must not read as live")
	}
}

func TestResolveRefFindsActorsByOperatorName(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if err := env.Couch.SetName("/repo", "refactor thing"); err != nil {
		t.Fatalf("SetName: %v", err)
	}
	got, _, err := env.Couch.ResolveRef("refactor")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if len(got) != 1 || got[0].ID != rec.ID {
		t.Fatalf("ResolveRef = %+v", got)
	}
}

func TestNameAndDescriptionChangeMidSession(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	_ = env.Couch.SetName("/repo", "first")
	_ = env.Couch.SetName("/repo", "second")
	if got, _, err := env.Couch.ResolveRef("second"); err != nil || len(got) != 1 {
		t.Fatalf("rename did not take effect: %v %v", got, err)
	}
	if got, _, err := env.Couch.ResolveRef("first"); err == nil && len(got) > 0 {
		t.Fatal("the old name must stop resolving")
	}
	_ = env.Couch.SetDescription("/repo", "reworking the composer gate")
	if got, _, err := env.Couch.ResolveRef("composer"); err != nil || len(got) != 1 {
		t.Fatalf("description did not take effect: %v %v", got, err)
	}
}

func TestNameSurvivesActorReplacement(t *testing.T) {
	// A real lifecycle: spawn, name, the child exits, the actor is forgotten,
	// a new one is spawned. The name must still resolve, because it hangs off
	// the tree rather than the incarnation.
	env := newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	_ = env.Couch.SetName("/repo", "long lived")

	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID)
	if err := env.Couch.Forget("/repo", first.ID); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	second, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if second.ID == first.ID {
		t.Fatal("the revival must be a new incarnation")
	}
	got, _, err := env.Couch.ResolveRef("long lived")
	if err != nil || len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("name lost across revival: %+v %v", got, err)
	}
}

// --- close-review regressions: each of these fails against the shipped code ---

func TestDeadActorDoesNotBlockItsTreeForever(t *testing.T) {
	// BR-1. `couch start` blocks until the child exits and nothing unregisters
	// on exit, so the ORDINARY end of a session used to leave a record that
	// refused its own tree permanently.
	env := newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})

	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID) // the process is gone; the record is not

	second, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("a dead actor still refused its tree: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("expected a fresh incarnation")
	}
	if got := env.Couch.Get("/repo"); len(got) != 1 {
		t.Fatalf("tree holds %d actors; the dead one should have been pruned", len(got))
	}
}

func TestLiveActorStillBlocksItsTree(t *testing.T) {
	// The complement of BR-1: pruning must not weaken the guard.
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("a live actor must still refuse its tree")
	}
}

func TestStopSignalsTheChildBeforeForgettingIt(t *testing.T) {
	// BR-2. Forgetting first frees the tree while the agent keeps running, so
	// the next start is allowed and two agents share one index lock.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})

	signalled, err := env.Couch.Stop(rec)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !signalled {
		t.Fatal("Stop must signal a live child, not merely forget it")
	}
	got := env.Proc.Signals[rec.PID]
	if len(got) != 1 || got[0] != TermSignal {
		t.Fatalf("signals = %v, want one SIGTERM", got)
	}
	if len(env.Couch.Get("/repo")) != 0 {
		t.Fatal("the record should be gone after Stop")
	}
}

func TestStopOnADeadActorForgetsWithoutSignalling(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(rec.PID)

	signalled, err := env.Couch.Stop(rec)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if signalled {
		t.Fatal("a dead actor must not be reported as signalled -- that implies a running agent was terminated")
	}
}

func TestShowFilterRestrictsRatherThanAdds(t *testing.T) {
	// BR-3. Summarize took a filter and then folded in every registry record,
	// so `show <ref>` printed exactly what `list` printed. The old test passed
	// only because its fixture had a single tree.
	env := newTestEnv(t, "/repo", "/other")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	env.spawn(t, StartArgs{Worktree: "/other"})

	got := env.Couch.Summarize([]Worktree{"/repo"})
	if len(got) != 1 {
		var trees []Worktree
		for _, s := range got {
			trees = append(trees, s.Tree)
		}
		t.Fatalf("Summarize([/repo]) returned %v; a filter must restrict, not add", trees)
	}
	if got[0].Tree != "/repo" {
		t.Fatalf("returned %q", got[0].Tree)
	}
	if len(env.Couch.Summarize(nil)) != 2 {
		t.Fatal("an empty filter must still list everything")
	}
}

func TestReplayPreservesSameTreeExactly(t *testing.T) {
	// BR-4. Load used to set SameTree=true on every record to dodge its own
	// re-register refusal, and the next Save persisted the lie -- after which
	// no reader could tell which actors really used the escape hatch.
	dir := t.TempDir()
	s := NewStore(dir)
	reg := NewRegistry().Insert(ActorRecord{ID: "plain", Args: StartArgs{Worktree: "/repo"}})
	reg = reg.Insert(ActorRecord{ID: "hatch", Args: StartArgs{Worktree: "/repo", SameTree: true}})
	if err := s.Save(reg, NewNamingTable()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, names, _, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Save(loaded, names); err != nil { // round two is where the lie used to stick
		t.Fatalf("re-Save: %v", err)
	}
	again, _, _, _ := s.Load()

	byID := map[ActorID]bool{}
	for _, r := range again.Records() {
		byID[r.ID] = r.Args.SameTree
	}
	if byID["plain"] {
		t.Error("SameTree fabricated on a record that never used the escape hatch")
	}
	if !byID["hatch"] {
		t.Error("SameTree lost on a record that did use it")
	}
}

func TestUnreadableRegistryErrorsRatherThanReadingAsFirstRun(t *testing.T) {
	// BR-5. Load discarded every ReadFile error, so an unreadable snapshot
	// looked like a fresh install and the next Save destroyed it.
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Save(NewRegistry().Insert(ActorRecord{ID: "a", Args: StartArgs{Worktree: "/repo"}}), NewNamingTable()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(filepath.Join(dir, "registry.json"), 0o000); err != nil {
		t.Skipf("cannot chmod in this environment: %v", err)
	}
	defer func() { _ = os.Chmod(filepath.Join(dir, "registry.json"), 0o644) }()

	if _, _, _, err := s.Load(); err == nil {
		t.Fatal("an unreadable registry must error, not read as an empty one")
	}
}

func TestAliveIsFalseForAnExitedChildWithoutCallingWait(t *testing.T) {
	// BR-8. This pins the reaper in the DEFAULT suite. procutil.Alive is
	// `kill -0`, which succeeds for a zombie, so the pre-fix implementation
	// reported an exited-but-unreaped child as running.
	h, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 0"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !h.Alive() {
			return // reaped without anyone calling Wait
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("Alive() stayed true for an exited child -- a zombie is being reported as running")
}

func TestSpawnTellsTheChildWhichTreeItIs(t *testing.T) {
	// BR-9. Without COUCH_TREE and COUCH_STORE_DIR the agent cannot publish a
	// description, and Describe's cache has nothing to cache from.
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	got := env.Runner.Child(env.Runner.order[0]).Env
	var tree, store bool
	for _, kv := range got {
		if kv == "COUCH_TREE=/repo" {
			tree = true
		}
		if len(kv) > len("COUCH_STORE_DIR=") && kv[:len("COUCH_STORE_DIR=")] == "COUCH_STORE_DIR=" {
			store = true
		}
	}
	if !tree || !store {
		t.Fatalf("child env = %v; needs COUCH_TREE and COUCH_STORE_DIR", got)
	}
}

func TestDescribePrefersTheAgentsPublishedLineOverTheOperators(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	if err := env.Couch.SetDescription("/repo", "what the operator typed"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}
	if err := env.Couch.PublishDescription("/repo", "what the agent is doing"); err != nil {
		t.Fatalf("PublishDescription: %v", err)
	}
	if got := env.Couch.Describe("/repo"); got != "what the agent is doing" {
		t.Fatalf("Describe = %q; the agent's own line must win", got)
	}
}

func TestPruneKeepsRecordsWhoseLivenessIsUnknown(t *testing.T) {
	// The smoke-test bug: a probe that could not answer read as "dead", the
	// record was pruned, and a second agent was let onto a tree that already
	// had a running one. Unknown must fail CLOSED.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.SetUnknown(rec.PID)

	if got := env.Couch.Liveness(rec); got != Unknown {
		t.Fatalf("Liveness = %v, want Unknown", got)
	}
	if err := env.Couch.PruneDead(); err != nil {
		t.Fatalf("PruneDead: %v", err)
	}
	if len(env.Couch.Get("/repo")) != 1 {
		t.Fatal("an unknown-liveness record was pruned; the guard now protects nothing")
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("a second agent was admitted while the incumbent's state was unknown")
	}
}

func TestUnreadableIdentityIsUnknownNotDead(t *testing.T) {
	// A process that exists but whose token cannot be read is not evidence of
	// anything, so it must not be treated as gone.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.IdentityErr[rec.PID] = true

	if got := env.Couch.Liveness(rec); got != Unknown {
		t.Fatalf("Liveness = %v, want Unknown", got)
	}
}

func TestStopSignalsEvenWhenLivenessIsUnknown(t *testing.T) {
	// Refusing to signal because we could not confirm liveness would free the
	// tree while leaving a running agent behind -- the hazard Stop closes.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.SetUnknown(rec.PID)

	signalled, err := env.Couch.Stop(rec)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !signalled {
		t.Fatal("Stop must attempt a signal when liveness is unknown")
	}
}

func TestKnownDeadIsStillPruned(t *testing.T) {
	// The complement: failing closed must not disable pruning entirely, or
	// BR-1 comes back.
	env := newTestEnv(t, "/repo")
	rec, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(rec.PID)

	if got := env.Couch.Liveness(rec); got != Dead {
		t.Fatalf("Liveness = %v, want Dead", got)
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err != nil {
		t.Fatalf("a known-dead actor still refused its tree: %v", err)
	}
}

func TestAgentPublishedDescriptionResolvesNotJustDisplays(t *testing.T) {
	// BR-23. Display derived from the agent's published line while resolution
	// still only searched the operator's -- half of Done-when 3.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if err := env.Couch.PublishDescription("/repo", "reworking the composer gate"); err != nil {
		t.Fatalf("PublishDescription: %v", err)
	}
	got, _, err := env.Couch.ResolveRef("composer")
	if err != nil {
		t.Fatalf("ResolveRef: %v -- the agent's own line must resolve, not only render", err)
	}
	if len(got) != 1 || got[0].ID != rec.ID {
		t.Fatalf("ResolveRef = %+v", got)
	}
}

func TestCoTenantsAreAddressableByActorID(t *testing.T) {
	// BR-24. --same-tree co-tenants share a path and a label, so without an
	// ActorID branch the escape hatch creates a state couch cannot exit.
	env := newTestEnv(t, "/repo")
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	second, _ := env.spawn(t, StartArgs{Worktree: "/repo", SameTree: true})
	if first.ID == second.ID {
		t.Fatal("expected two distinct actors")
	}

	if got, _, err := env.Couch.ResolveRef("/repo"); err != nil || len(got) != 2 {
		t.Fatalf("path ref resolved to %+v (%v), want both co-tenants", got, err)
	}
	got, _, err := env.Couch.ResolveRef(string(second.ID))
	if err != nil {
		t.Fatalf("ResolveRef by id: %v", err)
	}
	if len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("ResolveRef(%q) = %+v, want exactly that actor", second.ID, got)
	}
	if _, err := env.Couch.Stop(got[0]); err != nil {
		t.Fatalf("Stop by id: %v", err)
	}
	if len(env.Couch.Get("/repo")) != 1 {
		t.Fatal("stopping one co-tenant must leave the other")
	}
}

func TestUnknownRefSaysMissingNotAmbiguous(t *testing.T) {
	env := newTestEnv(t, "/repo")
	_, _, err := env.Couch.ResolveRef("nothing-like-this")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no actor or tree matches") {
		t.Fatalf("err = %v; absence must not read as ambiguity", err)
	}
}
