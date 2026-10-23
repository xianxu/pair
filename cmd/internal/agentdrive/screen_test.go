package agentdrive

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func capture(t *testing.T, name string) []string {
	t.Helper()
	b, e := os.ReadFile("../wrapcmd/testdata/drive/codex/0.162.1/" + name + ".screen")
	if e != nil {
		t.Fatal(e)
	}
	return strings.Split(string(b), "\n")
}
func TestReadModelScreenCaptured(t *testing.T) {
	model := ReadModelScreen("codex", Screen{Complete: true, Lines: capture(t, "model-picker")})
	if model.Phase != ScreenModel || model.Selected != "gpt-6.1-sol" || len(model.Labels) != 7 {
		t.Fatalf("%#v", model)
	}
	effort := ReadModelScreen("codex", Screen{Complete: true, Lines: capture(t, "effort-picker")})
	if effort.Phase != ScreenEffort || effort.Selected != "low" || effort.Model != "gpt-6.1-sol" {
		t.Fatalf("%#v", effort)
	}
	status := ReadModelScreen("codex", Screen{Complete: true, Idle: true, Lines: capture(t, "session-status")})
	if status.Phase != ScreenStatus || status.Model != "gpt-6.1-sol" || status.Effort != "low" || status.Scope != ScopeSession {
		t.Fatalf("%#v", status)
	}
}
func TestReadModelScreenDeclinesAmbiguity(t *testing.T) {
	base := []string{"Select Model and Effort", "‹rev›› 1. GPT-6.1-Sol‹/rev›", "2. GPT-6-Astra", "enter select · esc back"}
	for _, lines := range [][]string{
		{"Select Model and Effort", "‹‹rev›› 1. GPT-6.1-Sol‹‹/rev›", "enter select · esc back"},
		{"Select Model and Effort", "‹rev›› 1. GPT-6.1-Sol", "enter select · esc back"},
		append(append([]string{}, base...), "‹rev›› 3. GPT-6-Astra‹/rev›"),
		append(append([]string{}, base...), "4. GPT-6.1-Sol"),
	} {
		if got := ReadModelScreen("codex", Screen{Complete: true, Lines: lines}); got.Phase != ScreenUnknown {
			t.Fatalf("accepted %#v", got)
		}
	}
	if ReadModelScreen("claude", Screen{Complete: true, Lines: base}).Phase != ScreenUnknown {
		t.Fatal("invented Claude recipe")
	}
	if ReadModelScreen("codex", Screen{Lines: base}).Phase != ScreenUnknown {
		t.Fatal("incomplete paint")
	}
}

func claudeCapture(t *testing.T, name string) []string {
	t.Helper()
	b, e := os.ReadFile("../wrapcmd/testdata/drive/claude/2.1.299/" + name + ".screen")
	if e != nil {
		t.Fatal(e)
	}
	return strings.Split(string(b), "\n")
}
func TestReadModelScreenClaudeCaptured(t *testing.T) {
	got := ReadModelScreen("claude", Screen{Complete: true, Lines: claudeCapture(t, "session-before")})
	if got.Phase != ScreenCombined || got.Selected != "opus 5.5" || got.Effort != "medium" {
		t.Fatalf("%#v", got)
	}
	got = ReadModelScreen("claude", Screen{Complete: true, Idle: true, Lines: claudeCapture(t, "session-status")})
	if got.Phase != ScreenStatus || got.Model != "opus 5.5" || got.Effort != "medium" || got.Scope != ScopeSession {
		t.Fatalf("%#v", got)
	}
}
func TestReadModelScreenCodexAdvancedAndDefaultCaptured(t *testing.T) {
	entry := ReadModelScreen("codex", Screen{Complete: true, Lines: capture(t, "more-entry")})
	if entry.Phase != ScreenEffort || entry.Selected != "more" {
		t.Fatalf("entry %#v", entry)
	}
	advanced := ReadModelScreen("codex", Screen{Complete: true, Lines: capture(t, "more-expanded")})
	if advanced.Phase != ScreenPhase("advanced") || advanced.Selected != "max" || !reflect.DeepEqual(advanced.Labels, []string{"max", "ultra"}) {
		t.Fatalf("advanced %#v", advanced)
	}
	status := ReadModelScreen("codex", Screen{Complete: true, Idle: true, Lines: capture(t, "default-status")})
	if status.Phase != ScreenStatus || status.Scope != ScopeDefault || status.Effort != "low" {
		t.Fatalf("default %#v", status)
	}
}

func TestReadModelScreenCurrentStatusCountsHistoricalConfirmations(t *testing.T) {
	lines := capture(t, "session-status")
	history := append(append([]string{}, lines...), lines...)
	e := ReadModelScreen("codex", Screen{Complete: true, Idle: true, Lines: lines, History: history})
	if e.Phase != ScreenStatus || e.ConfirmationCount != 2 {
		t.Fatalf("history confirmation evidence %#v", e)
	}
	if e := ReadModelScreen("codex", Screen{Complete: true, Idle: true, Lines: []string{"unknown viewport"}, History: history}); e.Phase != ScreenUnknown {
		t.Fatalf("historical status actionable %#v", e)
	}
}

