package agentdrive

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestReduceDriveCommandEnterWaitsForEcho(t *testing.T) {
	now := time.Unix(100, 0)
	for _, structured := range []bool{false, true} {
		r := Request{Keys: KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyEnter}}}
		target := "/usage"
		if structured {
			r = Request{Model: &ModelRequest{Model: "gpt-6.1-sol"}}
			target = "/model"
		}
		s, err := NewDrive(r, "codex", now.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 4, Complete: true, Idle: true}})
		if structured {
			s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 4}})
		}
		if len(fx) != 1 || fx[0].Key.Kind != KeyText {
			t.Fatalf("text %#v", fx)
		}
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 5}})
		for _, f := range fx {
			if f.Kind == EffectWrite {
				t.Fatalf("command Enter before echo: %#v", f)
			}
		}
		for _, screen := range []Screen{
			{Epoch: 5, Complete: true, Recoverable: true, CommandKnown: true, CommandText: target},
			{Epoch: 6, Complete: true, Recoverable: true, CommandKnown: true, CommandText: target + "x"},
			{Epoch: 7, Complete: true, Idle: true},
			{Epoch: 8, Complete: false, Recoverable: true, CommandKnown: true, CommandText: target},
		} {
			s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: screen})
			for _, f := range fx {
				if f.Kind == EffectWrite {
					t.Fatalf("invalid echo wrote %#v", f)
				}
			}
		}
		echo := Screen{Epoch: 9, Complete: true, Recoverable: true, CommandKnown: true, CommandText: target}
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: echo})
		if len(fx) != 1 || fx[0].Key.Kind != KeyEnter || !fx[0].CommandEnter() {
			t.Fatalf("proved enter %#v", fx)
		}
		if fx[0].NotBefore != now.Add(TextSubmitDelay) {
			t.Fatal("lost paste suppression")
		}
		if !fx[0].MatchesScreen("codex", echo) {
			t.Fatal("exact proof rejected")
		}
		echo.Epoch++
		if !fx[0].MatchesScreen("codex", echo) {
			t.Fatal("identical repaint rejected")
		}
		for _, mutate := range []func(*Screen){func(s *Screen) { s.CommandText += "x" }, func(s *Screen) { s.Epoch = 8 }, func(s *Screen) { s.Complete = false }, func(s *Screen) { s.Recoverable = false }, func(s *Screen) { s.CommandKnown = false }, func(s *Screen) { s.TurnActive = true }} {
			changed := echo
			mutate(&changed)
			if fx[0].MatchesScreen("codex", changed) {
				t.Fatalf("changed proof accepted %#v", changed)
			}
		}
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventCancel, At: now})
		_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: echo})
		if len(fx) != 0 {
			t.Fatal("effect after cancel")
		}
	}
}

func TestCommandExpectationSupportedSpans(t *testing.T) {
	cases := []struct {
		keys KeySequence
		want string
	}{
		{KeySequence{{Kind: KeyText, Text: "/us"}, {Kind: KeyWait}, {Kind: KeyText, Text: "age"}, {Kind: KeyEnter}}, "/usage"},
		{KeySequence{{Kind: KeyClearLine}, {Kind: KeyText, Text: "/model"}, {Kind: KeyEnter}}, "/model"},
		{KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyLeft}, {Kind: KeyEnter}}, ""},
		{KeySequence{{Kind: KeyText, Text: "hello"}, {Kind: KeyEnter}}, ""},
		{KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyLeft}, {Kind: KeyClearLine}, {Kind: KeyText, Text: "/model"}, {Kind: KeyEnter}}, ""},
	}
	for _, c := range cases {
		got := commandExpectations(c.keys)
		if got[len(got)-1] != c.want {
			t.Fatalf("%#v got %#v want %q", c.keys, got, c.want)
		}
	}
}

