package couchcore

import (
	"encoding/json"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

const ThreadSchemaVersion = threadrecord.SchemaVersion

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
	Nonce         string         `json:"nonce"`
	OwnerPID      int            `json:"owner_pid"`
	OwnerIdentity string         `json:"owner_identity"`
	LaunchProfile *LaunchProfile `json:"launch_profile,omitempty"`
}

type ThreadIncarnation struct {
	PID       int              `json:"pid,omitempty"`
	Identity  string           `json:"identity,omitempty"`
	State     IncarnationState `json:"state"`
	StartedAt time.Time        `json:"started_at,omitempty"`
	// RepoIdentity is the Git common directory: the key under which this path's
	// successful launch profile is remembered. It used to be read out of the
	// fleet-policy record that admission attached here (pair#170 M4).
	RepoIdentity  string            `json:"repo_identity,omitempty"`
	Start         *ThreadStartClaim `json:"start,omitempty"`
	LaunchProfile *LaunchProfile    `json:"launch_profile,omitempty"`
}

type ThreadRecord struct {
	SchemaVersion       int                 `json:"schema_version"`
	Address             ThreadAddress       `json:"address"`
	StartingPath        string              `json:"starting_path"`
	WorkingPath         string              `json:"working_path"`
	CreatedAt           time.Time           `json:"created_at"`
	Revision            uint64              `json:"revision"`
	Reservation         bool                `json:"reservation,omitempty"`
	Name                string              `json:"name,omitempty"`
	Description         string              `json:"description,omitempty"`
	PublishedSummary    string              `json:"published_summary,omitempty"`
	Incarnations        []ThreadIncarnation `json:"incarnations,omitempty"`
	LatestLaunchProfile *LaunchProfile      `json:"latest_launch_profile,omitempty"`
	LastActiveAt        time.Time           `json:"last_active_at,omitempty"`
	Park                *ParkTransaction    `json:"park,omitempty"`
	VerifiedPark        *VerifiedPark       `json:"verified_park,omitempty"`
	ParkHistory         []ParkTransaction   `json:"park_history,omitempty"`
}

var threadRecordValidators = threadrecord.Validators{
	RepoScope: launcher.ValidateRepoScopeKey,
	Tag:       launcher.ValidatePairTag,
}

func ValidateThreadRecord(record ThreadRecord) error {
	return threadrecord.Validate(toPersistedThreadRecord(record), threadRecordValidators)
}

func validateThreadAddress(address ThreadAddress) error {
	return threadrecord.ValidateAddress(toPersistedThreadAddress(address), threadRecordValidators)
}

func toPersistedThreadAddress(address ThreadAddress) threadrecord.Address {
	return threadrecord.Address{RepoScope: address.RepoScope, Tag: string(address.Tag)}
}

func toPersistedThreadRecord(record ThreadRecord) threadrecord.Record {
	out := threadrecord.Record{
		SchemaVersion: record.SchemaVersion, Address: toPersistedThreadAddress(record.Address),
		StartingPath: record.StartingPath, WorkingPath: record.WorkingPath, CreatedAt: record.CreatedAt,
		Revision: record.Revision, Reservation: record.Reservation,
		Name: record.Name, Description: record.Description, PublishedSummary: record.PublishedSummary,
		Incarnations: make([]threadrecord.Incarnation, len(record.Incarnations)),
	}
	if record.LatestLaunchProfile != nil {
		profile := cloneLaunchProfile(*record.LatestLaunchProfile)
		out.LatestLaunchProfile = &threadrecord.LaunchProfile{Agent: profile.Agent, Argv: profile.Argv}
	}
	out.LastActiveAt = record.LastActiveAt
	if record.Park != nil {
		park := toPersistedParkTransaction(*record.Park)
		out.Park = &park
	}
	if record.VerifiedPark != nil {
		out.VerifiedPark = &threadrecord.VerifiedPark{
			Identity: toPersistedParkIdentity(record.VerifiedPark.Identity),
			Attempt:  record.VerifiedPark.Attempt, ParkedAt: record.VerifiedPark.ParkedAt,
		}
	}
	if record.ParkHistory != nil {
		out.ParkHistory = make([]threadrecord.ParkTransaction, len(record.ParkHistory))
		for i := range record.ParkHistory {
			out.ParkHistory[i] = toPersistedParkTransaction(record.ParkHistory[i])
		}
	}
	for i, incarnation := range record.Incarnations {
		out.Incarnations[i] = threadrecord.Incarnation{
			PID:      incarnation.PID,
			Identity: incarnation.Identity, State: string(incarnation.State), StartedAt: incarnation.StartedAt,
		}
		if incarnation.Start != nil {
			out.Incarnations[i].Start = &threadrecord.StartClaim{
				Nonce: incarnation.Start.Nonce, OwnerPID: incarnation.Start.OwnerPID, OwnerIdentity: incarnation.Start.OwnerIdentity,
			}
			if incarnation.Start.LaunchProfile != nil {
				profile := cloneLaunchProfile(*incarnation.Start.LaunchProfile)
				out.Incarnations[i].Start.LaunchProfile = &threadrecord.LaunchProfile{Agent: profile.Agent, Argv: profile.Argv}
			}
		}
		out.Incarnations[i].RepoIdentity = incarnation.RepoIdentity
		if incarnation.LaunchProfile != nil {
			profile := cloneLaunchProfile(*incarnation.LaunchProfile)
			out.Incarnations[i].LaunchProfile = &threadrecord.LaunchProfile{Agent: profile.Agent, Argv: profile.Argv}
		}
	}
	return out
}

