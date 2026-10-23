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
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 0, Complete: true, Idle: true}})
	s = acknowledgeModelCommand(t, s, now)
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
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 0, Idle: true, Complete: true}})
	s = acknowledgeModelCommand(t, s, now)
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

func TestDriveCodexDefaultCaptured(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol", Effort: "low", Scope: ScopeDefault})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "default-model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: capture(t, "default-effort-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEnter {
		t.Fatalf("apply %#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Idle: true, Lines: capture(t, "default-status")}})
	if s.Result().Outcome != OutcomeConfirmed || s.Result().Scope != ScopeDefault {
		t.Fatalf("%#v", s.Result())
	}
}

func advancedEntry(t *testing.T, effort string) (DriveState, time.Time) {
	t.Helper()
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol", Effort: effort})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "more-model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: capture(t, "more-effort-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyDown {
		t.Fatalf("navigate entry %#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Lines: capture(t, "more-entry")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEnter {
		t.Fatalf("open entry %#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	return s, now
}

func TestDriveCodexAdvancedCaptured(t *testing.T) {
	for _, effort := range []string{"max", "ultra"} {
		s, now := advancedEntry(t, effort)
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 5, Complete: true, Lines: capture(t, "more-expanded")}})
		if len(fx) != 1 {
			t.Fatalf("advanced %#v", fx)
		}
		if effort == "max" {
			if fx[0].Key.Text != "s" {
				t.Fatalf("max apply %#v", fx)
			}
		} else {
			if fx[0].Key.Kind != KeyDown {
				t.Fatalf("ultra navigate %#v", fx)
			}
			s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
			_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 5, Complete: true, Lines: capture(t, "more-expanded")}})
			for _, effect := range fx {
				if effect.Kind == EffectWrite {
					t.Fatal("advanced navigation replayed stale frame")
				}
			}
			lines := capture(t, "more-expanded")
			for i, line := range lines {
				if strings.Contains(line, "› 1. Max") {
					lines[i] = strings.ReplaceAll(strings.ReplaceAll(line, "‹rev››", "  "), "‹/rev›", "")
				}
				if strings.Contains(line, "2. Ultra") {
					lines[i] = "‹rev›› " + strings.TrimSpace(line) + "‹/rev›"
				}
			}
			_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 6, Complete: true, Lines: lines}})
			if len(fx) != 1 || fx[0].Key.Text != "s" {
				t.Fatalf("observed ultra apply %#v", fx)
			}
		}
	}
}

func TestDriveAdvancedRequiresObservedModelIdentity(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol", Effort: "max"})
	_, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "more-expanded")}})
	for _, effect := range fx {
		if effect.Kind == EffectWrite && effect.Key.Text == "s" {
			t.Fatal("advanced selection applied without owned model evidence")
		}
	}
}

func TestDriveCodexAdvancedRecoveryThreeEscapes(t *testing.T) {
	s, now := advancedEntry(t, "max")
	s.navigation = MaxNavigationSteps
	for i, name := range []string{"more-expanded", "more-back-effort", "more-model-picker"} {
		var fx []DriveEffect
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: uint64(5 + i), Complete: true, Lines: capture(t, name)}})
		if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
			t.Fatalf("escape %d %#v %#v", i, fx, s.Result())
		}
		s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 8, Complete: true, Idle: true, Lines: capture(t, "default-recovery-prompt")}})
	if s.Result().Outcome != OutcomeFailed || !strings.Contains(s.Result().Detail, "back at empty prompt") {
		t.Fatalf("%#v", s.Result())
	}
}

func TestDriveScreenDerivedWritesCarryObservationProof(t *testing.T) {
	for _, tc := range []struct {
		model string
		key   KeyKind
	}{{"gpt-6-astra", KeyDown}, {"gpt-6.1-sol", KeyEnter}, {"gpt-missing", KeyEsc}} {
		s, now := modelAtPicker(t, ModelRequest{Model: tc.model, Effort: "low"})
		_, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 17, Complete: true, Lines: capture(t, "model-picker")}})
		if len(fx) != 1 || fx[0].Key.Kind != tc.key || fx[0].ScreenEpoch != 17 {
			t.Fatalf("%s proof missing: %#v", tc.model, fx)
		}
	}
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol", Effort: "low"})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 17, Complete: true, Lines: capture(t, "model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 17}})
	_, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 18, Complete: true, Lines: capture(t, "effort-picker")}})
	if len(fx) != 1 || fx[0].Key.Text != "s" || fx[0].ScreenEpoch != 18 {
		t.Fatalf("apply proof missing: %#v", fx)
	}
}

