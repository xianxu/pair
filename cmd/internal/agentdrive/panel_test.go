package agentdrive

import (
	"os"
	"strings"
	"testing"
	"time"
)

func claude2304(t *testing.T, name string) []string {
	t.Helper()
	b, err := os.ReadFile("../wrapcmd/testdata/drive/claude/2.1.304/" + name + ".screen")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(b), "\n")
}

// usagePanel is the hub's live /usage capture: 39 rows, the panel below a rule
// with the earlier prompt row above it.
func usagePanel(t *testing.T) []string { return claude2304(t, "usage-panel") }

func TestDismissablePanelRecognizesTheUsagePanel(t *testing.T) {
	// The tall capture has scrolled its rule and prompt row off the screen.
	for _, name := range []string{"usage-panel", "usage-panel-tall"} {
		if !DismissablePanel("claude", claude2304(t, name)) {
			t.Errorf("%s not recognized", name)
		}
	}
}

// workingTurnScreens are real Claude 2.1.304 working-turn viewports, and the
// same screens carrying the footer words anywhere but as a panel's last row.
// Esc on any of them would interrupt real work (pair#691).
func workingTurnScreens(t *testing.T) map[string][]string {
	t.Helper()
	screens := map[string][]string{}
	for _, name := range []string{"working-thinking", "working-after-usage", "working-shell"} {
		lines := claude2304(t, name)
		screens[name] = lines
		inHistory := append([]string{}, lines...)
		inHistory[3] = "  Esc to cancel"
		screens[name+", footer words in history"] = inHistory
		// Walk back past the composer's closing rule to its opening rule, and
		// put the words on the row just above the box.
		rule := len(lines) - 1
		for rule > 0 && !strings.HasPrefix(lines[rule], "─") {
			rule--
		}
		rule--
		for rule > 0 && !strings.HasPrefix(lines[rule], "─") {
			rule--
		}
		if !strings.HasPrefix(lines[rule+1], "❯") {
			t.Fatalf("%s: row after the opening rule is not the prompt: %q", name, lines[rule+1])
		}
		aboveComposer := append(append(append([]string{}, lines[1:rule]...), "  Esc to cancel"), lines[rule:]...)
		screens[name+", footer words above the composer"] = aboveComposer
	}
	return screens
}

func TestDismissablePanelRefusesEveryWorkingTurn(t *testing.T) {
	for name, lines := range workingTurnScreens(t) {
		if DismissablePanel("claude", lines) {
			t.Errorf("%s: recognized as a dismissable panel", name)
		}
	}
}

func TestDismissablePanelDeclinesEverythingElse(t *testing.T) {
	panel := usagePanel(t)
	rule := strings.Repeat("─", 94)
	composer := []string{"previous answer", rule, "❯ draft", rule, "  ? for shortcuts"}
	permission := []string{rule, "  Bash command", "    rm -rf build", "  Do you want to proceed?", "  ❯ 1. Yes", "    2. No", "", "  Esc to cancel"}
	for name, c := range map[string]struct {
		agent string
		lines []string
	}{
		"composer":               {"claude", composer},
		"closed panel":           {"claude", claude2304(t, "usage-closed")},
		"model picker":           {"claude", claudeCapture(t, "model-picker")},
		"permission prompt":      {"claude", permission},
		"numbered row in panel":  {"claude", append(append([]string{}, panel[:len(panel)-2]...), "    1. Default", "  Esc to cancel")},
		"footer only in history": {"claude", append(append([]string{}, panel...), composer...)},
		"footer then other text": {"claude", append(append([]string{}, panel...), "  thinking…")},
		"other agent":            {"codex", panel},
		"unbalanced markup":      {"claude", append([]string{"‹rev›Usage"}, panel...)},
		"empty":                  {"claude", nil},
	} {
		if DismissablePanel(c.agent, c.lines) {
			t.Errorf("%s: recognized as a dismissable panel", name)
		}
	}
}

func TestDismissOnlyIsExactlyOneEsc(t *testing.T) {
	if !(Request{Keys: KeySequence{{Kind: KeyEsc}}}).DismissOnly() {
		t.Fatal("[esc] refused")
	}
	for _, r := range []Request{
		{},
		{Keys: KeySequence{{Kind: KeyEsc}, {Kind: KeyEsc}}},
		{Keys: KeySequence{{Kind: KeyClearLine}}},
		{Keys: KeySequence{{Kind: KeyEsc}, {Kind: KeyClearLine}}},
		{Keys: KeySequence{{Kind: KeyEsc}, {Kind: KeyText, Text: "x"}}},
		{Keys: KeySequence{{Kind: KeySubmit}}},
		{Keys: KeySequence{{Kind: KeyEsc, Text: "extra"}}},
		{Model: &ModelRequest{Model: "opus 5.5"}},
	} {
		if r.DismissOnly() {
			t.Errorf("dismissal admitted %#v", r)
		}
	}
}

