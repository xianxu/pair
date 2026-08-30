package threadrecord

import (
	"fmt"
	"time"
)

type ParkIdentity struct {
	Nonce           string  `json:"nonce"`
	Address         Address `json:"address"`
	PID             int     `json:"pid"`
	ProcessIdentity string  `json:"process_identity"`
}

type ParkFailure struct {
	Code       string `json:"code"`
	Diagnostic string `json:"diagnostic"`
}

type ParkAttempt struct {
	Number   uint64       `json:"number"`
	Failure  *ParkFailure `json:"failure,omitempty"`
	TimedOut bool         `json:"timed_out,omitempty"`
	Closed   bool         `json:"closed,omitempty"`
}

type ParkTransaction struct {
	Identity          ParkIdentity  `json:"identity"`
	BaseRevision      uint64        `json:"base_revision"`
	RecordRevision    uint64        `json:"record_revision"`
	Phase             string        `json:"phase"`
	Attempts          []ParkAttempt `json:"attempts"`
	Closed            bool          `json:"closed,omitempty"`
	Tombstoned        bool          `json:"tombstoned,omitempty"`
	SuccessfulAttempt uint64        `json:"successful_attempt,omitempty"`
}

type VerifiedPark struct {
	Identity ParkIdentity `json:"identity"`
	Attempt  uint64       `json:"attempt"`
	ParkedAt time.Time    `json:"parked_at"`
}

func validateLifecycle(record Record) error {
	if record.LatestLaunchProfile != nil {
		if err := validateLaunchProfile(*record.LatestLaunchProfile); err != nil {
			return fmt.Errorf("latest launch profile: %w", err)
		}
	}
	if record.Park != nil && record.VerifiedPark != nil {
		return fmt.Errorf("active and verified park cannot coexist")
	}

	nonces := make(map[string]struct{}, len(record.ParkHistory)+1)
	for i := range record.ParkHistory {
		transaction := &record.ParkHistory[i]
		if err := validateParkTransaction(*transaction, record.Address); err != nil {
			return fmt.Errorf("park history %d: %w", i, err)
		}
		if !transaction.Closed {
			return fmt.Errorf("park history %d is not closed", i)
		}
		if transaction.Tombstoned == (transaction.SuccessfulAttempt != 0) {
			return fmt.Errorf("park history %d must be exactly one of tombstoned or successful", i)
		}
		if _, exists := nonces[transaction.Identity.Nonce]; exists {
			return fmt.Errorf("duplicate park nonce %q", transaction.Identity.Nonce)
		}
		nonces[transaction.Identity.Nonce] = struct{}{}
	}

	if record.Park != nil {
		if err := validateParkTransaction(*record.Park, record.Address); err != nil {
			return fmt.Errorf("active park: %w", err)
		}
		if record.Park.Closed || record.Park.Tombstoned || record.Park.SuccessfulAttempt != 0 {
			return fmt.Errorf("active park cannot be closed, tombstoned, or successful")
		}
		if _, exists := nonces[record.Park.Identity.Nonce]; exists {
			return fmt.Errorf("active park nonce %q is already historical", record.Park.Identity.Nonce)
		}
		matches := 0
		for _, incarnation := range record.Incarnations {
			if incarnation.PID == record.Park.Identity.PID && incarnation.Identity == record.Park.Identity.ProcessIdentity {
				matches++
			}
		}
		replacementUnknown := matches == 0 && record.Park.Phase == "unknown" && transactionHasFailure(*record.Park, "replacement_incarnation")
		if matches != 1 && !replacementUnknown {
			return fmt.Errorf("active park identity matches %d incarnations", matches)
		}
	}

	if record.VerifiedPark != nil {
		if err := validateParkIdentity(record.VerifiedPark.Identity, record.Address); err != nil {
			return fmt.Errorf("verified park: %w", err)
		}
		if record.VerifiedPark.Attempt == 0 || record.VerifiedPark.ParkedAt.IsZero() {
			return fmt.Errorf("verified park attempt and parked_at are required")
		}
		if len(record.Incarnations) != 0 {
			return fmt.Errorf("verified parked thread still has an incarnation")
		}
		if record.LastActiveAt.IsZero() || record.LastActiveAt.Before(record.VerifiedPark.ParkedAt) {
			return fmt.Errorf("verified park requires monotonic last_active_at")
		}
		matched := false
		for _, transaction := range record.ParkHistory {
			if transaction.Identity == record.VerifiedPark.Identity &&
				transaction.SuccessfulAttempt == record.VerifiedPark.Attempt &&
				transaction.Closed && !transaction.Tombstoned {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("verified park has no matching closed success history")
		}
	}
	return nil
}

func validateParkTransaction(transaction ParkTransaction, address Address) error {
	if err := validateParkIdentity(transaction.Identity, address); err != nil {
		return err
	}
	if transaction.BaseRevision == 0 || transaction.RecordRevision <= transaction.BaseRevision {
		return fmt.Errorf("park base and record revisions are invalid")
	}
	switch transaction.Phase {
	case "requested", "awaiting_completion", "unknown":
	default:
		return fmt.Errorf("park phase %q is invalid", transaction.Phase)
	}
	if len(transaction.Attempts) == 0 || transaction.Attempts[0].Number != 1 {
		return fmt.Errorf("park attempts must begin at one")
	}
	for i := range transaction.Attempts {
		attempt := transaction.Attempts[i]
		if i > 0 && attempt.Number <= transaction.Attempts[i-1].Number {
			return fmt.Errorf("park attempts must be strictly increasing")
		}
		if attempt.Failure != nil {
			if !validFailureCode(attempt.Failure.Code) || attempt.Failure.Diagnostic == "" {
				return fmt.Errorf("park attempt %d has invalid failure", attempt.Number)
			}
		}
		if attempt.TimedOut && (attempt.Failure == nil || attempt.Failure.Code != "timeout") {
			return fmt.Errorf("park attempt %d timeout marker lacks timeout failure", attempt.Number)
		}
	}
	if transaction.Tombstoned && !transaction.Closed {
		return fmt.Errorf("park tombstone must be closed")
	}
	if transaction.SuccessfulAttempt != 0 {
		found := false
		for _, attempt := range transaction.Attempts {
			if attempt.Number == transaction.SuccessfulAttempt && attempt.Closed {
				found = true
			}
		}
		if !transaction.Closed || !found {
			return fmt.Errorf("park success must name a closed attempt on a closed transaction")
		}
	}
	return nil
}

func validateParkIdentity(identity ParkIdentity, address Address) error {
	if !componentPattern.MatchString(identity.Nonce) {
		return fmt.Errorf("park nonce %q is invalid", identity.Nonce)
	}
	if identity.Address != address {
		return fmt.Errorf("park identity address does not match thread")
	}
	if identity.PID <= 0 || identity.ProcessIdentity == "" {
		return fmt.Errorf("park identity pid and process identity are required")
	}
	return nil
}

func validFailureCode(code string) bool {
	switch code {
	case "request_publish_failed", "cleanup_failed", "timeout", "completion_missing",
		"stale_completion", "revision_conflict", "replacement_incarnation":
		return true
	default:
		return false
	}
}

func transactionHasFailure(transaction ParkTransaction, code string) bool {
	for _, attempt := range transaction.Attempts {
		if attempt.Failure != nil && attempt.Failure.Code == code {
			return true
		}
	}
	return false
}

func validateLaunchProfile(profile LaunchProfile) error {
	if profile.Agent == "" || profile.Argv == nil {
		return fmt.Errorf("agent and argv are required")
	}
	return nil
}
