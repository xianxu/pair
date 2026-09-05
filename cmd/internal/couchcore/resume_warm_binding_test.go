package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// warmPathResolverProbe counts resolutions and fails every one, the way a real
// IO error does -- a ZERO resolution alongside the error.
type warmPathResolverProbe struct {
	*FakeThreadArtifactCollisionChecker
	calls int
}

func (w *warmPathResolverProbe) ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error) {
	w.calls++
	return NativeBindingResolution{}, errors.New("resolve home directory: permission denied")
}

// A warm reattach's authority is its SURVIVING SESSION, not a native id, so it
// must neither pay for a binding resolution nor be failed by one.
//
// Consolidating resume's and relaunch's evidence-gathering into one helper made
// ResumeContext resolve a binding on the warm path too: a ListFiles, an up-to-8MB
// read, proof validation and a possible catalog write, all discarded -- plus a
// failure mode the path never had, since a resolver error refused the thread
// before DecideResume could decide anything. The path check is shared; the
// resolution is not, because only the cold path passes `--resume <native-id>`.
func TestWarmReattachNeitherConsultsNorIsFailedByTheBindingResolver(t *testing.T) {
	env := newTestEnv(t, "/repo")
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	// No verified park and no incarnation: the shape an alt+d detach leaves.
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	record.Reservation = false
	record.LatestLaunchProfile = &profile
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPairSession(created.Address, "pair-"+string(created.Address.Tag), true)
	env.Artifacts.SetDetachedSession(created.Address, "pair-"+string(created.Address.Tag))

	probe := &warmPathResolverProbe{FakeThreadArtifactCollisionChecker: env.Artifacts}
	env.Couch.Artifacts = probe

	_, _, err = env.Couch.ResumeContext(context.Background(), created.Address)

	if probe.calls != 0 {
		t.Errorf("warm reattach resolved the binding %d time(s); its authority is the surviving session", probe.calls)
	}
	// And the resolver's failure must not be the thing that refuses it.
	if got := ResumeDiagnosticOf(err); isBindingDiagnostic(got) {
		t.Errorf("a failing resolver refused a warm reattach with %q: %v", got, err)
	}
	if err != nil && strings.Contains(err.Error(), "permission denied") {
		t.Errorf("warm reattach failed on the resolver error it should never have made: %v", err)
	}
}
