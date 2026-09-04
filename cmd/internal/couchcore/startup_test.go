package couchcore

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The selector admits BOTH resumable states. Parked is cold (the session was
// torn down) and detached is warm (it survived its client), but `couch` in a
// directory means the same thing either way: reattach what is already there.
//
// A live row is never selected: couch is a singleton holding its supervisor
// lease for the whole run, so a live row at startup means THIS couch already
// hosts it.
func TestSelectResumableRoot(t *testing.T) {
	want := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	other := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000002"}
	row := func(address ThreadAddress, path string, state ActionableThreadState) ActionableThreadSummary {
		return ActionableThreadSummary{Address: address, WorkingPath: path, State: state}
	}
	parked := row(want, "/real/repo", ThreadParked)
	detached := row(want, "/real/repo", ThreadDetached)

	tests := []struct {
		name string
		rows []ActionableThreadSummary
		want ThreadAddress
		ok   bool
	}{
		{name: "one exact parked row", rows: []ActionableThreadSummary{parked}, want: want, ok: true},
		{name: "one exact detached row", rows: []ActionableThreadSummary{detached}, want: want, ok: true},
		{name: "empty", rows: nil},
		// These three cases USED to return nothing: two resumable rows at one
		// path were TWO matches, and a new thread was created. That exactness was
		// a ratchet -- the new thread made a third row, guaranteeing the next
		// startup created a fourth, which is how one repo reached six threads.
		// The selector now ranks: detached before parked, recent before old.
		{name: "repeated parked rows select one", rows: []ActionableThreadSummary{parked, parked}, want: want, ok: true},
		{name: "repeated detached rows select one", rows: []ActionableThreadSummary{detached, detached}, want: want, ok: true},
		{
			// Warm beats cold: the detached agent is already running, so
			// reattaching preserves what it was doing.
			name: "one parked and one detached at the same path prefers detached",
			rows: []ActionableThreadSummary{parked, row(other, "/real/repo", ThreadDetached)},
			want: other, ok: true,
		},
		{name: "live is never selected", rows: []ActionableThreadSummary{row(want, "/real/repo", ThreadLive)}},
		{name: "wrong scope", rows: []ActionableThreadSummary{row(ThreadAddress{RepoScope: "scope-b", Tag: want.Tag}, "/real/repo", ThreadParked)}},
		{name: "wrong path", rows: []ActionableThreadSummary{row(want, "/real/other", ThreadParked)}},
		{
			// Paths are compared by exact string, so a row still carrying an
			// unresolved alias does not match the physical target. This is what
			// makes physicalizing detached rows load-bearing rather than tidy.
			name: "an unresolved alias path does not match",
			rows: []ActionableThreadSummary{row(want, "/link/repo", ThreadDetached)},
		},
		{name: "one resumable among nonmatches", rows: []ActionableThreadSummary{
			row(want, "/real/repo", ThreadLive),
			detached,
			row(ThreadAddress{RepoScope: "scope-b", Tag: other.Tag}, "/real/repo", ThreadParked),
		}, want: want, ok: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := SelectResumableRoot(test.rows, "scope-a", "/real/repo")
			if ok != test.ok || got != test.want {
				t.Fatalf("SelectResumableRoot() = (%+v, %v), want (%+v, %v)", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestStartInteractiveResumesUniqueExactParkedRoot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --show-toplevel"}] = "/repo"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	parked := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	parked.StartingPath, parked.WorkingPath = "/repo", "/repo/sub"
	parked.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{"--verbose"}}
	markActionableParked(&parked, parked.LastActiveAt)
	var err error
	parked, err = env.Couch.Threads.CreateThread(parked)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo/sub"})
	if err != nil {
		t.Fatalf("StartInteractive: %v", err)
	}
	if start.Record.Thread != parked.Address {
		t.Fatalf("resumed address = %+v, want %+v", start.Record.Thread, parked.Address)
	}
	child := env.Runner.Child(start.Handle.ID())
	if !strings.Contains(strings.Join(child.Env, "\n"), "native-root-1") {
		t.Fatalf("resume env = %v, want saved native root", child.Env)
	}
}

func TestStartInteractiveCreatesNewRootWithoutExactCandidate(t *testing.T) {
	env := newTestEnv(t, "/repo")

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("StartInteractive: %v", err)
	}
	if start.Record.Thread == (ThreadAddress{}) || start.Handle == nil {
		t.Fatalf("new root = %+v, %v", start.Record, start.Handle)
	}
}

// This test used to assert that TWO parked candidates at one path made startup
// create a NEW thread. #181 reverses it on the operator's instruction: that
// exactness was a ratchet -- the new thread became a third candidate, so the
// next startup created a fourth, which is how one repo reached six threads.
// Startup now returns to the most recently active of them.
//
// Inverted rather than deleted, because the reversal is worth recording where
// the superseded claim lived.
func TestStartInteractiveResumesTheNewestOfSeveralParkedCandidates(t *testing.T) {
	env := newTestEnv(t, "/repo")
	// Distinct activity times, or "the newest" is decided by iteration order and
	// the ranking this test names is never exercised.
	first := seedStartupParked(t, env, "couch-0000000000000001", "/repo", time.Unix(100, 0).UTC())
	second := seedStartupParked(t, env, "couch-0000000000000002", "/repo", time.Unix(9000, 0).UTC())
	// Both must be genuinely resumable, or "it picked one" would be indistinguishable
	// from "the one it picked happened to work".
	env.Artifacts.SetPairSession(first.Address, "pair-"+string(first.Address.Tag), true)
	env.Artifacts.SetPairSession(second.Address, "pair-"+string(second.Address.Tag), true)

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("StartInteractive: %v", err)
	}
	// The NEWEST of them, which is what the name promises: asserting only "one
	// of the two" would pass with the ranking removed.
	if start.Record.Thread != second.Address {
		t.Fatalf("startup resumed %+v, want the more recently active %+v",
			start.Record.Thread, second.Address)
	}
}

func TestStartInteractiveInventoryFailureCreatesNoRoot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	if err := os.WriteFile(env.Couch.Threads.manifestPath(), []byte("{corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"}); err == nil {
		t.Fatal("StartInteractive accepted corrupt authoritative inventory")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("inventory failure started child: %v", env.Runner.Ops)
	}
}

type refusingStartupArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	resolutions int
}

