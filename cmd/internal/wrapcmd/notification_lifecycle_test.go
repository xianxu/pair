package wrapcmd

import (
	"testing"
	"time"
)

func TestNotificationLifecycleRequiresAnOpener(t *testing.T) {
	state, decision := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationNativeCompletion, Message: "done"})
	if decision.Notify || state.Active {
		t.Fatalf("idle completion = state %+v decision %+v", state, decision)
	}
}

func TestNotificationLifecycleDeduplicatesSourcesWithinTurn(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	state, first := Reduce(state, TurnObservation{Kind: ObservationNativeCompletion, Message: "finished"})
	state, duplicate := Reduce(state, TurnObservation{Kind: ObservationMarkerCompletion, Message: "richer marker"})
	if !first.Notify || first.Message != "finished" {
		t.Fatalf("first completion = %+v", first)
	}
	if duplicate.Notify {
		t.Fatalf("same-turn duplicate notified: %+v", duplicate)
	}
	if !state.Completed {
		t.Fatalf("completion did not leave tombstone: %+v", state)
	}
}

func TestNotificationLifecycleRapidSubmissionsOpenDistinctTurns(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	state, first := Reduce(state, TurnObservation{Kind: ObservationNativeCompletion, Message: "one"})
	state, _ = Reduce(state, TurnObservation{Kind: ObservationUserSubmission})
	_, second := Reduce(state, TurnObservation{Kind: ObservationNativeCompletion, Message: "two"})
	if !first.Notify || !second.Notify {
		t.Fatalf("rapid turns = first %+v second %+v", first, second)
	}
}

func TestNotificationLifecycleCompletionSourcesNotifyAtMostOnceInEitherOrder(t *testing.T) {
	for _, order := range [][]ObservationKind{
		{ObservationNativeCompletion, ObservationMarkerCompletion},
		{ObservationMarkerCompletion, ObservationNativeCompletion},
	} {
		state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
		notifications := 0
		for _, kind := range order {
			var decision LifecycleDecision
			state, decision = Reduce(state, TurnObservation{Kind: kind, Message: "done"})
			if decision.Notify {
				notifications++
			}
		}
		if notifications != 1 {
			t.Fatalf("order %v emitted %d notifications", order, notifications)
		}
	}
}

func TestNotificationLifecycleKeyedTurnsRejectMismatchedTerminals(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "one"})
	state, wrong := Reduce(state, TurnObservation{Kind: ObservationTranscriptCompletion, TurnID: "two", Message: "wrong"})
	if wrong.Notify {
		t.Fatalf("mismatched terminal notified: %+v", wrong)
	}
	state, _ = Reduce(state, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "two"})
	_, right := Reduce(state, TurnObservation{Kind: ObservationTranscriptCompletion, TurnID: "two", Message: "right"})
	if !right.Notify || right.Message != "right" {
		t.Fatalf("new keyed turn completion = %+v", right)
	}
}

func TestNotificationLifecycleAbortHasDistinctFallbackOutcome(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "one"})
	_, aborted := Reduce(state, TurnObservation{Kind: ObservationTranscriptAbort, TurnID: "one"})
	if !aborted.Notify || aborted.Message == "" || aborted.Message == "agent finished working" {
		t.Fatalf("abort outcome = %+v", aborted)
	}
}

func TestNotificationLifecycleProgressStopWaitsForRicherMessage(t *testing.T) {
	state, started := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationWorking})
	if started.WatchdogToken == 0 {
		t.Fatalf("working did not arm watchdog: %+v", started)
	}
	state, stopped := Reduce(state, TurnObservation{Kind: ObservationStopped})
	if stopped.Notify || stopped.GraceToken == 0 {
		t.Fatalf("stop did not enter grace: %+v", stopped)
	}
	_, marker := Reduce(state, TurnObservation{Kind: ObservationMarkerCompletion, Message: "Sautéed for 34s"})
	if !marker.Notify || marker.Message != "Sautéed for 34s" {
		t.Fatalf("richer completion = %+v", marker)
	}
}

func TestNotificationLifecycleWorkingCancelsPendingStopGrace(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationWorking})
	state, stopped := Reduce(state, TurnObservation{Kind: ObservationStopped})
	state, resumed := Reduce(state, TurnObservation{Kind: ObservationWorking})
	if state.GracePending || state.GraceToken != 0 {
		t.Fatalf("resumed working retained grace: %+v", state)
	}
	if resumed.WatchdogToken == 0 {
		t.Fatalf("resumed working did not rearm watchdog: %+v", resumed)
	}
	_, stale := Reduce(state, TurnObservation{Kind: ObservationGraceExpired, Token: stopped.GraceToken})
	if stale.Notify {
		t.Fatalf("stale grace expiry notified after work resumed: %+v", stale)
	}
}

