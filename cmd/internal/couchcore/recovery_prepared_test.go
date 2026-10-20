package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProvenRecoveryWithoutScreenUsesNativeConversationOnly(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = nil
	f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
	p, err := f.Couch.ResumeCandidates(context.Background(), ResumeTarget{Address: f.record.Address})
	if err != nil || p.Proven == nil || p.Proven.NativeID != "proven-native" || p.Fresh == nil {
		t.Fatalf("preview=%+v err=%v", p, err)
	}
	result, err := f.Couch.RecoverSelection(context.Background(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryProven})
	if err != nil {
		t.Fatal(err)
	}
	child := f.Runner.Child(result.Start.Handle.ID())
	var profile launcher.TrustedLaunchProfile
	if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
		t.Fatal(err)
	}
	if !profile.ResumeRequired || profile.RequiredSessionID != "proven-native" || profile.Prompt != "" {
		t.Fatalf("profile=%+v", profile)
	}
}

func TestPreparedFreshDoesNotRequireTranscriptScanOrScreen(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = func(context.Context, ThreadRecord, RecoveryObservationRequest) (RecoveryObservation, error) {
		return RecoveryObservation{}, errors.New("previous screen missing")
	}
	result, err := f.Couch.RecoverSelection(context.Background(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh, Reference: "pair:99"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Start.Handle == nil || !strings.Contains(result.Notice, "fresh") {
		t.Fatalf("result=%+v", result)
	}
	if f.preserved != 0 {
		t.Fatal("manufactured capture preservation")
	}
}

func TestPreparedRecoveryRequiresExactStopConfirmationBeforeEffects(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = nil
	f.Proc.Set(42, "old-helper")
	retired := 0
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 70, Identity: "owned-agent", Role: "native-agent"}}, Digest: "same-owned-generation", Check: func(context.Context) error { return nil }, Retire: func(context.Context) error { retired++; f.Proc.Kill(42); return nil }}, nil
	}
	ctx := WithRecoveryConfirmer(context.Background(), "hub")
	_, err := f.Couch.RecoverSelection(ctx, RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh, Reference: "pair:99"})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) || retired != 0 || len(f.Runner.Ops) != 0 {
		t.Fatalf("error=%v retired=%d operations=%v", err, retired, f.Runner.Ops)
	}
	result, err := f.Couch.ConfirmRecovery(ctx, required.Offer.Token)
	if err != nil || retired != 1 || result.Start.Handle == nil || result.Mode != Fresh {
		t.Fatalf("result=%+v retired=%d err=%v", result, retired, err)
	}
	if _, err := f.Couch.ConfirmRecovery(ctx, required.Offer.Token); err == nil {
		t.Fatal("confirmation launched again")
	}
}

func TestPreparedRecoveryChangedStopSetNeverSignals(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = nil
	digest := "original"
	retired := false
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 70, Identity: digest}}, Digest: digest, Check: func(context.Context) error { return nil }, Retire: func(context.Context) error { retired = true; return nil }}, nil
	}
	_, err := f.Couch.RecoverSelection(context.Background(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh, Reference: "pair:99"})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) {
		t.Fatal(err)
	}
	digest = "replacement"
	_, err = f.Couch.ConfirmRecovery(context.Background(), required.Offer.Token)
	if err == nil || retired || len(f.Runner.Ops) != 0 {
		t.Fatalf("changed source acted: %v", err)
	}
}

func TestPreparedRecoveryPreservationFailureNeverRetiresOrLaunches(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Proc.Set(42, "old-helper")
	retired := 0
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 42, Identity: "old-helper", Role: "wrapper"}}, Digest: "owned", Retire: func(context.Context) error { retired++; return f.Proc.Signal(42, os.Interrupt) }}, nil
	}
	selection := RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh}
	_, err := f.Couch.RecoverSelection(t.Context(), selection)
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) {
		t.Fatal(err)
	}
	before, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil {
		t.Fatal(err)
	}
	preserveErr := errors.New("preservation disk full")
	preserved := 0
	f.Couch.RecoveryPreserve = func(context.Context, ThreadRecord, RecoveryObservation) (RecoveryObservation, error) {
		preserved++
		return RecoveryObservation{}, preserveErr
	}
	_, err = f.Couch.ConfirmRecovery(t.Context(), required.Offer.Token)
	after, readErr := f.Couch.Threads.GetThread(f.record.Address)
	if !errors.Is(err, preserveErr) || preserved != 1 || retired != 0 || len(f.Proc.Signals[42]) != 0 || len(f.Runner.Ops) != 0 || readErr != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("preservation failure effects: err=%v preserved=%d retired=%d signals=%v starts=%v after=%+v read=%v", err, preserved, retired, f.Proc.Signals, f.Runner.Ops, after, readErr)
	}
}

