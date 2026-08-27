package couchcore

import (
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
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
	LaunchProfile *LaunchProfile    `json:"launch_profile,omitempty"`
}

type ThreadRecord struct {
	SchemaVersion    int                 `json:"schema_version"`
	Address          ThreadAddress       `json:"address"`
	StartingPath     string              `json:"starting_path"`
	WorkingPath      string              `json:"working_path"`
	CreatedAt        time.Time           `json:"created_at"`
	Revision         uint64              `json:"revision"`
	ClaimGeneration  uint64              `json:"claim_generation"`
	Reservation      bool                `json:"reservation,omitempty"`
	Name             string              `json:"name,omitempty"`
	Description      string              `json:"description,omitempty"`
	PublishedSummary string              `json:"published_summary,omitempty"`
	Incarnations     []ThreadIncarnation `json:"incarnations,omitempty"`
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
		Revision: record.Revision, ClaimGeneration: record.ClaimGeneration, Reservation: record.Reservation,
		Name: record.Name, Description: record.Description, PublishedSummary: record.PublishedSummary,
		Incarnations: make([]threadrecord.Incarnation, len(record.Incarnations)),
	}
	for i, incarnation := range record.Incarnations {
		out.Incarnations[i] = threadrecord.Incarnation{
			LegacyActorID: string(incarnation.LegacyActorID), PID: incarnation.PID,
			Identity: incarnation.Identity, State: string(incarnation.State), StartedAt: incarnation.StartedAt,
		}
		if incarnation.Start != nil {
			out.Incarnations[i].Start = &threadrecord.StartClaim{
				Nonce: incarnation.Start.Nonce, OwnerPID: incarnation.Start.OwnerPID, OwnerIdentity: incarnation.Start.OwnerIdentity,
			}
		}
		if incarnation.Policy != nil {
			out.Incarnations[i].Policy = &threadrecord.Policy{
				PolicyVersion: incarnation.Policy.PolicyVersion, PolicyDigest: incarnation.Policy.PolicyDigest,
				RepoIdentity: incarnation.Policy.RepoIdentity, AdmissionKey: incarnation.Policy.AdmissionKey,
				Capacity:   threadrecord.PolicyCapacity{Kind: string(incarnation.Policy.Capacity.Kind), Limit: incarnation.Policy.Capacity.Limit},
				OnCapacity: string(incarnation.Policy.OnCapacity),
			}
		}
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
		Revision: record.Revision, ClaimGeneration: record.ClaimGeneration, Reservation: record.Reservation,
		Name: record.Name, Description: record.Description, PublishedSummary: record.PublishedSummary,
		Incarnations: make([]ThreadIncarnation, len(record.Incarnations)),
	}
	for i, incarnation := range record.Incarnations {
		out.Incarnations[i] = ThreadIncarnation{
			LegacyActorID: ActorID(incarnation.LegacyActorID), PID: incarnation.PID,
			Identity: incarnation.Identity, State: IncarnationState(incarnation.State), StartedAt: incarnation.StartedAt,
		}
		if incarnation.Start != nil {
			out.Incarnations[i].Start = &ThreadStartClaim{
				Nonce: incarnation.Start.Nonce, OwnerPID: incarnation.Start.OwnerPID, OwnerIdentity: incarnation.Start.OwnerIdentity,
			}
		}
		if incarnation.Policy != nil {
			out.Incarnations[i].Policy = &PolicyResult{
				PolicyVersion: incarnation.Policy.PolicyVersion, PolicyDigest: incarnation.Policy.PolicyDigest,
				RepoIdentity: incarnation.Policy.RepoIdentity, AdmissionKey: incarnation.Policy.AdmissionKey,
				Capacity:   PolicyCapacity{Kind: PolicyCapacityKind(incarnation.Policy.Capacity.Kind), Limit: incarnation.Policy.Capacity.Limit},
				OnCapacity: CapacityAction(incarnation.Policy.OnCapacity),
			}
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
		if record.Incarnations[i].Policy != nil {
			policy := *record.Incarnations[i].Policy
			copy.Incarnations[i].Policy = &policy
		}
		if record.Incarnations[i].Start != nil {
			start := *record.Incarnations[i].Start
			copy.Incarnations[i].Start = &start
		}
		if record.Incarnations[i].LaunchProfile != nil {
			profile := cloneLaunchProfile(*record.Incarnations[i].LaunchProfile)
			copy.Incarnations[i].LaunchProfile = &profile
		}
	}
	return copy
}
