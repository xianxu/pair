package couchcore

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestSelectUniqueParkedRoot(t *testing.T) {
	want := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	matching := ActionableThreadSummary{
		Address:     want,
		WorkingPath: "/real/repo",
		State:       ThreadParked,
	}

	tests := []struct {
		name string
		rows []ActionableThreadSummary
		want ThreadAddress
		ok   bool
	}{
		{name: "one exact parked row", rows: []ActionableThreadSummary{matching}, want: want, ok: true},
		{name: "empty", rows: nil},
		{name: "ambiguous exact parked rows", rows: []ActionableThreadSummary{matching, matching}},
		{name: "live", rows: []ActionableThreadSummary{{Address: want, WorkingPath: "/real/repo", State: ThreadLive}}},
		{name: "wrong scope", rows: []ActionableThreadSummary{{Address: ThreadAddress{RepoScope: "scope-b", Tag: want.Tag}, WorkingPath: "/real/repo", State: ThreadParked}}},
		{name: "wrong path", rows: []ActionableThreadSummary{{Address: want, WorkingPath: "/real/other", State: ThreadParked}}},
		{name: "one among nonmatches", rows: []ActionableThreadSummary{
			{Address: want, WorkingPath: "/real/repo", State: ThreadLive},
			matching,
			{Address: ThreadAddress{RepoScope: "scope-b", Tag: "couch-0000000000000002"}, WorkingPath: "/real/repo", State: ThreadParked},
		}, want: want, ok: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := SelectUniqueParkedRoot(test.rows, "scope-a", "/real/repo")
			if ok != test.ok || got != test.want {
				t.Fatalf("SelectUniqueParkedRoot() = (%+v, %v), want (%+v, %v)", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestStartInteractiveResumesUniqueExactParkedRoot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --show-toplevel"}] = "/repo"
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

func TestStartInteractiveCreatesNewRootForAmbiguousParkedCandidates(t *testing.T) {
	env := newTestEnv(t, "/repo")
	first := seedStartupParked(t, env, "couch-0000000000000001", "/repo")
	second := seedStartupParked(t, env, "couch-0000000000000002", "/repo")

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("StartInteractive: %v", err)
	}
	if start.Record.Thread == first.Address || start.Record.Thread == second.Address {
		t.Fatalf("ambiguous startup selected parked candidate %+v", start.Record.Thread)
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

func seedStartupParked(t *testing.T, env *testEnv, tag ThreadTag, path string) ThreadRecord {
	t.Helper()
	record := actionableTestThread(tag, time.Unix(100, 0).UTC())
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
