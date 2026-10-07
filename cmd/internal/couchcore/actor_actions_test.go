package couchcore

import (
	"fmt"
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// actorActionRule is one row of #363's resume/reboot admission rules, written
// from its Spec rather than from the code. "" in a match field is a wildcard.
type actorActionRule struct {
	kind       ThreadTargetKind
	state      ActionableThreadState
	reason     ThreadReason
	unfinished string // "", or "any" for pending|running|failed, or "none" (no request, or a complete one)
	recover    string // "", "yes", "no"
	want       []string
}

// actorActionSpec is the literal table, first match wins.
var actorActionSpec = []actorActionRule{
	// Live and busy rows offer no actor operation: live gets the lifecycle
	// actions, and busy is a start another couch may still finish.
	{state: ThreadLive},
	{state: ThreadBusy},
	{state: ThreadArchived},
	{state: ThreadParked, want: []string{"resume", "reboot"}},
	{state: ThreadDetached, want: []string{"resume", "reboot"}},
	// unusable/unknown is no verdict this round: nothing is offered.
	{state: ThreadUnusable, reason: ReasonUnknown},
	// An orphaned server's agent may still be writing: resume would start a
	// second one, and reboot would archive a running conversation. Only a
	// confirmed reap is safe, and M2 adds it (#399).
	{state: ThreadUnusable, reason: ReasonOrphanedServer},
	// A :1+ record lives inside its directory; a :0 record outlives its
	// checkout, so reboot archives it alone.
	{state: ThreadUnusable, reason: ReasonPathMissing, kind: ThreadTargetSlot},
	{state: ThreadUnusable, reason: ReasonPathMissing, kind: ThreadTargetOrdinary, want: []string{"reboot"}},
	// A slot whose parked conversation cannot be resolved (binding-lost, e.g.
	// its agent never took a turn) has nothing to resume or adopt: a park
	// quiesced its session. Reboot only (pair#367 smoke test), unless an
	// unfinished continuation gives resume its own executor.
	{state: ThreadUnusable, reason: ReasonBindingLost, kind: ThreadTargetSlot, unfinished: "none", want: []string{"reboot"}},
	// A slot resume may adopt a still-running agent.
	{state: ThreadUnusable, kind: ThreadTargetSlot, want: []string{"resume", "reboot"}},
	// A primary resumes only through its own recovery or request executor.
	{state: ThreadUnusable, kind: ThreadTargetOrdinary, recover: "yes", want: []string{"resume", "reboot"}},
	{state: ThreadUnusable, kind: ThreadTargetOrdinary, unfinished: "any", want: []string{"resume", "reboot"}},
	{state: ThreadUnusable, kind: ThreadTargetOrdinary, unfinished: "none", recover: "no", want: []string{"reboot"}},
}

func (r actorActionRule) matches(kind ThreadTargetKind, state ActionableThreadState, reason ThreadReason, unfinished checkpoint.Phase, recover bool) bool {
	if r.kind != "" && r.kind != kind || r.state != state || r.reason != "" && r.reason != reason {
		return false
	}
	pending := unfinished != "" && unfinished != checkpoint.Complete
	switch r.unfinished {
	case "any":
		if !pending {
			return false
		}
	case "none":
		if pending {
			return false
		}
	}
	switch r.recover {
	case "yes":
		return recover
	case "no":
		return !recover
	}
	return true
}

// TestActorActionsFollowsTheSpecOverTheDerivedDomain crosses every state,
// reason, kind, unfinished-request phase and recovery verdict (each from its
// own enumeration) and fails any combination the literal table does not
// cover, so a new state or reason cannot slip past the admission table.
func TestActorActionsFollowsTheSpecOverTheDerivedDomain(t *testing.T) {
	// No request, plus every phase from the one vocabulary (a complete request
	// is not unfinished).
	phases := append(make([]checkpoint.Phase, 1), checkpoint.AllPhases()...)
	for _, state := range AllThreadStates() {
		for _, reason := range AllThreadReasons() {
			for _, kind := range []ThreadTargetKind{ThreadTargetOrdinary, ThreadTargetSlot} {
				for _, phase := range phases {
					for _, recover := range []bool{false, true} {
						name := fmt.Sprintf("%s/%s/%s/%q/recover=%v", kind, state, reason, phase, recover)
						var rule *actorActionRule
						for i := range actorActionSpec {
							if actorActionSpec[i].matches(kind, state, reason, phase, recover) {
								rule = &actorActionSpec[i]
								break
							}
						}
						if rule == nil {
							t.Errorf("%s: no spec row covers this combination", name)
							continue
						}
						row := ActionableThreadSummary{State: state, Target: ThreadTarget{Kind: kind}}
						if state == ThreadUnusable {
							row.Reason = reason
						}
						if phase != "" {
							row.Continuation = &ContinuationStatus{Phase: phase}
						}
						if recover {
							row.Recovery = &RecoveryDecision{Recover: true}
						}
						if got := ActorActions(ActorRowFactsOf(row)); !slices.Equal(got, rule.want) {
							t.Errorf("%s: ActorActions = %v, want %v", name, got, rule.want)
						}
					}
				}
			}
		}
	}
}

// A completed request is not unfinished: it routes nothing.
func TestActorRowFactsIgnoresACompletedRequest(t *testing.T) {
	row := ActionableThreadSummary{State: ThreadUnusable, Reason: ReasonSessionGone, Target: ThreadTarget{Kind: ThreadTargetOrdinary},
		Continuation: &ContinuationStatus{Phase: checkpoint.Complete}}
	if got := ActorActions(ActorRowFactsOf(row)); !slices.Equal(got, []string{"reboot"}) {
		t.Fatalf("ActorActions = %v, want [reboot]", got)
	}
}
