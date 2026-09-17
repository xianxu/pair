package couchcore

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// Recovery decisions describe an observation, never durable launch authority.
// Execution reobserves before every store or process boundary.
type RecoveryEvidence struct {
	Thread     ThreadRecord
	Helper     Liveness
	Session    string
	Presence   SessionPresence
	Detached   bool
	Checkpoint bool
}

type RecoveryDecision struct {
	Recover          bool   `json:"recover"`
	FromCheckpoint   bool   `json:"from_checkpoint"`
	Archive          bool   `json:"archive"`
	Diagnosis        string `json:"diagnosis"`
	CheckpointPath   string `json:"checkpoint_path,omitempty"`
	CheckpointDigest string `json:"checkpoint_digest,omitempty"`
}

func DecideRecovery(in RecoveryEvidence) RecoveryDecision {
	d := RecoveryDecision{}
	if r := in.Thread.Continuation; r != nil {
		d.CheckpointPath, d.CheckpointDigest = r.Checkpoint.SourcePath, r.Checkpoint.Digest
	}
	if in.Thread.Park != nil {
		d.Diagnosis = "a park transaction is still open; let lifecycle recovery finish"
		return d
	}
	if len(in.Thread.Incarnations) > 1 {
		d.Diagnosis = "multiple recorded helpers; ownership must be resolved"
		return d
	}
	if len(in.Thread.Incarnations) == 1 {
		i := in.Thread.Incarnations[0]
		if i.Start != nil || i.State != IncarnationLive {
			d.Diagnosis = "helper start or ownership is unresolved; let lifecycle recovery finish"
			return d
		}
		if in.Helper != Dead {
			d.Diagnosis = "the recorded helper is live or its death cannot be proved"
			return d
		}
	}
	if refusal := RecoverySessionRefusal(in); refusal != "" {
		d.Diagnosis = refusal
		return d
	}
	switch in.Presence {
	case PresencePresent:
		d.Recover, d.Archive = true, true
		d.Diagnosis = "session survives; reattach to the running agent"
	case PresenceAbsent:
		d.Recover, d.FromCheckpoint, d.Archive = in.Checkpoint, true, true
		d.Diagnosis = "session is gone; select a checkpoint to start a new conversation, or archive"
		if in.Checkpoint {
			d.Diagnosis = "session is gone; recover a new conversation from the retained checkpoint"
		}
	}
	return d
}

// RecoverySessionRefusal is the SESSION half of DecideRecovery, on its own so
// archive can ask it before it writes anything.
//
// Everything else DecideRecovery refuses on is debris -- an open park, a start
// claim, an incarnation shape -- and clearing debris is a durable write. An
// irreversible step must never precede a revocable check (#256 M1, round 2), so
// archive asks this first, refuses on it with nothing changed, and only then
// clears. Keeping it here rather than restating it at the call site is what
// stops the two from drifting into different answers about the same session.
//
// "" means the session does not stand in the way.
func RecoverySessionRefusal(in RecoveryEvidence) string {
	switch in.Presence {
	case PresencePresent:
		if !in.Detached {
			return "the session has an active client or ambiguous ownership"
		}
		return ""
	case PresenceAbsent:
		return ""
	}
	return "session state could not be checked; inspect and retry"
}

// ProjectRecoveryChoices offers inspection, not execution authority. It uses
// only the existing snapshot so a recovery feature adds no refresh-time IO.
func ProjectRecoveryChoices(record ThreadRecord, evidence ThreadEvidence, state ActionableThreadState, reason ThreadReason) *RecoveryDecision {
	if record.Park != nil || state == ThreadLive || state == ThreadBusy || len(evidence.Live) != 0 {
		return nil
	}
	if len(record.Incarnations) > 1 {
		return nil
	}
	if len(record.Incarnations) == 1 && (record.Incarnations[0].Start != nil || record.Incarnations[0].State != IncarnationLive) {
		return nil
	}
	// `stale-incarnation` was retired in #256 -- a record whose launcher died
	// while its session survived is now `detached` and needs no recovery offer at
	// all, and one whose session is also gone reads `session-gone`.
	if reason != ReasonSessionGone && record.Continuation == nil {
		return nil
	}
	d := &RecoveryDecision{Recover: true, FromCheckpoint: record.LatestLaunchProfile != nil, Archive: true, Diagnosis: "inspect the helper and session before recovering"}
	if r := record.Continuation; r != nil {
		d.CheckpointPath, d.CheckpointDigest = r.Checkpoint.SourcePath, r.Checkpoint.Digest
	}
	return d
}

// AdmitRecoveryGeneration keeps original source identity immutable. Only an
// exact target witness owned by this request can authorize a later generation.
func AdmitRecoveryGeneration(request checkpoint.Request, current ContinuationSource) error {
	if current.Agent == request.Source.Agent && current.Session == request.Source.Session {
		if current.LaunchOrdinal == request.Source.LaunchOrdinal {
			return nil
		}
		matches := func(g *checkpoint.TargetGeneration) bool {
			return g != nil && g.Agent == current.Agent && g.Session == current.Session && g.LaunchOrdinal == current.LaunchOrdinal && g.LaunchOrdinal > request.Source.LaunchOrdinal && g.Attempt != ""
		}
		if g := request.PreviousTargetGeneration; matches(g) && g.Attempt != request.Attempt {
			return nil
		}
		if request.Target != nil {
			if g := request.Target.Generation; matches(g) && g.Attempt == request.Attempt {
				return nil
			}
		}
	}
	return fmt.Errorf("recovery source generation changed without an exact request-owned target; inspect or archive")
}