func TestAdmitsIsTheOneAdmissionRule(t *testing.T) {
	esc := Request{Keys: KeySequence{{Kind: KeyEsc}}}
	clear := Request{Keys: KeySequence{{Kind: KeyClearLine}}}
	text := Request{Keys: KeySequence{{Kind: KeyText, Text: "x"}}}
	panel := Admission{Dismissable: true}
	for _, c := range []struct {
		name string
		r    Request
		a    Admission
		want bool
	}{
		{"esc on panel", esc, panel, true},
		{"clear-line on panel", clear, panel, false},
		{"text on panel", text, panel, false},
		{"esc then clear on panel", Request{Keys: KeySequence{{Kind: KeyEsc}, {Kind: KeyClearLine}}}, panel, false},
		{"text when idle", text, Admission{Idle: true}, true},
		{"clear-line on occupied composer", clear, Admission{Recoverable: true}, true},
		{"text on occupied composer", text, Admission{Recoverable: true}, false},
		{"esc with no proof", esc, Admission{}, false},
	} {
		if got := c.r.Admits(c.a); got != c.want {
			t.Errorf("%s: Admits = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDriveDismissalEndsDoneOrUnconfirmed(t *testing.T) {
	now := time.Unix(100, 0)
	esc := Request{Keys: KeySequence{{Kind: KeyEsc}}}
	start := func(t *testing.T, screen Screen) DriveState {
		t.Helper()
		s, _ := NewDrive(esc, "claude", now.Add(time.Minute))
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: screen})
		if s.Terminal() || len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
			t.Fatalf("esc refused on %+v: %#v", screen, s.Result())
		}
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 2}})
		return s
	}
	panel := Screen{Epoch: 1, Complete: true, Dismissable: true}

	t.Run("composer returns", func(t *testing.T) {
		s := start(t, panel)
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Dismissable: true}})
		if s.Terminal() {
			t.Fatalf("a panel repaint ended the dismissal: %#v", s.Result())
		}
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Idle: true}})
		if s.Result().Outcome != OutcomeConfirmed {
			t.Fatalf("composer return %#v", s.Result())
		}
	})
	t.Run("panel persists", func(t *testing.T) {
		s := start(t, panel)
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Dismissable: true, Lines: []string{"Esc to cancel"}}})
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(time.Minute), Screen: Screen{Epoch: 3, Complete: true, Dismissable: true, Lines: []string{"Esc to cancel"}}})
		r := s.Result()
		if r.Outcome != OutcomeUnconfirmed || !strings.Contains(r.Detail, "panel still shows") || len(r.Tail) == 0 || len(fx) != 0 {
			t.Fatalf("persisting panel %#v %#v", r, fx)
		}
	})
	t.Run("unrecognized frame", func(t *testing.T) {
		s := start(t, panel)
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true}})
		if s.Terminal() {
			t.Fatalf("an unknown frame confirmed the dismissal: %#v", s.Result())
		}
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(time.Minute), Screen: Screen{Epoch: 3, Complete: true}})
		if s.Result().Outcome != OutcomeUnconfirmed {
			t.Fatalf("deadline %#v", s.Result())
		}
	})
	t.Run("esc on an occupied composer keeps the raw rule", func(t *testing.T) {
		s := start(t, Screen{Epoch: 1, Complete: true, Recoverable: true})
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Recoverable: true}})
		if s.Result().Outcome != OutcomeConfirmed {
			t.Fatalf("raw esc %#v", s.Result())
		}
	})
	t.Run("other keys refuse on a panel", func(t *testing.T) {
		for _, keys := range []KeySequence{{{Kind: KeyClearLine}}, {{Kind: KeyEsc}, {Kind: KeyEsc}}, {{Kind: KeyText, Text: "x"}}, {{Kind: KeySubmit}}} {
			s, _ := NewDrive(Request{Keys: keys}, "claude", now.Add(time.Minute))
			s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: panel})
			if s.Result().Outcome != OutcomeRefused {
				t.Errorf("%v admitted on a panel: %#v", keys, s.Result())
			}
		}
	})
}