func TestNotificationLifecycleTranscriptStartMarksSubmittedTurnActive(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	state, started := Reduce(state, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "turn-1"})
	if state.TurnID != "turn-1" || !state.ActivitySeen {
		t.Fatalf("transcript start did not activate submitted turn: %+v", state)
	}
	if started.WatchdogToken == 0 {
		t.Fatalf("transcript start did not arm watchdog: %+v", started)
	}
}

func TestNotificationLifecycleTimersAreActivityGatedAndTokenized(t *testing.T) {
	idle, ignored := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationWatchdogExpired, Token: 1})
	if ignored.Notify || idle.Active {
		t.Fatalf("idle watchdog = %+v %+v", idle, ignored)
	}
	active, armed := Reduce(idle, TurnObservation{Kind: ObservationWorking})
	active, _ = Reduce(active, TurnObservation{Kind: ObservationUserSubmission})
	_, stale := Reduce(active, TurnObservation{Kind: ObservationWatchdogExpired, Token: armed.WatchdogToken})
	if stale.Notify {
		t.Fatalf("stale watchdog notified: %+v", stale)
	}
}

func TestProxyLifecycleUsesOneOwnedResettableTimer(t *testing.T) {
	p := &proxy{lifecycleTimer: time.NewTimer(time.Hour)}
	defer p.lifecycleTimer.Stop()
	p.processLifecycleObservation(TurnObservation{Kind: ObservationWorking})
	watchdog := p.lifecycleTimerToken
	if p.lifecycleTimerKind != ObservationWatchdogExpired || watchdog == 0 {
		t.Fatalf("working timer = kind %v token %d", p.lifecycleTimerKind, watchdog)
	}
	p.processLifecycleObservation(TurnObservation{Kind: ObservationStopped})
	if p.lifecycleTimerKind != ObservationGraceExpired || p.lifecycleTimerToken == 0 || p.lifecycleTimerToken == watchdog {
		t.Fatalf("stopped timer = kind %v token %d", p.lifecycleTimerKind, p.lifecycleTimerToken)
	}
	p.processLifecycleObservation(TurnObservation{Kind: ObservationMarkerCompletion, Message: "done"})
	if p.lifecycleTimerKind != ObservationUnknown || p.lifecycleTimerToken != 0 {
		t.Fatalf("completed timer = kind %v token %d", p.lifecycleTimerKind, p.lifecycleTimerToken)
	}
}

func FuzzNotificationLifecycleAtMostOncePerGeneration(f *testing.F) {
	f.Add([]byte{0, 2, 3, 4, 5})
	f.Add([]byte{6, 7, 8, 9, 1, 4})
	f.Add([]byte{1, 11, 11, 12, 5})
	f.Add([]byte{0, 10, 11, 10, 4}) // reaches IdleExpired + BareReturn as seeds
	f.Fuzz(func(t *testing.T, input []byte) {
		state := NotificationLifecycle{}
		// One completion per generation; and, since the idle floor is an alert
		// that leaves the turn open, at most one alert per idle EPOCH. Not per
		// generation: a bare return re-arms a spent floor inside the same turn
		// (BR-14), minting a fresh IdleToken, so the same generation may
		// legitimately alert again. The seeds caught exactly that.
		completed := make(map[uint64]bool)
		alerted := make(map[uint64]bool)
		for _, raw := range input {
			// Bound derived from the enum sentinel, so a kind added to
			// ObservationKind is covered here without editing this line.
			kind := ObservationKind(raw%byte(observationKindCount-1) + 1)
			turnID := "turn-a"
			if raw&0x80 != 0 {
				turnID = "turn-b"
			}
			observation := TurnObservation{Kind: kind, TurnID: turnID, Message: "done"}
			switch kind {
			case ObservationWatchdogExpired:
				observation.Token = state.WatchdogToken
			case ObservationGraceExpired:
				observation.Token = state.GraceToken
			case ObservationIdleExpired:
				observation.Token = state.IdleToken
			}
			var decision LifecycleDecision
			state, decision = Reduce(state, observation)
			if !decision.Notify {
				continue
			}
			if state.Generation == 0 {
				t.Fatal("notification emitted without an opened generation")
			}
			if kind == ObservationIdleExpired {
				// Tokens are monotonic, so the epoch is a unique key.
				if alerted[observation.Token] {
					t.Fatalf("idle epoch %d alerted more than once", observation.Token)
				}
				alerted[observation.Token] = true
				continue
			}
			if completed[state.Generation] {
				t.Fatalf("generation %d completed more than once", state.Generation)
			}
			completed[state.Generation] = true
		}
	})
}