func (a *refusingStartupArtifacts) ResolveEstablished(ctx context.Context, scope, tag, agent string) (NativeBindingResolution, error) {
	a.resolutions++
	if a.resolutions > 1 {
		return NativeBindingResolution{Status: sessioninventory.BindingUnbound}, refuseResume(ResumeBindingUnbound, "binding changed")
	}
	return a.FakeThreadArtifactCollisionChecker.ResolveEstablished(ctx, scope, tag, agent)
}

func TestStartInteractiveResumeRefusalDoesNotCreateFallbackRoot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := seedStartupParked(t, env, "couch-0000000000000001", "/repo")
	env.Couch.Artifacts = &refusingStartupArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts}

	if _, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"}); ResumeDiagnosticOf(err) != ResumeBindingUnbound {
		t.Fatalf("StartInteractive refusal = %v, want %s", err, ResumeBindingUnbound)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("resume refusal started fallback child: %v", env.Runner.Ops)
	}
	got, err := env.Couch.Threads.GetThread(parked.Address)
	if err != nil || got.VerifiedPark == nil {
		t.Fatalf("refused parked thread = %+v, %v", got, err)
	}
}

func seedStartupParked(t *testing.T, env *testEnv, tag ThreadTag, path string, active ...time.Time) ThreadRecord {
	t.Helper()
	at := time.Unix(100, 0).UTC()
	if len(active) > 0 {
		at = active[0]
	}
	record := actionableTestThread(tag, at)
	record.StartingPath, record.WorkingPath = "/repo", path
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-"+string(tag))
	return created
}

