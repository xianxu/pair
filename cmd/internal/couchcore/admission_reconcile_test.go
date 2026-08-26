package couchcore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func allocateAdmissionCandidate(t *testing.T, store *ThreadStore, ns CouchNamespace, scope string, suffix byte, created time.Time) ThreadRecord {
	t.Helper()
	record, err := store.AllocateThreadTag(scope, ns.Dir(), created, bytes.NewReader(bytes.Repeat([]byte{suffix}, 8)))
	if err != nil {
		t.Fatalf("AllocateThreadTag: %v", err)
	}
	return record
}

func TestAdmissionReconcileResolvesOutsideStoreLockAndCommitsCreatingOccupant(t *testing.T) {
	store, ns := newTestThreadStore(t)
	candidate := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, time.Now())
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	resolver := PolicyResolverFunc(func(_ context.Context, path string) (PolicyResult, error) {
		if path != candidate.WorkingPath {
			t.Fatalf("resolved path = %q", path)
		}
		// This would deadlock if ReconcileAdmission held the store lock across
		// provider IO.
		if _, err := NewThreadStore(ns).ManifestGeneration(); err != nil {
			t.Fatalf("store access from resolver: %v", err)
		}
		return policy, nil
	})
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), candidate.Address, time.Now())
	if err != nil {
		t.Fatalf("ReconcileAdmission: %v", err)
	}
	if len(claimed.Incarnations) != 1 || claimed.Incarnations[0].State != IncarnationCreating || claimed.Incarnations[0].Policy == nil {
		t.Fatalf("claimed thread = %+v", claimed)
	}
}

func TestAdmissionReconcileCountsUnknownAndRollsBackRefusedCandidate(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	incumbent := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	incumbent, err := store.UpdateExistingThread(incumbent.Address, incumbent.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "unknown", State: IncarnationUnknown, StartedAt: now}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	candidate := allocateAdmissionCandidate(t, store, ns, incumbent.Address.RepoScope, 2, now.Add(time.Second))
	policy := admissionPolicy(CapacityBounded, 1, CapacityReject)
	resolver := NewFakePolicyResolver()
	resolver.Queue(candidate.WorkingPath, policy, nil)
	resolver.Queue(incumbent.WorkingPath, policy, nil)
	proc := NewFakeProcOps()
	proc.SetUnknown(42)

	_, err = ReconcileAdmission(context.Background(), store, resolver, proc, candidate.Address, now)
	var full *CapacityExceededError
	if !errors.As(err, &full) {
		t.Fatalf("err = %T %v, want capacity refusal", err, err)
	}
	if _, err := store.GetThread(candidate.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("refused pristine candidate was not rolled back: %v", err)
	}
	kept, err := store.GetThread(incumbent.Address)
	if err != nil || len(kept.Incarnations) != 1 {
		t.Fatalf("unknown incumbent was pruned: %+v, %v", kept, err)
	}
}

func TestAdmissionReconcilePrunesOnlyProvenDeadIncarnation(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	incumbent := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	incumbent, err := store.UpdateExistingThread(incumbent.Address, incumbent.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "dead", State: IncarnationLive, StartedAt: now}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	candidate := allocateAdmissionCandidate(t, store, ns, incumbent.Address.RepoScope, 2, now.Add(time.Second))
	policy := admissionPolicy(CapacityBounded, 1, CapacityReject)
	resolver := NewFakePolicyResolver()
	resolver.Queue(candidate.WorkingPath, policy, nil)
	resolver.Queue(incumbent.WorkingPath, policy, nil)

	claimed, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), candidate.Address, now)
	if err != nil {
		t.Fatalf("ReconcileAdmission: %v", err)
	}
	if len(claimed.Incarnations) != 1 || claimed.Incarnations[0].State != IncarnationCreating {
		t.Fatalf("candidate = %+v", claimed)
	}
	pruned, err := store.GetThread(incumbent.Address)
	if err != nil || len(pruned.Incarnations) != 0 {
		t.Fatalf("proven-dead incumbent = %+v, %v", pruned, err)
	}
}

