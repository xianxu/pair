package couchcore

import (
	"reflect"
	"testing"
	"time"
)

func admittedStartRecord(t *testing.T) ThreadRecord {
	t.Helper()
	record := validThreadRecord(t)
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{State: IncarnationCreating, StartedAt: record.CreatedAt}}
	return record
}

func TestAdvanceStartTransactionClaimHelperAndRegistrationSequence(t *testing.T) {
	original := admittedStartRecord(t)
	claimed, err := AdvanceStartTransaction(original, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if original.Incarnations[0].Start != nil || claimed.Incarnations[0].Start == nil {
		t.Fatal("claim mutated input or failed to copy start state")
	}

	helper, err := AdvanceStartTransaction(claimed, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	if helper.Incarnations[0].PID != 42 || helper.Incarnations[0].State != IncarnationCreating {
		t.Fatalf("helper state = %+v", helper.Incarnations[0])
	}

	registered, err := AdvanceStartTransaction(helper, StartEvent{
		Kind: StartRegistered, Nonce: "start-0123456789abcdef",
	})
	if err != nil {
		t.Fatalf("registered: %v", err)
	}
	incarnation := registered.Incarnations[0]
	if incarnation.State != IncarnationLive || incarnation.Start != nil || incarnation.PID != 42 || incarnation.Identity != "helper-token" {
		t.Fatalf("registered state = %+v", incarnation)
	}
}

func TestAdvanceStartTransactionRejectsOutOfOrderOrWrongNonce(t *testing.T) {
	base := admittedStartRecord(t)
	claim := StartEvent{Kind: StartClaimed, Nonce: "start-0123456789abcdef", Owner: SupervisorOwner{PID: 41, Identity: "owner-token"}}
	claimed, err := AdvanceStartTransaction(base, claim)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []StartEvent{
		{Kind: StartHelperRecorded, Nonce: "wrong", Helper: ProcessIdentity{PID: 42, Identity: "helper-token"}},
		{Kind: StartRegistered, Nonce: claim.Nonce},
		{Kind: StartClaimed, Nonce: "second", Owner: claim.Owner},
	} {
		before := cloneThreadRecord(claimed)
		if _, err := AdvanceStartTransaction(claimed, event); err == nil {
			t.Fatalf("accepted event %+v", event)
		}
		if !reflect.DeepEqual(claimed, before) {
			t.Fatalf("failed event mutated input: %+v", event)
		}
	}
}

func TestReconcileStartPreservesOccupiedOrProvenFreeInvariant(t *testing.T) {
	base := admittedStartRecord(t)
	claimed, _ := AdvanceStartTransaction(base, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	helper, _ := AdvanceStartTransaction(claimed, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})

	tests := []struct {
		name   string
		record ThreadRecord
		obs    StartObservation
		want   StartReconcileAction
	}{
		{name: "prefork owner live remains occupied", record: claimed, obs: StartObservation{Owner: Live, Registration: RegistrationAbsent}, want: StartKeepOccupied},
		{name: "prefork owner unknown remains occupied", record: claimed, obs: StartObservation{Owner: Unknown, Registration: RegistrationAbsent}, want: StartKeepOccupied},
		{name: "prefork dead owner is proven free", record: claimed, obs: StartObservation{Owner: Dead, Registration: RegistrationAbsent}, want: StartRollback},
		{name: "live helper without registration remains occupied", record: helper, obs: StartObservation{Helper: Live, Registration: RegistrationAbsent}, want: StartKeepOccupied},
		{name: "unknown helper remains occupied", record: helper, obs: StartObservation{Helper: Unknown, Registration: RegistrationAbsent}, want: StartKeepOccupied},
		{name: "dead helper without registration is proven free", record: helper, obs: StartObservation{Helper: Dead, Registration: RegistrationAbsent}, want: StartRollback},
		{name: "registered live helper promotes live", record: helper, obs: StartObservation{Helper: Live, Registration: RegistrationEstablished}, want: StartPromoteLive},
		{name: "registered dead helper stays occupied unknown", record: helper, obs: StartObservation{Helper: Dead, Registration: RegistrationEstablished}, want: StartPromoteUnknown},
		{name: "unreadable registration remains occupied", record: helper, obs: StartObservation{Helper: Dead, Registration: RegistrationUnknown}, want: StartKeepOccupied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := ReconcileStart(tt.record, tt.obs)
			if err != nil || decision.Action != tt.want || decision.Nonce != "start-0123456789abcdef" {
				t.Fatalf("ReconcileStart = %+v, %v", decision, err)
			}
		})
	}
}

func TestAdvanceStartTransactionRunsAgainstFakeRunnerLifecycle(t *testing.T) {
	runner := NewFakeRunner()
	record := admittedStartRecord(t)
	record, err := AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	h, err := runner.StartBlocked("/repo", []string{"pair"}, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	record, err = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: h.PID(), Identity: h.Identity()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if runner.Child(h.ID()).ExecCount != 0 {
		t.Fatal("fake target execed before durable helper transition")
	}
	if err := h.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	record, err = AdvanceStartTransaction(record, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
	if err != nil || record.Incarnations[0].State != IncarnationLive || runner.Child(h.ID()).ExecCount != 1 {
		t.Fatalf("final record/child = %+v / %+v, %v", record.Incarnations[0], runner.Child(h.ID()), err)
	}
}
