package couchmessage

import (
	"strings"
	"testing"
)

func pastedPeerState(t *testing.T) PeerDeliveryState {
	t.Helper()
	s, fx := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
	if fx != PeerPaste {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerPasteCompleted, Written: 12, Expected: 12})
	if fx != PeerAwaitRender || !s.BodyWritten {
		t.Fatal(s, fx)
	}
	return s
}

// pair#427: no render match gates the submit. The fixed delay (Ready) does,
// and the outcome waits for after-the-fact confirmation.
func TestPeerDeliverySubmitsAfterDelayThenConfirms(t *testing.T) {
	s := pastedPeerState(t)
	s, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved})
	if fx != PeerNoEffect {
		t.Fatal("submitted before the delay", s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Ready: true})
	if fx != PeerSubmit {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerSubmitCompleted, Written: 1, Expected: 1})
	if fx != PeerNoEffect || s.Phase != PeerDeliveryConfirming {
		t.Fatal("a submit write is not yet a landed submission", s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerConfirmObserved})
	if fx != PeerNoEffect || s.Terminal() {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerConfirmObserved, Ready: true})
	if fx != PeerPublish || s.Outcome() != Submitted {
		t.Fatal(s, fx)
	}
}

// Once text may be in the recipient's composer, every stop is uncertain,
// never a plain expired or cancelled (pair#427 Spec 2).
func TestPeerDeliveryStopsAfterPasteAreUncertain(t *testing.T) {
	stops := []PeerDeliveryEventKind{PeerOperatorInput, PeerImageInput, PeerOverlayObserved, PeerDeadlineElapsed, PeerChildExited}
	for _, phase := range []string{"pasted", "confirming"} {
		for _, kind := range stops {
			s := pastedPeerState(t)
			if phase == "confirming" {
				s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Ready: true})
				s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerSubmitCompleted, Written: 1, Expected: 1})
			}
			s, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: kind})
			if s.Outcome() != Indeterminate || fx != PeerPublish || !strings.HasPrefix(s.Reason, "uncertain: ") || !strings.HasSuffix(s.Reason, " after the "+map[string]string{"pasted": "paste", "confirming": "submit"}[phase]) {
				t.Fatal(phase, kind, s, fx)
			}
			next, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerConfirmObserved, Ready: true})
			if next != s || fx != PeerNoEffect {
				t.Fatal(phase, kind, next, fx)
			}
		}
	}
}

func TestPeerDeliveryStopsBeforePasteKeepTheirOutcome(t *testing.T) {
	want := map[PeerDeliveryEventKind]Status{PeerOperatorInput: Cancelled, PeerImageInput: Cancelled, PeerOverlayObserved: Cancelled, PeerDeadlineElapsed: Expired, PeerChildExited: Cancelled}
	for kind, outcome := range want {
		s, fx := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: kind})
		if s.Outcome() != outcome || fx != PeerPublish || s.BodyWritten {
			t.Fatal(kind, s, fx)
		}
	}
}

func TestPeerDeliveryPartialWritesNeverRetry(t *testing.T) {
	for _, submit := range []bool{false, true} {
		s, _ := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
		kind := PeerPasteCompleted
		if submit {
			s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerPasteCompleted, Written: 12, Expected: 12})
			s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Ready: true})
			kind = PeerSubmitCompleted
		}
		s, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: kind, Written: 1, Expected: 12, Failed: true})
		if s.Phase != PeerDeliveryIndeterminate || fx != PeerPublish {
			t.Fatal(s, fx)
		}
		next, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
		if next != s || fx != PeerNoEffect {
			t.Fatal(next, fx)
		}
	}
}

func TestPeerDeliveryIgnoresOutOfOrderEvents(t *testing.T) {
	for _, e := range []PeerDeliveryEvent{{Kind: PeerRenderObserved, Ready: true}, {Kind: PeerSubmitCompleted, Written: 1, Expected: 1}, {Kind: PeerConfirmObserved, Ready: true}, {Kind: PeerComposerObserved}} {
		s, fx := AdvancePeerDelivery(PeerDeliveryState{}, e)
		if s != (PeerDeliveryState{}) || fx != PeerNoEffect {
			t.Fatal(s, fx)
		}
	}
}

func TestPeerDeliveryRetrySubmitBudget(t *testing.T) {
	s, fx := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
	pastes, submits := 0, 0
	count := func(fx PeerDeliveryEffect) {
		if fx == PeerPaste {
			pastes++
		}
		if fx == PeerSubmit {
			submits++
		}
	}
	count(fx)
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerPasteCompleted, Written: 12, Expected: 12})
	count(fx)
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Ready: true})
	count(fx)
	for attempt := 1; attempt <= 5; attempt++ {
		if s.SubmitAttempts != submits {
			t.Fatalf("attempt count=%d, actual submit effects=%d", s.SubmitAttempts, submits)
		}
		s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerSubmitCompleted, Written: 1, Expected: 1})
		count(fx)
		before := s
		s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRetrySubmit})
		if s != before || fx != PeerNoEffect {
			t.Fatal("unauthorized retry", s, fx)
		}
		s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRetrySubmit, Ready: true})
		count(fx)
		if attempt < 3 && (fx != PeerSubmit || s.Phase != PeerDeliverySubmitting) {
			t.Fatal("retry missing", s, fx)
		}
		if attempt >= 3 && fx != PeerNoEffect {
			t.Fatal("budget exceeded", s, fx)
		}
	}
	if pastes != 1 || submits != 3 {
		t.Fatalf("paste=%d submit=%d", pastes, submits)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerConfirmObserved, Ready: true})
	if s.Outcome() != Submitted || fx != PeerPublish {
		t.Fatal(s, fx)
	}
	assertPeerTerminalNoEffects(t, s)
}

