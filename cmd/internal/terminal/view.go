package terminal

import "fmt"

type ViewState uint8

const (
	Ready ViewState = iota
	Presenting
	Failed
	Released
)

// GestureOwner records who accepted the physical press. Parent gestures include
// chrome, panels, orphan motion, and canceled child gestures until release.
type GestureOwner uint8

const (
	GestureNone GestureOwner = iota
	GestureParent
	GestureChild
)

// View is the confirmed parent ownership, separate from a product's logical
// operation target. Empty Selected denotes the compositor's panel.
type View struct {
	State                               ViewState
	Selected, Admitted, DragDestination string
	Token, Generation, GeometryEpoch    uint64
	Gesture                             GestureOwner
	Button                              int
}
type ViewEventKind uint8

const (
	SelectView ViewEventKind = iota
	PresentView
	FailView
	ReleaseView
	PublishFrame
	PressMouse
	ReleaseMouse
	ParentPressMouse
	CancelMouse
)

type ViewEvent struct {
	Kind                             ViewEventKind
	Button                           int
	EndpointID                       string
	Token, Generation, GeometryEpoch uint64
}
type ViewEffects struct {
	CancelDrag     string
	Present, Admit bool
}

// Transition never performs IO. CancelDrag must be delivered before the
// presenter's new selection can be admitted; an unsuccessful write is FailView.
func Transition(v View, e ViewEvent) (View, ViewEffects, error) {
	var out ViewEffects
	reject := func() (View, ViewEffects, error) {
		return v, out, fmt.Errorf("terminal: inadmissible view event %d in state %d", e.Kind, v.State)
	}
	if v.State == Released {
		return reject()
	}
	cancel := func() {
		if v.DragDestination != "" {
			out.CancelDrag = v.DragDestination
			v.DragDestination = ""
			v.Gesture = GestureParent
		}
	}
	switch e.Kind {
	case SelectView:
		if v.State == Failed || e.Token <= v.Token {
			return reject()
		}
		cancel()
		v.State = Presenting
		v.Selected = e.EndpointID
		v.Admitted = ""
		v.Token = e.Token
		v.Generation = e.Generation
		v.GeometryEpoch = e.GeometryEpoch
		out.Present = true
	case PresentView:
		if v.State != Presenting || e.Token != v.Token || e.EndpointID != v.Selected || e.GeometryEpoch != v.GeometryEpoch || e.Generation < v.Generation {
			return reject()
		}
		v.State = Ready
		v.Generation = e.Generation
		v.Admitted = v.Selected
		out.Admit = v.Admitted != ""
	case PublishFrame:
		if v.State != Ready || e.EndpointID != v.Selected || e.GeometryEpoch != v.GeometryEpoch || e.Generation < v.Generation {
			return reject()
		}
		v.State = Presenting
		v.Admitted = ""
		v.Generation = e.Generation
		out.Present = true
	case FailView, ReleaseView:
		cancel()
		v.Admitted = ""
		v.State = Failed
		if e.Kind == ReleaseView {
			v.State = Released
		}
	case PressMouse:
		if v.State != Ready || v.Admitted == "" || v.Gesture != GestureNone {
			return reject()
		}
		v.DragDestination = v.Admitted
		v.Gesture = GestureChild
		v.Button = e.Button
	case ParentPressMouse:
		if v.State != Ready || v.Gesture != GestureNone {
			return reject()
		}
		v.Gesture = GestureParent
		v.Button = e.Button
	case CancelMouse:
		cancel()
	case ReleaseMouse:
		if v.Gesture != GestureNone && e.Button != 0 && v.Button != 0 && e.Button != v.Button {
			return reject()
		}
		v.DragDestination = ""
		v.Gesture = GestureNone
		v.Button = 0
	default:
		return reject()
	}
	return v, out, nil
}
