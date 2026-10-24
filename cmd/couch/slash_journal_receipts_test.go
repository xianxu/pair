package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

func refreshSlashJournalProofs(current []agentfixture.AcceptedInput, baselineSize int, proofs map[string]couchmessage.Receipt, hub, worker couchmessage.Binding, lookup func(string) (couchmessage.Receipt, error)) error {
	if baselineSize < 0 || baselineSize > len(current) || len(current) > agentfixture.MaxRequests || proofs == nil {
		return fmt.Errorf("invalid accepted journal or receipt store")
	}
	for _, entry := range current[baselineSize:] {
		id, candidate := systemEnvelopeID(entry.Body)
		if !candidate {
			continue // The pure oracle rejects every other non-command addition.
		}
		if proof, known := proofs[id]; known && proof.Status == couchmessage.Submitted {
			if _, err := slashSystemProofSubmitted(entry, proof, hub, worker); err != nil {
				return err
			}
			continue
		}
		proof, err := lookup(id)
		if err != nil {
			return fmt.Errorf("system receipt %q: %w", id, err)
		}
		if _, err := slashSystemProofSubmitted(entry, proof, hub, worker); err != nil {
			return err
		}
		proofs[id] = proof
	}
	return nil
}

func journalProofBindings837() (couchmessage.Binding, couchmessage.Binding) {
	hub := couchmessage.Binding{Slot: "fixture:0", Repository: "fixture.git", Scope: "hub-scope", Tag: "hub-tag", Session: "hub-session", Nonce: "hub-nonce", Agent: "claude", Version: "fixture", PID: 11, Start: "11.1"}
	worker := couchmessage.Binding{Slot: "fixture:1", Repository: "worker.git", Scope: "worker-scope", Tag: "worker-tag", Session: "worker-session", Nonce: "worker-nonce", Agent: "claude", Version: "fixture", PID: 12, Start: "12.1"}
	return hub, worker
}

func TestSlashJournalReceiptLookupsScaleWithDistinctDeliveries(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			hub, worker := journalProofBindings837()
			var rows []agentfixture.AcceptedInput
			server := map[string]couchmessage.Receipt{}
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("system-%d", i)
				m := couchmessage.Message{Origin: couchmessage.OriginSystem, ID: id, From: hub, To: worker, Body: fmt.Sprintf("system notice %d", i)}
				server[id] = couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}
				rows = append(rows, agentfixture.AcceptedInput{Body: couchmessage.Envelope(m)})
			}
			calls := 0
			lookup := func(id string) (couchmessage.Receipt, error) { calls++; return server[id], nil }
			proofs := map[string]couchmessage.Receipt{}
			if err := refreshSlashJournalProofs(rows, 0, proofs, hub, worker, lookup); err != nil {
				t.Fatal(err)
			}
			if calls != n || len(proofs) != n {
				t.Fatalf("first observation: lookups=%d proofs=%d want=%d", calls, len(proofs), n)
			}
			if err := refreshSlashJournalProofs(rows, 0, proofs, hub, worker, lookup); err != nil {
				t.Fatal(err)
			}
			if calls != n {
				t.Fatalf("stable submitted receipts repeated lookup: got=%d want=%d", calls, n)
			}
		})
	}
}

func TestSlashJournalReceiptRefreshObservesSubmissionTransition(t *testing.T) {
	hub, worker := journalProofBindings837()
	m := couchmessage.Message{Origin: couchmessage.OriginSystem, ID: "delayed", From: hub, To: worker, Body: "restart notice"}
	rows := []agentfixture.AcceptedInput{{Body: "/model"}, {Body: couchmessage.Envelope(m)}}
	proofs := map[string]couchmessage.Receipt{}
	status, calls := couchmessage.Delivering, 0
	lookup := func(string) (couchmessage.Receipt, error) {
		calls++
		return couchmessage.Receipt{Message: m, Status: status}, nil
	}
	oracle := newSlashJournalOracle(nil, hub, worker)
	if err := refreshSlashJournalProofs(rows, 0, proofs, hub, worker, lookup); err != nil {
		t.Fatal(err)
	}
	if ready, err := oracle.Observe(rows, proofs); err != nil || ready {
		t.Fatalf("in-flight ready=%t err=%v", ready, err)
	}
	status = couchmessage.Submitted
	if err := refreshSlashJournalProofs(rows, 0, proofs, hub, worker, lookup); err != nil {
		t.Fatal(err)
	}
	if ready, err := oracle.Observe(rows, proofs); err != nil || !ready {
		t.Fatalf("submitted ready=%t err=%v", ready, err)
	}
	if err := refreshSlashJournalProofs(rows, 0, proofs, hub, worker, lookup); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("lookups=%d want2 across pending, submitted, stable", calls)
	}
}

func TestSlashJournalReceiptRefreshRefusesUnprovedCacheEntries(t *testing.T) {
	hub, worker := journalProofBindings837()
	m := couchmessage.Message{Origin: couchmessage.OriginSystem, ID: "delayed", From: hub, To: worker, Body: "restart notice"}
	row := agentfixture.AcceptedInput{Body: couchmessage.Envelope(m)}
	for _, mutate := range []func(*couchmessage.Receipt){
		func(r *couchmessage.Receipt) { r.Message.Origin = couchmessage.OriginPeer },
		func(r *couchmessage.Receipt) { r.Message.From.Nonce = "replaced" },
		func(r *couchmessage.Receipt) { r.Message.To.Nonce = "replaced" },
		func(r *couchmessage.Receipt) { r.Message.Body += "!" },
		func(r *couchmessage.Receipt) { r.Message.ID = "other" },
		func(r *couchmessage.Receipt) { r.Status = couchmessage.Indeterminate },
	} {
		proof := couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}
		mutate(&proof)
		proofs := map[string]couchmessage.Receipt{}
		if err := refreshSlashJournalProofs([]agentfixture.AcceptedInput{row}, 0, proofs, hub, worker, func(string) (couchmessage.Receipt, error) { return proof, nil }); err == nil || len(proofs) != 0 {
			t.Fatalf("unproved receipt cached: err=%v proofs=%+v", err, proofs)
		}
	}
	lookupErr := errors.New("receipt unavailable")
	if err := refreshSlashJournalProofs([]agentfixture.AcceptedInput{row}, 0, map[string]couchmessage.Receipt{}, hub, worker, func(string) (couchmessage.Receipt, error) { return couchmessage.Receipt{}, lookupErr }); !errors.Is(err, lookupErr) {
		t.Fatalf("lookup failure lost: %v", err)
	}
}