func assertPeerTerminalNoEffects(t *testing.T, s PeerDeliveryState) {
	t.Helper()
	for kind := PeerComposerObserved; kind <= PeerAutomaticInputBlocked; kind++ {
		next, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: kind, Ready: true, Written: 1, Expected: 1})
		if next != s || fx != PeerNoEffect {
			t.Fatalf("terminal event %d: %+v %v", kind, next, fx)
		}
	}
}

func TestPeerDeliveryRetryOnlyWhileConfirming(t *testing.T) {
	for _, phase := range []PeerDeliveryPhase{PeerDeliveryWaiting, PeerDeliveryPasting, PeerDeliveryPasted, PeerDeliverySubmitting} {
		s := PeerDeliveryState{Phase: phase}
		next, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRetrySubmit, Ready: true})
		if next != s || fx != PeerNoEffect {
			t.Fatal(phase, next, fx)
		}
	}
}

func TestPeerDeliveryRetryStopsAndShortWrites(t *testing.T) {
	for _, kind := range []PeerDeliveryEventKind{PeerOperatorInput, PeerImageInput, PeerOverlayObserved, PeerDeadlineElapsed, PeerChildExited, PeerSubmitCompleted} {
		for attempt := 1; attempt <= 3; attempt++ {
			s := pastedPeerState(t)
			s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Ready: true})
			for n := 1; n < attempt; n++ {
				s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerSubmitCompleted, Written: 1, Expected: 1})
				s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRetrySubmit, Ready: true})
			}
			s, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: kind, Written: 0, Expected: 1})
			if s.Outcome() != Indeterminate || fx != PeerPublish {
				t.Fatal(kind, attempt, s, fx)
			}
			assertPeerTerminalNoEffects(t, s)
		}
	}
}

func TestPeerDeliveryBlockedAutomaticInputRefusesBeforePaste(t *testing.T) {
	s, fx := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerAutomaticInputBlocked, Ready: true})
	if s.Outcome() != Cancelled || fx != PeerPublish || s.BodyWritten || s.Reason != "composer holds an unsubmitted automatic message; press Alt+Enter or clear it" {
		t.Fatal(s, fx)
	}
	assertPeerTerminalNoEffects(t, s)
	for _, phase := range []PeerDeliveryPhase{PeerDeliveryPasting, PeerDeliveryPasted, PeerDeliverySubmitting, PeerDeliveryConfirming} {
		before := PeerDeliveryState{Phase: phase, BodyWritten: true}
		next, fx := AdvancePeerDelivery(before, PeerDeliveryEvent{Kind: PeerAutomaticInputBlocked})
		if next != before || fx != PeerNoEffect {
			t.Fatal("replaced existing interruption policy", next, fx)
		}
	}
}

// Generated event sequences count observable effects independently of reducer state.
func TestPeerDeliveryGeneratedEventSequences(t *testing.T) {
	for seed := uint32(1); seed <= 1000; seed++ {
		s := PeerDeliveryState{}
		pastes, submits := 0, 0
		rng := seed
		for step := 0; step < 100; step++ {
			rng = 1664525*rng + 1013904223
			e := PeerDeliveryEvent{Kind: PeerDeliveryEventKind(1 + rng%uint32(PeerAutomaticInputBlocked)), Ready: rng&16 != 0, Written: 1, Expected: 1}
			// Include successful delivery prefixes before adversarial events.
			if step < 4 {
				e = PeerDeliveryEvent{Kind: []PeerDeliveryEventKind{PeerComposerObserved, PeerPasteCompleted, PeerRenderObserved, PeerSubmitCompleted}[step], Ready: true, Written: 1, Expected: 1}
			}
			if rng&32 != 0 && step >= 4 {
				e.Written = 0
			}
			before := s
			next, fx := AdvancePeerDelivery(s, e)
			if s.Terminal() && (next != before || fx != PeerNoEffect) {
				t.Fatalf("seed=%d step=%d terminal mutation", seed, step)
			}
			if fx == PeerPaste {
				pastes++
			}
			if fx == PeerSubmit {
				submits++
			}
			if pastes > 1 || submits > 3 || next.SubmitAttempts != submits {
				t.Fatalf("seed=%d step=%d paste=%d submit=%d state=%+v", seed, step, pastes, submits, next)
			}
			s = next
		}
	}
}

func TestPeerAutomaticInputBlockedPreservesUnknownOutcome(t *testing.T) {
	s, fx := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerAutomaticInputBlocked})
	if fx != PeerPublish || s.Outcome() != Cancelled || !strings.Contains(s.Reason, "unconfirmed") {
		t.Fatalf("unknown residue claimed unsubmitted: %+v", s)
	}
}
