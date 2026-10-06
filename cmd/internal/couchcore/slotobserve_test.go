package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// observedSlot is a provisioned slot 1 on a ProvisionFixture, with helpers
// that break exactly one resource.
type observedSlot struct {
	f      *ProvisionFixture
	layout SlotLayout
	io     ProvisionIO
}

func newObservedSlot(t *testing.T) *observedSlot {
	t.Helper()
	f := newProvisionFixture(t)
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	common := f.git(f.Primary, "rev-parse", "--path-format=absolute", "--git-common-dir")
	return &observedSlot{f: f, layout: NewSlotLayout(f.Primary, common, 1), io: f}
}

func (s *observedSlot) observe(t *testing.T) SlotObservation {
	t.Helper()
	return ObserveSlot(context.Background(), SlotObserveInput{IO: s.io, Layout: s.layout, Agent: AgentNone})
}

func (s *observedSlot) admin(t *testing.T) string {
	t.Helper()
	return s.f.git(s.layout.Host(), "rev-parse", "--absolute-git-dir")
}

// addDep declares substrate ../<name> in the host and clones a layer there.
func (s *observedSlot) addDep(t *testing.T, name string) string {
	t.Helper()
	write(t, filepath.Join(s.layout.Host(), "construct", "deps"), "substrate ../"+name+"\n")
	dep := filepath.Join(s.layout.Env(), name)
	if err := os.MkdirAll(dep, 0o755); err != nil {
		t.Fatal(err)
	}
	s.f.git(dep, "init", "-q", "-b", "main")
	write(t, filepath.Join(dep, "construct", "base.manifest"), "# layer\n")
	s.f.git(dep, "add", ".")
	s.f.git(dep, "commit", "-q", "-m", "layer")
	return dep
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type wantObservation struct {
	id    SlotResourceID
	state ObservedState
	sub   string
}

// assertObserved checks the named resources and that every other resource of
// the table keeps its healthy reading, so a break shows up exactly where made.
func assertObserved(t *testing.T, got SlotObservation, want ...wantObservation) {
	t.Helper()
	healthy := map[SlotResourceID]ObservedState{
		ResourceEnv: StatePresent, ResourceStore: StatePresent, ResourceIntent: StateAbsent, ResourceBranch: StatePresent,
		ResourceUpstream: StatePresent, ResourceRegistration: StatePresent, ResourceHost: StatePresent,
		ResourceDeps: StatePresent, ResourceSetup: StatePresent, ResourceAgent: StateAbsent,
	}
	expect := map[SlotResourceID]wantObservation{}
	for _, w := range want {
		expect[w.id] = w
	}
	for id, state := range healthy {
		if _, named := expect[id]; !named {
			expect[id] = wantObservation{id: id, state: state}
		}
	}
	for id, w := range expect {
		r, ok := got.Get(id)
		if !ok {
			t.Errorf("%s not observed", id)
			continue
		}
		if r.State != w.state || r.Sub != w.sub {
			t.Errorf("%s = %s/%q (%s), want %s/%q", id, r.State, r.Sub, r.Reason, w.state, w.sub)
		}
	}
}

func TestObserveHealthySlot(t *testing.T) {
	s := newObservedSlot(t)
	if err := os.MkdirAll(s.layout.Store(), 0o700); err != nil {
		t.Fatal(err)
	}
	assertObserved(t, s.observe(t))
}

func TestObserveEachBrokenResource(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(t *testing.T, s *observedSlot)
		want   []wantObservation
	}{
		{"env deleted", func(t *testing.T, s *observedSlot) { os.RemoveAll(s.layout.Env()) }, []wantObservation{
			{ResourceEnv, StateAbsent, ""}, {ResourceStore, StatePending, ""}, {ResourceRegistration, StateBroken, SubStale},
			{ResourceHost, StatePending, ""}, {ResourceDeps, StatePending, ""}, {ResourceSetup, StatePending, ""}}},
		{"host deleted", func(t *testing.T, s *observedSlot) { os.RemoveAll(s.layout.Host()) }, []wantObservation{
			{ResourceRegistration, StateBroken, SubStale}, {ResourceHost, StateAbsent, ""}, {ResourceDeps, StatePending, ""}, {ResourceSetup, StatePending, ""}}},
		{"marker deleted", func(t *testing.T, s *observedSlot) { os.Remove(SetupMarkerPath(s.admin(t))) }, []wantObservation{
			{ResourceSetup, StateAbsent, ""}}},
		{"marker for another slot", func(t *testing.T, s *observedSlot) {
			path := SetupMarkerPath(s.admin(t))
			raw, _ := os.ReadFile(path)
			os.WriteFile(path, []byte(strings.Replace(string(raw), `"slot":1`, `"slot":2`, 1)), 0o600)
		}, []wantObservation{{ResourceSetup, StateBroken, ""}}},
		{"weave lock held", func(t *testing.T, s *observedSlot) { holdLock(t, s.layout.SetupLock()) }, []wantObservation{
			{ResourceSetup, StatePresent, SubLockHeld}}},
		{"stale creation intent", func(t *testing.T, s *observedSlot) { write(t, s.layout.Intent(), "{}") }, []wantObservation{
			{ResourceIntent, StatePresent, ""}}},
		{"resting branch checked out elsewhere", func(t *testing.T, s *observedSlot) {
			s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
			s.f.git(s.f.Primary, "worktree", "add", "-q", filepath.Join(t.TempDir(), "elsewhere"), s.layout.RestingBranch())
		}, []wantObservation{{ResourceBranch, StateBroken, SubElsewhere}, {ResourceUpstream, StatePending, ""}}},
		{"upstream merges another branch", func(t *testing.T, s *observedSlot) {
			s.f.git(s.f.Primary, "config", "branch."+s.layout.RestingBranch()+".merge", "refs/heads/other")
		}, []wantObservation{{ResourceUpstream, StateBroken, SubConflict}}},
		{"host checkout unreadable", func(t *testing.T, s *observedSlot) {
			write(t, filepath.Join(s.layout.Host(), ".git"), "gitdir: /nonexistent/admin\n")
		}, []wantObservation{{ResourceHost, StateBroken, SubUnreadable}, {ResourceDeps, StatePending, ""}, {ResourceSetup, StatePending, ""}}},
		{"malformed construct/deps", func(t *testing.T, s *observedSlot) {
			write(t, filepath.Join(s.layout.Host(), "construct", "deps"), "substrate\n")
		}, []wantObservation{{ResourceDeps, StateUnknown, ""}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newObservedSlot(t)
			os.MkdirAll(s.layout.Store(), 0o700)
			c.break_(t, s)
			assertObserved(t, s.observe(t), c.want...)
		})
	}
}

