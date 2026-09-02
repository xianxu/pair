package couchtty

// Focus is where the operator's terminal is pointed: at one actor, or at
// couch's panel.
//
// A comparable value type, not an interface, so a caller can `==` two focuses
// and switch on one. The zero value is the panel, which is the safe default:
// couch with nothing attached shows the operator a list rather than a blank
// screen.
type Focus struct {
	// kind is what makes FocusActor("") distinguishable from FocusPanel().
	//
	// Without it the two compare EQUAL, so a bug that produced an empty actor
	// id would silently become "show the panel" -- a wrong screen that looks
	// deliberate. With it, the zero value is still the panel (the safe default
	// for a console with nothing attached) while an empty-id actor stays a
	// detectable state.
	kind  focusKind
	actor string
}

type focusKind uint8

const (
	focusPanel focusKind = iota
	focusActor
)

// FocusPanel is couch's own screen.
func FocusPanel() Focus { return Focus{kind: focusPanel} }

// FocusActor is one hosted session.
func FocusActor(id string) Focus { return Focus{kind: focusActor, actor: id} }

// IsPanel reports whether the focus is couch's panel.
func (f Focus) IsPanel() bool { return f.kind == focusPanel }

// Actor returns the focused actor's id, empty for the panel.
func (f Focus) Actor() string { return f.actor }

func (f Focus) String() string {
	if f.IsPanel() {
		return "panel"
	}
	return "actor:" + f.actor
}
