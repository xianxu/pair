package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// relaunchEnv is a couch that can both PARK and RESUME, which no existing
// fixture provides: newTestEnv wires a Runner and Artifacts but no
// PairLifecycle, and the park fixtures wire a controller but no Runner to spawn
// the resumed child.
type relaunchEnv struct {
	*testEnv
	Lifecycle *fakeControllerLifecycle
	model     *pairlifecycletest.Fake
}

func envWithLiveThread(t *testing.T) (*relaunchEnv, ThreadRecord) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	now := env.Now

	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.Reservation = false
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "pair-helper", State: IncarnationLive, LaunchProfile: &profile,
	}}
	record.LatestLaunchProfile = &profile
	live, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	model := pairlifecycletest.New(now)
	model.SetSession("pair-exact", true)
	lifecycle := &fakeControllerLifecycle{model: model, store: env.Couch.Threads}
	env.Artifacts.SetPairSession(live.Address, "pair-exact", true)
	env.Artifacts.SetNativeBinding(live.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Proc.Set(42, "pair-helper")

	// A park that completes: the trigger delivers the completion and the exact
	// child dies, which is what the real Pair does.
	env.Artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
		if err := model.DeliverTrigger(lifecycle.lastRequest); err != nil {
			return err
		}
		completion := successCompletion(lifecycle.lastRequest, now)
		lifecycle.completion = &completion
		env.Proc.Kill(42)
		return model.CommitCompletion(lifecycle.lastRequest, pairlifecycle.CleanupResult{
			Outcome: pairlifecycle.CompletionSuccess, CompletedAt: now,
		})
	}
	env.Couch.PairLifecycle = &PairLifecycleController{
		Threads: env.Couch.Threads, DataDir: t.TempDir(), Lifecycle: lifecycle,
		Sessions: env.Artifacts, Proc: env.Proc, Clock: FixedClock{T: now},
		Nonce:             func() (string, error) { return "park-relaunch-0123", nil },
		CompletionTimeout: time.Second, PollInterval: time.Millisecond,
	}
	return &relaunchEnv{testEnv: env, Lifecycle: lifecycle, model: model}, live
}

// The property that makes relaunch safe to offer at all. Park is destructive
// and resume can refuse, so a relaunch whose resume could not succeed must not
// park -- otherwise the operator trades a working session for a cold one.
func TestRelaunchRefusesBeforeParkingWhenTheResumeCouldNotSucceed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		breaks func(*relaunchEnv, ThreadRecord)
		want   ResumeDiagnosticCode
	}{
		{
			name: "no established binding",
			breaks: func(env *relaunchEnv, live ThreadRecord) {
				env.Artifacts.SetNativeBinding(live.Address, "claude", sessioninventory.BindingProvisional, "")
			},
			want: ResumeBindingProvisional,
		},
		{
			name: "working path is gone",
			breaks: func(env *relaunchEnv, live ThreadRecord) {
				env.Couch.Path.(*FakePathOps).Fail("/repo")
			},
			want: ResumePathMissing,
		},
		{
			name: "two incarnations -- park's own precondition",
			breaks: func(env *relaunchEnv, live ThreadRecord) {
				_, err := env.Couch.Threads.UpdateExistingThread(live.Address, live.Revision, func(r *ThreadRecord) error {
					r.Incarnations = append(r.Incarnations, ThreadIncarnation{
						PID: 43, Identity: "pair-second", State: IncarnationLive,
					})
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			},
			want: ResumeLive,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, live := envWithLiveThread(t)
			tc.breaks(env, live)

			result, err := env.Couch.Relaunch(context.Background(), live.Address)

			if result.Outcome != RefusedBeforePark {
				t.Fatalf("outcome = %q, want RefusedBeforePark (err %v)", result.Outcome, err)
			}
			if got := ResumeDiagnosticOf(err); got != tc.want {
				t.Fatalf("diagnostic = %q, want %q (err %v)", got, tc.want, err)
			}
			// NOTHING happened. This is the assertion that matters: "it refuses"
			// is cheap, "it refuses and nothing was destroyed" is the property.
			if got := env.Lifecycle.trace; len(got) != 0 {
				t.Fatalf("a refused relaunch performed lifecycle work: %v", got)
			}
			if got := env.Artifacts.TriggeredQuits(); len(got) != 0 {
				t.Fatalf("a refused relaunch triggered Pair quits: %v", got)
			}
			after, err := env.Couch.Threads.GetThread(live.Address)
			if err != nil {
				t.Fatal(err)
			}
			if len(after.Incarnations) == 0 || after.VerifiedPark != nil || after.Park != nil {
				t.Fatalf("a refused relaunch changed the thread: %+v", after)
			}
		})
	}
}

