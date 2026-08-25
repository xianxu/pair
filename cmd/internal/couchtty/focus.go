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

// Up moves the focus one level toward couch: a child goes HOME to the root
// actor, the root actor goes to the panel, the panel stays.
//
// The child -> root-actor step is the property the whole project rests on. The
// obvious wrong version is "up = panel", which costs the operator a second
// keystroke every time they come home -- and they come home constantly, so a
// switcher that charges two keys for it is one they stop using. Richer
// navigation lives in the panel, where there is typeahead and a screen to read.
//
// alive is consulted rather than assumed: landing on a dead actor gives the
// operator a frozen screen with no way to tell it is frozen, which is worse
// than landing on the panel. Passed in rather than looked up so this stays pure
// -- liveness is the console's to know.
func Up(cur Focus, rootActor string, alive func(string) bool) Focus {
	if cur.IsPanel() {
		return FocusPanel()
	}
	// The root actor's own step is UP to the panel, including when it is the
	// only child -- otherwise couch's first session could never reach the
	// panel and the operator would have no way to start a second one.
	if cur.Actor() == rootActor {
		return FocusPanel()
	}
	if rootActor == "" || alive == nil || !alive(rootActor) {
		return FocusPanel()
	}
	return FocusActor(rootActor)
}