func TestDriveCommandWritesHaveNoPickerProof(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Model: &ModelRequest{Model: "gpt-6.1-sol"}}, "codex", now.Add(time.Minute))
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 10, Complete: true, Idle: true}})
	for i := 0; i < 2; i++ {
		if len(fx) != 1 || fx[0].Kind != EffectWrite || fx[0].ScreenEpoch != 0 {
			t.Fatalf("command depends on picker proof %#v", fx)
		}
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: uint64(10 + i)}})
	}
}

func TestDriveRawFinalFenceUsesActualEmissionEpoch(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: "a"}, {Kind: KeyEnter}}}, "codex", now.Add(time.Minute))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 10}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 11}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 11, Complete: true, Idle: true}})
	if s.Terminal() {
		t.Fatal("emission epoch mistaken for fresh final paint")
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 12, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeConfirmed {
		t.Fatalf("%#v", s.Result())
	}
}

func TestDriveModelDoesNotActOnPreEmissionFrame(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol", Effort: "low"})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 10}})
	_, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: capture(t, "effort-picker")}})
	for _, effect := range fx {
		if effect.Kind == EffectWrite {
			t.Fatalf("pre-emission frame authorized effect %#v", effect)
		}
	}
}

func TestDriveCancelPreservesReason(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventCancel, At: now, Error: "screen changed before write"})
	if len(fx) != 0 || !strings.Contains(s.Result().Detail, "screen changed before write") || !strings.Contains(s.Result().Detail, "inspect") {
		t.Fatalf("reason lost %#v", s.Result())
	}
}

func TestDriveEffectSemanticRepaintProof(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6-astra"})
	lines := capture(t, "model-picker")
	_, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: lines}})
	if len(fx) != 1 {
		t.Fatal("effect missing")
	}
	effect := fx[0]
	if !effect.MatchesScreen("codex", Screen{Epoch: 3, Complete: true, Lines: lines}) {
		t.Fatal("identical parsed repaint refused")
	}
	for _, screen := range []Screen{{Epoch: 1, Complete: true, Lines: lines}, {Epoch: 3, Lines: lines}, {Epoch: 3, Complete: true, TurnActive: true, Lines: lines}, {Epoch: 3, Complete: true, Lines: []string{"unknown"}}} {
		if effect.MatchesScreen("codex", screen) {
			t.Fatalf("unsafe repaint accepted %#v", screen)
		}
	}
	changed := append([]string{}, lines...)
	for i, line := range changed {
		changed[i] = strings.ReplaceAll(line, "GPT-6-Astra", "GPT-6-Changed")
	}
	if effect.MatchesScreen("codex", Screen{Epoch: 3, Complete: true, Lines: changed}) {
		t.Fatal("changed choices accepted")
	}
	changed = append([]string{}, lines...)
	for i, line := range changed {
		changed[i] = strings.ReplaceAll(line, "enter ‹dim›select ·‹/dim› esc ‹dim›back‹/dim›", "enter ‹dim›default ·‹/dim› s ‹dim›session ·‹/dim› esc ‹dim›back‹/dim›")
	}
	if effect.MatchesScreen("codex", Screen{Epoch: 3, Complete: true, Lines: changed}) {
		t.Fatal("changed footer accepted")
	}
	changed = append([]string{}, lines...)
	for i, line := range changed {
		if strings.Contains(line, "1. GPT-6.1-Sol") {
			changed[i] = strings.ReplaceAll(strings.ReplaceAll(line, "‹rev››", "  "), "‹/rev›", "")
		}
		if strings.Contains(line, "2. GPT-6-Astra") {
			changed[i] = "‹rev›› " + strings.TrimSpace(line) + "‹/rev›"
		}
	}
	if effect.MatchesScreen("codex", Screen{Epoch: 3, Complete: true, Lines: changed}) {
		t.Fatal("changed selection accepted")
	}
}

