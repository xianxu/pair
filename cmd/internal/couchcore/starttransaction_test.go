package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"reflect"
	"testing"
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
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	claimed, err := AdvanceStartTransaction(original, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner:   SupervisorOwner{PID: 41, Identity: "owner-token"},
		Profile: &profile,
	})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if original.Incarnations[0].Start != nil || claimed.Incarnations[0].Start == nil {
		t.Fatal("claim mutated input or failed to copy start state")
	}
	if claimed.Incarnations[0].LaunchProfile != nil || claimed.Incarnations[0].Start.LaunchProfile == nil {
		t.Fatalf("unregistered claim committed profile early: %+v", claimed.Incarnations[0])
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
	if helper.Incarnations[0].LaunchProfile != nil {
		t.Fatalf("helper-recorded state committed profile early: %+v", helper.Incarnations[0])
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
	if incarnation.LaunchProfile == nil || !reflect.DeepEqual(*incarnation.LaunchProfile, profile) {
		t.Fatalf("registered profile = %+v, want %+v", incarnation.LaunchProfile, profile)
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

func TestAdvanceStartTransactionCanRecoverEstablishedDeadHelperAsUnknown(t *testing.T) {
	record := admittedStartRecord(t)
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	recovered, err := AdvanceStartTransaction(record, StartEvent{
		Kind: StartRecoveredUnknown, Nonce: "start-0123456789abcdef",
	})
	if err != nil || recovered.Incarnations[0].State != IncarnationUnknown || recovered.Incarnations[0].Start != nil {
		t.Fatalf("recovered = %+v, %v", recovered.Incarnations[0], err)
	}
}

func TestStartRegisteredCopiesLatestLaunchProfile(t *testing.T) {
	record := admittedStartRecord(t)
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"}, Profile: &profile,
	})
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	registered, err := AdvanceStartTransaction(record, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if registered.LatestLaunchProfile == nil || !reflect.DeepEqual(*registered.LatestLaunchProfile, profile) {
		t.Fatalf("latest profile = %+v, want %+v", registered.LatestLaunchProfile, profile)
	}
	registered.Incarnations[0].LaunchProfile.Argv[0] = "changed"
	if registered.LatestLaunchProfile.Argv[0] != "--sandbox" {
		t.Fatal("thread and incarnation profiles alias")
	}
}

func TestFailedStartDoesNotReplaceLatestLaunchProfile(t *testing.T) {
	oldProfile := LaunchProfile{Agent: "claude", Argv: []string{"--resume", "old"}}
	newProfile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	record := admittedStartRecord(t)
	record.LatestLaunchProfile = &oldProfile
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"}, Profile: &newProfile,
	})
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	recovered, err := AdvanceStartTransaction(record, StartEvent{Kind: StartRecoveredUnknown, Nonce: "start-0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.LatestLaunchProfile == nil || !reflect.DeepEqual(*recovered.LatestLaunchProfile, oldProfile) {
		t.Fatalf("failed start latest profile = %+v, want %+v", recovered.LatestLaunchProfile, oldProfile)
	}
	if _, err := AdvanceStartTransaction(record, StartEvent{Kind: StartRegistered, Nonce: "wrong"}); err == nil {
		t.Fatal("wrong nonce registration succeeded")
	}
	if !reflect.DeepEqual(*record.LatestLaunchProfile, oldProfile) {
		t.Fatal("failed registration mutated input profile")
	}
}

func TestReconcileRegisteredTargetRetiresUnknownAtomically(t *testing.T) {
	record := validThreadRecord(t)
	cp, err := checkpoint.New("/checkpoint.md", "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nResume exact task.\n")
	if err != nil {
		t.Fatal(err)
	}
	record.Continuation = &checkpoint.Request{Version: checkpoint.Version, ID: checkpoint.RequestID(record.Address.RepoScope, string(record.Address.Tag), 2, cp.Digest), Checkpoint: cp, Source: checkpoint.Source{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2, Helper: checkpoint.Process{PID: 42, Identity: "source"}}, CreatedAt: record.CreatedAt, Phase: checkpoint.Running, Attempt: "attempt", SourceAbsence: &checkpoint.SourceAbsence{Session: "pair-exact", LaunchOrdinal: 2, ObservedAt: record.CreatedAt, RecordRevision: record.Revision}, Target: &checkpoint.Target{Process: checkpoint.Process{PID: 43, Identity: "target"}, ObservedAt: record.CreatedAt}}
	record.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown, PID: 43, Identity: "target", StartedAt: record.CreatedAt}}
	if err := ValidateThreadRecord(record); err != nil {
		t.Fatal(err)
	}
	proof := RegisteredTargetProof{RequestID: record.Continuation.ID, Agent: record.Continuation.Source.Agent, Session: record.Continuation.Source.Session, Attempt: record.Continuation.Attempt, Helper: ProcessIdentity{PID: record.Incarnations[0].PID, Identity: record.Incarnations[0].Identity}}
	for _, tc := range []struct {
		name   string
		mutate func(*ThreadRecord, *RegisteredTargetProof)
	}{
		{"exact", func(*ThreadRecord, *RegisteredTargetProof) {}},
		{"open-park", func(r *ThreadRecord, _ *RegisteredTargetProof) { r.Park = &ParkTransaction{} }},
		{"complete", func(r *ThreadRecord, _ *RegisteredTargetProof) { r.Continuation.Phase = checkpoint.Complete }},
		{"foreign-attempt", func(_ *ThreadRecord, p *RegisteredTargetProof) { p.Attempt = "foreign" }},
		{"foreign-agent", func(_ *ThreadRecord, p *RegisteredTargetProof) { p.Agent = "codex" }},
		{"foreign-session", func(_ *ThreadRecord, p *RegisteredTargetProof) { p.Session = "foreign" }},
		{"foreign-request", func(_ *ThreadRecord, p *RegisteredTargetProof) { p.RequestID = "foreign" }},
		{"foreign-helper", func(_ *ThreadRecord, p *RegisteredTargetProof) { p.Helper.Identity = "foreign" }},
		{"live", func(r *ThreadRecord, _ *RegisteredTargetProof) { r.Incarnations[0].State = IncarnationLive }},
		{"open-start", func(r *ThreadRecord, _ *RegisteredTargetProof) {
			r.Incarnations[0].Start = &ThreadStartClaim{Nonce: "open"}
		}},
		{"multiple", func(r *ThreadRecord, _ *RegisteredTargetProof) {
			r.Incarnations = append(r.Incarnations, r.Incarnations[0])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := cloneThreadRecord(record)
			p := proof
			tc.mutate(&input, &p)
			if tc.name == "complete" {
				if err := ValidateThreadRecord(input); err != nil {
					t.Fatalf("complete fixture invalid: %v", err)
				}
			}
			before := cloneThreadRecord(input)
			next, err := ReconcileRegisteredTarget(input, p)
			if !reflect.DeepEqual(input, before) {
				t.Fatal("transition mutated input")
			}
			if tc.name != "exact" {
				if err == nil {
					t.Fatal("unproved retirement accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			before.Incarnations = nil
			if !reflect.DeepEqual(next, before) {
				t.Fatalf("retirement changed unrelated facts: %+v", next)
			}
		})
	}
}
