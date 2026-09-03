// Package threadrecord owns Couch's structural acceptance contract for
// persisted thread records.
package threadrecord

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const SchemaVersion = 2

type Validators struct {
	RepoScope func(string) error
	Tag       func(string) error
}

type Address struct {
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
}

type StartClaim struct {
	Nonce         string         `json:"nonce"`
	OwnerPID      int            `json:"owner_pid"`
	OwnerIdentity string         `json:"owner_identity"`
	LaunchProfile *LaunchProfile `json:"launch_profile,omitempty"`
}

type PolicyCapacity struct {
	Kind  string `json:"kind"`
	Limit int    `json:"limit,omitempty"`
}

type LaunchProfile struct {
	Agent string   `json:"agent"`
	Argv  []string `json:"argv"`
}

type Incarnation struct {
	// DeprecatedLegacyActorID is a TOMBSTONE. Only the one-time registry
	// cutover ever wrote it, and pair#170 M4 deleted the cutover. The
	// operator's live store has none left (the imported records were parked,
	// which clears incarnations), but "none in ONE store" is not "none
	// anywhere", and the cost of being wrong is an undecodable record.
	DeprecatedLegacyActorID json.RawMessage `json:"legacy_actor_id,omitempty"`
	PID                     int             `json:"pid,omitempty"`
	Identity                string          `json:"identity,omitempty"`
	State                   string          `json:"state"`
	StartedAt               time.Time       `json:"started_at,omitempty"`
	// RepoIdentity is the Git common directory, the key for this path's saved
	// launch preference. It replaces the fleet-policy record that used to carry
	// it (pair#170 M4).
	RepoIdentity string `json:"repo_identity,omitempty"`
	// DeprecatedPolicy is a TOMBSTONE, not a field. Records written before
	// pair#170 M4 carry a `policy` object, and strictjson disallows unknown
	// fields -- so removing it outright would make every such record
	// undecodable, and an undecodable record is a thread that vanishes from
	// every view at once. It is decoded, never read, and never written, so a
	// record sheds it on its next write with no migration pass.
	DeprecatedPolicy json.RawMessage `json:"policy,omitempty"`
	Start            *StartClaim     `json:"start,omitempty"`
	LaunchProfile    *LaunchProfile  `json:"launch_profile,omitempty"`
}

type Record struct {
	SchemaVersion int       `json:"schema_version"`
	Address       Address   `json:"address"`
	StartingPath  string    `json:"starting_path"`
	WorkingPath   string    `json:"working_path"`
	CreatedAt     time.Time `json:"created_at"`
	Revision      uint64    `json:"revision"`
	// DeprecatedClaimGeneration is a TOMBSTONE for the same reason as
	// Incarnation.DeprecatedPolicy -- and a more urgent one: it appeared in
	// EVERY record in the operator's store, so deleting it would have made the
	// whole store unreadable at once.
	DeprecatedClaimGeneration uint64            `json:"claim_generation,omitempty"`
	Reservation               bool              `json:"reservation,omitempty"`
	Name                      string            `json:"name,omitempty"`
	Description               string            `json:"description,omitempty"`
	PublishedSummary          string            `json:"published_summary,omitempty"`
	Incarnations              []Incarnation     `json:"incarnations,omitempty"`
	LatestLaunchProfile       *LaunchProfile    `json:"latest_launch_profile,omitempty"`
	LastActiveAt              time.Time         `json:"last_active_at,omitempty"`
	Park                      *ParkTransaction  `json:"park,omitempty"`
	VerifiedPark              *VerifiedPark     `json:"verified_park,omitempty"`
	ParkHistory               []ParkTransaction `json:"park_history,omitempty"`
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
		if incarnation.LaunchProfile != nil {
			if incarnation.State == "creating" {
				return fmt.Errorf("incarnation %d has launch profile before registration", i)
			}
			if err := validateLaunchProfile(*incarnation.LaunchProfile); err != nil {
				return fmt.Errorf("incarnation %d has incomplete launch profile", i)
			}
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
			if incarnation.Start.LaunchProfile != nil {
				if err := validateLaunchProfile(*incarnation.Start.LaunchProfile); err != nil {
					return fmt.Errorf("incarnation %d has incomplete pending launch profile", i)
				}
			}
		}
	}
	if trackedStarts > 1 {
		return fmt.Errorf("thread has %d tracked start transactions", trackedStarts)
	}
	return validateLifecycle(record)
}

func ValidatePersisted(record Record, expected Address, validators Validators) error {
	if err := Validate(record, validators); err != nil {
		return err
	}
	if record.Address != expected {
		return fmt.Errorf("thread record path/address mismatch")
	}
	return nil
}

func DecodePersisted(raw []byte, expected Address, validators Validators) (Record, error) {
	var envelope map[string]json.RawMessage
	if err := strictjson.Decode(raw, &envelope); err != nil {
		return Record{}, err
	}
	versionRaw, ok := envelope["schema_version"]
	if !ok {
		return Record{}, fmt.Errorf("thread schema version is required")
	}
	var version int
	if err := strictjson.Decode(versionRaw, &version); err != nil {
		return Record{}, fmt.Errorf("invalid thread schema version: %w", err)
	}

	var record Record
	switch version {
	case 1:
		var legacy recordV1
		if err := strictjson.Decode(raw, &legacy); err != nil {
			return Record{}, err
		}
		record = migrateV1(legacy)
	case SchemaVersion:
		if err := strictjson.Decode(raw, &record); err != nil {
			return Record{}, err
		}
	default:
		return Record{}, fmt.Errorf("unsupported thread schema version %d", version)
	}
	if err := ValidatePersisted(record, expected, validators); err != nil {
		return Record{}, err
	}
	return record, nil
}

// recordV1 is frozen. Keeping its exact field set makes migration strict and
// ensures rollback to an old binary fails closed on newly-written v2 fields.
type recordV1 struct {
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

func migrateV1(legacy recordV1) Record {
	return Record{
		SchemaVersion: SchemaVersion, Address: legacy.Address,
		StartingPath: legacy.StartingPath, WorkingPath: legacy.WorkingPath,
		CreatedAt: legacy.CreatedAt, Revision: legacy.Revision,
		Reservation: legacy.Reservation,
		Name:        legacy.Name, Description: legacy.Description,
		PublishedSummary: legacy.PublishedSummary, Incarnations: legacy.Incarnations,
	}
}