// expectationOracle deliberately scans the complete preceding span at each
// boundary; production cannot do so because its predictor receives one step.
func expectationOracle(steps KeySequence, end int) string {
	if steps[end].Kind != KeyEnter {
		return ""
	}
	start := 0
	for i := end - 1; i >= 0; i-- {
		if steps[i].Kind == KeyClearLine {
			start = i + 1
			break
		}
	}
	lastEnter := -1
	for i, step := range steps[:end] {
		switch step.Kind {
		case KeyClearLine, KeyText, KeyWait:
		case KeyEnter:
			if i < start {
				lastEnter = i
			}
		default:
			return ""
		}
	}
	if lastEnter >= 0 && expectationOracle(steps, lastEnter) == "" {
		return ""
	}
	var parts []string
	for _, step := range steps[start:end] {
		switch step.Kind {
		case KeyText:
			parts = append(parts, step.Text)
		case KeyWait:
		default:
			return ""
		}
	}
	text := strings.Join(parts, "")
	if slashCommandOracle.MatchString(text) {
		return text
	}
	return ""
}

func TestCommandExpectationGeneratedAgainstSpanOracle(t *testing.T) {
	for seed := int64(0); seed < 256; seed++ {
		rng := rand.New(rand.NewSource(seed))
		var keys KeySequence
		choices := []KeyStep{{Kind: KeyClearLine}, {Kind: KeyText, Text: "/usage"}, {Kind: KeyText, Text: "/mo"}, {Kind: KeyText, Text: "del"}, {Kind: KeyText, Text: " --help"}, {Kind: KeyText, Text: " /model"}, {Kind: KeyWait}, {Kind: KeyEnter}, {Kind: KeyLeft}, {Kind: KeySubmit}, {Kind: "ctrl-a"}, {Kind: "backspace"}}
		for i := 0; i < MaxKeySteps; i++ {
			keys = append(keys, choices[rng.Intn(len(choices))])
		}
		got := commandExpectations(keys)
		for i := range keys {
			if want := expectationOracle(keys, i); got[i] != want {
				t.Fatalf("seed %d step %d: got %q want %q", seed, i, got[i], want)
			}
		}
	}
}

func TestReduceDriveGeneratedCommandEchoInvariants(t *testing.T) {
	for seed := int64(0); seed < 256; seed++ {
		rng := rand.New(rand.NewSource(seed))
		now := time.Unix(100, 0)
		structured := seed%2 == 0
		req := Request{Keys: KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyEnter}}}
		target := "/usage"
		if structured {
			req = Request{Model: &ModelRequest{Model: "gpt-6.1-sol"}}
			target = "/model"
		}
		s, _ := NewDrive(req, "codex", now.Add(time.Second))
		event := DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}}
		enters := 0
		fence := uint64(1)
		lastWrite := KeyKind("")
		pending := false
		textAck := false
		epoch := uint64(1)
		for n := 0; n < 160; n++ {
			terminal := s.Terminal()
			if event.Kind == EventWritten && pending && lastWrite == KeyText {
				textAck = true
				if event.Screen.Epoch > fence {
					fence = event.Screen.Epoch
				}
			}
			if event.Kind == EventWritten {
				pending = false
			}
			var fx []DriveEffect
			s, fx = ReduceDrive(s, event)
			for _, effect := range fx {
				if effect.Kind != EffectWrite {
					continue
				}
				if terminal {
					t.Fatalf("seed %d write after terminal", seed)
				}
				lastWrite = effect.Key.Kind
				pending = true
				if effect.CommandEnter() {
					enters++
					sc := event.Screen
					if !textAck || event.Kind != EventScreen || !sc.Complete || sc.TurnActive || !sc.Recoverable || !sc.CommandKnown || sc.CommandText != target || sc.Epoch <= fence {
						t.Fatalf("seed %d early Enter %#v fence=%d", seed, event, fence)
					}
				}
			}
			if enters > 1 {
				t.Fatalf("seed %d duplicate command Enter", seed)
			}
			event = DriveEvent{At: now.Add(time.Duration(n+1) * 10 * time.Millisecond)}
			switch rng.Intn(12) {
			case 0:
				event.Kind = EventCancel
			case 1, 2, 3, 4:
				event.Kind = EventWritten
				event.Screen.Epoch = epoch
			case 5:
				event.Kind = EventTick
			default:
				event.Kind = EventScreen
				if rng.Intn(2) == 0 {
					epoch++
				}
				event.Screen = Screen{Epoch: epoch, Complete: rng.Intn(4) != 0, Recoverable: rng.Intn(4) != 0, CommandKnown: rng.Intn(4) != 0, CommandText: target}
				if rng.Intn(4) == 0 {
					event.Screen.CommandText = "/different"
				}
			}
		}
	}
}

