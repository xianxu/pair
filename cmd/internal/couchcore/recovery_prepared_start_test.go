package couchcore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func prepareRecoveryStartFixture(t *testing.T, age time.Duration) (*recoveryExecutionFixture, string) {
	t.Helper()
	f := newRecoveryExecutionFixture(t)
	oldNonce := "old-exact-start-nonce"
	record, err := f.Couch.Threads.updateExistingThread(f.record.Address, f.record.Revision, func(next *ThreadRecord) error {
		next.Incarnations[0].State = IncarnationCreating
		next.Incarnations[0].LaunchProfile = nil
		next.Incarnations[0].StartedAt = f.Now.Add(-age)
		next.Incarnations[0].Start = &ThreadStartClaim{Nonce: oldNonce, OwnerPID: 4242, OwnerIdentity: "live-supervisor", LaunchProfile: next.LatestLaunchProfile}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	f.record = record
	f.Proc.Set(4242, "live-supervisor")
	f.Artifacts.SetNativeBinding(record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
	return f, oldNonce
}

func TestPreparedNativeRecoveryExpiredLiveOwnerFencesExactStartBeforeOneLaunch(t *testing.T) {
	f, oldNonce := prepareRecoveryStartFixture(t, 5*time.Minute)
	f.Proc.Set(42, "old-helper")
	retired, fences, newClaims, launches := 0, 0, 0, 0
	nextNonce := ""
	f.Couch.Threads.hooks.BeforeUpdate = func(current, next ThreadRecord) error {
		if len(current.Incarnations) == 1 && current.Incarnations[0].Start != nil && current.Incarnations[0].Start.Nonce == oldNonce && len(next.Incarnations) == 0 {
			if retired != 1 || next.RecoveryIntent == nil || next.RecoveryIntent.NativeID != "proven-native" {
				t.Fatalf("fence before durable native selection/retirement: retired=%d next=%+v", retired, next)
			}
			fences++
		}
		if len(next.Incarnations) == 1 && next.Incarnations[0].Start != nil && next.Incarnations[0].Start.Nonce != oldNonce {
			if len(current.Incarnations) == 0 {
				if fences != 1 {
					t.Fatalf("replacement start preceded exact old fence: %d", fences)
				}
				newClaims++
				nextNonce = next.Incarnations[0].Start.Nonce
			}
		}
		return nil
	}
	f.Couch.RecoveryRetirement = func(_ context.Context, record ThreadRecord) (RecoveryRetirement, error) {
		if record.Incarnations[0].Start.Nonce != oldNonce {
			t.Fatal("prepared a different old attempt")
		}
		return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 42, Identity: "old-helper", Role: "wrapper"}}, Digest: "exact-start-generation", Check: func(context.Context) error { return nil }, Retire: func(context.Context) error { retired++; f.Proc.Kill(42); return nil }}, nil
	}
	f.Runner.AfterBlockedStart = func(string) {
		launches++
		current, err := f.Couch.Threads.GetThread(f.record.Address)
		if err != nil {
			t.Fatal(err)
		}
		if fences != 1 || newClaims != 1 || nextNonce == "" || nextNonce == oldNonce || len(current.Incarnations) != 1 || current.Incarnations[0].Start == nil || current.Incarnations[0].Start.Nonce != nextNonce {
			t.Fatalf("launch without new exact claim: fences=%d claims=%d nonce=%q record=%+v", fences, newClaims, nextNonce, current)
		}
		_, err = f.Couch.Threads.AdvanceStart(current.Address, current.Revision, StartEvent{Kind: StartHelperRecorded, Nonce: oldNonce, Helper: ProcessIdentity{PID: 9000, Identity: "late-old-driver"}})
		if err == nil {
			t.Fatal("old supervisor resurrected fenced nonce at current revision")
		}
		after, err := f.Couch.Threads.GetThread(current.Address)
		if err != nil || !reflect.DeepEqual(current, after) {
			t.Fatalf("late old nonce altered current claim: %+v err=%v", after, err)
		}
	}
	ctx := WithRecoveryConfirmer(t.Context(), "headless-hub")
	_, err := f.Couch.RecoverSelection(ctx, RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryProven})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) || retired != 0 || fences != 0 || newClaims != 0 || launches != 0 {
		t.Fatalf("unconfirmed effects: err=%v retired=%d fences=%d claims=%d launches=%d", err, retired, fences, newClaims, launches)
	}
	result, err := f.Couch.ConfirmRecovery(ctx, required.Offer.Token)
	if err != nil || result.Start.Handle == nil || retired != 1 || fences != 1 || newClaims != 1 || launches != 1 {
		t.Fatalf("recovery result=%+v err=%v retired=%d fences=%d claims=%d launches=%d", result, err, retired, fences, newClaims, launches)
	}
	actualStarts := 0
	for _, operation := range f.Runner.Ops {
		if strings.HasPrefix(operation, "start ") {
			actualStarts++
		}
	}
	if actualStarts != 1 {
		t.Fatalf("native recovery started %d helpers: %v", actualStarts, f.Runner.Ops)
	}
	if observeExactProcess(f.Proc, ProcessIdentity{PID: 4242, Identity: "live-supervisor"}) != Live || len(f.Proc.Signals[4242]) != 0 || len(f.Proc.GroupSignals[4242]) != 0 {
		t.Fatal("recovery signalled the live old supervisor")
	}
	if _, err := f.Couch.ConfirmRecovery(ctx, required.Offer.Token); err == nil || launches != 1 {
		t.Fatalf("confirmation reused: launches=%d err=%v", launches, err)
	}
}

