package couchtty

import "testing"

// The single most important property in the project: from anywhere inside a
// child, ONE key goes home. The easy wrong implementation is "up = panel",
// which would make the operator take two keys to reach the session they roam
// back to constantly -- and the whole design rests on getting home being free.
func TestUpFromANonRootChildGoesToTheRootActor(t *testing.T) {
	got := Up(FocusActor("worker"), "root", aliveExcept())
	if got != FocusActor("root") {
		t.Fatalf("Up(worker) = %v, want the root actor — not the panel", got)
	}
}

func TestUpFromTheRootActorGoesToThePanel(t *testing.T) {
	if got := Up(FocusActor("root"), "root", aliveExcept()); got != FocusPanel() {
		t.Fatalf("Up(root) = %v, want the panel", got)
	}
}

// The panel is the top. Pressing again must not cycle back into a child --
// "up" that wraps is a trapdoor, not a ladder.
func TestUpFromThePanelStays(t *testing.T) {
	if got := Up(FocusPanel(), "root", aliveExcept()); got != FocusPanel() {
		t.Fatalf("Up(panel) = %v, want the panel", got)
	}
}

// Landing on a dead actor is worse than landing on the panel: the operator gets
// a frozen screen with no way to tell it is frozen.
func TestUpSkipsADeadRootActor(t *testing.T) {
	if got := Up(FocusActor("worker"), "root", aliveExcept("root")); got != FocusPanel() {
		t.Fatalf("Up(worker) with a dead root = %v, want the panel", got)
	}
}

// With no root actor at all -- couch started, its first child already gone --
// there is nowhere to go but the panel.
func TestUpWithNoRootActorGoesToThePanel(t *testing.T) {
	if got := Up(FocusActor("worker"), "", aliveExcept()); got != FocusPanel() {
		t.Fatalf("Up(worker) with no root = %v, want the panel", got)
	}
}

// A child that IS the root actor takes the root branch, not the child branch --
// otherwise the very first session couch starts can never reach the panel.
func TestUpFromTheOnlyChildReachesThePanel(t *testing.T) {
	if got := Up(FocusActor("root"), "root", aliveExcept()); got != FocusPanel() {
		t.Fatalf("Up(root-as-only-child) = %v, want the panel", got)
	}
}

func TestFocusEquality(t *testing.T) {
	if FocusActor("a") == FocusActor("b") {
		t.Fatal("different actors compare equal")
	}
	if FocusPanel() == FocusActor("") {
		t.Fatal("the panel compares equal to an empty actor — a switch on Focus would confuse them")
	}
}

// aliveExcept builds the liveness predicate Up consults.
func aliveExcept(dead ...string) func(string) bool {
	gone := map[string]bool{}
	for _, d := range dead {
		gone[d] = true
	}
	return func(id string) bool { return !gone[id] }
}