func fromPersistedThreadRecord(record threadrecord.Record) ThreadRecord {
	out := ThreadRecord{
		SchemaVersion: record.SchemaVersion,
		Address:       ThreadAddress{RepoScope: record.Address.RepoScope, Tag: ThreadTag(record.Address.Tag)},
		StartingPath:  record.StartingPath, WorkingPath: record.WorkingPath, CreatedAt: record.CreatedAt,
		Revision: record.Revision, Reservation: record.Reservation,
		Name: record.Name, Description: record.Description, PublishedSummary: record.PublishedSummary,
		Incarnations: make([]ThreadIncarnation, len(record.Incarnations)),
	}
	if record.LatestLaunchProfile != nil {
		profile := LaunchProfile{Agent: record.LatestLaunchProfile.Agent, Argv: cloneArgv(record.LatestLaunchProfile.Argv)}
		out.LatestLaunchProfile = &profile
	}
	out.LastActiveAt = record.LastActiveAt
	if record.Park != nil {
		park := fromPersistedParkTransaction(*record.Park)
		out.Park = &park
	}
	if record.VerifiedPark != nil {
		out.VerifiedPark = &VerifiedPark{
			Identity: fromPersistedParkIdentity(record.VerifiedPark.Identity),
			Attempt:  record.VerifiedPark.Attempt, ParkedAt: record.VerifiedPark.ParkedAt,
		}
	}
	if record.ParkHistory != nil {
		out.ParkHistory = make([]ParkTransaction, len(record.ParkHistory))
		for i := range record.ParkHistory {
			out.ParkHistory[i] = fromPersistedParkTransaction(record.ParkHistory[i])
		}
	}
	for i, incarnation := range record.Incarnations {
		out.Incarnations[i] = ThreadIncarnation{
			PID:      incarnation.PID,
			Identity: incarnation.Identity, State: IncarnationState(incarnation.State), StartedAt: incarnation.StartedAt,
		}
		if incarnation.Start != nil {
			out.Incarnations[i].Start = &ThreadStartClaim{
				Nonce: incarnation.Start.Nonce, OwnerPID: incarnation.Start.OwnerPID, OwnerIdentity: incarnation.Start.OwnerIdentity,
			}
			if incarnation.Start.LaunchProfile != nil {
				profile := LaunchProfile{Agent: incarnation.Start.LaunchProfile.Agent, Argv: cloneArgv(incarnation.Start.LaunchProfile.Argv)}
				out.Incarnations[i].Start.LaunchProfile = &profile
			}
		}
		// The tombstone has to be READ, not merely tolerated. A pre-M4
		// incarnation carries its repository identity inside `policy` and has
		// no top-level `repo_identity`, so ignoring the tombstone loads it as
		// "" -- and advanceSuccessfulStart refuses an empty identity. That
		// fires whenever an interrupted start written by the old binary is
		// promoted after the upgrade: reconcileInterruptedStarts runs inside
		// New(), so couch would refuse to START AT ALL. Decoding without
		// carrying the value forward is the same whole-store blast radius the
		// tombstone exists to prevent.
		out.Incarnations[i].RepoIdentity = incarnation.RepoIdentity
		if out.Incarnations[i].RepoIdentity == "" {
			out.Incarnations[i].RepoIdentity = deprecatedPolicyRepoIdentity(incarnation.DeprecatedPolicy)
		}
		if incarnation.LaunchProfile != nil {
			profile := LaunchProfile{Agent: incarnation.LaunchProfile.Agent, Argv: cloneArgv(incarnation.LaunchProfile.Argv)}
			out.Incarnations[i].LaunchProfile = &profile
		}
	}
	return out
}

