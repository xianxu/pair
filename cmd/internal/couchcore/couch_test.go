package couchcore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err != nil {
		t.Fatalf("first: %v", err)
	}
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
	_, h, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
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
	rec, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	env.Proc.Set(rec.PID, rec.Identity)
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
	rec, _, _ := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
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
	_, _, _ = env.Couch.Spawn(StartArgs{Worktree: "/repo"})
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
	first, h, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	_ = env.Couch.SetName("/repo", "long lived")

	env.Runner.SetExited(h.ID(), 0)
	if err := env.Couch.Forget("/repo", first.ID); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	second, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("respawn: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("the revival must be a new incarnation")
	}
	got, _, err := env.Couch.ResolveRef("long lived")
	if err != nil || len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("name lost across revival: %+v %v", got, err)
	}
}
