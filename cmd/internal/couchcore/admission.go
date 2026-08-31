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

const admissionReconcileAttempts = 3

type PolicyUnstableError struct {
	Attempts     int
	RepoIdentity string
}

func (e *PolicyUnstableError) Error() string {
	return fmt.Sprintf("fleet policy unstable for %q after %d cohort attempts", e.RepoIdentity, e.Attempts)
}

// ReconcileAdmission resolves provider evidence without the store lock, then
// commits pruning and the candidate's creating incarnation only if the exact
// snapshot is still current. Refused pristine reservations are rolled back.
func ReconcileAdmission(ctx context.Context, store *ThreadStore, resolver PolicyResolver, candidateAddress ThreadAddress, startedAt time.Time) (ThreadRecord, error) {
	return reconcileAdmission(ctx, store, resolver, candidateAddress, startedAt, nil)
}

// ReconcileAdmissionPrepared admits with candidate policy evidence already
// accepted by start revalidation. Resolver remains available only for stale
// incumbents; the candidate authority is never read again.
func ReconcileAdmissionPrepared(ctx context.Context, store *ThreadStore, resolver PolicyResolver, candidateAddress ThreadAddress, startedAt time.Time, candidatePolicy PolicyResult) (ThreadRecord, error) {
	return reconcileAdmission(ctx, store, resolver, candidateAddress, startedAt, &candidatePolicy)
}

