package couchmessage

import (
	"reflect"
	"testing"
)

// Tails rendered by the wrapper from pair#464's recorded screens
// (wrapcmd/testdata/peer/claude): a Claude Code 2.1.296 permission prompt and
// an AskUserQuestion picker, trimmed above the picker.
var (
	recordedPermissionTail = []string{
		"⏺ Bash(P=/private/tmp/proto; T=/private/tmp/fx…)",
		"  ⎿  Waiting…",
		"",
		"──────────────────────────────────────────────────────────────",
		" Bash command",
		" Copy fixtures into testdata and save the prototype patch",
		"╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌",
		" ‹dim›│‹/dim› D=cmd/internal/webcast/recording/testdata/fixtures; mkdir -p $D && for n in stress app",
		" ‹dim›│‹/dim› blog segments; do rm -rf $D/$n; done",
		"╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌",
		" ‹dim›│‹/dim› Dangerous rm operation on possibly-empty variable path: $D/$n in `rm -rf $D/$n`",
		" ⚠ Claude Code will automatically deny this request in 1:53, to avoid blocking progress on an",
		" unattended session",
		"",
		"",
		" Do you want to proceed?",
		" ❯ 1. Yes",
		"   2. No",
		"",
		" Esc to cancel · Tab to amend",
	}
	recordedQuestionTail = []string{
		"⏺ While that blocks, one thing worth surfacing now rather than at session end,",
		"  since it could still run today:",
		"────────────────────────────────────────────────────────────────────────────────",
		"←  ☐ API creds  ☐ Batch size  ✔ Submit  →",
		"",
		"‹dim›│‹/dim› The API sampler arm is blocked on credentials — no ANTHROPIC_API_KEY, no `ant`",
		"‹dim›│‹/dim› CLI, no `anthropic` SDK on this box. How do you want to unblock it?",
		"",
		"  1. You run `ant auth login` yourself",
		"     Type `! ant auth login` in the prompt so the interactive login lands in",
		"     this session.",
		"  2. Export a key into the environment",
		"     You set ANTHROPIC_API_KEY yourself and I use it.",
		"  3. Leave the API arm for next session",
		"❯ 4. ‹cursor›‹dim›Type something.‹/dim›",
		"────────────────────────────────────────────────────────────────────────────────",
		"  5. Chat about this",
		"",
		"Enter to select · Tab/Arrow keys to navigate · ctrl+g to edit in Nvim · Esc to",
		"cancel",
	}
	// A boxed prompt, the shape pair#451's fixtures took.
	boxedPermissionTail = []string{
		"  I'll push the branch now.",
		"",
		"╭──────────────────────────────────────────────╮",
		"│ Bash command                                 │",
		"│   git push origin HEAD                       │",
		"│ Do you want to proceed?                      │",
		"│ ❯ 1. Yes                                     │",
		"│   2. No, and tell Claude what to do (esc)    │",
		"╰──────────────────────────────────────────────╯",
	}
)

func TestReadPicker(t *testing.T) {
	cases := []struct {
		name string
		tail []string
		want Picker
	}{
		{"recorded permission prompt", recordedPermissionTail, Picker{
			Kind:    PickerPermission,
			Title:   "Do you want to proceed?",
			Detail:  "Bash command: D=cmd/internal/webcast/recording/testdata/fixtures; mkdir -p $D && for n in stress app blog segments; do rm -rf $D/$n; done",
			Options: []string{"1. Yes", "2. No"},
		}},
		{"recorded question picker", recordedQuestionTail, Picker{
			Kind:    PickerQuestion,
			Title:   "The API sampler arm is blocked on credentials — no ANTHROPIC_API_KEY, no `ant` CLI, no `anthropic` SDK on this box. How do you want to unblock it?",
			Options: []string{"1. You run `ant auth login` yourself", "2. Export a key into the environment", "3. Leave the API arm for next session", "4. Type something.", "5. Chat about this"},
		}},
		{"boxed permission prompt", boxedPermissionTail, Picker{
			Kind:    PickerPermission,
			Title:   "Do you want to proceed?",
			Detail:  "Bash command: git push origin HEAD",
			Options: []string{"1. Yes", "2. No, and tell Claude what to do (esc)"},
		}},
		// Wording the reader doesn't know is a picker, never a guessed question.
		{"unrecognized wording", []string{"Pick a model", "› 1. gpt-6", "  2. gpt-6-mini"}, Picker{
			Kind: PickerUnknown, Title: "Pick a model", Options: []string{"1. gpt-6", "2. gpt-6-mini"},
		}},
		{"numbered prose is no picker", []string{"Plan:", "1. run tests", "2. push", "", "❯ ‹cursor›"}, Picker{}},
		{"codex composer is no picker", []string{"  Not merged.", "", "› ‹cursor›‹dim›Ask Codex to do anything‹/dim›"}, Picker{}},
	}
	for _, c := range cases {
		if got := ReadPicker(c.tail); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %#v\nwant %#v", c.name, got, c.want)
		}
	}
}

// The picker is the screen's last one: a numbered list in the prose above it
// is neither its options nor its title.
func TestReadPickerIgnoresNumberedProseAbove(t *testing.T) {
	tail := []string{"Plan:", "1. run tests", "2. push", "", "Do you want to proceed?", "  1. Yes", "❯ 2. No, and tell Claude what to do (esc)"}
	got := ReadPicker(tail)
	if got.Title != "Do you want to proceed?" || !reflect.DeepEqual(got.Options, []string{"1. Yes", "2. No, and tell Claude what to do (esc)"}) {
		t.Fatalf("got %#v", got)
	}
}