// The likeliest failure, and the one whose leftover state is worst: Pair has
// already been sent its quit intent, the transaction is open, and `Enter` will
// NOT recover the row because DecideResume refuses ResumeParking.
func TestRelaunchStopsAtAFailedParkAndNamesTheRecovery(t *testing.T) {
	for _, tc := range []struct {
		name   string
		breaks func(*relaunchEnv)
	}{
		{
			name: "the completion never arrives",
			breaks: func(env *relaunchEnv) {
				// The trigger fires but no completion is committed, so the park
				// times out waiting for proof it worked.
				env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error { return nil }
				env.Couch.PairLifecycle.CompletionTimeout = 5 * time.Millisecond
			},
		},
		{
			name: "the exact child never dies",
			breaks: func(env *relaunchEnv) {
				// The completion commits but the process stays alive, so the
				// park cannot prove the child is gone.
				env.Artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
					if err := env.model.DeliverTrigger(env.Lifecycle.lastRequest); err != nil {
						return err
					}
					completion := successCompletion(env.Lifecycle.lastRequest, env.Now)
					env.Lifecycle.completion = &completion
					// Deliberately NOT killing pid 42.
					return env.model.CommitCompletion(env.Lifecycle.lastRequest, pairlifecycle.CleanupResult{
						Outcome: pairlifecycle.CompletionSuccess, CompletedAt: env.Now,
					})
				}
				env.Couch.PairLifecycle.CompletionTimeout = 5 * time.Millisecond
			},
		},
		{
			name: "the request cannot be published",
			breaks: func(env *relaunchEnv) {
				env.Lifecycle.publishErr = errors.New("publish refused")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, live := envWithLiveThread(t)
			tc.breaks(env)

			result, err := env.Couch.Relaunch(context.Background(), live.Address)

			if result.Outcome != ParkIncomplete || err == nil {
				t.Fatalf("outcome = %q, err = %v; want ParkIncomplete and a refusal", result.Outcome, err)
			}
			// The resume must NOT be attempted. A thread with an open park
			// transaction is not resumable, so trying reports a second,
			// confusing failure stacked on the first.
			// Ops records every spawn attempt, so an empty log is the proof
			// that no child was started -- the resume never ran.
			if got := env.Runner.Ops; len(got) != 0 {
				t.Fatalf("a failed park still tried to start a child: %v", got)
			}
			// And the message names PARK's recovery, not Enter -- Enter is
			// refused with ResumeParking, so naming it would be the
			// unnavigable-refusal class (pair#181 M3).
			for _, want := range []string{"retry", "recover", "abandon"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q does not name park's recovery: missing %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "Enter") {
				t.Fatalf("error %q tells the operator to press Enter, which refuses here", err)
			}
		})
	}
}

// A Pair cleanup failure is NOT a park failure, and relaunch must not treat it
// as one. The park is proven by the durable completion and the final CAS
// (park.go:642-643); CleanupAttempt runs afterwards and its error lands in
// ParkResult.CleanupError with the park returning nil. Discovered by asserting
// the opposite and watching the relaunch succeed.
func TestRelaunchProceedsWhenOnlyPairsCleanupFailed(t *testing.T) {
	env, live := envWithLiveThread(t)
	env.Lifecycle.cleanupErr = errors.New("cleanup refused")

	result, err := env.Couch.Relaunch(context.Background(), live.Address)

	if err != nil || result.Outcome != Relaunched {
		t.Fatalf("Relaunch = %+v, %v; a cleanup failure must not stop the restart", result, err)
	}
}