func TestObserveHostBrokenKeepsTheRecordedBranch(t *testing.T) {
	s := newObservedSlot(t)
	s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
	write(t, filepath.Join(s.layout.Host(), ".git"), "gitdir: /nonexistent/admin\n")
	host, _ := s.observe(t).Get(ResourceHost)
	if host.State != StateBroken || host.Branch != "issue-work" {
		t.Fatalf("host = %+v, want broken with the registration's branch issue-work", host)
	}
}

func TestObserveDependencies(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(t *testing.T, s *observedSlot, dep string)
		dep    wantObservation
		setup  wantObservation
	}{
		{"healthy", func(*testing.T, *observedSlot, string) {}, wantObservation{DepResource("ariadne"), StatePresent, ""}, wantObservation{ResourceSetup, StatePresent, ""}},
		{"deleted", func(t *testing.T, s *observedSlot, dep string) { os.RemoveAll(dep) },
			wantObservation{DepResource("ariadne"), StateAbsent, ""}, wantObservation{ResourceSetup, StateAbsent, SubMarkerValid}},
		{"corrupt .git", func(t *testing.T, s *observedSlot, dep string) { os.Remove(filepath.Join(dep, ".git", "HEAD")) },
			wantObservation{DepResource("ariadne"), StateBroken, SubUnreadable}, wantObservation{ResourceSetup, StateAbsent, SubMarkerValid}},
		{"dirty with a local commit", func(t *testing.T, s *observedSlot, dep string) {
			write(t, filepath.Join(dep, "untracked.txt"), "x")
			write(t, filepath.Join(dep, "construct", "base.manifest"), "# edited\n")
			s.f.git(dep, "commit", "-q", "--allow-empty", "-m", "local only")
		}, wantObservation{DepResource("ariadne"), StatePresent, ""}, wantObservation{ResourceSetup, StatePresent, ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newObservedSlot(t)
			os.MkdirAll(s.layout.Store(), 0o700)
			dep := s.addDep(t, "ariadne")
			c.break_(t, s, dep)
			assertObserved(t, s.observe(t), c.dep, c.setup)
		})
	}
}

