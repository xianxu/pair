// Package threadrecord owns the structural acceptance contract for persisted
// Couch thread records. Couch and standalone Pair project this same wire model
// into their richer local types; neither reader maintains a shadow schema.
package threadrecord

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const SchemaVersion = 1

type Validators struct {
	RepoScope func(string) error
	Tag       func(string) error
}

type Address struct {
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
}

type StartClaim struct {
	Nonce         string `json:"nonce"`
	OwnerPID      int    `json:"owner_pid"`
	OwnerIdentity string `json:"owner_identity"`
}

type PolicyCapacity struct {
	Kind  string `json:"kind"`
	Limit int    `json:"limit,omitempty"`
}

type Policy struct {
	PolicyVersion int            `json:"policy_version"`
	PolicyDigest  string         `json:"policy_digest"`
	RepoIdentity  string         `json:"repo_identity"`
	AdmissionKey  string         `json:"admission_key"`
	Capacity      PolicyCapacity `json:"capacity"`
	OnCapacity    string         `json:"on_capacity,omitempty"`
}

type Incarnation struct {
	LegacyActorID string      `json:"legacy_actor_id,omitempty"`
	PID           int         `json:"pid,omitempty"`
	Identity      string      `json:"identity,omitempty"`
	State         string      `json:"state"`
	StartedAt     time.Time   `json:"started_at,omitempty"`
	Policy        *Policy     `json:"policy,omitempty"`
	Start         *StartClaim `json:"start,omitempty"`
}

type Record struct {
	SchemaVersion    int           `json:"schema_version"`
	Address          Address       `json:"address"`
	StartingPath     string        `json:"starting_path"`
	WorkingPath      string        `json:"working_path"`
	CreatedAt        time.Time     `json:"created_at"`
	Revision         uint64        `json:"revision"`
	ClaimGeneration  uint64        `json:"claim_generation"`
	Reservation      bool          `json:"reservation,omitempty"`
	Name             string        `json:"name,omitempty"`
	Description      string        `json:"description,omitempty"`
	PublishedSummary string        `json:"published_summary,omitempty"`
	Incarnations     []Incarnation `json:"incarnations,omitempty"`
}

var componentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ValidateAddress(address Address, validators Validators) error {
	if validators.RepoScope == nil || validators.Tag == nil {
		return fmt.Errorf("thread record component validators are unavailable")
	}
	if err := validators.RepoScope(address.RepoScope); err != nil {
		return fmt.Errorf("invalid thread repo scope %q: %w", address.RepoScope, err)
	}
	if err := validators.Tag(address.Tag); err != nil {
		return fmt.Errorf("invalid thread tag %q: %w", address.Tag, err)
	}
	return nil
}

// Validate enumerates every structural invariant shared by in-memory and
// persisted records. Policy values are evidence interpreted elsewhere; their
// exact JSON shape is still enforced by Record plus strictjson.Decode.
func Validate(record Record, validators Validators) error {
	if record.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported thread schema version %d", record.SchemaVersion)
	}
	if err := ValidateAddress(record.Address, validators); err != nil {
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
		case "unknown", "creating", "live":
		default:
			return fmt.Errorf("incarnation %d has invalid state %q", i, incarnation.State)
		}
		if incarnation.PID < 0 {
			return fmt.Errorf("incarnation %d has negative pid", i)
		}
		if incarnation.Start != nil {
			trackedStarts++
			if incarnation.State != "creating" {
				return fmt.Errorf("incarnation %d has start claim outside creating state", i)
			}
			if !componentPattern.MatchString(incarnation.Start.Nonce) {
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

func ValidatePersisted(record Record, expected Address, validators Validators) error {
	if err := Validate(record, validators); err != nil {
		return err
	}
	if record.ClaimGeneration == 0 {
		return fmt.Errorf("stored thread has zero claim generation")
	}
	if record.Address != expected {
		return fmt.Errorf("thread record path/address mismatch")
	}
	return nil
}

func DecodePersisted(raw []byte, expected Address, validators Validators) (Record, error) {
	var record Record
	if err := strictjson.Decode(raw, &record); err != nil {
		return Record{}, err
	}
	if err := ValidatePersisted(record, expected, validators); err != nil {
		return Record{}, err
	}
	return record, nil
}