// Walks every kind the sentinel declares, so a kind added to ObservationKind is
// exercised here without editing this test. Replaces an earlier guard that
// asserted observationKindCount == ObservationBareReturn+1 — which failed on
// exactly the change the sentinel exists to absorb (BR-4).
func TestReducePreservesTheGenerationInvariantForEveryObservationKind(t *testing.T) {
	if observationKindCount <= ObservationBareReturn {
		t.Fatalf("sentinel %d does not follow the declared kinds", observationKindCount)
	}
	reached := 0
	for kind := ObservationKind(1); kind < observationKindCount; kind++ {
		reached++
		open, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
		open, _ = Reduce(open, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "turn-a"})
		observation := TurnObservation{Kind: kind, TurnID: "turn-a", Message: "m"}
		switch kind {
		case ObservationWatchdogExpired:
			observation.Token = open.WatchdogToken
		case ObservationGraceExpired:
			observation.Token = open.GraceToken
		case ObservationIdleExpired:
			observation.Token = open.IdleToken
		}
		for _, from := range []NotificationLifecycle{{}, open} {
			next, decision := Reduce(from, observation)
			if decision.Notify && next.Generation == 0 {
				t.Fatalf("kind %d notified without an opened generation", kind)
			}
			if decision.Notify && decision.Message == "" {
				t.Fatalf("kind %d notified with an empty message", kind)
			}
		}
	}
	if reached != int(observationKindCount)-1 {
		t.Fatalf("walked %d kinds, want %d", reached, observationKindCount-1)
	}
}

// ----- Idle floor (#171) ------------------------------------------------------
//
// The floor's contract lives here, in the pure reducer, rather than in the
// master loop: whether a timeout reports a turn is a state-machine property,
// and Reduce is testable without goroutines, a timer, or a terminal.

func TestIdleFloorCoversTurnThatWasNeverRecognizedAsWorking(t *testing.T) {
	// The dominant real case: no progress OSC ever arrives, so ActivitySeen
	// stays false and the watchdog is never armed. The floor must still fire.
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	if state.ActivitySeen {
		t.Fatalf("submission should not claim activity: %+v", state)
	}
	if state.IdleToken == 0 {
		t.Fatalf("submission did not mint an idle epoch: %+v", state)
	}
	_, decision := Reduce(state, TurnObservation{
		Kind: ObservationIdleExpired, Token: state.IdleToken, Message: "no agent output for 60s",
	})
	if !decision.Notify || decision.Message != "no agent output for 60s" {
		t.Fatalf("idle expiry on unrecognized turn = %+v", decision)
	}
}

func TestIdleFloorIsSilentOnACompletedTurn(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	token := state.IdleToken
	state, done := Reduce(state, TurnObservation{Kind: ObservationMarkerCompletion, Message: "finished"})
	if !done.Notify {
		t.Fatalf("completion did not notify: %+v", done)
	}
	if state.IdleToken != 0 {
		t.Fatalf("completion left the idle epoch armed: %+v", state)
	}
	_, decision := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: token, Message: "no agent output for 60s"})
	if decision.Notify {
		t.Fatalf("idle expiry duplicated a completed turn: %+v", decision)
	}
}

func TestIdleFloorIsAnAlertSoARealCompletionStillNotifies(t *testing.T) {
	// The reason ObservationIdleExpired does not call complete(): a 60s silence
	// does not know the turn ended. If it set the Completed tombstone, the
	// agent's genuine end-of-turn would be swallowed and the operator would be
	// pulled in early and then never told it actually finished.
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	state, alert := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: state.IdleToken, Message: "no agent output for 60s"})
	if !alert.Notify {
		t.Fatalf("idle alert did not fire: %+v", alert)
	}
	if !state.Active || state.Completed {
		t.Fatalf("idle alert closed the turn: %+v", state)
	}
	_, finished := Reduce(state, TurnObservation{Kind: ObservationMarkerCompletion, Message: "agent finished working"})
	if !finished.Notify || finished.Message != "agent finished working" {
		t.Fatalf("real completion after an idle alert = %+v", finished)
	}
}