func TestObserveHalfClonedDependency(t *testing.T) {
	s := newObservedSlot(t)
	os.MkdirAll(s.layout.Store(), 0o700)
	write(t, filepath.Join(s.layout.Host(), "construct", "deps"), "substrate ../ariadne\n")
	if err := os.MkdirAll(filepath.Join(s.layout.Env(), "ariadne", "partial"), 0o755); err != nil {
		t.Fatal(err)
	}
	assertObserved(t, s.observe(t), wantObservation{DepResource("ariadne"), StateBroken, SubUnreadable}, wantObservation{ResourceSetup, StateAbsent, SubMarkerValid})
}

// failingGit fails a matching git command; everything else reaches the fixture.
type failingGit struct {
	inner ProvisionIO
	match func(ProvisionCommand) bool
}

func (f failingGit) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	if c.Program == "git" && f.match(c) {
		return []byte("fatal: unable to read index: Resource temporarily unavailable"), errors.New("exit status 128")
	}
	return f.inner.Run(ctx, c)
}

func TestObserveFailedProbesAreUnknownNeverAbsent(t *testing.T) {
	t.Run("worktree list", func(t *testing.T) {
		s := newObservedSlot(t)
		os.MkdirAll(s.layout.Store(), 0o700)
		s.io = failingGit{inner: s.f, match: func(c ProvisionCommand) bool { return len(c.Args) > 0 && c.Args[0] == "worktree" }}
		got := s.observe(t)
		for _, id := range []SlotResourceID{ResourceBranch, ResourceRegistration} {
			if r, _ := got.Get(id); r.State != StateUnknown {
				t.Errorf("%s = %s, want unknown after a failed git worktree list", id, r.State)
			}
		}
	})
	t.Run("dependency rev-parse", func(t *testing.T) {
		s := newObservedSlot(t)
		os.MkdirAll(s.layout.Store(), 0o700)
		dep := s.addDep(t, "ariadne")
		s.io = failingGit{inner: s.f, match: func(c ProvisionCommand) bool { return c.Dir == dep }}
		if r, _ := s.observe(t).Get(DepResource("ariadne")); r.State != StateUnknown {
			t.Fatalf("dep = %s/%q, want unknown: a failed probe is not evidence that the clone is broken", r.State, r.Sub)
		}
	})
}

func holdLock(t *testing.T, path string) {
	t.Helper()
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unix.Close(fd) })
}

func TestParseWorktreeList(t *testing.T) {
	raw := "worktree /a\x00HEAD 1111111111111111111111111111111111111111\x00branch refs/heads/main\x00\x00" +
		"worktree /b\x00HEAD 2222222222222222222222222222222222222222\x00detached\x00prunable gitdir file points to non-existent location\x00\x00" +
		"worktree /c\x00HEAD 3333333333333333333333333333333333333333\x00branch refs/heads/main-slot1\x00locked\x00\x00"
	got := parseWorktreeList(raw)
	want := []worktreeEntry{
		{Path: "/a", Head: "1111111111111111111111111111111111111111", Branch: "main"},
		{Path: "/b", Head: "2222222222222222222222222222222222222222", Detached: true, Prunable: true},
		{Path: "/c", Head: "3333333333333333333333333333333333333333", Branch: "main-slot1", Locked: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestObserveSlotFollowsTheResourceOrder: every table resource is observed,
// in SlotResourceOrder (instances under their template), so --show and the
// plan read one order with no second list.
func TestObserveSlotFollowsTheResourceOrder(t *testing.T) {
	s := newObservedSlot(t)
	os.MkdirAll(s.layout.Store(), 0o700)
	s.addDep(t, "ariadne")
	order, err := SlotResourceOrder()
	if err != nil {
		t.Fatal(err)
	}
	var got []SlotResourceID
	for _, r := range s.observe(t).Resources {
		if id := templateOf(r.ID); len(got) == 0 || got[len(got)-1] != id {
			got = append(got, id)
		}
	}
	if len(got) != len(order) {
		t.Fatalf("observed %v, want %v", got, order)
	}
	for i := range order {
		if got[i] != order[i] {
			t.Fatalf("observed %v, want %v", got, order)
		}
	}
}
