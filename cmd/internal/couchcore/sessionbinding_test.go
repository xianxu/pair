package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"reflect"
	"testing"
)

func TestStartBindingSurvivesRegistrationAndUnknownRecovery(t *testing.T) {
	for _, kind := range []StartEventKind{StartRegistered, StartRecoveredUnknown} {
		t.Run(string(kind), func(t *testing.T) {
			r := admittedStartRecord(t)
			old := couchidentity.SessionBinding{C: 1, M: 1, Name: "📁1-1", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: "old"}
			proposed := old
			proposed.M = 2
			proposed.Name = "📁1-2"
			proposed.StartNonce = "start-0123456789abcdef"
			r.SessionBinding = &old
			r, err := AdvanceStartTransaction(r, StartEvent{Kind: StartClaimed, Nonce: proposed.StartNonce, Owner: SupervisorOwner{PID: 41, Identity: "owner"}})
			if err != nil {
				t.Fatal(err)
			}
			bound, err := AdvanceStartTransaction(r, StartEvent{Kind: StartSessionBound, Nonce: proposed.StartNonce, Binding: &proposed})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(bound.SessionBinding, &old) || r.Incarnations[0].Start.SessionBinding != nil {
				t.Fatal("pending binding replaced current or mutated source")
			}
			round := fromPersistedThreadRecord(toPersistedThreadRecord(bound))
			if !reflect.DeepEqual(bound, round) {
				t.Fatal("binding lost in persistence")
			}
			helper, err := AdvanceStartTransaction(bound, StartEvent{Kind: StartHelperRecorded, Nonce: proposed.StartNonce, Helper: ProcessIdentity{PID: 42, Identity: "helper"}})
			if err != nil {
				t.Fatal(err)
			}
			decision, err := ReconcileStart(helper, StartObservation{Helper: Dead, Registration: RegistrationUnknown})
			if err != nil || decision.Action != StartKeepOccupied {
				t.Fatalf("unknown lost claim: %+v %v", decision, err)
			}
			got, err := AdvanceStartTransaction(helper, StartEvent{Kind: kind, Nonce: proposed.StartNonce})
			if err != nil {
				t.Fatal(err)
			}
			if got.Incarnations[0].Start != nil || !reflect.DeepEqual(got.SessionBinding, &proposed) {
				t.Fatalf("binding lost on %s: %+v", kind, got)
			}
			got.SessionBinding.Name = "mutated"
			if helper.Incarnations[0].Start.SessionBinding.Name != "📁1-2" {
				t.Fatal("binding aliases input")
			}
		})
	}
}

func TestStartBindingRejectsWrongAddressAndReplacement(t *testing.T) {
	r := admittedStartRecord(t)
	r, err := AdvanceStartTransaction(r, StartEvent{Kind: StartClaimed, Nonce: "start-0123456789abcdef", Owner: SupervisorOwner{PID: 41, Identity: "owner"}})
	if err != nil {
		t.Fatal(err)
	}
	b := couchidentity.SessionBinding{C: 1, M: 1, Name: "📁1-1", ScopeKey: r.Address.RepoScope, Tag: "other", StartNonce: r.Incarnations[0].Start.Nonce}
	if _, err := AdvanceStartTransaction(r, StartEvent{Kind: StartSessionBound, Nonce: b.StartNonce, Binding: &b}); err == nil {
		t.Fatal("accepted foreign binding")
	}
	b.Tag = string(r.Address.Tag)
	r, err = AdvanceStartTransaction(r, StartEvent{Kind: StartSessionBound, Nonce: b.StartNonce, Binding: &b})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdvanceStartTransaction(r, StartEvent{Kind: StartSessionBound, Nonce: b.StartNonce, Binding: &b}); err == nil {
		t.Fatal("replaced pending binding")
	}
}

func TestReconcilePendingTerminalRequiresAbsenceBeforeRollback(t *testing.T) {
	r := admittedStartRecord(t)
	nonce := "start-0123456789abcdef"
	r, _ = AdvanceStartTransaction(r, StartEvent{Kind: StartClaimed, Nonce: nonce, Owner: SupervisorOwner{PID: 41, Identity: "owner"}})
	binding := couchidentity.SessionBinding{C: 1, M: 1, Name: "📁1-1", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: nonce}
	r, _ = AdvanceStartTransaction(r, StartEvent{Kind: StartSessionBound, Nonce: nonce, Binding: &binding})
	r, _ = AdvanceStartTransaction(r, StartEvent{Kind: StartHelperRecorded, Nonce: nonce, Helper: ProcessIdentity{PID: 42, Identity: "helper"}})
	for _, presence := range []SessionPresence{PresenceUnobserved, PresencePresent, PresenceAbsent} {
		want := StartKeepOccupied
		if presence == PresenceAbsent {
			want = StartRollback
		}
		got, err := ReconcileStart(r, StartObservation{Owner: Dead, Helper: Dead, Registration: RegistrationAbsent, Session: presence})
		if err != nil || got.Action != want {
			t.Fatalf("presence=%v got=%+v err=%v", presence, got, err)
		}
	}
}

func TestReconcilePendingTerminalDoesNotReuseHistoricalRegistration(t *testing.T) {
	env := newTestEnv(t, "/repo")
	r := admittedStartRecord(t)
	r.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	nonce := "start-0123456789abcdef"
	old := couchidentity.SessionBinding{C: 1, M: 1, Name: "📁1-1", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: "old"}
	r.SessionBinding = &old
	r, _ = AdvanceStartTransaction(r, StartEvent{Kind: StartClaimed, Nonce: nonce, Owner: SupervisorOwner{PID: 41, Identity: "dead-owner"}})
	pending := old
	pending.M = 2
	pending.Name = "📁1-2"
	pending.StartNonce = nonce
	r, _ = AdvanceStartTransaction(r, StartEvent{Kind: StartSessionBound, Nonce: nonce, Binding: &pending})
	r, _ = AdvanceStartTransaction(r, StartEvent{Kind: StartHelperRecorded, Nonce: nonce, Helper: ProcessIdentity{PID: 42, Identity: "dead-helper"}})
	if _, err := env.Couch.Threads.CreateThread(r); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetRegistration(r.Address, RegistrationEstablished, nil)
	// The durable Pair address was registered by its previous launch; this new
	// terminal never existed. The old marker cannot promote the proposed M.
	if err := env.Couch.reconcileInterruptedStarts(); err != nil {
		t.Fatal(err)
	}
	got, err := env.Couch.Threads.GetThread(r.Address)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Incarnations) != 0 || got.SessionBinding.Name != old.Name {
		t.Fatalf("historical registration promoted new terminal: %+v", got)
	}
}