func TestDriveUnknownRecoveryFrameStopsWithoutCleanup(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-missing"})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: []string{"unknown replacement screen"}}})
	if !s.Terminal() || s.Result().Outcome != OutcomeUnconfirmed || len(fx) != 0 {
		t.Fatalf("unknown recovery received cleanup: %#v %#v", s.Result(), fx)
	}
}

func TestDriveEffectHistoryCannotAuthorizeCurrentInput(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6-astra"})
	lines := capture(t, "model-picker")
	_, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: lines}})
	effect := fx[0]
	if effect.MatchesScreen("codex", Screen{Epoch: 3, Complete: true, Lines: []string{"unknown current viewport"}, History: lines}) {
		t.Fatal("old historical picker authorized input")
	}
	history := append(append([]string{}, lines...), lines...)
	if !effect.MatchesScreen("codex", Screen{Epoch: 3, Complete: true, Lines: lines, History: history}) {
		t.Fatal("duplicate historical headings defeated identical current viewport")
	}
}

func TestDriveRetainsHistoryWithoutParsingItAsCurrentPicker(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	history := append(capture(t, "model-picker"), "retained prose")
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: []string{"unknown current viewport"}, History: history}})
	if s.Terminal() || len(fx) != 1 || fx[0].Kind != EffectObserve {
		t.Fatal("historical picker was actionable")
	}
	tail := s.Result().Tail
	if len(tail) == 0 || tail[len(tail)-1] != "retained prose" {
		t.Fatal("history tail was not retained")
	}
}

func TestDriveUnknownPostSubmitFrameObservesUntilDeadline(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: []string{"› /model‹cursor›"}}})
	if s.Terminal() || len(fx) != 1 || fx[0].Kind != EffectObserve {
		t.Fatalf("transient frame terminated: %#v %#v", s.Result(), fx)
	}
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(time.Minute)})
	if !s.Terminal() || s.Result().Outcome != OutcomeUnconfirmed || len(fx) != 0 {
		t.Fatalf("unknown screen exceeded deadline: %#v %#v", s.Result(), fx)
	}
}

func TestDriveRecoveryOnlyAdmissionAndFinalNoop(t *testing.T) {
	now := time.Unix(100, 0)
	request := Request{Keys: KeySequence{{Kind: KeyClearLine}, {Kind: KeyEsc}}}
	if !request.RecoveryOnly() {
		t.Fatal("recovery shape refused")
	}
	for _, r := range []Request{{}, {Keys: KeySequence{{Kind: KeyText, Text: "x"}}}, {Keys: KeySequence{{Kind: KeyWait}}}, {Keys: KeySequence{{Kind: KeySubmit}}}, {Keys: KeySequence{{Kind: KeyEsc, Text: "extra"}}}} {
		if r.RecoveryOnly() {
			t.Fatalf("unsafe recovery %#v", r)
		}
	}
	s, _ := NewDrive(request, "codex", now.Add(time.Minute))
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Recoverable: true, ComposerText: "occupied"}})
	if s.Terminal() || len(fx) != 1 || fx[0].Key.Kind != KeyClearLine {
		t.Fatalf("occupied recovery refused %#v", s.Result())
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 2}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 3}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeConfirmed {
		t.Fatalf("final noop recovery %#v", s.Result())
	}
	generic, _ := NewDrive(Request{Keys: KeySequence{{Kind: KeyEnter}}}, "codex", now.Add(time.Minute))
	generic, _ = ReduceDrive(generic, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Recoverable: true}})
	if generic.Result().Outcome != OutcomeRefused {
		t.Fatal("generic sequence admitted occupied composer")
	}
}