func TestPreparedRecoveryCancellationBeforeEffectsNeverSignals(t *testing.T) {
	for _, at := range []string{"before-admission", "during-preservation", "after-durable-preservation"} {
		t.Run(at, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			f.Proc.Set(42, "old-helper")
			retired, checked := 0, 0
			var cancel context.CancelFunc
			f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
				return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 42, Identity: "old-helper"}}, Digest: "owned", Check: func(context.Context) error {
					checked++
					if at == "after-durable-preservation" && checked == 2 {
						cancel()
					}
					return nil
				}, Retire: func(context.Context) error { retired++; return f.Proc.Signal(42, os.Interrupt) }}, nil
			}
			_, err := f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
			var required *RecoveryConfirmationRequired
			if !errors.As(err, &required) {
				t.Fatal(err)
			}
			ctx, stop := context.WithCancel(t.Context())
			cancel = stop
			defer stop()
			if at == "before-admission" {
				cancel()
			}
			f.Couch.RecoveryPreserve = func(_ context.Context, _ ThreadRecord, observation RecoveryObservation) (RecoveryObservation, error) {
				if at == "during-preservation" {
					cancel()
				}
				return observation, nil
			}
			_, err = f.Couch.ConfirmRecovery(ctx, required.Offer.Token)
			if !errors.Is(err, context.Canceled) || retired != 0 || len(f.Proc.Signals[42]) != 0 || len(f.Runner.Ops) != 0 {
				t.Fatalf("cancel at%s: err=%v retired=%d checks=%d signals=%v starts=%v", at, err, retired, checked, f.Proc.Signals, f.Runner.Ops)
			}
		})
	}
}

func TestPreparedRecoveryConfirmationRejectsChangedSourceWithoutEffects(t *testing.T) {
	for _, changed := range []string{"actor", "revision", "native-binding"} {
		t.Run(changed, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			f.Couch.RecoveryEvidence = nil
			f.Proc.Set(42, "old-helper")
			f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "original-native")
			retired := 0
			f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
				return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 42, Identity: "old-helper"}}, Digest: "same-generation", Retire: func(context.Context) error { retired++; return f.Proc.Signal(42, os.Interrupt) }}, nil
			}
			ctx := WithRecoveryConfirmer(t.Context(), "authenticated-original-hub")
			_, err := f.Couch.RecoverSelection(ctx, RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryProven})
			var required *RecoveryConfirmationRequired
			if !errors.As(err, &required) {
				t.Fatal(err)
			}
			switch changed {
			case "actor":
				ctx = WithRecoveryConfirmer(ctx, "replacement-hub")
			case "revision":
				name := "changed-source"
				if _, err := f.Couch.Threads.ApplyThreadMetadata(f.record.Address, f.record.Revision, ThreadMetadataPatch{Name: &name}); err != nil {
					t.Fatal(err)
				}
			case "native-binding":
				f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "replacement-native")
			}
			_, err = f.Couch.ConfirmRecovery(ctx, required.Offer.Token)
			if err == nil || retired != 0 || len(f.Proc.Signals[42]) != 0 || len(f.Runner.Ops) != 0 {
				t.Fatalf("changed%s acted: err=%v retired=%d signals=%v starts=%v", changed, err, retired, f.Proc.Signals, f.Runner.Ops)
			}
		})
	}
}

