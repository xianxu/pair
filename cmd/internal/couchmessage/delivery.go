package couchmessage

// Peer delivery is IO-free. Only the wrapper input owner executes effects.
type PeerDeliveryPhase string

const (
	PeerDeliveryWaiting       PeerDeliveryPhase = ""
	PeerDeliveryPasting       PeerDeliveryPhase = "pasting"
	PeerDeliveryPasted        PeerDeliveryPhase = "pasted"
	PeerDeliverySubmitting    PeerDeliveryPhase = "submitting"
	PeerDeliveryConfirming    PeerDeliveryPhase = "confirming"
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
	PeerConfirmObserved
)

// Ready is the adapter's verdict for the event's phase: on ComposerObserved,
// the pre-paste gates hold; on RenderObserved, the fixed post-paste delay has
// passed; on ConfirmObserved, agent-agnostic evidence shows the submission
// landed. Pair renders, never classifies (pair#427): nothing here inspects
// how the agent drew the pasted text.
type PeerDeliveryEvent struct {
	Kind              PeerDeliveryEventKind
	Ready             bool
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
		reason := map[PeerDeliveryEventKind]string{PeerOperatorInput: "operator input interrupted delivery", PeerImageInput: "image input interrupted delivery", PeerOverlayObserved: "dialog interrupted delivery", PeerDeadlineElapsed: "delivery deadline elapsed", PeerChildExited: "recipient exited"}[e.Kind]
		switch {
		case s.BodyWritten:
			// The text may sit in the recipient's composer or may have been
			// submitted: never a plain expired or cancelled (pair#427).
			after := " after the paste"
			if s.Phase == PeerDeliveryConfirming {
				after = " after the submit"
			}
			s.Phase = PeerDeliveryIndeterminate
			reason = "uncertain: " + reason + after
		case e.Kind == PeerDeadlineElapsed:
			s.Phase = PeerDeliveryExpired
		default:
			s.Phase = PeerDeliveryCancelled
		}
		s.Reason = reason
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
			s.Phase = PeerDeliveryPasted
			return s, PeerAwaitRender
		}
	case PeerDeliveryPasted:
		if e.Kind == PeerRenderObserved && e.Ready {
			s.Phase = PeerDeliverySubmitting
			return s, PeerSubmit
		}
	case PeerDeliverySubmitting:
		if e.Kind == PeerSubmitCompleted {
			if e.Failed || e.Expected <= 0 || e.Written != e.Expected {
				s.Phase = PeerDeliveryIndeterminate
				s.Reason = "uncertain: submit write incomplete; inspect recipient before retrying"
				return s, PeerPublish
			}
			s.Phase = PeerDeliveryConfirming
		}
	case PeerDeliveryConfirming:
		if e.Kind == PeerConfirmObserved && e.Ready {
			s.Phase = PeerDeliverySubmitted
			return s, PeerPublish
		}
	}
	return s, PeerNoEffect
}