func reconcileAdmission(ctx context.Context, store *ThreadStore, resolver PolicyResolver, candidateAddress ThreadAddress, startedAt time.Time, prepared *PolicyResult) (ThreadRecord, error) {
	if store == nil || resolver == nil {
		return ThreadRecord{}, errors.New("reconcile admission: nil dependency")
	}
	rollback := func(primary error) (ThreadRecord, error) {
		return ThreadRecord{}, errors.Join(primary, store.DeletePristineThread(candidateAddress))
	}
	lastRepoIdentity := ""
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
		var candidatePolicy PolicyResult
		if prepared == nil {
			candidatePolicy, err = resolver.ResolvePolicy(ctx, candidate.WorkingPath)
			if err != nil {
				return rollback(err)
			}
		} else {
			candidatePolicy = *prepared
		}
		if err := ValidatePolicyResult(candidatePolicy); err != nil {
			return rollback(fmt.Errorf("candidate provider result: %w", err))
		}
		lastRepoIdentity = candidatePolicy.RepoIdentity

		occupants, replacements, epochChanged, err := reconcileAdmissionIncumbents(ctx, resolver, snapshot, candidate, candidatePolicy)
		if err != nil {
			return rollback(err)
		}
		if epochChanged {
			if prepared != nil {
				return rollback(&PolicyUnstableError{Attempts: attempt + 1, RepoIdentity: lastRepoIdentity})
			}
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
	return rollback(&PolicyUnstableError{Attempts: admissionReconcileAttempts, RepoIdentity: lastRepoIdentity})
}

// ReconcileResumeAdmission admits an already verified parked address without
// allocating or replacing it. Refusal leaves the parked record unchanged; a
// successful admission retains VerifiedPark until Pair registration succeeds.
type ResumeAdmissionInput struct {
	Address   ThreadAddress
	StartedAt time.Time
	Owner     SupervisorOwner
	Nonce     string
	Profile   LaunchProfile
}

func ReconcileResumeAdmission(ctx context.Context, store *ThreadStore, resolver PolicyResolver, input ResumeAdmissionInput) (ThreadRecord, error) {
	if store == nil || resolver == nil {
		return ThreadRecord{}, errors.New("reconcile resume admission: nil dependency")
	}
	if input.Nonce == "" || input.Profile.Agent == "" || input.Profile.Argv == nil {
		return ThreadRecord{}, errors.New("reconcile resume admission: incomplete start authority")
	}
	lastRepoIdentity := ""
	for attempt := 0; attempt < admissionReconcileAttempts; attempt++ {
		snapshot, err := store.Snapshot()
		if err != nil {
			return ThreadRecord{}, err
		}
		candidate, ok := snapshotThread(snapshot, input.Address)
		if !ok {
			return ThreadRecord{}, fmt.Errorf("resume candidate thread %+v is absent", input.Address)
		}
		if candidate.Reservation || candidate.Park != nil || candidate.VerifiedPark == nil || len(candidate.Incarnations) != 0 || candidate.LatestLaunchProfile == nil {
			return ThreadRecord{}, fmt.Errorf("resume candidate thread %+v is not verified parked", input.Address)
		}
		candidatePolicy, err := resolver.ResolvePolicy(ctx, candidate.WorkingPath)
		if err != nil {
			return ThreadRecord{}, err
		}
		if err := ValidatePolicyResult(candidatePolicy); err != nil {
			return ThreadRecord{}, fmt.Errorf("resume candidate provider result: %w", err)
		}
		lastRepoIdentity = candidatePolicy.RepoIdentity

		occupants, replacements, epochChanged, err := reconcileAdmissionIncumbents(ctx, resolver, snapshot, candidate, candidatePolicy)
		if err != nil {
			return ThreadRecord{}, err
		}
		if epochChanged {
			continue
		}
		if _, err := (Admission{}).Decide(candidatePolicy, occupants); err != nil {
			return ThreadRecord{}, err
		}
		candidate.Revision++
		policyCopy := candidatePolicy
		candidate.Incarnations = []ThreadIncarnation{{
			State: IncarnationCreating, StartedAt: input.StartedAt, Policy: &policyCopy,
		}}
		candidate, err = AdvanceStartTransaction(candidate, StartEvent{
			Kind: StartClaimed, Nonce: input.Nonce, Owner: input.Owner, Profile: &input.Profile,
		})
		if err != nil {
			return ThreadRecord{}, err
		}
		replacements = append(replacements, candidate)
		if err := store.CommitThreadReplacements(snapshot, replacements); err != nil {
			var conflict *ThreadSnapshotConflictError
			if errors.As(err, &conflict) {
				continue
			}
			return ThreadRecord{}, err
		}
		return cloneThreadRecord(candidate), nil
	}
	return ThreadRecord{}, &PolicyUnstableError{Attempts: admissionReconcileAttempts, RepoIdentity: lastRepoIdentity}
}

// reconcileAdmissionIncumbents builds the capacity cohort and refreshes stale
// incumbent policy evidence. Callers retain their distinct candidate and
// rollback semantics while sharing the policy-epoch algorithm.
func reconcileAdmissionIncumbents(ctx context.Context, resolver PolicyResolver, snapshot ThreadSnapshot, candidate ThreadRecord, candidatePolicy PolicyResult) ([]AdmissionOccupant, []ThreadRecord, bool, error) {
	occupants := []AdmissionOccupant{}
	replacements := []ThreadRecord{}
	for _, current := range snapshot.Records {
		if current.Address == candidate.Address {
			continue
		}
		next := cloneThreadRecord(current)
		count := len(next.Incarnations)
		changed := false
		if count == 0 && next.Reservation && next.ClaimGeneration < candidate.ClaimGeneration {
			count = 1
		}
		if count > 0 {
			policy, hasCurrent := coherentCurrentPolicy(next, candidatePolicy)
			if !hasCurrent {
				var err error
				policy, err = resolver.ResolvePolicy(ctx, next.WorkingPath)
				if err != nil {
					return nil, nil, false, err
				}
				if err := ValidatePolicyResult(policy); err != nil {
					return nil, nil, false, fmt.Errorf("incumbent provider result: %w", err)
				}
				if policy.RepoIdentity == candidatePolicy.RepoIdentity && (policy.PolicyVersion != candidatePolicy.PolicyVersion || policy.PolicyDigest != candidatePolicy.PolicyDigest) {
					return nil, nil, true, nil
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
	return occupants, replacements, false, nil
}

func snapshotThread(snapshot ThreadSnapshot, address ThreadAddress) (ThreadRecord, bool) {
	for _, record := range snapshot.Records {
		if record.Address == address {
			return cloneThreadRecord(record), true
		}
	}
	return ThreadRecord{}, false
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