func TestAdmissionReconcileEarlierReservationWinsBoundedRace(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	first := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	second := allocateAdmissionCandidate(t, store, ns, first.Address.RepoScope, 2, now.Add(time.Second))
	policy := admissionPolicy(CapacityBounded, 1, CapacityReject)
	resolver := NewFakePolicyResolver()
	resolver.Queue(first.WorkingPath, policy, nil)
	resolver.Queue(second.WorkingPath, policy, nil)
	resolver.Queue(first.WorkingPath, policy, nil)

	if _, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), second.Address, now); err == nil {
		t.Fatal("later reservation bypassed earlier bounded claim")
	}
	if _, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), first.Address, now); err != nil {
		t.Fatalf("earlier reservation did not win: %v", err)
	}
}

func TestAdmissionReconcileRetriesWholeCohortAcrossPolicyEpochChange(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	incumbent := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	incumbent, err := store.UpdateExistingThread(incumbent.Address, incumbent.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{State: IncarnationCreating, StartedAt: now}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	candidate := allocateAdmissionCandidate(t, store, ns, incumbent.Address.RepoScope, 2, now.Add(time.Second))
	epochA := admissionPolicy(CapacityBounded, 2, CapacityReject)
	epochB := epochA
	epochB.PolicyDigest = strings.Repeat("b", 64)
	resolver := NewFakePolicyResolver()
	// Attempt one sees a mixed cohort. Attempt two resolves every member at B.
	resolver.Queue(candidate.WorkingPath, epochA, nil)
	resolver.Queue(incumbent.WorkingPath, epochB, nil)
	resolver.Queue(candidate.WorkingPath, epochB, nil)
	resolver.Queue(incumbent.WorkingPath, epochB, nil)
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), candidate.Address, now)
	if err != nil {
		t.Fatalf("ReconcileAdmission: %v", err)
	}
	if claimed.Incarnations[0].Policy.PolicyDigest != epochB.PolicyDigest {
		t.Fatalf("candidate retained mixed epoch: %+v", claimed.Incarnations[0].Policy)
	}
}

func TestAdmissionReconcileRetriesStaleSnapshotWithoutLosingConcurrentMetadata(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	candidate := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	calls := 0
	resolver := PolicyResolverFunc(func(_ context.Context, _ string) (PolicyResult, error) {
		calls++
		if calls == 1 {
			current, err := store.GetThread(candidate.Address)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.UpdateExistingThread(candidate.Address, current.Revision, func(next *ThreadRecord) error {
				next.Description = "concurrent metadata"
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return policy, nil
	})
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), candidate.Address, now)
	if err != nil {
		t.Fatalf("ReconcileAdmission: %v", err)
	}
	if calls != 2 || claimed.Description != "concurrent metadata" {
		t.Fatalf("calls=%d claimed=%+v", calls, claimed)
	}
}

func TestDeleteUnstartedThreadRefusesAfterConcurrentMetadata(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	candidate := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	resolver := NewFakePolicyResolver()
	resolver.Queue(candidate.WorkingPath, policy, nil)
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), candidate.Address, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateExistingThread(claimed.Address, claimed.Revision, func(next *ThreadRecord) error {
		next.Description = "operator named this concurrently"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteUnstartedThread(claimed.Address, claimed.Revision); err == nil {
		t.Fatal("stale rollback deleted or accepted a concurrently edited claim")
	}
	if _, err := store.GetThread(claimed.Address); err != nil {
		t.Fatalf("concurrently edited claim was deleted: %v", err)
	}
}

func TestActivateCreatingThreadRequiresExactClaimRevision(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	candidate := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	resolver := NewFakePolicyResolver()
	resolver.Queue(candidate.WorkingPath, policy, nil)
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, NewFakeProcOps(), candidate.Address, now)
	if err != nil {
		t.Fatal(err)
	}
	live, err := store.ActivateCreatingThread(claimed.Address, claimed.Revision, 42, "process-start")
	if err != nil {
		t.Fatal(err)
	}
	if live.Incarnations[0].State != IncarnationLive || live.Incarnations[0].PID != 42 || live.Incarnations[0].Identity != "process-start" {
		t.Fatalf("activated thread = %+v", live)
	}
	if _, err := store.ActivateCreatingThread(claimed.Address, claimed.Revision, 43, "other"); err == nil {
		t.Fatal("stale activation succeeded")
	}
}

type PolicyResolverFunc func(context.Context, string) (PolicyResult, error)

func (f PolicyResolverFunc) ResolvePolicy(ctx context.Context, path string) (PolicyResult, error) {
	return f(ctx, path)
}
