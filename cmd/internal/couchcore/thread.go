package couchcore

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

const ThreadSchemaVersion = 1

type ThreadTag string

type ThreadAddress struct {
	RepoScope string    `json:"repo_scope"`
	Tag       ThreadTag `json:"tag"`
}

type IncarnationState string

const (
	IncarnationUnknown  IncarnationState = "unknown"
	IncarnationCreating IncarnationState = "creating"
	IncarnationLive     IncarnationState = "live"
)

// ThreadStartClaim is the durable recovery anchor for one creating
// incarnation. The supervisor identity proves which couch process initiated
// the transaction; ThreadIncarnation PID/Identity is empty before fork and
// names the blocked helper after fork. Exec preserves that PID/start token.
type ThreadStartClaim struct {
	Nonce         string `json:"nonce"`
	OwnerPID      int    `json:"owner_pid"`
	OwnerIdentity string `json:"owner_identity"`
}

type ThreadIncarnation struct {
	LegacyActorID ActorID           `json:"legacy_actor_id,omitempty"`
	PID           int               `json:"pid,omitempty"`
	Identity      string            `json:"identity,omitempty"`
	State         IncarnationState  `json:"state"`
	StartedAt     time.Time         `json:"started_at,omitempty"`
	Policy        *PolicyResult     `json:"policy,omitempty"`
	Start         *ThreadStartClaim `json:"start,omitempty"`
}

type ThreadRecord struct {
	SchemaVersion   int                 `json:"schema_version"`
	Address         ThreadAddress       `json:"address"`
	StartingPath    string              `json:"starting_path"`
	WorkingPath     string              `json:"working_path"`
	CreatedAt       time.Time           `json:"created_at"`
	Revision        uint64              `json:"revision"`
	ClaimGeneration uint64              `json:"claim_generation"`
	Reservation     bool                `json:"reservation,omitempty"`
	Description     string              `json:"description,omitempty"`
	Incarnations    []ThreadIncarnation `json:"incarnations,omitempty"`
}

var threadComponentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ValidateThreadRecord(record ThreadRecord) error {
	if record.SchemaVersion != ThreadSchemaVersion {
		return fmt.Errorf("unsupported thread schema version %d", record.SchemaVersion)
	}
	if err := validateThreadAddress(record.Address); err != nil {
		return err
	}
	if !filepath.IsAbs(record.StartingPath) || !filepath.IsAbs(record.WorkingPath) {
		return fmt.Errorf("thread paths must be absolute")
	}
	if record.CreatedAt.IsZero() {
		return fmt.Errorf("thread creation time is required")
	}
	if record.Revision == 0 {
		return fmt.Errorf("thread revision must be positive")
	}
	trackedStarts := 0
	for i, incarnation := range record.Incarnations {
		switch incarnation.State {
		case IncarnationUnknown, IncarnationCreating, IncarnationLive:
		default:
			return fmt.Errorf("incarnation %d has invalid state %q", i, incarnation.State)
		}
		if incarnation.PID < 0 {
			return fmt.Errorf("incarnation %d has negative pid", i)
		}
		if incarnation.Start != nil {
			trackedStarts++
			if incarnation.State != IncarnationCreating {
				return fmt.Errorf("incarnation %d has start claim outside creating state", i)
			}
			if !threadComponentPattern.MatchString(incarnation.Start.Nonce) {
				return fmt.Errorf("incarnation %d has invalid start nonce %q", i, incarnation.Start.Nonce)
			}
			if incarnation.Start.OwnerPID <= 0 || incarnation.Start.OwnerIdentity == "" {
				return fmt.Errorf("incarnation %d start owner pid and identity are required", i)
			}
			if (incarnation.PID == 0) != (incarnation.Identity == "") {
				return fmt.Errorf("incarnation %d helper pid and identity must be recorded together", i)
			}
		}
	}
	if trackedStarts > 1 {
		return fmt.Errorf("thread has %d tracked start transactions", trackedStarts)
	}
	return nil
}

func validateThreadAddress(address ThreadAddress) error {
	if err := launcher.ValidateRepoScopeKey(address.RepoScope); err != nil {
		return fmt.Errorf("invalid thread repo scope %q: %w", address.RepoScope, err)
	}
	if err := launcher.ValidatePairTag(string(address.Tag)); err != nil {
		return fmt.Errorf("invalid thread tag %q: %w", address.Tag, err)
	}
	return nil
}

func cloneThreadRecord(record ThreadRecord) ThreadRecord {
	copy := record
	copy.Incarnations = append([]ThreadIncarnation{}, record.Incarnations...)
	for i := range copy.Incarnations {
		if record.Incarnations[i].Policy != nil {
			policy := *record.Incarnations[i].Policy
			copy.Incarnations[i].Policy = &policy
		}
		if record.Incarnations[i].Start != nil {
			start := *record.Incarnations[i].Start
			copy.Incarnations[i].Start = &start
		}
	}
	return copy
}
