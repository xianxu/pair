package couchcore

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func recoveryFenceFixture(t *testing.T) (*testEnv, ThreadRecord, threadrecord.RecoveryIntent) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	record := staleStartRecord(env.Now.Add(-5 * time.Minute))
	markActionableParked(&record, record.Incarnations[0].StartedAt)
	record.Incarnations[0].PID, record.Incarnations[0].Identity = 4243, "dead-helper"
	intent := threadrecord.RecoveryIntent{Mode: threadrecord.RecoveryProven, Address: toPersistedThreadAddress(record.Address), SourceRevision: record.Revision, CreatedAt: env.Now, Agent: "claude", SelectedKey: "explicit", NativeID: "native"}
	record.RecoveryIntent = threadrecord.CloneRecoveryIntent(&intent)
	record, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(4242, "supervisor")
	env.Artifacts.SetPairSession(record.Address, "old-terminal", false)
	return env, record, intent
}

func TestRecoveryFenceTransitionPreservesRetainedChoice(t *testing.T) {
	_, record, intent := recoveryFenceFixture(t)
	original := cloneThreadRecord(record)
	got, err := AdvanceStartTransaction(record, StartEvent{Kind: StartEventKind("recovery-fenced"), Nonce: record.Incarnations[0].Start.Nonce, RecoveryIntent: &intent})
	if err != nil {
		t.Fatalf("fence transition missing: %v", err)
	}
	if len(got.Incarnations) != 0 || !reflect.DeepEqual(got.RecoveryIntent, original.RecoveryIntent) || got.Address != original.Address || !reflect.DeepEqual(got.VerifiedPark, original.VerifiedPark) {
		t.Fatalf("fence discarded recovery choice or thread: %+v", got)
	}
	if !reflect.DeepEqual(record, original) {
		t.Fatal("pure fence mutated input")
	}
}

func TestRecoveryFenceTransitionRejectsChangedNonceOrChoice(t *testing.T) {
	for _, scenario := range []string{"nonce", "intent", "missing intent", "missing retained choice"} {
		t.Run(scenario, func(t *testing.T) {
			_, record, intent := recoveryFenceFixture(t)
			event := StartEvent{Kind: StartEventKind("recovery-fenced"), Nonce: record.Incarnations[0].Start.Nonce, RecoveryIntent: &intent}
			switch scenario {
			case "nonce":
				event.Nonce = "other"
			case "intent":
				event.RecoveryIntent.NativeID = "other"
			case "missing intent":
				event.RecoveryIntent = nil
			case "missing retained choice":
				record.RecoveryIntent = nil
			}
			before := cloneThreadRecord(record)
			if _, err := AdvanceStartTransaction(record, event); err == nil {
				t.Fatal("fence accepted changed authority")
			}
			if !reflect.DeepEqual(record, before) {
				t.Fatal("refusal mutated original")
			}
		})
	}
}

func TestRecoveryStartFenceReleasesExpiredOwnedClaim(t *testing.T) {
	for _, unlaunched := range []bool{false, true} {
		t.Run(map[bool]string{false: "dead helper", true: "unlaunched"}[unlaunched], func(t *testing.T) {
			env, record, intent := recoveryFenceFixture(t)
			if unlaunched {
				var err error
				record, err = env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
					next.Incarnations[0].PID = 0
					next.Incarnations[0].Identity = ""
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			nonce := record.Incarnations[0].Start.Nonce
			got, err := env.Couch.fenceRecoveryStart(context.Background(), record, intent)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Incarnations) != 0 || got.Revision <= record.Revision || !reflect.DeepEqual(got.RecoveryIntent, record.RecoveryIntent) || !reflect.DeepEqual(got.VerifiedPark, record.VerifiedPark) || !reflect.DeepEqual(got.ParkHistory, record.ParkHistory) {
				t.Fatalf("fence lost record authority: %+v", got)
			}
			if observeExactProcess(env.Proc, ProcessIdentity{PID: 4242, Identity: "supervisor"}) != Live || len(env.Proc.Signals) != 0 || len(env.Proc.GroupSignals) != 0 {
				t.Fatal("fence signalled supervisor or foreign process")
			}
			if _, err := env.Couch.Threads.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartHelperRecorded, Nonce: nonce, Helper: ProcessIdentity{PID: 5000, Identity: "late"}}); err == nil {
				t.Fatal("late driver advanced stale revision")
			}
			if _, err := env.Couch.Threads.AdvanceStart(got.Address, got.Revision, StartEvent{Kind: StartHelperRecorded, Nonce: nonce, Helper: ProcessIdentity{PID: 5000, Identity: "late"}}); err == nil {
				t.Fatal("late driver resurrected fenced nonce")
			}
			again, err := env.Couch.fenceRecoveryStart(context.Background(), got, intent)
			if err != nil || !reflect.DeepEqual(again, got) {
				t.Fatalf("no-start fence changed record: %+v %v", again, err)
			}
		})
	}
}

func TestRecoveryStartFenceRefusesChangedOrUnprovenSource(t *testing.T) {
	for _, scenario := range []string{"young", "future", "live helper", "unknown helper", "live session", "unreadable session", "changed revision", "changed nonce", "changed choice", "cancelled", "cancelled after observation"} {
		t.Run(scenario, func(t *testing.T) {
			env, record, intent := recoveryFenceFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mutate := func(change func(*ThreadRecord) error, refresh bool) {
				t.Helper()
				got, err := env.Couch.Threads.updateExistingThread(record.Address, record.Revision, change)
				if err != nil {
					t.Fatal(err)
				}
				if refresh {
					record = got
				}
			}
			switch scenario {
			case "young":
				mutate(func(next *ThreadRecord) error { next.Incarnations[0].StartedAt = env.Now.Add(-time.Minute); return nil }, true)
			case "future":
				mutate(func(next *ThreadRecord) error { next.Incarnations[0].StartedAt = env.Now.Add(time.Minute); return nil }, true)
			case "live helper":
				env.Proc.Set(4243, "dead-helper")
			case "unknown helper":
				env.Proc.SetUnknown(4243)
			case "live session":
				env.Artifacts.SetPairSession(record.Address, "old-terminal", true)
			case "unreadable session":
				env.Artifacts.BeforePairSession = func(ThreadAddress) error { return fmt.Errorf("session observation unavailable") }
			case "changed revision":
				mutate(func(next *ThreadRecord) error { next.Name = "concurrent metadata"; return nil }, false)
			case "changed nonce":
				mutate(func(next *ThreadRecord) error { next.Incarnations[0].Start.Nonce = "other-attempt"; return nil }, false)
			case "changed choice":
				intent.NativeID = "other"
			case "cancelled":
				cancel()
			case "cancelled after observation":
				env.Artifacts.BeforePairSession = func(ThreadAddress) error { cancel(); return nil }
			}
			before, err := env.Couch.Threads.GetThread(record.Address)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.Couch.fenceRecoveryStart(ctx, record, intent); err == nil {
				t.Fatal("fenced unproven or changed source")
			}
			after, err := env.Couch.Threads.GetThread(record.Address)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, before) {
				t.Fatal("refusal modified exact claim")
			}
			if len(env.Proc.Signals) != 0 || len(env.Proc.GroupSignals) != 0 {
				t.Fatal("refusal signalled a process")
			}
		})
	}
}