func TestDriveModelReservedCleanupOwnCommand(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	screen := Screen{Epoch: 2, Complete: true, Recoverable: true, ComposerText: "/model"}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(57 * time.Second), Screen: screen})
	if s.Terminal() || len(fx) != 1 || fx[0].Key.Kind != KeyClearLine || !fx[0].MatchesScreen("codex", screen) {
		t.Fatalf("owned command cleanup missing %#v %#v", s.Result(), fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now.Add(57 * time.Second), Screen: screen})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now.Add(58 * time.Second), Screen: Screen{Epoch: 3, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeFailed || !strings.Contains(s.Result().Detail, "back at empty prompt") {
		t.Fatalf("recovery proof missing %#v", s.Result())
	}
}

func TestDriveReservedCleanupRefusesUnsafeFrames(t *testing.T) {
	for _, screen := range []Screen{{Epoch: 2, Complete: true, Lines: []string{"unknown"}}, {Epoch: 2, Complete: true, Recoverable: true, ComposerText: "operator"}, {Epoch: 2, Complete: true, Recoverable: true, ComposerText: "/model\n"}, {Epoch: 2, Complete: true, Recoverable: true, ComposerText: "/model", TurnActive: true}} {
		s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
		s, fx := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(57 * time.Second), Screen: screen})
		if !s.Terminal() || len(fx) != 0 || s.Result().Outcome == OutcomeConfirmed {
			t.Fatalf("unsafe cleanup %#v %#v", s.Result(), fx)
		}
	}
}

func TestDriveReservedCleanupPickerAndCancellation(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	screen := Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(57 * time.Second), Screen: screen})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc || !fx[0].MatchesScreen("codex", screen) {
		t.Fatalf("picker recovery missing %#v", fx)
	}
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventCancel, At: now.Add(57 * time.Second), Error: "operator input"})
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(58 * time.Second), Screen: screen})
	if !s.Terminal() || len(fx) != 0 {
		t.Fatal("recovery continued after human cancellation")
	}
	s, now = modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWriteFailed, At: now, Error: "partial submit"})
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(57 * time.Second), Screen: Screen{Epoch: 2, Complete: true, Recoverable: true, ComposerText: "/model"}})
	if !s.Terminal() || len(fx) != 0 {
		t.Fatal("uncertain write received cleanup")
	}
}

func TestDriveShortBudgetReservesFraction(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Model: &ModelRequest{Model: "gpt-6.1-sol"}}, "codex", now.Add(time.Second))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if !s.recoverAt.Equal(now.Add(900 * time.Millisecond)) {
		t.Fatalf("reserve moved budget %#v", s.recoverAt)
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
	if s.Started() {
		t.Fatal("new drive already started")
	}
	req.Keys[0].Kind = KeySubmit
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
		t.Fatalf("%#v", fx)
	}
	if !s.Started() {
		t.Fatal("started drive still queued")
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
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 0, Idle: true, Complete: true}})
	for _, want := range []KeyKind{KeyClearLine, KeyText} {
		if len(fx) != 1 || fx[0].Key.Kind != want || !fx[0].Control {
			t.Fatalf("%#v", fx)
		}
		s, fx = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	}
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 1, Complete: true, Recoverable: true, CommandKnown: true, CommandText: "/model"}})
	if len(fx) != 1 || !fx[0].CommandEnter() {
		t.Fatalf("missing command Enter: %#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 1}})
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

func TestDriveTextQuietStartsAtAcknowledgmentAndOverlapsWait(t *testing.T) {
	now := time.Unix(100, 0)
	s, err := NewDrive(Request{Keys: KeySequence{{Kind: KeyText, Text: "/model"}, {Kind: KeyWait, WaitMS: 50}, {Kind: KeySubmit}}}, "codex", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 1, Complete: true, Idle: true}})
	ack := now.Add(time.Second)
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: ack})
	_, effects := ReduceDrive(s, DriveEvent{Kind: EventTick, At: ack.Add(50 * time.Millisecond)})
	if len(effects) != 1 || effects[0].Key.Kind != KeySubmit || !effects[0].NotBefore.Equal(ack.Add(TextSubmitDelay)) {
		t.Fatalf("quiet period ignored acknowledgment or explicit wait: %#v", effects)
	}
}

