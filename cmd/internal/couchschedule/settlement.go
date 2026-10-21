package couchschedule

import (
	"reflect"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// SettlementKind identifies the single retained task phase.
type SettlementKind string

const (
	SettlementStable          SettlementKind = "stable"
	SettlementExecuting       SettlementKind = "executing"
	SettlementAwaitingReceipt SettlementKind = "awaiting-receipt"
	SettlementSettling        SettlementKind = "settling"
	SettlementQuarantined     SettlementKind = "quarantined"
	SettlementUnknown         SettlementKind = "unknown"
)

type SettlementAction string

const (
	SettlementNone            SettlementAction = "none"
	SettlementPersist         SettlementAction = "persist"
	SettlementReadHistory     SettlementAction = "read-history"
	SettlementAwaitResult     SettlementAction = "await-result"
	SettlementAwaitReceipt    SettlementAction = "await-receipt"
	SettlementPublishHistory  SettlementAction = "publish-history"
	SettlementFinalizeHistory SettlementAction = "finalize-history"
)

// Settlement is a pure decision. The owner must finish Next before releasing
// Effects or acknowledging State; State is only a candidate until persisted.
type Settlement struct {
	Kind    SettlementKind
	State   TaskState
	Effects []Effect
	Busy    bool
	Next    SettlementAction
	Reason  string
}

// PlanSettlement receives only canonical evidence whose durability the IO owner
// has confirmed. A nil canonical value conveys no completion authority.
func PlanSettlement(state TaskState, canonical *RunSummary, event Event) Settlement {
	original := state
	state = cloneSettlementState(state)
	result := Settlement{State: state, Next: SettlementNone}
	quarantine := func(reason string) Settlement {
		return Settlement{Kind: SettlementQuarantined, State: cloneSettlementState(original), Next: SettlementReadHistory, Reason: reason}
	}
	if err := validateTaskPhase(state); err != nil {
		return quarantine(err.Error())
	}
	if event.Kind == ResultRecovered && canonical == nil {
		return quarantine("recovery requires canonical durable evidence")
	}
	if canonical != nil {
		if event.At.IsZero() {
			return quarantine("canonical adoption requires observation time")
		}
		if !validRecoveredSummary(*canonical) || !settlementCanonicalMatches(state, *canonical) {
			return quarantine("canonical history does not match retained occurrence")
		}
		attempt, revision := canonical.ID, canonical.Revision
		if state.Active != nil {
			attempt, revision = state.Active.ID, state.Active.Revision
		}
		state, _ = Advance(state, NewRecoveryEvent(event.At, attempt, revision, *canonical))
		// Invalid source/phase combinations must not silently discard evidence.
		if state.Last == nil || !settlementKnownSummaryEqual(*state.Last, *canonical) {
			return quarantine("canonical history cannot settle retained phase")
		}
	}
	if event.Kind == CatalogChanged && state.Active != nil {
		if settlementCatalogueDiffers(state, event.Declaration) {
			result.Busy = true
			result.Reason = "catalogue change pending: task busy"
		}
		// Identical discovery is inert, including the retained declaration's due.
	} else if event.Kind != "" && event.Kind != ResultRecovered {
		state, result.Effects = Advance(state, event)
	}
	result.State = state
	result.Kind = settlementKind(state)
	switch result.Kind {
	case SettlementExecuting:
		result.Next = SettlementAwaitResult
	case SettlementAwaitingReceipt:
		result.Next = SettlementAwaitReceipt
	case SettlementUnknown:
		result.Next = SettlementNone
	}
	if !reflect.DeepEqual(original, state) {
		result.Next = SettlementPersist
		// The owner must durably publish/finalize newly-derived facts before their
		// matching ledger commit. Recovery already supplied confirmed history.
		if state.Last != nil && !reflect.DeepEqual(original.Last, state.Last) && state.Last.Outcome != "unknown" && (canonical == nil || !settlementKnownSummaryEqual(*state.Last, *canonical)) {
			result.Kind = SettlementSettling
			result.Next = SettlementPublishHistory
			if original.Last != nil && original.Last.ID == state.Last.ID && original.Last.Notice == "pending" && state.Last.Notice != "pending" {
				result.Next = SettlementFinalizeHistory
			}
		}
	}
	return result
}

func settlementKind(state TaskState) SettlementKind {
	if state.Active == nil {
		return SettlementStable
	}
	switch state.Active.Stage {
	case "uncertain":
		return SettlementUnknown
	case "notifying", "delivery":
		return SettlementAwaitingReceipt
	default:
		return SettlementExecuting
	}
}

func settlementCatalogueDiffers(state TaskState, d *Declaration) bool {
	if d == nil {
		return state.RemovedAt.IsZero()
	}
	return state.Key != d.Key || state.Revision != d.Revision || state.Spec != d.Spec || state.Allowed != d.Allowed || state.Refusal != d.Reason || !state.RemovedAt.IsZero()
}

func settlementCanonicalMatches(state TaskState, canonical RunSummary) bool {
	if a := state.Active; a != nil {
		return canonical.ID == a.ID && canonical.Revision == a.Revision && canonical.Source == a.Source && canonical.DueAt.Equal(a.DueAt) && canonical.StartedAt.Equal(a.StartedAt)
	}
	if last := state.Last; last != nil {
		return canonical.ID == last.ID && (last.Revision == "" || canonical.Revision == last.Revision) && canonical.Source == last.Source && canonical.DueAt.Equal(last.DueAt) && canonical.StartedAt.Equal(last.StartedAt)
	}
	return false
}

func settlementKnownSummaryEqual(a, b RunSummary) bool {
	a.Unknown, b.Unknown = nil, nil
	return reflect.DeepEqual(a, b)
}

func cloneSettlementState(state TaskState) TaskState {
	state.Unknown = strictjson.CloneRaw(state.Unknown)
	if state.Active != nil {
		state.Active = cloneAttempt(state.Active)
	}
	if state.Last != nil {
		state.Last = cloneRunSummary(state.Last)
	}
	return state
}