// The invariant #167's no-fallback startup rule rests on: a row the inventory
// OFFERS must be one resume can actually take. Startup has no fallback by
// design -- a Resume refusal stops it rather than creating a thread -- so
// offering a row that cannot resume does not degrade to "start something new",
// it kills `couch` in that tree.
//
// Detached rows must therefore clear the same native-binding gate parked rows
// already do. Without it a thread whose agent session data was pruned, rotated
// or raced is auto-selected and startup exits 1 with no way through -- and M2
// made detached the NORMAL resting state, so that is the ordinary row at the
// operator's own path.
func TestStartInteractiveSkipsDetachedRowsWithoutAResumableBinding(t *testing.T) {
	for _, test := range []struct {
		name    string
		binding func(*testEnv, ThreadAddress)
		want    bool
	}{
		{
			name: "established binding is resumable",
			binding: func(env *testEnv, a ThreadAddress) {
				env.Artifacts.SetNativeBinding(a, "claude", sessioninventory.BindingEstablished, "native-root-1")
			},
			want: true,
		},
		{
			name:    "no binding at all",
			binding: func(*testEnv, ThreadAddress) {},
		},
		{
			name: "ambiguous binding",
			binding: func(env *testEnv, a ThreadAddress) {
				env.Artifacts.SetNativeBinding(a, "claude", sessioninventory.BindingAmbiguous, "native-root-1")
			},
		},
		{
			name: "provisional binding",
			binding: func(env *testEnv, a ThreadAddress) {
				env.Artifacts.SetNativeBinding(a, "claude", sessioninventory.BindingProvisional, "native-root-1")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			test.binding(env, created.Address)

			rows, err := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			_, selected := SelectResumableRoot(rows, created.Address.RepoScope, "/repo")
			if selected != test.want {
				t.Fatalf("row offered for selection = %v, want %v (rows = %+v) -- an offered row must be resumable",
					selected, test.want, rows)
			}
		})
	}
}

// The twin of TestStartInteractiveResumesUniqueExactParkedRoot, driving
// StartInteractive ITSELF -- which is the wiring the M3 review found unpinned.
//
// The earlier test in this file exercises the inventory and the selector
// directly, so mutating StartInteractive's own call (filtering its rows to
// parked) left the suite green. Only the couchcmd acceptance test covered it,
// and that one hard-fails wherever a pty is unavailable -- which is every CI and
// agent context, so in practice nothing covered it at all.
func TestStartInteractiveResumesUniqueDetachedRoot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --show-toplevel"}] = "/repo"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	detached := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	detached.StartingPath, detached.WorkingPath = "/repo", "/repo/sub"
	detached.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{"--verbose"}}
	// Deliberately NOT parked: no verified park, no incarnation -- the shape an
	// alt+d detach leaves behind.
	var err error
	detached, err = env.Couch.Threads.CreateThread(detached)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(detached.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPairSession(detached.Address, "pair-"+string(detached.Address.Tag), true)
	// The surviving session is the resume authority.
	env.Artifacts.SetDetachedSession(detached.Address, "pair-"+string(detached.Address.Tag))

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo/sub"})
	if err != nil {
		t.Fatalf("StartInteractive: %v", err)
	}
	if start.Record.Thread != detached.Address {
		t.Fatalf("resumed address = %+v, want the detached thread %+v", start.Record.Thread, detached.Address)
	}
}

// Its negative: with no surviving session there is no resume authority, so
// startup must create a NEW thread rather than reattach one it cannot prove.
func TestStartInteractiveStartsNewWhenNoSessionSurvives(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --show-toplevel"}] = "/repo"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	stale := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	stale.StartingPath, stale.WorkingPath = "/repo", "/repo/sub"
	stale.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{"--verbose"}}
	var err error
	stale, err = env.Couch.Threads.CreateThread(stale)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(stale.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	// No SetDetachedSession: the session did not survive.

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo/sub"})
	if err != nil {
		t.Fatalf("StartInteractive: %v", err)
	}
	if start.Record.Thread == stale.Address {
		t.Fatalf("startup reattached %+v with no surviving session", stale.Address)
	}
}