func TestCommandExpectationCountsBoundedGroups(t *testing.T) {
	for _, groups := range []int{8, 16} {
		prediction := commandExpectation{known: true}
		consumed, textBytes, commands := 0, 0, 0
		// Every group fills a quarter of the 64-step shell; the 16-group case
		// consumes exactly the existing 8 KiB aggregate text allowance.
		chunk := "/usage " + strings.Repeat("x", MaxKeyTextBytes/16-len("/usage "))
		for i := 0; i < groups; i++ {
			for _, key := range []KeyStep{{Kind: KeyClearLine}, {Kind: KeyText, Text: chunk}, {Kind: KeyWait}, {Kind: KeyEnter}} {
				consumed++
				textBytes += len(key.Text)
				if target := prediction.advance(key); target != "" {
					commands++
					if target != chunk {
						t.Fatal("wrong target")
					}
				}
			}
		}
		if consumed != 4*groups || textBytes != 512*groups || commands != groups {
			t.Fatalf("work: keys=%d bytes=%d commands=%d", consumed, textBytes, commands)
		}
	}
}

func TestReduceDriveCommandEchoDeadlineAndCleanupScope(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Model: &ModelRequest{Model: "gpt-6.1-sol"}}, "codex", now.Add(time.Minute))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	// General command evidence must not turn /usage into model-cleanup authority.
	screen := Screen{Epoch: 2, Complete: true, Recoverable: true, CommandKnown: true, CommandText: "/usage"}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(57 * time.Second), Screen: screen})
	if !s.Terminal() || s.Result().Outcome != OutcomeUnconfirmed || len(fx) != 0 {
		t.Fatalf("foreign cleanup: %#v %#v", s.Result(), fx)
	}
	raw, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyEnter}}}, "codex", now.Add(time.Second))
	raw, _ = ReduceDrive(raw, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	raw, _ = ReduceDrive(raw, DriveEvent{Kind: EventWritten, At: now})
	raw, fx = ReduceDrive(raw, DriveEvent{Kind: EventScreen, At: now.Add(time.Second), Screen: screen})
	if !raw.Terminal() || len(fx) != 0 {
		t.Fatal("echo at deadline wrote Enter")
	}
}

func FuzzCommandExpectation(f *testing.F) {
	f.Add([]byte{1, 2, 6, 3, 7})
	f.Add([]byte{0, 1, 7, 0, 2, 3, 7})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxKeySteps {
			data = data[:MaxKeySteps]
		}
		choices := []KeyStep{{Kind: KeyClearLine}, {Kind: KeyText, Text: "/usage"}, {Kind: KeyText, Text: "/mo"}, {Kind: KeyText, Text: "del"}, {Kind: KeyText, Text: " --help"}, {Kind: KeyText, Text: " /model"}, {Kind: KeyWait}, {Kind: KeyEnter}, {Kind: KeyLeft}, {Kind: KeySubmit}, {Kind: "ctrl-a"}, {Kind: "backspace"}}
		keys := make(KeySequence, len(data))
		for i, b := range data {
			keys[i] = choices[int(b)%len(choices)]
		}
		got := commandExpectations(keys)
		for i := range keys {
			if want := expectationOracle(keys, i); got[i] != want {
				t.Fatalf("step %d: got %q want %q", i, got[i], want)
			}
		}
	})
}

func TestReduceDriveCommandEchoObservedDuringWaitRemainsCurrent(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: "/usage"}, {Kind: KeyWait, WaitMS: 50}, {Kind: KeyEnter}}}, "codex", now.Add(time.Second))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 1}})
	echo := Screen{Epoch: 2, Complete: true, Recoverable: true, CommandKnown: true, CommandText: "/usage"}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now.Add(10 * time.Millisecond), Screen: echo})
	for _, effect := range fx {
		if effect.Kind == EffectWrite {
			t.Fatal("wait skipped")
		}
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(50 * time.Millisecond), Screen: echo})
	_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now.Add(50 * time.Millisecond), Screen: echo})
	if len(fx) != 1 || !fx[0].CommandEnter() {
		t.Fatalf("current fresh echo lost during wait: %#v", fx)
	}
}
