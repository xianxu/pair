package agentdrive

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestDriveGeneratedEventsNeverConfirmWithoutApplyOrWriteAfterCancel(t *testing.T) {
	for seed := int64(0); seed < 64; seed++ {
		rng := rand.New(rand.NewSource(seed))
		now := time.Unix(100, 0)
		s, _ := NewDrive(Request{Model: &ModelRequest{Model: "gpt-6.1-sol", Effort: "low"}}, "codex", now.Add(time.Minute))
		applied, canceled := false, false
		epoch := uint64(1)
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Idle: true, Complete: true}})
		for n := 0; n < 100; n++ {
			for _, effect := range fx {
				if effect.Kind == EffectWrite {
					if canceled {
						t.Fatalf("seed %d wrote after cancellation", seed)
					}
					if effect.Key.Kind == KeyText && effect.Key.Text == "s" {
						applied = true
					}
				}
			}
			e := DriveEvent{At: now.Add(time.Duration(n) * time.Millisecond)}
			switch rng.Intn(8) {
			case 0:
				e.Kind = EventCancel
				canceled = true
			case 1:
				e.Kind = EventWriteFailed
			case 2, 3:
				e.Kind = EventWritten
			case 4:
				e.Kind = EventTick
			default:
				e.Kind = EventScreen
				if rng.Intn(2) == 0 {
					epoch++
				}
				e.Screen = Screen{Epoch: epoch, Complete: rng.Intn(3) != 0}
				switch rng.Intn(3) {
				case 0:
					e.Screen.Lines = capture(t, "model-picker")
				case 1:
					e.Screen.Lines = capture(t, "effort-picker")
				case 2:
					e.Screen.Lines = capture(t, "session-status")
					e.Screen.Idle = true
				}
			}
			s, fx = ReduceDrive(s, e)
			if s.Result().Outcome == OutcomeConfirmed && !applied {
				t.Fatalf("seed %d confirmed without apply", seed)
			}
		}
	}
}

func TestDriveClaudeSessionCaptured(t *testing.T) {
	now := time.Unix(100, 0)
	s, e := NewDrive(Request{Model: &ModelRequest{Model: "opus 5.5", Effort: "medium"}}, "claude", now.Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	for i := 0; i < 3; i++ {
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: claudeCapture(t, "session-before")}})
	if len(fx) != 1 || fx[0].Key.Text != "s" {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Idle: true, Lines: claudeCapture(t, "session-status")}})
	if s.Result().Outcome != OutcomeConfirmed || s.Result().Model != "opus 5.5" {
		t.Fatalf("%#v", s.Result())
	}
}

func modelAtPicker(t *testing.T, req ModelRequest) (DriveState, time.Time) {
	t.Helper()
	now := time.Unix(100, 0)
	s, e := NewDrive(Request{Model: &req}, "codex", now.Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Idle: true, Complete: true}})
	for i := 0; i < 3; i++ {
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	}
	return s, now
}
func TestDriveFreshNavigationRejectsStaleScreen(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6-astra", Effort: "medium"})
	lines := capture(t, "model-picker")
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: lines}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyDown {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: lines}})
	for _, e := range fx {
		if e.Kind == EffectWrite {
			t.Fatal("stale navigation")
		}
	}
}

func TestDriveClaudeDefaultScopeRefusedBeforeInput(t *testing.T) {
	_, err := NewDrive(Request{Model: &ModelRequest{Model: "opus 5.5", Scope: ScopeDefault}}, "claude", time.Now().Add(time.Minute))
	if err == nil {
		t.Fatal("unverified Claude default scope admitted")
	}
}
func TestDriveOmittedEffortAndOldConfirmation(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	s.priorConfirmations = confirmationCounts(capture(t, "session-status"))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: capture(t, "effort-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Idle: true, Lines: capture(t, "session-status")}})
	if s.Terminal() {
		t.Fatal("old confirmation reused")
	}
	lines := append([]string{"• Model changed to gpt-6.1-sol low for this session only"}, capture(t, "session-status")...)
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 5, Complete: true, Idle: true, Lines: lines}})
	if s.Result().Outcome != OutcomeConfirmed || s.Result().Effort != "low" {
		t.Fatalf("%#v", s.Result())
	}
}
func TestDriveUnknownChoiceBackoutAndTailBound(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-missing"})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Idle: true, Lines: []string{strings.Repeat("界", 20000)}}})
	if s.Result().Outcome != OutcomeFailed {
		t.Fatalf("%#v", s.Result())
	}
	n := 0
	for _, line := range s.Result().Tail {
		n += len(line) + 1
	}
	if n > 32768 {
		t.Fatal("tail overflow")
	}
}

func TestDriveRawSequenceAndCancellation(t *testing.T) {
	now := time.Unix(100, 0)
	req := Request{Keys: KeySequence{{Kind: KeyEsc}, {Kind: KeyEnter}, {Kind: KeyWait, WaitMS: 10}}}
	s, e := NewDrive(req, "codex", now.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	req.Keys[0].Kind = KeySubmit
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
		t.Fatalf("%#v", fx)
	}
	stale, _ := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if stale.Terminal() {
		t.Fatal("screen acknowledged write")
	}
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEnter {
		t.Fatalf("%#v", fx)
	}
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	if len(fx) != 1 || fx[0].Kind != EffectWait {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(10 * time.Millisecond)})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if s.Terminal() {
		t.Fatal("stale final success")
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeConfirmed {
		t.Fatalf("%#v", s.Result())
	}
}
func TestDriveNoWritesAfterCancelGenerated(t *testing.T) {
	now := time.Unix(100, 0)
	for _, kind := range []EventKind{EventScreen, EventWritten, EventWriteFailed, EventCancel, EventTick, EventStart} {
		s, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyEnter}}}, "codex", now.Add(time.Second))
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Idle: true, Complete: true}})
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventCancel, At: now})
		s, fx := ReduceDrive(s, DriveEvent{Kind: kind, At: now, Screen: Screen{Epoch: 2, Idle: true, Complete: true}})
		if !s.Terminal() || len(fx) != 0 || s.Result().Outcome == OutcomeConfirmed {
			t.Fatalf("%#v %#v", s.Result(), fx)
		}
	}
}
func TestDriveCapturedModelSession(t *testing.T) {
	now := time.Unix(100, 0)
	s, e := NewDrive(Request{Model: &ModelRequest{Model: "gpt-6.1-sol", Effort: "low"}}, "codex", now.Add(time.Second))
	if e != nil {
		t.Fatal(e)
	}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Idle: true, Complete: true}})
	for _, want := range []KeyKind{KeyClearLine, KeyText, KeySubmit} {
		if len(fx) != 1 || fx[0].Key.Kind != want || !fx[0].Control {
			t.Fatalf("%#v", fx)
		}
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	}
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEnter {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: capture(t, "effort-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyText || fx[0].Key.Text != "s" {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	stale, _ := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Idle: true, Complete: true, Lines: capture(t, "session-status")}})
	if stale.Terminal() {
		t.Fatal("stale confirmation")
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Idle: true, Complete: true, Lines: capture(t, "session-status")}})
	if s.Result().Outcome != OutcomeConfirmed {
		t.Fatalf("%#v", s.Result())
	}
}