func TestReadModelScreenCodexExplicitNativeSelection(t *testing.T) {
	native := capture(t, "native-model-picker")
	gotNative := ReadModelScreen("codex", Screen{Complete: true, Lines: native})
	if gotNative.Phase != ScreenModel || gotNative.Selected != "gpt-6.1-sol" {
		t.Fatalf("actual native capture declined %#v", gotNative)
	}
	// Actual Run/native relay conformance on Codex 0.162.1 reports no SGR7/27
	// before Pair receives output. Preserve the captured row text and explicit
	// cursor prefix while removing only the older capture's reverse styling.
	for _, tc := range []struct {
		name     string
		phase    ScreenPhase
		selected string
	}{{"model-picker", ScreenModel, "gpt-6.1-sol"}, {"effort-picker", ScreenEffort, "low"}, {"more-expanded", ScreenAdvanced, "max"}} {
		lines := capture(t, tc.name)
		for i, line := range lines {
			lines[i] = strings.ReplaceAll(strings.ReplaceAll(line, "‹rev›", ""), "‹/rev›", "")
		}
		got := ReadModelScreen("codex", Screen{Complete: true, Lines: lines})
		if got.Phase != tc.phase || got.Selected != tc.selected {
			t.Fatalf("%s native prefix declined: %#v", tc.name, got)
		}
		if got := ReadModelScreen("codex", Screen{Lines: lines}); got.Phase != ScreenUnknown {
			t.Fatal("incomplete native picker accepted")
		}
	}
}

func TestReadModelScreenCodexSelectionRequiresUniqueExplicitPrefix(t *testing.T) {
	base := []string{"Select Model and Effort", "› 1. GPT-6.1-Sol (current)", "2. GPT-6-Astra", "enter select · esc back"}
	for _, lines := range [][]string{
		{base[0], "‹rev›1. GPT-6.1-Sol (current)‹/rev›", base[2], base[3]},
		{base[0], base[1], "› 2. GPT-6-Astra", base[3]},
		{base[0], base[1], "2. GPT-6.1-Sol", base[3]},
		{base[0], "‹rev›› 1. GPT-6.1-Sol", base[2], base[3]},
		{base[0], base[1], base[2], "unrecognized footer"},
	} {
		if got := ReadModelScreen("codex", Screen{Complete: true, Lines: lines}); got.Phase != ScreenUnknown {
			t.Fatalf("ambiguous native picker accepted %#v", got)
		}
	}
}

func TestNativePrefixSelectionRetainsDriveFreshnessGuard(t *testing.T) {
	now := time.Unix(100, 0)
	s, err := NewDrive(Request{Model: &ModelRequest{Model: "gpt-6-astra"}}, "codex", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventStart, At: now, Screen: Screen{Epoch: 0, Complete: true, Idle: true}})
	s = acknowledgeModelCommand(t, s, now)
	screen := Screen{Epoch: 2, Complete: true, Lines: capture(t, "native-model-picker")}
	s, fx := ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: screen})
	if len(fx) != 1 || fx[0].Key.Kind != KeyDown {
		t.Fatalf("native selection not actionable %#v", fx)
	}
	s, _ = ReduceDrive(s, DriveEvent{Kind: EventWritten, At: now, Screen: screen})
	_, fx = ReduceDrive(s, DriveEvent{Kind: EventScreen, At: now, Screen: screen})
	for _, e := range fx {
		if e.Kind == EffectWrite {
			t.Fatal("same native frame replayed navigation")
		}
	}
}

func FuzzReadModelScreen(f *testing.F) {
	b, err := os.ReadFile("../wrapcmd/testdata/drive/codex/0.162.1/native-model-picker.screen")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(string(b))
	for _, path := range []string{"../wrapcmd/testdata/drive/codex/0.162.1/model-picker.screen", "../wrapcmd/testdata/drive/codex/0.162.1/effort-picker.screen", "../wrapcmd/testdata/drive/codex/0.162.1/session-status.screen", "../wrapcmd/testdata/drive/codex/0.162.1/more-entry.screen", "../wrapcmd/testdata/drive/codex/0.162.1/more-expanded.screen", "../wrapcmd/testdata/drive/codex/0.162.1/default-status.screen", "../wrapcmd/testdata/drive/claude/2.1.299/session-before.screen", "../wrapcmd/testdata/drive/claude/2.1.299/session-status.screen"} {
		b, e := os.ReadFile(path)
		if e != nil {
			f.Fatal(e)
		}
		f.Add(string(b))
	}
	for _, s := range []string{"Select Model and Effort\n‹rev›› 1. GPT-6.1-Sol‹/rev›\nenter select · esc back", "‹‹rev›", "‹rev›x", "‹dim›‹rev›x‹/dim›‹/rev›"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := ReadModelScreen("codex", Screen{Complete: true, Lines: strings.Split(s, "\n")})
		if len(got.Labels) > 128 {
			t.Fatal("labels overflow")
		}
		seen := map[string]bool{}
		for _, l := range got.Labels {
			if seen[l] || len(l) > 128 {
				t.Fatal("invalid labels")
			}
			seen[l] = true
		}
		if got.Phase == ScreenModel || got.Phase == ScreenEffort || got.Phase == ScreenAdvanced {
			if !seen[got.Selected] {
				t.Fatal("selection absent")
			}
		}
	})
}