func TestPreparedRecoveryForeignFreshLeavesRemnantsAndUsesNewTag(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	slot := *f.Couch.Threads.slot
	f.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{slot.PrimaryRoot: {Slots: []SlotCandidate{{Identity: slot, Verified: true}}}}}
	f.Proc.Set(42, "foreign-process-generation")
	f.Artifacts.SetPairSession(f.record.Address, "foreign-terminal", true)
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{}, errors.New("unattributable foreign terminal owner")
	}
	oldBinding, err := f.Artifacts.PairSession(f.record.Address)
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
	if err != nil {
		t.Fatal(err)
	}
	if result.Start.Handle == nil || result.Start.Record.Thread == f.record.Address || !strings.Contains(result.Notice, "untouched") {
		t.Fatalf("Fresh did not isolate foreign generation: %+v", result)
	}
	newBinding, readErr := f.Artifacts.PairSession(f.record.Address)
	identity, identityErr := f.Proc.Identity(42)
	if readErr != nil || identityErr != nil || identity != "foreign-process-generation" || !reflect.DeepEqual(oldBinding, newBinding) || len(f.Proc.Signals[42]) != 0 || len(f.Proc.GroupSignals[42]) != 0 || len(f.Artifacts.Quiesces()) != 0 {
		t.Fatalf("foreign remnants changed: before=%+v after=%+v identity=%q signals=%v groups=%v quiesces=%v errs=%v/%v", oldBinding, newBinding, identity, f.Proc.Signals, f.Proc.GroupSignals, f.Artifacts.Quiesces(), readErr, identityErr)
	}
	if f.prepared != 0 {
		t.Fatalf("normal workspace convergence ran beneath foreign owner: %d", f.prepared)
	}
	if !strings.Contains(result.Notice, "workspace setup was not refreshed") {
		t.Fatalf("missing read-only readiness notice: %s", result.Notice)
	}
	var profile launcher.TrustedLaunchProfile
	if err := json.Unmarshal([]byte(environmentValue(f.Runner.Child(result.Start.Handle.ID()).Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
		t.Fatal(err)
	}
	if !profile.FreshRequired || profile.ResumeRequired || profile.RequiredSessionID != "" || profile.FreshNonce == "" || profile.Prompt == "" || profile.Tag == string(f.record.Address.Tag) || !strings.Contains(profile.Prompt, f.observation.Capture.Raw) {
		t.Fatalf("foreign Fresh profile: %+v", profile)
	}
}

func TestPreparedFreshRepresentationsBypassOnlyNativeGuards(t *testing.T) {
	for _, scenario := range []string{"missing saved profile", "unfinished park", "continuation", "young start"} {
		for _, route := range []string{"explicit", "candidate", "operation"} {
			t.Run(scenario+"/"+route, func(t *testing.T) {
				f := newRecoveryExecutionFixture(t)
				f.Couch.RootAgent = "codex"
				var err error
				switch scenario {
				case "missing saved profile":
					f.record, err = f.Couch.Threads.updateExistingThread(f.record.Address, f.record.Revision, func(r *ThreadRecord) error {
						r.LatestLaunchProfile = nil
						r.Incarnations[0].LaunchProfile = nil
						return nil
					})
				case "unfinished park":
					f.record, err = f.Couch.Threads.BeginPark(f.record.Address, f.record.Revision, ParkIdentity{Nonce: "park-fresh-equivalence", Address: f.record.Address, PID: 42, ProcessIdentity: "old-helper"})
				case "continuation":
					request := testContinuationRequest(t, f.record)
					f.record, err = f.Couch.Threads.PublishContinuation(f.record.Address, f.record.Revision, request)
				case "young start":
					f, _ = prepareRecoveryStartFixture(t, time.Minute)
					f.Couch.RootAgent = "codex"
				}
				if err != nil {
					t.Fatal(err)
				}
				preview := f.preview(t)
				if preview.Fresh.Key == "" {
					t.Fatal("fixture did not advertise keyed Fresh")
				}
				before, err := f.Couch.Threads.PreviewThreadContext(t.Context(), f.record.Address)
				if err != nil {
					t.Fatal(err)
				}
				// The same native-only restrictions still refuse real transcript resumes.
				if _, err := f.Couch.ResumeCandidate(t.Context(), ResumeTarget{Address: f.record.Address}, preview.Candidates[0].Key); err == nil {
					t.Fatal("native recovery bypassed its guard")
				}
				after, err := f.Couch.Threads.PreviewThreadContext(t.Context(), f.record.Address)
				if err != nil || !reflect.DeepEqual(before, after) || len(f.Runner.Ops) != 0 || f.preserved != 0 {
					t.Fatalf("native refusal had effects: %v", err)
				}
				target := ResumeTarget{Address: f.record.Address}
				var result RecoveryResumeResult
				switch route {
				case "explicit":
					result, err = f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: target, Kind: RecoveryFresh})
				case "candidate":
					result, err = f.Couch.ResumeCandidate(t.Context(), target, preview.Fresh.Key)
				case "operation":
					var value any
					value, err = DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(f.Couch)}, OperationCall{Name: "resume", Context: t.Context(), Implicit: true, Args: map[string]string{"repo-scope": f.record.Address.RepoScope, "tag": string(f.record.Address.Tag), "candidate": preview.Fresh.Key}})
					if err == nil {
						result = value.(RecoveryResumeResult)
					}
				}
				if err != nil {
					t.Fatalf("advertised Fresh refused through %s: %v", route, err)
				}
				if result.Start.Handle == nil {
					t.Fatal("Fresh returned no launched helper")
				}
				child := f.Runner.Child(result.Start.Handle.ID())
				var profile launcher.TrustedLaunchProfile
				if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
					t.Fatal(err)
				}
				if !profile.FreshRequired || profile.ResumeRequired || profile.RequiredSessionID != "" || profile.Prompt == "" {
					t.Fatalf("not one oriented Fresh launch: %+v", profile)
				}
				starts := 0
				for _, op := range f.Runner.Ops {
					if strings.HasPrefix(op, "start ") {
						starts++
					}
				}
				if starts != 1 || len(f.Proc.Signals) != 0 || len(f.Proc.GroupSignals) != 0 {
					t.Fatalf("unexpected Fresh effects: starts=%d signals=%v", starts, f.Proc.Signals)
				}
			})
		}
	}
}
