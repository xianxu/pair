package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Admission struct{}

type AdmissionOccupant struct {
	Address ThreadAddress
	Policy  PolicyResult
}

type AdmissionDecision struct {
	Admitted bool
}

type CapacityExceededError struct {
	RepoIdentity    string
	AdmissionKey    string
	Limit           int
	Action          CapacityAction
	Incumbents      []ThreadAddress
	ProvisionedPath string
}

func (e *CapacityExceededError) Error() string {
	return fmt.Sprintf("admission key %q is at capacity %d (%s)", e.AdmissionKey, e.Limit, e.Action)
}

func (Admission) Decide(candidate PolicyResult, occupants []AdmissionOccupant) (AdmissionDecision, error) {
	if err := ValidatePolicyResult(candidate); err != nil {
		return AdmissionDecision{}, fmt.Errorf("candidate policy: %w", err)
	}
	if candidate.Capacity.Kind == CapacityUnbounded {
		return AdmissionDecision{Admitted: true}, nil
	}
	matching := make([]ThreadAddress, 0, len(occupants))
	for _, occupant := range occupants {
		if err := ValidatePolicyResult(occupant.Policy); err != nil {
			return AdmissionDecision{}, fmt.Errorf("occupant %+v policy: %w", occupant.Address, err)
		}
		if occupant.Policy.RepoIdentity == candidate.RepoIdentity && occupant.Policy.AdmissionKey == candidate.AdmissionKey {
			matching = append(matching, occupant.Address)
		}
	}
	if len(matching) >= candidate.Capacity.Limit {
		return AdmissionDecision{}, &CapacityExceededError{
			RepoIdentity: candidate.RepoIdentity,
			AdmissionKey: candidate.AdmissionKey,
			Limit:        candidate.Capacity.Limit,
			Action:       candidate.OnCapacity,
			Incumbents:   append([]ThreadAddress{}, matching...),
		}
	}
	return AdmissionDecision{Admitted: true}, nil
}

const admissionReconcileAttempts = 4

var errPolicyEpochChanged = errors.New("fleet policy epoch changed during admission")

