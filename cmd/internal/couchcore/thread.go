package couchcore

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"
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

type ThreadIncarnation struct {
	LegacyActorID ActorID          `json:"legacy_actor_id,omitempty"`
	PID           int              `json:"pid,omitempty"`
	Identity      string           `json:"identity,omitempty"`
	State         IncarnationState `json:"state"`
	StartedAt     time.Time        `json:"started_at,omitempty"`
	Policy        *PolicyResult    `json:"policy,omitempty"`
}

type ThreadRecord struct {
	SchemaVersion   int                 `json:"schema_version"`
	Address         ThreadAddress       `json:"address"`
	StartingPath    string              `json:"starting_path"`
	WorkingPath     string              `json:"working_path"`
	CreatedAt       time.Time           `json:"created_at"`
	Revision        uint64              `json:"revision"`
	ClaimGeneration uint64              `json:"claim_generation"`
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
	for i, incarnation := range record.Incarnations {
		switch incarnation.State {
		case IncarnationUnknown, IncarnationCreating, IncarnationLive:
		default:
			return fmt.Errorf("incarnation %d has invalid state %q", i, incarnation.State)
		}
		if incarnation.PID < 0 {
			return fmt.Errorf("incarnation %d has negative pid", i)
		}
	}
	return nil
}

func validateThreadAddress(address ThreadAddress) error {
	if !threadComponentPattern.MatchString(address.RepoScope) {
		return fmt.Errorf("invalid thread repo scope %q", address.RepoScope)
	}
	if !threadComponentPattern.MatchString(string(address.Tag)) {
		return fmt.Errorf("invalid thread tag %q", address.Tag)
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
	}
	return copy
}
