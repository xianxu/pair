package couchtty

import "testing"

// The invariant focus.go's own comment calls load-bearing: FocusPanel() and
// FocusActor("") must not compare equal, so a bug that produced an empty actor
// id becomes a detectable state rather than silently meaning "show the panel" --
// a wrong screen that looks deliberate.
//
// Three live comparisons depend on it, including alt+x's panel branch. This
// test survived the deletion of the focus ladder (#170 retired `Up`); it was
// dropped with focus_test.go by accident and is restored here, because `Focus`
// itself is unchanged and still the console's focus authority.
func TestFocusEquality(t *testing.T) {
	if FocusPanel() == FocusActor("") {
		t.Fatal("FocusPanel() == FocusActor(\"\") -- an empty actor id would silently mean the panel")
	}
	if FocusPanel() != (Focus{}) {
		t.Fatal("the zero Focus must be the panel: a console with nothing attached shows a list, not a blank screen")
	}
	if !FocusPanel().IsPanel() || FocusActor("c1").IsPanel() {
		t.Fatal("IsPanel does not distinguish the panel from an actor")
	}
	if got := FocusActor("c1").Actor(); got != "c1" {
		t.Fatalf("Actor() = %q, want c1", got)
	}
	if FocusActor("c1") == FocusActor("c2") {
		t.Fatal("distinct actors compare equal")
	}
	if got, want := FocusActor("c1").String(), "actor:c1"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got, want := FocusPanel().String(), "panel"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
