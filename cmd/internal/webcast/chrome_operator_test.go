package webcast

import "testing"

func TestOperatorTargetGuardSurvivesAnUncertainInput(t *testing.T) {
	c, p := startPeer(t, nil)
	_ = p
	if err := c.uiGuard("S1", "acquire"); err != nil {
		t.Fatal(err)
	}
	if err := c.uiGuard("S1", "unknown"); err != nil {
		t.Fatal(err)
	}
	if err := c.uiGuard("S1", "acquire"); err == nil {
		t.Fatal("uncertain input allowed another batch")
	}
	waitState(t, c, "recovery instruction", func(s State) bool { return s.Recovery != "" })
}

func TestOperatorModeRejectsInvalidOrUnavailableWithoutFocus(t *testing.T) {
	c, _ := startPeer(t, nil)
	if err := c.SetAnnotationMode("bogus"); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if err := c.SetAnnotationMode("point"); err == nil {
		t.Fatal("unrecorded annotation accepted")
	}
}
