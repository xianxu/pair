package slugline

import "testing"

func TestValid(t *testing.T) {
	good := []string{
		"=== pair | doing tests ===",
		"=== #42 winbar-recap | wiring the recap ===",
	}
	bad := []string{
		"KEEP",
		"=== pair ===",           // no pipe
		"=== | right ===",        // empty left
		"=== left | ===",         // empty right
		"pair | doing tests",     // no fence
		"Sandbox restriction...", // hijack garbage
		"",
	}
	for _, s := range good {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if Valid(s) {
			t.Errorf("Valid(%q) = true, want false", s)
		}
	}
}

func TestFormatRoundTrips(t *testing.T) {
	line := Format("main-slot3", "couch switcher click-select feature")
	if line != "=== main-slot3 | couch switcher click-select feature ===" {
		t.Fatalf("Format = %q", line)
	}
	if !Valid(line) {
		t.Fatalf("Format output %q is not Valid", line)
	}
	if got := Focus(line); got != "couch switcher click-select feature" {
		t.Errorf("Focus = %q", got)
	}
	if got := Unfenced(line); got != "main-slot3 | couch switcher click-select feature" {
		t.Errorf("Unfenced = %q", got)
	}
}

func TestFocus(t *testing.T) {
	if got := Focus("=== a | b c d ==="); got != "b c d" {
		t.Errorf("Focus = %q", got)
	}
}
