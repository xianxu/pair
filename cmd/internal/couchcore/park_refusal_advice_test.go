package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

func TestOpenParkRefusalsNameRecovery(t *testing.T) {
	for _, operation := range []string{"resume", "resume-pending", "resume-running", "recover", "continue", "switch", "switch-pending", "switch-running", "detach"} {
		t.Run(operation, func(t *testing.T) {
			env, record := interruptedParkEnv(t, false)
			env.Couch.PairLifecycle.CompletionTimeout = 0
			if operation == "resume-pending" || operation == "resume-running" || operation == "continue" || operation == "switch-pending" || operation == "switch-running" {
				request := testContinuationRequest(t, record)
				var err error
				record, err = env.Couch.Threads.PublishContinuation(record.Address, record.Revision, request)
				if err != nil {
					t.Fatal(err)
				}
				if operation != "resume-pending" && operation != "switch-pending" {
					record, err = env.Couch.Threads.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: "start-0102030405060708"})
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return false, nil }
			env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return ContinuationSource{}, nil }
			before := cloneThreadRecord(record)
			var err error
			switch operation {
			case "resume", "resume-pending", "resume-running":
				_, err = env.Couch.ResumeTarget(context.Background(), ResumeTarget{Address: record.Address})
			case "recover":
				_, err = env.Couch.RecoverThread(context.Background(), record.Address, "")
			case "continue":
				_, err = env.Couch.Continue(context.Background(), record.Address, record.Continuation.ID)
			case "switch", "switch-pending", "switch-running":
				_, err = env.Couch.PrepareAgentSwitch(context.Background(), record.Address, "codex", nil)
			case "detach":
				_, err = env.Couch.Detach(context.Background(), record.Address)
			}
			want := env.Couch.ParkRecoveryAdvice(context.Background(), record.Address)
			if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "retry/recover/abandon") {
				t.Fatalf("%s refusal = %v; want exact recovery advice %q", operation, err, want)
			}
			after, getErr := env.Couch.Threads.GetThread(record.Address)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if after.Park == nil || before.Park.Identity != after.Park.Identity || !reflect.DeepEqual(before.Incarnations, after.Incarnations) || len(env.Runner.Ops) != 0 {
				t.Fatalf("refusal changed park or owner or launched a child: before=%+v after=%+v", before, after)
			}
		})
	}
}

func TestOpenParkSwitchFailureNamesRecovery(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	args := []string{}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	env.Couch.SwitchGuard = switchGuardFunc(func(context.Context, ThreadAddress, string) error {
		calls++
		if calls > 1 {
			env.Proc.Kill(42)
			env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
		}
		return nil
	})
	env.Couch.PairLifecycle.CompletionTimeout = 0
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: prepared.Fingerprint})
	if err == nil || result.Outcome != SwitchParkIncomplete || !strings.Contains(err.Error(), env.Couch.ParkRecoveryAdvice(context.Background(), source.Address)) || strings.Contains(err.Error(), "retry/recover/abandon") {
		t.Fatalf("incomplete switch park lacks recovery advice: %+v %v", result, err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("failed source park launched a target")
	}
}

func TestOpenParkRecoveryRefusalNamesVerifiedSlot(t *testing.T) {
	env, original := envWithLiveThread(t)
	local := testLocalThreadStore(t)
	record := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	record.LatestLaunchProfile, record.Incarnations = original.LatestLaunchProfile, original.Incarnations
	created, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "original-park", Address: created.Address, PID: 42, ProcessIdentity: "pair-helper"}
	if _, err := local.BeginPark(created.Address, created.Revision, identity); err != nil {
		t.Fatal(err)
	}
	env.Couch.Threads = local
	env.Artifacts.SetPairSession(created.Address, "pair-exact", false)
	_, err = env.Couch.RecoverThread(context.Background(), created.Address, "")
	const want = "couch --recover-park pair:1 --confirm"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("verified slot refusal = %v; want %s", err, want)
	}
}

