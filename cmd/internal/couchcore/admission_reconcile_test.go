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
	record, err := store.AllocateThreadTag(scope, ns.Dir(), created, bytes.NewReader(bytes.Repeat([]byte{suffix}, 8)), NoThreadArtifactCollisions{})
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
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, candidate.Address, time.Now())
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
	_, err = ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
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

func TestAdmissionReconcileRetainsDeadClientWithoutWholeIncarnationProof(t *testing.T) {
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

	_, err = ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
	var full *CapacityExceededError
	if !errors.As(err, &full) {
		t.Fatalf("dead client err = %T %v, want occupied capacity", err, err)
	}
	kept, err := store.GetThread(incumbent.Address)
	if err != nil || len(kept.Incarnations) != 1 {
		t.Fatalf("dead client lost whole-incarnation occupancy: %+v, %v", kept, err)
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

	if _, err := ReconcileAdmission(context.Background(), store, resolver, second.Address, now); err == nil {
		t.Fatal("later reservation bypassed earlier bounded claim")
	}
	if _, err := ReconcileAdmission(context.Background(), store, resolver, first.Address, now); err != nil {
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
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
	if err != nil {
		t.Fatalf("ReconcileAdmission: %v", err)
	}
	if claimed.Incarnations[0].Policy.PolicyDigest != epochB.PolicyDigest {
		t.Fatalf("candidate retained mixed epoch: %+v", claimed.Incarnations[0].Policy)
	}
}

func TestAdmissionReconcileReturnsTypedPolicyUnstableAfterThreeCohorts(t *testing.T) {
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
	for range 3 {
		resolver.Queue(candidate.WorkingPath, epochA, nil)
		resolver.Queue(incumbent.WorkingPath, epochB, nil)
	}
	_, err = ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
	var unstable *PolicyUnstableError
	if !errors.As(err, &unstable) || unstable.Attempts != 3 {
		t.Fatalf("err = %T %v, want three-attempt *PolicyUnstableError", err, err)
	}
	if got := len(resolver.Calls()); got != 6 {
		t.Fatalf("provider calls = %d, want 6 for three whole cohorts", got)
	}
	if _, err := store.GetThread(candidate.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("unstable candidate reservation leaked: %v", err)
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
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
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
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
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

func TestAdvanceStartRequiresExactClaimRevision(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Now()
	candidate := allocateAdmissionCandidate(t, store, ns, "0123456789abcdef", 1, now)
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	resolver := NewFakePolicyResolver()
	resolver.Queue(candidate.WorkingPath, policy, nil)
	claimed, err := ReconcileAdmission(context.Background(), store, resolver, candidate.Address, now)
	if err != nil {
		t.Fatal(err)
	}
	started, err := store.AdvanceStart(claimed.Address, claimed.Revision, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-start"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceStart(claimed.Address, claimed.Revision, StartEvent{
		Kind: StartClaimed, Nonce: "start-other", Owner: SupervisorOwner{PID: 43, Identity: "other"},
	}); err == nil {
		t.Fatal("stale start claim succeeded")
	}
	helper, err := store.AdvanceStart(claimed.Address, started.Revision, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "process-start"},
	})
	if err != nil {
		t.Fatal(err)
	}
	live, err := store.AdvanceStart(claimed.Address, helper.Revision, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
	if err != nil || live.Incarnations[0].State != IncarnationLive {
		t.Fatalf("registered thread = %+v, %v", live, err)
	}
}

type PolicyResolverFunc func(context.Context, string) (PolicyResult, error)

func (f PolicyResolverFunc) ResolvePolicy(ctx context.Context, path string) (PolicyResult, error) {
	return f(ctx, path)
}