func TestPreparedNativeRecoveryYoungStartingNeverRetiresClaimsOrLaunches(t *testing.T) {
	for _, owner := range []Liveness{Live, Unknown} {
		t.Run(owner.String(), func(t *testing.T) {
			f, _ := prepareRecoveryStartFixture(t, 5*time.Minute-time.Nanosecond)
			if owner == Unknown {
				f.Proc.SetUnknown(4242)
			}
			before, err := f.Couch.Threads.GetThread(f.record.Address)
			if err != nil {
				t.Fatal(err)
			}
			retired := 0
			f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
				return RecoveryRetirement{Digest: "empty-exact-terminal", Check: func(context.Context) error { return nil }, Retire: func(context.Context) error { retired++; return nil }}, nil
			}
			_, err = f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryProven})
			after, readErr := f.Couch.Threads.GetThread(f.record.Address)
			if err == nil || !strings.Contains(err.Error(), "starting") || retired != 0 || f.preserved != 0 || len(f.Runner.Ops) != 0 || readErr != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("young start effects: err=%v retire=%d preserve=%d starts=%v before=%+v after=%+v read=%v", err, retired, f.preserved, f.Runner.Ops, before, after, readErr)
			}
			if len(f.Proc.Signals) != 0 || len(f.Proc.GroupSignals) != 0 {
				t.Fatal("young start signalled a process")
			}
		})
	}
}

func TestPreparedNativeRecoveryOpenParkNeverPreparesRetirementOrLaunches(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Proc.Set(42, "old-helper")
	record, err := f.Couch.Threads.BeginPark(f.record.Address, f.record.Revision, ParkIdentity{Nonce: "park-exact-open", Address: f.record.Address, PID: 42, ProcessIdentity: "old-helper"})
	if err != nil {
		t.Fatal(err)
	}
	f.record = record
	f.Artifacts.SetNativeBinding(record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
	preparations, retired := 0, 0
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		preparations++
		return RecoveryRetirement{Retire: func(context.Context) error { retired++; return nil }}, nil
	}
	_, err = f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: record.Address}, Kind: RecoveryProven})
	after, readErr := f.Couch.Threads.GetThread(record.Address)
	if err == nil || !strings.Contains(err.Error(), "park transaction") || preparations != 0 || retired != 0 || f.preserved != 0 || len(f.Runner.Ops) != 0 || readErr != nil || !reflect.DeepEqual(record, after) {
		t.Fatalf("open park effects: err=%v preparations=%d retired=%d preserve=%d starts=%v after=%+v read=%v", err, preparations, retired, f.preserved, f.Runner.Ops, after, readErr)
	}
	if len(f.Proc.Signals) != 0 || len(f.Proc.GroupSignals) != 0 {
		t.Fatal("open park signalled a process")
	}
}
