package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

// neverTurnedThread is a parked thread whose agent never took a turn: Pair
// chose its conversation id at launch and the agent never wrote it (the
// pair#367 smoke test's 1-tools-15, created by a reboot and resumed before
// its first turn). Registration is modeled as production checks it: Pair
// mints its own launch nonce unless it is handed one, and the ready file must
// carry the nonce Couch awaits.
func neverTurnedThread(t *testing.T) (*testEnv, ThreadRecord) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	native := sessioninventorytest.NewFakeRuntime()
	pair := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
	native.SetPairDataRoot(pair)
	row := `{"v":3,"kind":"launch","scope_key":"` + parked.Address.RepoScope + `","tag":"` + string(parked.Address.Tag) +
		`","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"11111111-1111-4111-8111-111111111111","request_origin":"chosen-id","baseline_complete":true}` + "\n"
	native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pair.Name, RelativePath: "ledger-" + string(parked.Address.Tag) + ".jsonl"}}, []byte(row))
	env.Couch.Artifacts = chosenRestartArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, resolver: SessionInventoryNativeBindingResolver{Runtime: native}}
	const pairMinted = "pair-minted-nonce"
	env.Couch.FreshRegistration = func(_ context.Context, _ ThreadAddress, _, attempt string) (bool, error) {
		return attempt == pairMinted, nil
	}
	return env, parked
}

// A thread whose agent never took a turn has no conversation to resume: resume
// refuses at once, naming reboot, and launches nothing, instead of restarting
// fresh under a nonce Pair never learns and waiting out the registration
// budget (pair#367 smoke test, operator decision).
func TestResumeRefusesAThreadThatNeverTookATurn(t *testing.T) {
	t.Parallel()

	env, parked := neverTurnedThread(t)
	// "At once" is "before the registration wait", observed where the wait
	// takes its budget rather than timed against it (#480).
	waited := false
	env.Couch.registrationDeadline = func(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		waited = true
		return context.WithTimeout(ctx, d)
	}
	_, _, err := env.Couch.ResumeContext(context.Background(), parked.Address)
	if waited {
		t.Fatalf("the refusal waited on the registration budget: %v", err)
	}
	if ResumeDiagnosticOf(err) != ResumeBindingUnbound {
		t.Fatalf("resume = %v, want the unbound refusal", err)
	}
	if n := len(env.Runner.Ops); n != 0 {
		t.Fatalf("a refused resume launched %d children: %v", n, env.Runner.Ops)
	}
	// Through the operation table the same: the route's refusal carries the
	// reboot advice.
	_, err = dispatchResume(env, map[string]string{"repo-scope": parked.Address.RepoScope, "tag": string(parked.Address.Tag)})
	if ResumeDiagnosticOf(err) != ResumeBindingUnbound || !strings.Contains(err.Error(), "reboot") {
		t.Fatalf("dispatched resume = %v", err)
	}
}

// Relaunch is park then resume, so it is refused before its park for the same
// thread: there is nothing to come back to.
func TestRelaunchRefusesAThreadThatNeverTookATurn(t *testing.T) {
	t.Parallel()

	env, live := envWithLiveThread(t)
	native := sessioninventorytest.NewFakeRuntime()
	pair := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
	native.SetPairDataRoot(pair)
	row := `{"v":3,"kind":"launch","scope_key":"` + live.Address.RepoScope + `","tag":"` + string(live.Address.Tag) +
		`","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"11111111-1111-4111-8111-111111111111","request_origin":"chosen-id","baseline_complete":true}` + "\n"
	native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pair.Name, RelativePath: "ledger-" + string(live.Address.Tag) + ".jsonl"}}, []byte(row))
	env.Couch.Artifacts = chosenRestartArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, resolver: SessionInventoryNativeBindingResolver{Runtime: native}}
	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if ResumeDiagnosticOf(err) != ResumeBindingUnbound || result.Outcome == Relaunched {
		t.Fatalf("relaunch = %+v, %v; want the unbound refusal", result, err)
	}
	if current, readErr := env.Couch.Threads.GetThread(live.Address); readErr != nil || current.VerifiedPark != nil {
		t.Fatalf("a refused relaunch parked the thread: %+v %v", current.VerifiedPark, readErr)
	}
}

// Every fresh launch waits for a ready file carrying the nonce Couch awaits;
// Pair learns that nonce through its trusted launch profile, and an empty
// nonce is refused before any child starts.
func TestFreshLaunchMustHandPairItsNonce(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name string
		in   trackedThreadLaunch
		ok   bool
	}{
		{"not fresh", trackedThreadLaunch{Nonce: "n"}, true},
		{"fresh with its nonce and a prompt", trackedThreadLaunch{Nonce: "n", Fresh: true, Prompt: "read it"}, true},
		{"fresh without a prompt", trackedThreadLaunch{Nonce: "n", Fresh: true}, true},
		{"fresh without a nonce", trackedThreadLaunch{Fresh: true}, false},
	} {
		err := freshNonceReachesPair(c.in)
		if (err == nil) != c.ok {
			t.Errorf("%s: %v", c.name, err)
		}
		if err != nil && !errors.Is(err, errFreshNonceUnreachable) {
			t.Errorf("%s: untyped %v", c.name, err)
		}
	}
}
