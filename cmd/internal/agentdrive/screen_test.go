package agentdrive

import (
	"os"
	"strings"
	"testing"
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
func FuzzReadModelScreen(f *testing.F) {
	for _, path := range []string{"../wrapcmd/testdata/drive/codex/0.162.1/model-picker.screen", "../wrapcmd/testdata/drive/codex/0.162.1/effort-picker.screen", "../wrapcmd/testdata/drive/codex/0.162.1/session-status.screen", "../wrapcmd/testdata/drive/claude/2.1.299/session-before.screen", "../wrapcmd/testdata/drive/claude/2.1.299/session-status.screen"} {
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
		if got.Phase == ScreenModel || got.Phase == ScreenEffort {
			if !seen[got.Selected] {
				t.Fatal("selection absent")
			}
		}
	})
}
