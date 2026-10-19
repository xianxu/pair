// Package threadrecord owns Couch's structural acceptance contract for
// persisted thread records.
package threadrecord

import (
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const SchemaVersion = 4

type Validators struct {
	RepoScope func(string) error
	Tag       func(string) error
}

type Address struct {
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
}

type StartClaim struct {
	RecoveryIntent *RecoveryIntent               `json:"recovery_intent,omitempty"`
	SessionBinding *couchidentity.SessionBinding `json:"session_binding,omitempty"`
	Nonce          string                        `json:"nonce"`
	OwnerPID       int                           `json:"owner_pid"`
	OwnerIdentity  string                        `json:"owner_identity"`
	LaunchProfile  *LaunchProfile                `json:"launch_profile,omitempty"`
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
	RecoveryIntent  *RecoveryIntent               `json:"recovery_intent,omitempty"`
	RecoveryContext *RecoveryContext              `json:"recovery_context,omitempty"`
	PreviousSession *PreviousSession              `json:"previous_session,omitempty"`
	SessionBinding  *couchidentity.SessionBinding `json:"session_binding,omitempty"`
	Continuation    *checkpoint.Request           `json:"continuation,omitempty"`
	// ContinuationMissed is the one visible line a compaction that didn't
	// finish leaves (#522). It blocks nothing.
	ContinuationMissed *checkpoint.Miss `json:"continuation_missed,omitempty"`
	SchemaVersion      int              `json:"schema_version"`
	Address            Address          `json:"address"`
	StartingPath       string           `json:"starting_path"`
	WorkingPath        string           `json:"working_path"`
	CreatedAt          time.Time        `json:"created_at"`
	Revision           uint64           `json:"revision"`
	// DeprecatedClaimGeneration is a TOMBSTONE for the same reason as
	// Incarnation.DeprecatedPolicy -- and a more urgent one: it appeared in
	// EVERY record in the operator's store, so deleting it would have made the
	// whole store unreadable at once.
	DeprecatedClaimGeneration uint64 `json:"claim_generation,omitempty"`
	Reservation               bool   `json:"reservation,omitempty"`
	Name                      string `json:"name,omitempty"`
	Description               string `json:"description,omitempty"`
	PublishedSummary          string `json:"published_summary,omitempty"`
	// Layout is the thread's witnessed pair layout, kept as a raw string here:
	// this package is the persistence mirror and does not own the vocabulary.
	// Absent on every record written before pair#198.
	Layout              string            `json:"layout,omitempty"`
	Incarnations        []Incarnation     `json:"incarnations,omitempty"`
	LatestLaunchProfile *LaunchProfile    `json:"latest_launch_profile,omitempty"`
	LastActiveAt        time.Time         `json:"last_active_at,omitempty"`
	Park                *ParkTransaction  `json:"park,omitempty"`
	VerifiedPark        *VerifiedPark     `json:"verified_park,omitempty"`
	ParkHistory         []ParkTransaction `json:"park_history,omitempty"`
	// Transcript is the agent's exact native conversation (pair#695 P2).
	// Absent on records from before it; added without a schema bump because
	// a reader that ignores it falls back to the ledger, as before. Decode
	// checks its types only: its location is checked where it is used (Bind),
	// so one bad or newer-build binding refuses a resume, never the record.
	Transcript *TranscriptBinding `json:"transcript,omitempty"`
	// ProvenDownAt is when a teardown proved this record's tree gone, stamped
	// on the record as it is archived (pair#695); a lifecycle success.
	ProvenDownAt *time.Time `json:"proven_down_at,omitempty"`
	// Unknown holds top-level members this build does not own, captured on
	// decode and written back verbatim by EncodePersisted (pair#695 §2a), so a
	// rewrite by an older build cannot erase a newer build's field. Only the
	// current schema tolerates them. Nested types older than pair#695 stay
	// strict; a nested type born additive (TranscriptBinding) preserves its
	// own unknown members.
	Unknown map[string]json.RawMessage `json:"-"`
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
	if record.RecoveryIntent != nil {
		if err := ValidateRecoveryIntent(*record.RecoveryIntent, record.Address); err != nil {
			return err
		}
	}
	if record.RecoveryContext != nil {
		if err := ValidateRecoveryContext(*record.RecoveryContext, record.Address); err != nil {
			return err
		}
	}
	if record.PreviousSession != nil {
		if err := record.PreviousSession.Validate(record.Address); err != nil {
			return fmt.Errorf("previous session: %w", err)
		}
	}

	if record.ContinuationMissed != nil {
		if err := record.ContinuationMissed.Validate(); err != nil {
			return err
		}
	}
	if record.Continuation != nil {
		if err := record.Continuation.Validate(); err != nil {
			return fmt.Errorf("continuation: %w", err)
		}
		if record.Continuation.ID != checkpoint.RequestID(record.Address.RepoScope, record.Address.Tag, record.Continuation.Source.LaunchOrdinal, record.Continuation.Checkpoint.Digest) {
			return fmt.Errorf("continuation request belongs to another address")
		}
		if absence := record.Continuation.SourceAbsence; absence != nil && absence.RecordRevision > record.Revision {
			return fmt.Errorf("continuation source absence refers to a future record revision")
		}
		if request := record.Continuation; request.SourcePark != "" {
			found := false
			for _, park := range record.ParkHistory {
				if park.Identity.Nonce == request.SourcePark && park.Closed && !park.Tombstoned && park.SuccessfulAttempt > 0 && park.Identity.PID == request.Source.Helper.PID && park.Identity.ProcessIdentity == request.Source.Helper.Identity {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("continuation source park receipt is unavailable or mismatched")
			}
		}
	}
	if err := validateSessionBinding(record.SessionBinding, record.Address); err != nil {
		return err
	}
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
			if incarnation.Start.RecoveryIntent != nil {
				if err := ValidateRecoveryIntent(*incarnation.Start.RecoveryIntent, record.Address); err != nil {
					return err
				}
			}
			if err := validateSessionBinding(incarnation.Start.SessionBinding, record.Address); err != nil {
				return err
			}
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

	if version == 1 || version == 2 {
		if err := rejectLegacyRecoveryFields(envelope); err != nil {
			return Record{}, err
		}
	}
	if version <= 3 {
		if err := rejectTaggedRecoveryFields(envelope); err != nil {
			return Record{}, err
		}
	}
	var record Record
	switch version {
	case 1:
		var legacy recordV1
		if err := strictjson.Decode(raw, &legacy); err != nil {
			return Record{}, err
		}
		record = migrateV1(legacy)
	case 2, 3:
		// Earlier schemas migrate only in memory; field-presence guards above
		// preserve each version's original recovery semantics.
		if err := strictjson.Decode(raw, &record); err != nil {
			return Record{}, err
		}
		record.SchemaVersion = SchemaVersion
	case SchemaVersion:
		decoded, err := decodeCurrent(raw)
		if err != nil {
			return Record{}, err
		}
		record = decoded
	default:
		return Record{}, fmt.Errorf("unsupported thread schema version %d", version)
	}
	if err := ValidatePersisted(record, expected, validators); err != nil {
		return Record{}, err
	}
	return record, nil
}

var recordKnownFields = strictjson.KnownFields(reflect.TypeOf(Record{}))

// decodeCurrent decodes the owned members strictly and keeps the rest.
// Owned members may be null exactly as strictjson.Decode allows.
func decodeCurrent(raw []byte) (Record, error) {
	var record Record
	unknown, err := strictjson.DecodeKnown(raw, recordKnownFields, &record)
	if err != nil {
		return Record{}, err
	}
	if len(unknown) > 0 {
		record.Unknown = unknown
	}
	return record, nil
}

// EncodePersisted is the one encoding of a persisted record: indented owned
// members, then the preserved unknown ones verbatim.
func EncodePersisted(record Record) ([]byte, error) {
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, record.Unknown, recordKnownFields)
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

func validateSessionBinding(binding *couchidentity.SessionBinding, address Address) error {
	if binding == nil {
		return nil
	}
	if err := binding.Validate(); err != nil {
		return fmt.Errorf("session binding: %w", err)
	}
	if binding.ScopeKey != address.RepoScope || binding.Tag != address.Tag {
		return fmt.Errorf("session binding belongs to another address")
	}
	return nil
}

// Legacy versions cannot smuggle new recovery semantics through shared nested
// types. Presence is rejected, including null, before migration.
func rejectLegacyRecoveryFields(envelope map[string]json.RawMessage) error {
	if _, ok := envelope["recovery_context"]; ok {
		return fmt.Errorf("legacy schema cannot contain recovery context")
	}
	var incarnations []map[string]json.RawMessage
	if raw, ok := envelope["incarnations"]; ok {
		if err := strictjson.Decode(raw, &incarnations); err != nil {
			return err
		}
	}
	for _, incarnation := range incarnations {
		if raw, ok := incarnation["start"]; ok && string(raw) != "null" {
			var start map[string]json.RawMessage
			if err := strictjson.Decode(raw, &start); err != nil {
				return err
			}
			if _, ok := start["recovery_intent"]; ok {
				return fmt.Errorf("legacy schema cannot contain recovery intent")
			}
		}
	}
	return nil
}

// Schema 3 knows only capture-dependent recovery. Reject newer fields by
// presence, including null, before the shared decoder can erase their version.
func rejectTaggedRecoveryFields(envelope map[string]json.RawMessage) error {
	if _, ok := envelope["recovery_intent"]; ok {
		return fmt.Errorf("legacy schema cannot contain retained recovery intent")
	}
	var incarnations []map[string]json.RawMessage
	if raw, ok := envelope["incarnations"]; ok {
		if err := strictjson.Decode(raw, &incarnations); err != nil {
			return err
		}
	}
	for _, incarnation := range incarnations {
		raw, ok := incarnation["start"]
		if !ok || string(raw) == "null" {
			continue
		}
		var start map[string]json.RawMessage
		if err := strictjson.Decode(raw, &start); err != nil {
			return err
		}
		raw, ok = start["recovery_intent"]
		if !ok || string(raw) == "null" {
			continue
		}
		var intent map[string]json.RawMessage
		if err := strictjson.Decode(raw, &intent); err != nil {
			return err
		}
		for _, field := range []string{"mode", "address", "source_revision", "created_at", "agent", "orientation"} {
			if _, ok := intent[field]; ok {
				return fmt.Errorf("legacy recovery intent cannot contain %s", field)
			}
		}
	}
	return nil
}
