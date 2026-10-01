package couchmessage

// Peer delivery is IO-free. Only the wrapper input owner executes effects.
type PeerDeliveryPhase string

const (
	PeerDeliveryWaiting       PeerDeliveryPhase = ""
	PeerDeliveryPasting       PeerDeliveryPhase = "pasting"
	PeerDeliveryRendering     PeerDeliveryPhase = "rendering"
	PeerDeliverySubmitting    PeerDeliveryPhase = "submitting"
	PeerDeliverySubmitted     PeerDeliveryPhase = "submitted"
	PeerDeliveryCancelled     PeerDeliveryPhase = "cancelled"
	PeerDeliveryExpired       PeerDeliveryPhase = "expired"
	PeerDeliveryIndeterminate PeerDeliveryPhase = "indeterminate"
	PeerDeliveryFailed        PeerDeliveryPhase = "failed"
)

type PeerDeliveryState struct {
	Phase       PeerDeliveryPhase
	BodyWritten bool
	Reason      string
}

func (s PeerDeliveryState) Terminal() bool { return s.Outcome() != "" }
func (s PeerDeliveryState) Outcome() Status {
	switch s.Phase {
	case PeerDeliverySubmitted:
		return Submitted
	case PeerDeliveryExpired:
		return Expired
	case PeerDeliveryCancelled, PeerDeliveryFailed:
		return Cancelled
	case PeerDeliveryIndeterminate:
		return Indeterminate
	}
	return ""
}

type PeerDeliveryEventKind uint8

const (
	PeerComposerObserved PeerDeliveryEventKind = iota + 1
	PeerPasteCompleted
	PeerRenderObserved
	PeerSubmitCompleted
	PeerOperatorInput
	PeerImageInput
	PeerOverlayObserved
	PeerDeadlineElapsed
	PeerChildExited
)

// Matches is positive evidence from a fresh post-paste render. The adapter
// owns render generations and arbitration; a pre-paste snapshot cannot match.
type PeerDeliveryEvent struct {
	Kind              PeerDeliveryEventKind
	Ready, Matches    bool
	Written, Expected int
	Failed            bool
}
type PeerDeliveryEffect uint8

const (
	PeerNoEffect PeerDeliveryEffect = iota
	PeerPaste
	PeerAwaitRender
	PeerSubmit
	PeerPublish
)

func AdvancePeerDelivery(s PeerDeliveryState, e PeerDeliveryEvent) (PeerDeliveryState, PeerDeliveryEffect) {
	if s.Terminal() {
		return s, PeerNoEffect
	}
	switch e.Kind {
	case PeerOperatorInput, PeerImageInput, PeerOverlayObserved, PeerDeadlineElapsed, PeerChildExited:
		s.Phase = PeerDeliveryCancelled
		if e.Kind == PeerDeadlineElapsed {
			s.Phase = PeerDeliveryExpired
		}
		s.Reason = map[PeerDeliveryEventKind]string{PeerOperatorInput: "operator input interrupted delivery", PeerImageInput: "image input interrupted delivery", PeerOverlayObserved: "dialog interrupted delivery", PeerDeadlineElapsed: "delivery deadline elapsed", PeerChildExited: "recipient exited"}[e.Kind]
		return s, PeerPublish
	}
	switch s.Phase {
	case PeerDeliveryWaiting:
		if e.Kind == PeerComposerObserved && e.Ready {
			s.Phase = PeerDeliveryPasting
			return s, PeerPaste
		}
	case PeerDeliveryPasting:
		if e.Kind == PeerPasteCompleted {
			s.BodyWritten = e.Written > 0
			if e.Failed || e.Expected <= 0 || e.Written != e.Expected {
				s.Phase = PeerDeliveryFailed
				if s.BodyWritten {
					s.Phase = PeerDeliveryIndeterminate
				}
				s.Reason = "paste incomplete; inspect composer before retrying"
				return s, PeerPublish
			}
			s.Phase = PeerDeliveryRendering
			return s, PeerAwaitRender
		}
	case PeerDeliveryRendering:
		if e.Kind == PeerRenderObserved && e.Ready && e.Matches {
			s.Phase = PeerDeliverySubmitting
			return s, PeerSubmit
		}
	case PeerDeliverySubmitting:
		if e.Kind == PeerSubmitCompleted {
			s.Phase = PeerDeliverySubmitted
			if e.Failed || e.Expected <= 0 || e.Written != e.Expected {
				s.Phase = PeerDeliveryIndeterminate
				s.Reason = "submission uncertain; inspect recipient before retrying"
			}
			return s, PeerPublish
		}
	}
	return s, PeerNoEffect
}
