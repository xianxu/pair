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
	f.Fuzz(func(t *testing.T, input []byte) {
		state := NotificationLifecycle{}
		notified := make(map[uint64]bool)
		for _, raw := range input {
			kind := ObservationKind(raw%byte(ObservationGraceExpired) + 1)
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
			}
			var decision LifecycleDecision
			state, decision = Reduce(state, observation)
			if !decision.Notify {
				continue
			}
			if state.Generation == 0 {
				t.Fatal("notification emitted without an opened generation")
			}
			if notified[state.Generation] {
				t.Fatalf("generation %d notified more than once", state.Generation)
			}
			notified[state.Generation] = true
		}
	})
}