func TestStepsTableIsTheParsersAcceptedSet(t *testing.T) {
	examples := map[KeyKind]string{KeyText: "text:hello", KeyWait: "wait:10"}
	for _, step := range Steps {
		spec, ok := examples[step.Kind]
		if !ok {
			spec = string(step.Kind)
		}
		if _, err := ParseKeySequence([]string{spec}); err != nil {
			t.Errorf("table step %s refused: %v", step.Syntax, err)
		}
	}
	for _, spec := range []string{"tab", "left", string(KeyLeft), string(KeyRight)} {
		if _, err := ParseKeySequence([]string{spec}); err == nil {
			t.Errorf("%q accepted but not in the table", spec)
		}
	}
	if err := (KeySequence{{Kind: KeyLeft}}).Validate(); err == nil {
		t.Error("internal recipe key validated as a caller step")
	}
}

func TestPanelRefusalNamesEsc(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyClearLine}}}, "claude", now.Add(time.Minute))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Dismissable: true}})
	if r := s.Result(); r.Outcome != OutcomeRefused || r.Detail != PanelRefusal {
		t.Fatalf("panel refusal %#v", r)
	}
	s, _ = NewDrive(Request{Keys: KeySequence{{Kind: KeyClearLine}}}, "claude", now.Add(time.Minute))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true}})
	if r := s.Result(); r.Detail == PanelRefusal {
		t.Fatalf("panel hint without a panel %#v", r)
	}
	// A lone esc refused on a panel (an incomplete frame) must not be told to send esc.
	s, _ = NewDrive(Request{Keys: KeySequence{{Kind: KeyEsc}}}, "claude", now.Add(time.Minute))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Dismissable: true}})
	if r := s.Result(); r.Outcome != OutcomeRefused || r.Detail == PanelRefusal {
		t.Fatalf("lone esc told to send esc %#v", r)
	}
}

// TestDismissalToleratesThePanelsTurnFlag: Claude reports progress as working
// while its panel shows, so the dismissal must not stop on TurnActive, while
// every other drive still does (pair#691).
func TestDismissalToleratesThePanelsTurnFlag(t *testing.T) {
	now := time.Unix(100, 0)
	esc := Request{Keys: KeySequence{{Kind: KeyEsc}}}
	s, _ := NewDrive(esc, "claude", now.Add(time.Minute))
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Dismissable: true, TurnActive: true}})
	if s.Terminal() || len(fx) != 1 || fx[0].Key.Kind != KeyEsc || !s.Dismissal() {
		t.Fatalf("esc refused on a panel with the turn flag: %#v", s.Result())
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 2, TurnActive: true}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, TurnActive: true}})
	if s.Terminal() {
		t.Fatalf("composer back but turn still flagged ended the drive: %#v", s.Result())
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeConfirmed {
		t.Fatalf("idle after dismissal %#v", s.Result())
	}

	for name, c := range map[string]struct {
		r      Request
		screen Screen
	}{
		"esc on an occupied composer during a turn": {esc, Screen{Epoch: 1, Complete: true, Recoverable: true, TurnActive: true}},
		"esc with no proof during a turn":           {esc, Screen{Epoch: 1, Complete: true, TurnActive: true}},
		"clear-line on a panel during a turn":       {Request{Keys: KeySequence{{Kind: KeyClearLine}}}, Screen{Epoch: 1, Complete: true, Dismissable: true, TurnActive: true}},
		"double esc on a panel during a turn":       {Request{Keys: KeySequence{{Kind: KeyEsc}, {Kind: KeyEsc}}}, Screen{Epoch: 1, Complete: true, Dismissable: true, TurnActive: true}},
	} {
		d, _ := NewDrive(c.r, "claude", now.Add(time.Minute))
		d, fx := ReduceDrive(d, DriveEvent{Kind: EventStart, At: now, Screen: c.screen})
		if d.Result().Outcome != OutcomeRefused || len(fx) != 0 {
			t.Errorf("%s: %#v %#v", name, d.Result(), fx)
		}
	}

	// A raw drive that has written still stops when a turn starts.
	raw, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: "x"}, {Kind: KeyEnter}}}, "claude", now.Add(time.Minute))
	raw, _ = ReduceDrive(raw, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	raw, _ = ReduceDrive(raw, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 2}})
	raw, _ = ReduceDrive(raw, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, TurnActive: true}})
	if !raw.Terminal() {
		t.Fatalf("raw drive kept going in a turn: %#v", raw.Result())
	}
}