func cloneThreadRecord(record ThreadRecord) ThreadRecord {
	copy := record
	copy.Incarnations = append([]ThreadIncarnation{}, record.Incarnations...)
	for i := range copy.Incarnations {
		if record.Incarnations[i].Start != nil {
			start := *record.Incarnations[i].Start
			if start.LaunchProfile != nil {
				profile := cloneLaunchProfile(*start.LaunchProfile)
				start.LaunchProfile = &profile
			}
			copy.Incarnations[i].Start = &start
		}
		if record.Incarnations[i].LaunchProfile != nil {
			profile := cloneLaunchProfile(*record.Incarnations[i].LaunchProfile)
			copy.Incarnations[i].LaunchProfile = &profile
		}
	}
	if record.LatestLaunchProfile != nil {
		profile := cloneLaunchProfile(*record.LatestLaunchProfile)
		copy.LatestLaunchProfile = &profile
	}
	if record.Park != nil {
		park := cloneParkTransaction(*record.Park)
		copy.Park = &park
	}
	if record.VerifiedPark != nil {
		verified := *record.VerifiedPark
		copy.VerifiedPark = &verified
	}
	if record.ParkHistory != nil {
		copy.ParkHistory = make([]ParkTransaction, len(record.ParkHistory))
		for i := range record.ParkHistory {
			copy.ParkHistory[i] = cloneParkTransaction(record.ParkHistory[i])
		}
	}
	return copy
}

func toPersistedParkIdentity(identity ParkIdentity) threadrecord.ParkIdentity {
	return threadrecord.ParkIdentity{
		Nonce: identity.Nonce, Address: toPersistedThreadAddress(identity.Address),
		PID: identity.PID, ProcessIdentity: identity.ProcessIdentity,
	}
}

func fromPersistedParkIdentity(identity threadrecord.ParkIdentity) ParkIdentity {
	return ParkIdentity{
		Nonce:   identity.Nonce,
		Address: ThreadAddress{RepoScope: identity.Address.RepoScope, Tag: ThreadTag(identity.Address.Tag)},
		PID:     identity.PID, ProcessIdentity: identity.ProcessIdentity,
	}
}

func toPersistedParkTransaction(transaction ParkTransaction) threadrecord.ParkTransaction {
	out := threadrecord.ParkTransaction{
		Identity:     toPersistedParkIdentity(transaction.Identity),
		BaseRevision: transaction.BaseRevision, RecordRevision: transaction.RecordRevision,
		Phase: string(transaction.Phase), Closed: transaction.Closed,
		Tombstoned: transaction.Tombstoned, SuccessfulAttempt: transaction.SuccessfulAttempt,
		Attempts: make([]threadrecord.ParkAttempt, len(transaction.Attempts)),
	}
	for i, attempt := range transaction.Attempts {
		out.Attempts[i] = threadrecord.ParkAttempt{Number: attempt.Number, TimedOut: attempt.TimedOut, Closed: attempt.Closed}
		if attempt.Failure != nil {
			out.Attempts[i].Failure = &threadrecord.ParkFailure{Code: string(attempt.Failure.Code), Diagnostic: attempt.Failure.Diagnostic}
		}
	}
	return out
}

func fromPersistedParkTransaction(transaction threadrecord.ParkTransaction) ParkTransaction {
	out := ParkTransaction{
		Identity:     fromPersistedParkIdentity(transaction.Identity),
		BaseRevision: transaction.BaseRevision, RecordRevision: transaction.RecordRevision,
		Phase: ParkPhase(transaction.Phase), Closed: transaction.Closed,
		Tombstoned: transaction.Tombstoned, SuccessfulAttempt: transaction.SuccessfulAttempt,
		Attempts: make([]ParkAttempt, len(transaction.Attempts)),
	}
	for i, attempt := range transaction.Attempts {
		out.Attempts[i] = ParkAttempt{Number: attempt.Number, TimedOut: attempt.TimedOut, Closed: attempt.Closed}
		if attempt.Failure != nil {
			out.Attempts[i].Failure = &ParkFailure{Code: pairlifecycle.FailureCode(attempt.Failure.Code), Diagnostic: attempt.Failure.Diagnostic}
		}
	}
	return out
}

// deprecatedPolicyRepoIdentity recovers the one field anything still wants from
// the pre-pair#170 `policy` object. Best-effort by design: a malformed or
// absent tombstone yields "", which is exactly the state the caller already
// handles. It never resurrects the rest of the record -- capacity, provider
// version and declaration digest went with admission and have no reader.
func deprecatedPolicyRepoIdentity(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var legacy struct {
		RepoIdentity string `json:"repo_identity"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return ""
	}
	return legacy.RepoIdentity
}
