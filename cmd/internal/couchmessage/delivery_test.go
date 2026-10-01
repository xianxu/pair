package couchmessage

import "testing"

func TestPeerDeliveryRequiresMatchingRender(t *testing.T) {
	s, fx := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
	if fx != PeerPaste {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerPasteCompleted, Written: 12, Expected: 12})
	if fx != PeerAwaitRender || !s.BodyWritten {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Matches: false})
	if fx != PeerNoEffect {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Matches: true, Ready: true})
	if fx != PeerSubmit {
		t.Fatal(s, fx)
	}
	s, fx = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerSubmitCompleted, Written: 1, Expected: 1})
	if fx != PeerPublish || s.Phase != PeerDeliverySubmitted {
		t.Fatal(s, fx)
	}
}

func TestPeerDeliveryCancellationNeverResurrects(t *testing.T) {
	for _, kind := range []PeerDeliveryEventKind{PeerOperatorInput, PeerImageInput, PeerOverlayObserved, PeerDeadlineElapsed, PeerChildExited} {
		s, _ := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
		s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerPasteCompleted, Written: 12, Expected: 12})
		s, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: kind})
		if (s.Phase != PeerDeliveryCancelled && s.Phase != PeerDeliveryExpired) || !s.BodyWritten || fx != PeerPublish {
			t.Fatal(kind, s, fx)
		}
		next, fx := AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Matches: true, Ready: true})
		if next != s || fx != PeerNoEffect {
			t.Fatal(kind, next, fx)
		}
	}
}

func TestPeerDeliveryPartialWritesNeverRetry(t *testing.T) {
	for _, submit := range []bool{false, true} {
		s, _ := AdvancePeerDelivery(PeerDeliveryState{}, PeerDeliveryEvent{Kind: PeerComposerObserved, Ready: true})
		kind := PeerPasteCompleted
		if submit {
			s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerPasteCompleted, Written: 12, Expected: 12})
			s, _ = AdvancePeerDelivery(s, PeerDeliveryEvent{Kind: PeerRenderObserved, Ready: true, Matches: true})
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
	for _, e := range []PeerDeliveryEvent{{Kind: PeerRenderObserved, Ready: true, Matches: true}, {Kind: PeerSubmitCompleted, Written: 1, Expected: 1}, {Kind: PeerComposerObserved}} {
		s, fx := AdvancePeerDelivery(PeerDeliveryState{}, e)
		if s != (PeerDeliveryState{}) || fx != PeerNoEffect {
			t.Fatal(s, fx)
		}
	}
}
