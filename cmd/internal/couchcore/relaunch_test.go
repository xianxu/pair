package couchcore

import (
	"context"
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
