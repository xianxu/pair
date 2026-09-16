package terminalqualify

import "testing"

func TestInputCases(t *testing.T) {
	seen := map[string]Case{}
	for _, c := range InputCases() {
		if _, ok := seen[c.ID]; ok {
			t.Fatalf("duplicate ID %s", c.ID)
		}
		seen[c.ID] = c
		if c.ID == "" || c.Source == "" || len(c.Expected) == 0 || c.Uncovered != "" {
			t.Errorf("invalid executable fixture %s", c.ID)
		}
	}
	for _, id := range []string{"keyboard-query", "ctrl-return", "alt-up", "key-repeat", "key-release", "application-cursor", "application-keypad", "mouse-off", "mouse-click", "mouse-click-suppresses-motion", "mouse-drag", "mouse-all-motion", "mouse-replace-mode", "focus", "paste", "cursor-query", "status-query", "title", "cwd", "bell", "sync-query"} {
		if _, ok := seen[id]; !ok {
			t.Errorf("missing obligation %s", id)
		}
	}
	// These bytes come from the protocol, independent of the candidate encoder.
	if seen["ctrl-return"].Expected["replies"] != "\x1b[13;5u" {
		t.Fatal("Ctrl-Return oracle lost")
	}
	if seen["mouse-drag"].Expected["replies"] != "\x1b[<32;3;2M" {
		t.Fatal("SGR drag oracle lost")
	}
}

func TestCoverage(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range append(append(ScreenCases(), InputCases()...), Coverage()...) {
		if seen[c.ID] {
			t.Fatalf("duplicate global ID %s", c.ID)
		}
		seen[c.ID] = true
	}
	for _, c := range Coverage() {
		if c.Uncovered == "" || c.Source == "" || len(c.Expected) != 0 {
			t.Errorf("unobservable obligation claimed executable: %s", c.ID)
		}
	}
	for _, id := range []string{"composition-switch", "hidden-origin", "reply-backpressure-routing", "sync-publication", "sync-recovery", "parser-memory-bound", "wrapper-composition", "clipboard-policy", "notification-origin"} {
		if !seen[id] {
			t.Errorf("missing explicit scope gap %s", id)
		}
	}
}