func TestDriveReservedRecoveryNeedsFreshEmptyProof(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol"})
	s, effects := ReduceDrive(s, DriveEvent{Kind: EventTick, At: now.Add(57 * time.Second), Screen: Screen{Epoch: s.writeEpoch, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeUnconfirmed || len(effects) != 0 {
		t.Fatalf("unchanged pre-text prompt claimed recovery: %+v %#v", s.Result(), effects)
	}
}

func TestDriveMissingChoicesSurviveRecoveredPrompt(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			now := time.Unix(100, 0)
			s, err := NewDrive(Request{Model: &ModelRequest{Model: "missing-model"}}, agent, now.Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 0, Complete: true, Idle: true}})
			s = acknowledgeModelCommand(t, s, now)
			lines := capture(t, "model-picker")
			if agent == "claude" {
				lines = claudeCapture(t, "model-picker")
			}
			labels := ReadModelScreen(agent, Screen{Complete: true, Lines: lines}).Labels
			if len(labels) == 0 {
				t.Fatal("fixture has no choices")
			}
			s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: lines}})
			if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
				t.Fatalf("missing backout: %#v", fx)
			}
			s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
			s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Idle: true, Lines: []string{"empty prompt"}}})
			result := s.Result()
			if result.Outcome != OutcomeFailed || !strings.Contains(result.Detail, "back at empty prompt") {
				t.Fatalf("%#v", result)
			}
			for _, label := range labels {
				if !strings.Contains(result.Detail, label) {
					t.Fatalf("recovery lost choice %q: %#v", label, result)
				}
			}
			if strings.Join(result.Tail, " ") != "empty prompt" {
				t.Fatalf("tail not current: %#v", result)
			}
		})
	}
}

func TestDriveMissingEffortChoicesSurviveRecoveredPrompt(t *testing.T) {
	s, now := modelAtPicker(t, ModelRequest{Model: "gpt-6.1-sol", Effort: "missing-effort"})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: capture(t, "model-picker")}})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Lines: capture(t, "effort-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 4, Complete: true, Lines: capture(t, "model-picker")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 5, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeFailed {
		t.Fatalf("%#v", s.Result())
	}
	for _, label := range []string{"low", "medium", "high"} {
		if !strings.Contains(s.Result().Detail, label) {
			t.Fatalf("lost effort %q: %#v", label, s.Result())
		}
	}
}

func TestChoiceFailureBoundsWholeLabels(t *testing.T) {
	labels := make([]string, MaxObservedLabels)
	for i := range labels {
		labels[i] = strings.Repeat("界", 42)
	}
	detail := choiceFailure("model", labels)
	if len(detail) > 900 || !strings.Contains(detail, "further choices omitted") || !strings.Contains(detail, labels[0]) {
		t.Fatalf("bad bounded choices: %q", detail)
	}
	// Recovery suffixes and the 1 KiB broker/endpoint detail bound must coexist.
	if len(detail+"; prompt recovery unconfirmed") > 1024 {
		t.Fatal("detail envelope exceeded")
	}
}

func TestDriveClaudeUnsupportedEffortRetainsObservedChoice(t *testing.T) {
	now := time.Unix(100, 0)
	s, _ := NewDrive(Request{Model: &ModelRequest{Model: "opus 5.5", Effort: "missing-effort"}}, "claude", now.Add(time.Minute))
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 0, Complete: true, Idle: true}})
	s = acknowledgeModelCommand(t, s, now)
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 2, Complete: true, Lines: claudeCapture(t, "session-before")}})
	if len(fx) != 1 || fx[0].Key.Kind != KeyEsc {
		t.Fatalf("%#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 3, Complete: true, Idle: true}})
	if s.Result().Outcome != OutcomeFailed || !strings.Contains(s.Result().Detail, "medium") {
		t.Fatalf("%#v", s.Result())
	}
}

// acknowledgeModelCommand paints the authored command after its text write,
// before acknowledging native Return. Picker fixtures begin at epoch two.
func acknowledgeModelCommand(t *testing.T, s DriveState, now time.Time) DriveState {
	t.Helper()
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now})
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: Screen{Epoch: 1, Complete: true, Recoverable: true, CommandKnown: true, CommandText: "/model"}})
	if len(fx) != 1 || !fx[0].CommandEnter() {
		t.Fatalf("missing command proof: %#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: Screen{Epoch: 1}})
	return s
}
