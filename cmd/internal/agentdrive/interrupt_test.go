package agentdrive

import (
	"strings"
	"testing"
	"time"
)

func TestInterruptIsItsOwnRequestShape(t *testing.T) {
	interrupt := Request{Interrupt: true}
	if interrupt.Validate() != nil || !interrupt.InterruptOnly() || interrupt.RecoveryOnly() || interrupt.DismissOnly() {
		t.Fatalf("interrupt shape: %v", interrupt.Validate())
	}
	for _, r := range []Request{
		{Interrupt: true, Keys: KeySequence{{Kind: KeyEsc}}},
		{Interrupt: true, Model: &ModelRequest{Model: "opus 5.5"}},
		{},
	} {
		if r.Validate() == nil {
			t.Errorf("mixed or empty shape validated: %#v", r)
		}
	}
	if err := (KeySequence{{Kind: KeyInterrupt}}).Validate(); err == nil {
		t.Error("the internal Ctrl+C validated inside a key sequence")
	}
	if _, err := ParseKeySequence([]string{"interrupt"}); err == nil {
		t.Error("interrupt parsed as a caller step")
	}
	if got := (KeyStep{Kind: KeyInterrupt}).InputBytes(); string(got) != "\x03" {
		t.Errorf("interrupt bytes %q", got)
	}
	if (Request{Interrupt: true}).IntentDigest() == (Request{Keys: KeySequence{{Kind: KeyEsc}}}).IntentDigest() {
		t.Error("interrupt intent digest collides")
	}
}

func TestAdmitsInterruptExclusively(t *testing.T) {
	interrupt := Request{Interrupt: true}
	all := Admission{Idle: true, Recoverable: true, Dismissable: true}
	if interrupt.Admits(all) || interrupt.Admits(Admission{Idle: true}) {
		t.Error("an interrupt admitted without a working turn")
	}
	if !interrupt.Admits(Admission{Interruptible: true}) {
		t.Error("an interrupt refused on a working turn")
	}
	for _, r := range []Request{{Keys: KeySequence{{Kind: KeyEsc}}}, {Keys: KeySequence{{Kind: KeyText, Text: "x"}}}, {Model: &ModelRequest{Model: "opus 5.5"}}} {
		if r.Admits(Admission{Interruptible: true}) {
			t.Errorf("%#v admitted by a working-turn proof", r)
		}
	}
	if interrupt.WriteAdmits(Admission{}) || !interrupt.WriteAdmits(Admission{Interruptible: true}) {
		t.Error("interrupt pre-write recheck")
	}
	if !(Request{Keys: KeySequence{{Kind: KeyText, Text: "x"}}}).WriteAdmits(Admission{}) {
		t.Error("generic keys gained a per-write proof")
	}
	if interrupt.Refusal(Admission{Dismissable: true}, "other") != InterruptRefusal {
		t.Error("interrupt refusal wording")
	}
}

func TestDriveInterruptOutcomes(t *testing.T) {
	now := time.Unix(100, 0)
	working := Screen{Epoch: 1, Complete: true, TurnActive: true, Interruptible: true}
	start := func(t *testing.T) DriveState {
		t.Helper()
		s, _ := NewDrive(Request{Interrupt: true}, "claude", now.Add(time.Minute))
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: working})
		if s.Terminal() || len(fx) != 1 || fx[0].Kind != EffectWrite || fx[0].Key.Kind != KeyInterrupt || !s.TurnTolerant() {
			t.Fatalf("interrupt start %#v %#v", s.Result(), fx)
		}
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 2, TurnActive: true}})
		if s.Terminal() || len(fx) != 1 || fx[0].Kind != EffectObserve {
			t.Fatalf("after the write %#v %#v", s.Result(), fx)
		}
		return s
	}
	t.Run("turn ends", func(t *testing.T) {
		s := start(t)
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, TurnActive: true}})
		if s.Terminal() {
			t.Fatalf("a working frame confirmed the interrupt %#v", s.Result())
		}
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Idle: true}})
		if s.Result().Outcome != OutcomeConfirmed || len(fx) != 0 {
			t.Fatalf("turn ended %#v %#v", s.Result(), fx)
		}
	})
	t.Run("turn still working at the deadline", func(t *testing.T) {
		s := start(t)
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(time.Minute), Screen: Screen{Epoch: 3, Complete: true, TurnActive: true, Lines: []string{"✳ Working…"}}})
		r := s.Result()
		if r.Outcome != OutcomeUnconfirmed || !strings.Contains(r.Detail, "never resent") || len(r.Tail) == 0 || len(fx) != 0 {
			t.Fatalf("deadline %#v %#v", r, fx)
		}
	})
	t.Run("refused without a working turn", func(t *testing.T) {
		for _, screen := range []Screen{{Epoch: 1, Complete: true, Idle: true}, {Epoch: 1, Complete: true, TurnActive: true}, {Epoch: 1, Complete: true, Dismissable: true}} {
			s, _ := NewDrive(Request{Interrupt: true}, "claude", now.Add(time.Minute))
			s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: screen})
			if s.Result().Outcome != OutcomeRefused || len(fx) != 0 || s.Result().Detail != InterruptRefusal {
				t.Errorf("%+v: %#v %#v", screen, s.Result(), fx)
			}
		}
	})
}