// ReconcileAdmission resolves provider evidence without the store lock, then
// commits pruning and the candidate's creating incarnation only if the exact
// snapshot is still current. Refused pristine reservations are rolled back.
func ReconcileAdmission(ctx context.Context, store *ThreadStore, resolver PolicyResolver, proc ProcOps, candidateAddress ThreadAddress, startedAt time.Time) (ThreadRecord, error) {
	if store == nil || resolver == nil || proc == nil {
		return ThreadRecord{}, errors.New("reconcile admission: nil dependency")
	}
	rollback := func(primary error) (ThreadRecord, error) {
		return ThreadRecord{}, errors.Join(primary, store.DeletePristineThread(candidateAddress))
	}
	for attempt := 0; attempt < admissionReconcileAttempts; attempt++ {
		snapshot, err := store.Snapshot()
		if err != nil {
			return rollback(err)
		}
		candidate, ok := snapshotThread(snapshot, candidateAddress)
		if !ok {
			return ThreadRecord{}, fmt.Errorf("candidate thread %+v is absent", candidateAddress)
		}
		if !candidate.Reservation || len(candidate.Incarnations) != 0 {
			return ThreadRecord{}, fmt.Errorf("candidate thread %+v is not a pristine reservation", candidateAddress)
		}
		candidatePolicy, err := resolver.ResolvePolicy(ctx, candidate.WorkingPath)
		if err != nil {
			return rollback(err)
		}
		if err := ValidatePolicyResult(candidatePolicy); err != nil {
			return rollback(fmt.Errorf("candidate provider result: %w", err))
		}

		occupants := []AdmissionOccupant{}
		replacements := []ThreadRecord{}
		epochChanged := false
		for _, current := range snapshot.Records {
			if current.Address == candidateAddress {
				continue
			}
			next, count, changed := pruneDeadIncarnations(current, proc)
			if count == 0 && next.Reservation && next.ClaimGeneration < candidate.ClaimGeneration {
				count = 1
			}
			if count > 0 {
				policy, hasCurrent := coherentCurrentPolicy(next, candidatePolicy)
				if !hasCurrent {
					policy, err = resolver.ResolvePolicy(ctx, next.WorkingPath)
					if err != nil {
						return rollback(err)
					}
					if err := ValidatePolicyResult(policy); err != nil {
						return rollback(fmt.Errorf("incumbent provider result: %w", err))
					}
					if policy.RepoIdentity == candidatePolicy.RepoIdentity && (policy.PolicyVersion != candidatePolicy.PolicyVersion || policy.PolicyDigest != candidatePolicy.PolicyDigest) {
						epochChanged = true
						break
					}
					if len(next.Incarnations) > 0 {
						for i := range next.Incarnations {
							copy := policy
							next.Incarnations[i].Policy = &copy
						}
						changed = true
					}
				}
				for i := 0; i < count; i++ {
					occupants = append(occupants, AdmissionOccupant{Address: next.Address, Policy: policy})
				}
			}
			if changed {
				next.Revision++
				replacements = append(replacements, next)
			}
		}
		if epochChanged {
			continue
		}
		if _, err := (Admission{}).Decide(candidatePolicy, occupants); err != nil {
			return rollback(err)
		}
		candidate.Reservation = false
		candidate.Revision++
		policyCopy := candidatePolicy
		candidate.Incarnations = append(candidate.Incarnations, ThreadIncarnation{
			State:     IncarnationCreating,
			StartedAt: startedAt,
			Policy:    &policyCopy,
		})
		replacements = append(replacements, candidate)
		if err := store.CommitThreadReplacements(snapshot, replacements); err != nil {
			var conflict *ThreadSnapshotConflictError
			if errors.As(err, &conflict) {
				continue
			}
			return rollback(err)
		}
		return cloneThreadRecord(candidate), nil
	}
	return rollback(fmt.Errorf("reconcile admission: exhausted %d retries", admissionReconcileAttempts))
}

func snapshotThread(snapshot ThreadSnapshot, address ThreadAddress) (ThreadRecord, bool) {
	for _, record := range snapshot.Records {
		if record.Address == address {
			return cloneThreadRecord(record), true
		}
	}
	return ThreadRecord{}, false
}

func pruneDeadIncarnations(record ThreadRecord, proc ProcOps) (ThreadRecord, int, bool) {
	next := cloneThreadRecord(record)
	next.Incarnations = next.Incarnations[:0]
	changed := false
	for _, incarnation := range record.Incarnations {
		if incarnationProvenDead(incarnation, proc) {
			changed = true
			continue
		}
		next.Incarnations = append(next.Incarnations, incarnation)
	}
	return next, len(next.Incarnations), changed
}

func incarnationProvenDead(incarnation ThreadIncarnation, proc ProcOps) bool {
	if incarnation.State == IncarnationCreating || incarnation.PID <= 0 || incarnation.Identity == "" {
		return false
	}
	switch proc.Exists(incarnation.PID) {
	case Dead:
		return true
	case Unknown:
		return false
	}
	identity, err := proc.Identity(incarnation.PID)
	return err == nil && identity != incarnation.Identity
}

func coherentCurrentPolicy(record ThreadRecord, candidate PolicyResult) (PolicyResult, bool) {
	if len(record.Incarnations) == 0 {
		return PolicyResult{}, false
	}
	var result PolicyResult
	for i, incarnation := range record.Incarnations {
		if incarnation.Policy == nil || ValidatePolicyResult(*incarnation.Policy) != nil {
			return PolicyResult{}, false
		}
		policy := *incarnation.Policy
		if policy.RepoIdentity == candidate.RepoIdentity && (policy.PolicyVersion != candidate.PolicyVersion || policy.PolicyDigest != candidate.PolicyDigest) {
			return PolicyResult{}, false
		}
		if i == 0 {
			result = policy
			continue
		}
		if policy != result {
			return PolicyResult{}, false
		}
	}
	return result, true
}