// The window that cannot be closed: the park succeeded and the resume failed.
// The thread is PARKED -- listed, and Enter resumes it -- so the message must
// say that. "relaunch failed" reads as data loss when the work is one keystroke
// away.
func TestRelaunchThatParksThenFailsToResumeLeavesARecoverableThread(t *testing.T) {
	env, live := envWithLiveThread(t)
	// The park runs to completion; the spawn that follows it does not.
	env.Runner.FailNextStart(errors.New("exec: no such file"))

	result, err := env.Couch.Relaunch(context.Background(), live.Address)

	if result.Outcome != ParkedNotResumed || err == nil {
		t.Fatalf("outcome = %q, err = %v; want ParkedNotResumed and a refusal", result.Outcome, err)
	}
	if !strings.Contains(err.Error(), "parked") || !strings.Contains(err.Error(), "Enter") {
		t.Fatalf("error %q does not tell the operator the work is recoverable", err)
	}
	// The store agrees, which is what makes the message true rather than
	// merely reassuring.
	after, err := env.Couch.Threads.GetThread(live.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after.VerifiedPark == nil || after.Park != nil {
		t.Fatalf("thread = %+v, want a completed verified park", after)
	}
	rows, err := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := findInventoryRow(rows, live.Address)
	if !ok || row.State != ThreadParked {
		t.Fatalf("row = %+v (found %v), want a parked row the operator can resume", row, ok)
	}
}

// The thing being built. Every failure above is scaffolding for this: one live
// incarnation at the SAME address afterwards, a new process, the row still in
// the switcher, and the conversation continued rather than restarted.
func TestASuccessfulRelaunchKeepsTheAddressTheRowAndTheConversation(t *testing.T) {
	env, live := envWithLiveThread(t)
	before, err := env.Couch.Threads.GetThread(live.Address)
	if err != nil {
		t.Fatal(err)
	}

	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if err != nil || result.Outcome != Relaunched {
		t.Fatalf("Relaunch = %+v, %v", result, err)
	}

	after, err := env.Couch.Threads.GetThread(live.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after.Address != before.Address {
		t.Fatalf("relaunch changed the address: %+v -> %+v", before.Address, after.Address)
	}
	if len(after.Incarnations) != 1 || after.Incarnations[0].State != IncarnationLive {
		t.Fatalf("after = %+v, want exactly one live incarnation", after)
	}
	if after.Incarnations[0].PID == before.Incarnations[0].PID {
		t.Fatalf("the Pair process was not replaced (still pid %d)", after.Incarnations[0].PID)
	}
	// The conversation continued: the resume carried the thread's OWN native
	// session id, which is the evidence pair#181 M2 used for reattach. A fresh
	// conversation would carry none.
	child := env.Runner.Child(result.Handle.ID())
	if !strings.Contains(strings.Join(child.Env, "\n"), "native-root-1") {
		t.Fatalf("resume env = %v, want the conversation's own native session id", child.Env)
	}
	// And the row survives, which is what the operator sees.
	rows, err := env.Couch.ActionableThreadInventoryContext(context.Background(),
		[]LiveTTYObservation{{Address: live.Address, Process: ProcessIdentity{
			PID: after.Incarnations[0].PID, Identity: after.Incarnations[0].Identity,
		}}})
	if err != nil {
		t.Fatal(err)
	}
	row, ok := findInventoryRow(rows, live.Address)
	if !ok || row.State != ThreadLive {
		t.Fatalf("row = %+v (found %v), want a live row at the same address", row, ok)
	}
}

// The seam neither a store test nor a menu test crosses: the switcher
// dispatches {repo-scope, tag} through threadEffect, and an executor reading
// only `ref` is how Tab → archive shipped broken (pair#181 M3, C-1).
//
// It runs through DispatchOperation with the production executors, because that
// is where the dialect is actually resolved. A couchcmd test cannot reach it:
// relaunch is ExecuteLiveOwner, so the CLI runtime returns its routing refusal
// before any of this code runs — which is exactly what a first attempt at this
// test did, passing for a reason unrelated to what it claimed.
func TestRelaunchResolvesTheSwitchersDialectThroughTheRealDispatcher(t *testing.T) {
	env, live := envWithLiveThread(t)

	result, err := DispatchOperation(OperationExecutors{
		DirectStore: DirectStoreExecutor(env.Couch),
		LiveOwner:   CouchLiveOwnerExecutor(env.Couch),
	}, OperationCall{
		Name: "relaunch", Implicit: true,
		Args: map[string]string{
			"repo-scope": live.Address.RepoScope,
			// tag, not ref: exactly what threadEffect sends.
			"tag": string(live.Address.Tag),
		},
	})
	if err != nil {
		t.Fatalf("relaunch through the switcher's dialect: %v", err)
	}
	relaunched, ok := result.(RelaunchResult)
	if !ok || relaunched.Outcome != Relaunched {
		t.Fatalf("result = %#v, want a completed RelaunchResult", result)
	}
}
