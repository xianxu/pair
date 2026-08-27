package couchcore

import (
	"errors"
	"testing"
)

func admissionPolicy(kind PolicyCapacityKind, limit int, action CapacityAction) PolicyResult {
	return PolicyResult{
		PolicyVersion: 1,
		PolicyDigest:  testPolicyDigest,
		RepoIdentity:  "/repo/.git",
		AdmissionKey:  "/repo",
		Capacity:      PolicyCapacity{Kind: kind, Limit: limit},
		OnCapacity:    action,
	}
}

func TestAdmissionDecideCountsOnlyMatchingNormalizedKey(t *testing.T) {
	policy := admissionPolicy(CapacityBounded, 1, CapacityReject)
	otherKey := policy
	otherKey.AdmissionKey = "/repo/other"
	decision, err := (Admission{}).Decide(policy, []AdmissionOccupant{
		{Address: ThreadAddress{RepoScope: "a", Tag: "other"}, Policy: otherKey},
	})
	if err != nil || !decision.Admitted {
		t.Fatalf("different key decision = %+v, %v", decision, err)
	}

	_, err = (Admission{}).Decide(policy, []AdmissionOccupant{
		{Address: ThreadAddress{RepoScope: "a", Tag: "incumbent"}, Policy: policy},
	})
	var full *CapacityExceededError
	if !errors.As(err, &full) {
		t.Fatalf("matching key err = %v, want *CapacityExceededError", err)
	}
	if full.Action != CapacityReject || full.Limit != 1 || len(full.Incumbents) != 1 {
		t.Fatalf("capacity refusal = %+v", full)
	}
}

func TestAdmissionDecideReturnsProvisionWorktreeWithoutInventingPath(t *testing.T) {
	policy := admissionPolicy(CapacityBounded, 1, CapacityProvisionWorktree)
	_, err := (Admission{}).Decide(policy, []AdmissionOccupant{{Address: ThreadAddress{RepoScope: "a", Tag: "incumbent"}, Policy: policy}})
	var full *CapacityExceededError
	if !errors.As(err, &full) || full.Action != CapacityProvisionWorktree {
		t.Fatalf("err = %T %v", err, err)
	}
	if full.ProvisionedPath != "" {
		t.Fatalf("#149 fabricated provisioned path %q", full.ProvisionedPath)
	}
}

func TestAdmissionDecideAllowsUnboundedRegardlessOfOccupancy(t *testing.T) {
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	occupants := make([]AdmissionOccupant, 20)
	for i := range occupants {
		occupants[i] = AdmissionOccupant{Address: ThreadAddress{RepoScope: "a", Tag: ThreadTag(string(rune('a' + i)))}, Policy: policy}
	}
	decision, err := (Admission{}).Decide(policy, occupants)
	if err != nil || !decision.Admitted {
		t.Fatalf("unbounded decision = %+v, %v", decision, err)
	}
}

func TestAdmissionDecideRejectsInvalidProviderValues(t *testing.T) {
	invalid := admissionPolicy(CapacityUnknown, 0, CapacityActionUnknown)
	if _, err := (Admission{}).Decide(invalid, nil); err == nil {
		t.Fatal("invalid zero capacity authorized admission")
	}
}