func TestIdleFloorRaisesAtMostOneAlertPerTurnAndRearmsOnTheNext(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	state, first := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: state.IdleToken, Message: "first"})
	state, second := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: state.IdleToken, Message: "second"})
	if !first.Notify || second.Notify {
		t.Fatalf("alerts per turn = first %+v second %+v", first, second)
	}
	// A new turn is a new floor.
	state, _ = Reduce(state, TurnObservation{Kind: ObservationUserSubmission})
	if state.IdleNotified {
		t.Fatalf("new turn inherited the spent floor: %+v", state)
	}
	_, next := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: state.IdleToken, Message: "next turn"})
	if !next.Notify {
		t.Fatalf("floor did not re-arm for the next turn: %+v", next)
	}
}

func TestIdleFloorRejectsExpiryFromAStaleEpoch(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	stale := state.IdleToken
	state, _ = Reduce(state, TurnObservation{Kind: ObservationMarkerCompletion, Message: "finished"})
	state, _ = Reduce(state, TurnObservation{Kind: ObservationUserSubmission})
	if state.IdleToken == stale {
		t.Fatalf("second turn reused the first turn's idle epoch: %+v", state)
	}
	for name, token := range map[string]uint64{"stale": stale, "zero": 0} {
		if _, decision := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: token, Message: "x"}); decision.Notify {
			t.Fatalf("%s token notified: %+v", name, decision)
		}
	}
}

func TestIdleFloorFallsBackToAMessageThatClaimsOnlySilence(t *testing.T) {
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	_, decision := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: state.IdleToken})
	if decision.Message != defaultIdleMessage {
		t.Fatalf("default idle message = %q, want %q", decision.Message, defaultIdleMessage)
	}
}

func TestBareReturnOpensATurnOnlyWhenNoneIsOpen(t *testing.T) {
	// A bare CR is the only submission signal when the composer gate reports
	// inactive, so it must open a turn — but answering a menu inside an open
	// turn must not reset that turn's identity or clear its tombstone.
	fresh, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationBareReturn})
	if !fresh.Active || fresh.IdleToken == 0 {
		t.Fatalf("bare return did not open a turn: %+v", fresh)
	}

	open, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	open, _ = Reduce(open, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "turn-1"})
	midTurn, decision := Reduce(open, TurnObservation{Kind: ObservationBareReturn})
	if decision.Notify {
		t.Fatalf("bare return mid-turn notified: %+v", decision)
	}
	if midTurn.Generation != open.Generation || midTurn.TurnID != "turn-1" || midTurn.IdleToken != open.IdleToken {
		t.Fatalf("bare return mid-turn disturbed the turn: got %+v want %+v", midTurn, open)
	}

	closed, _ := Reduce(open, TurnObservation{Kind: ObservationTranscriptCompletion, TurnID: "turn-1", Message: "done"})
	reopened, _ := Reduce(closed, TurnObservation{Kind: ObservationBareReturn})
	if !reopened.Active || reopened.Completed {
		t.Fatalf("bare return after completion did not open a new turn: %+v", reopened)
	}
}

func TestBareReturnRearmsAFloorThatAlreadyFired(t *testing.T) {
	// BR-14. Answering a menu is a new attention window. Without this the turn
	// that already alerted is left with no floor at all, while a mid-turn
	// Alt+Enter — which re-opens the turn — would have re-armed it.
	state, _ := Reduce(NotificationLifecycle{}, TurnObservation{Kind: ObservationUserSubmission})
	state, _ = Reduce(state, TurnObservation{Kind: ObservationTranscriptStarted, TurnID: "turn-a"})
	spent, alert := Reduce(state, TurnObservation{Kind: ObservationIdleExpired, Token: state.IdleToken, Message: "quiet"})
	if !alert.Notify || !spent.IdleNotified {
		t.Fatalf("floor did not fire: state %+v decision %+v", spent, alert)
	}

	rearmed, decision := Reduce(spent, TurnObservation{Kind: ObservationBareReturn})
	if decision.Notify {
		t.Fatalf("re-arm notified: %+v", decision)
	}
	if rearmed.IdleNotified || rearmed.IdleToken == 0 || rearmed.IdleToken == spent.IdleToken {
		t.Fatalf("floor not re-armed on a fresh epoch: %+v", rearmed)
	}
	if rearmed.Generation != spent.Generation || rearmed.TurnID != "turn-a" {
		t.Fatalf("re-arm disturbed the turn: got %+v want gen %d turn-a", rearmed, spent.Generation)
	}
	_, again := Reduce(rearmed, TurnObservation{Kind: ObservationIdleExpired, Token: rearmed.IdleToken, Message: "quiet again"})
	if !again.Notify {
		t.Fatalf("re-armed floor did not fire: %+v", again)
	}
}
