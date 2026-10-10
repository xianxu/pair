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