func TestOpenParkContinuationPublicationNamesRecovery(t *testing.T) {
	env, record := interruptedParkEnv(t, false)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "continuation.md")
	if err := os.WriteFile(path, []byte(testContinuationRequest(t, record).Checkpoint.Body), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := env.Couch.RequestContinuation(context.Background(), record.Address, proof, path)
	if err == nil || !strings.Contains(err.Error(), env.Couch.ParkRecoveryAdvice(context.Background(), record.Address)) {
		t.Fatalf("publication refusal has no recovery advice: %v", err)
	}
}

func TestOpenParkDebrisRefusalsNameRecovery(t *testing.T) {
	for _, refusal := range []string{"live-park-owner", "unknown-park-owner", "abandon-conflict"} {
		t.Run(refusal, func(t *testing.T) {
			env, record := interruptedParkEnv(t, false)
			if refusal != "abandon-conflict" {
				var err error
				record, err = env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error {
					r.Park.Phase = ParkUnknown
					r.Park.Identity.PID, r.Park.Identity.ProcessIdentity = 43, "original-owner"
					r.Park.Attempts[0].Failure = &ParkFailure{Code: pairlifecycle.FailureReplacementIncarnation, Diagnostic: "replacement"}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if refusal == "live-park-owner" {
					env.Proc.Set(43, "original-owner")
				} else {
					env.Proc.SetUnknown(43)
				}
			} else {
				if _, err := env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error { r.Description = "concurrent edit"; return nil }); err != nil {
					t.Fatal(err)
				}
			}
			before, err := env.Couch.Threads.GetThread(record.Address)
			if err != nil {
				t.Fatal(err)
			}
			_, err = env.Couch.clearLifecycleDebris(record)
			if err == nil || !strings.Contains(err.Error(), env.Couch.ParkRecoveryAdvice(context.Background(), record.Address)) {
				t.Fatalf("%s refusal has no recovery advice: %v", refusal, err)
			}
			if refusal == "abandon-conflict" {
				var revisionErr *ThreadRevisionError
				if !errors.As(err, &revisionErr) {
					t.Fatalf("advice lost original revision error: %v", err)
				}
			}
			after, getErr := env.Couch.Threads.GetThread(record.Address)
			if getErr != nil || !reflect.DeepEqual(before, after) || len(env.Runner.Ops) != 0 {
				t.Fatalf("refusal changed retained state or launched a child: %+v %v", after, getErr)
			}
		})
	}
}

func TestOpenParkSourceStopFailuresNameRecovery(t *testing.T) {
	for _, operation := range []string{"park-new", "park-existing", "leave-new", "leave-existing", "prepared", "continuation-new"} {
		t.Run(operation, func(t *testing.T) {
			var env *relaunchEnv
			var source ThreadRecord
			var continuationID string
			if strings.HasSuffix(operation, "existing") {
				env, source = interruptedParkEnv(t, false)
			} else if operation == "continuation-new" {
				f := newContinuationFixture(t)
				env, source, continuationID = f.env, f.source, f.status.RequestID
			} else {
				env, source = envWithLiveThread(t)
			}
			env.Couch.PairLifecycle.CompletionTimeout = 0
			if !strings.HasSuffix(operation, "existing") {
				env.Artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error { return errors.New("submit interrupted") }
			}
			var err error
			switch operation {
			case "park-new", "park-existing":
				_, err = env.Couch.Park(context.Background(), source.Address, "normal")
			case "leave-new", "leave-existing":
				_, err = env.Couch.Leave(context.Background(), LeavePark)
			case "prepared":
				_, err = env.Couch.parkPreparedRecovery(context.Background(), source, &source.Incarnations[0])
			case "continuation-new":
				_, err = env.Couch.Continue(context.Background(), source.Address, continuationID)
			}
			if err == nil || !strings.Contains(err.Error(), env.Couch.ParkRecoveryAdvice(context.Background(), source.Address)) {
				t.Fatalf("%s missing recovery action: %v", operation, err)
			}
			after, getErr := env.Couch.Threads.GetThread(source.Address)
			if getErr != nil || after.Park == nil || after.VerifiedPark != nil || len(env.Runner.Ops) != 0 {
				t.Fatalf("failed source stop lost its transaction or launched a target: %+v %v", after, getErr)
			}
		})
	}
}
