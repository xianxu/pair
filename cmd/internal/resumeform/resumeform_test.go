package resumeform_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/resumeform"
)

// The table is the contract: every spelling must extract, strip and read as a
// selector for its own agent — and only its own agent.
func TestEveryTableSpellingRoundTrips(t *testing.T) {
	for agent, form := range resumeform.Forms() {
		for _, spelling := range form.Space {
			pinned := []string{"--model", "m", spelling, "sid"}
			if got := resumeform.Extract(agent, pinned); got != "sid" {
				t.Errorf("%s space %q extracted %q, want sid", agent, spelling, got)
			}
			if got := resumeform.Strip(agent, pinned); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s space %q stripped to %v", agent, spelling, got)
			}
			if !resumeform.Selector(agent, spelling) {
				t.Errorf("%s space %q not a selector", agent, spelling)
			}
		}
		for _, prefix := range form.Inline {
			tok := prefix + "sid"
			if got := resumeform.Extract(agent, []string{"--model", "m", tok}); got != "sid" {
				t.Errorf("%s inline %q extracted %q, want sid", agent, tok, got)
			}
			if got := resumeform.Strip(agent, []string{"--model", "m", tok}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s inline %q stripped to %v", agent, tok, got)
			}
			if !resumeform.Selector(agent, tok) {
				t.Errorf("%s inline %q not a selector", agent, tok)
			}
		}
		for _, spelling := range form.Glued {
			tok := spelling + "sid"
			if got := resumeform.Extract(agent, []string{"--model", "m", tok}); got != "sid" {
				t.Errorf("%s glued %q extracted %q, want sid", agent, tok, got)
			}
			if got := resumeform.Strip(agent, []string{"--model", "m", tok}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s glued %q stripped to %v", agent, tok, got)
			}
			if !resumeform.Selector(agent, tok) {
				t.Errorf("%s glued %q not a selector", agent, tok)
			}
		}
	}
}

// BR-22: strip is strictly per-agent — a spelling only counts as a resume
// binding for the agent whose table carries it.
func TestStripIsPerAgent(t *testing.T) {
	if got := resumeform.Strip("codex", []string{"-rsid", "--model", "m"}); !reflect.DeepEqual(got, []string{"-rsid", "--model", "m"}) {
		t.Errorf("codex -rsid stripped: %v", got)
	}
	if got := resumeform.Strip("codex", []string{"--resume", "sid"}); !reflect.DeepEqual(got, []string{"--resume", "sid"}) {
		t.Errorf("codex --resume stripped: %v", got)
	}
	if got := resumeform.Strip("agy", []string{"-rsid"}); !reflect.DeepEqual(got, []string{"-rsid"}) {
		t.Errorf("agy -rsid stripped: %v", got)
	}
	if got := resumeform.Strip("claude", []string{"--conversation", "cid"}); !reflect.DeepEqual(got, []string{"--conversation", "cid"}) {
		t.Errorf("claude --conversation stripped: %v", got)
	}
}

// ShortLetters derives the cluster-forbidden letters from the Glued spellings
// (the launcher's fresh-arg validator consults it — BR-17).
func TestShortLettersDeriveFromGlued(t *testing.T) {
	for agent, want := range map[string]string{"claude": "r", "qoder": "r", "agy": "", "codex": ""} {
		if got := resumeform.ShortLetters(agent); got != want {
			t.Errorf("ShortLetters(%q) = %q, want %q", agent, got, want)
		}
	}
}

func TestSelectorIsPerAgent(t *testing.T) {
	if resumeform.Selector("agy", "-rsid") {
		t.Error("agy has no short resume form; agy must not accept claude's glued spelling")
	}
	if resumeform.Selector("codex", "--resume") {
		t.Error("codex resumes via the subcommand; --resume must not read as its selector")
	}
	if resumeform.Selector("qoder", "--conversation") || resumeform.Selector("qoder", "--fork-session") {
		t.Error("qoder must not accept other agents' spellings")
	}
}

func TestValuelessSpaceFormKeepsTheNextFlag(t *testing.T) {
	for agent, form := range resumeform.Forms() {
		for _, spelling := range form.Space {
			got := resumeform.Strip(agent, []string{spelling, "--model", "m"})
			if !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s: valueless %q ate the next flag: %v", agent, spelling, got)
			}
			if got := resumeform.Extract(agent, []string{spelling, "--model", "m"}); got != "" {
				t.Errorf("%s: valueless %q pinned %q", agent, spelling, got)
			}
		}
	}
	if got := resumeform.Strip("claude", []string{"-r", "--model", "m"}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
		t.Errorf("valueless short form ate the next flag: %v", got)
	}
}

func TestStripLeavesUnrelatedTokens(t *testing.T) {
	args := []string{"--model", "m", "a prompt about resume", "--", "-x"}
	if got := resumeform.Strip("qoder", args); !reflect.DeepEqual(got, args) {
		t.Fatalf("strip = %v", got)
	}
	if got := resumeform.Extract("qoder", []string{"hello", "world"}); got != "" {
		t.Fatalf("prompt pinned %q", got)
	}
}

func TestBareInlineAndEmptyGluedDoNotPin(t *testing.T) {
	if got := resumeform.Extract("qoder", []string{"--resume=", "--model", "m"}); got != "" {
		t.Errorf("bare --resume= pinned %q", got)
	}
	if got := resumeform.Extract("qoder", []string{"-r="}); got != "" {
		t.Errorf("-r= pinned %q", got)
	}
	if got := resumeform.Extract("qoder", []string{"-r"}); got != "" {
		t.Errorf("bare -r pinned %q", got)
	}
}

// The context-selector groups (SessionID, Continue) are part of the same
// contract: every spelling strips from persisted args, reads as a context
// selector, and contributes its single letter to the cluster-forbidden set.
func TestEveryContextSelectorSpellingRoundTrips(t *testing.T) {
	for agent, form := range resumeform.Forms() {
		for _, spelling := range form.SessionID {
			for _, args := range [][]string{{"--model", "m", spelling, "sid"}, {"--model", "m", spelling + "=sid"}} {
				if got := resumeform.Strip(agent, args); !reflect.DeepEqual(got, []string{"--model", "m"}) {
					t.Errorf("%s session-id %v stripped to %v", agent, args, got)
				}
				if !resumeform.HasSessionID(agent, args) {
					t.Errorf("%s session-id %v not detected", agent, args)
				}
			}
			if !resumeform.ContextSelector(agent, spelling) {
				t.Errorf("%s session-id %q not a context selector", agent, spelling)
			}
		}
		for _, spelling := range form.Continue {
			if got := resumeform.Strip(agent, []string{spelling, "--model", "m"}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s continue %q stripped to %v", agent, spelling, got)
			}
			if !resumeform.ContextSelector(agent, spelling) {
				t.Errorf("%s continue %q not a context selector", agent, spelling)
			}
		}
		for _, spelling := range append(append([]string(nil), form.SessionID...), form.Continue...) {
			if letter, ok := strings.CutPrefix(spelling, "-"); ok && len(letter) == 1 && !strings.Contains(resumeform.ContextShortLetters(agent), letter) {
				t.Errorf("%s short %q missing from ContextShortLetters %q", agent, spelling, resumeform.ContextShortLetters(agent))
			}
		}
	}
}

// A continue spelling is valueless: the token after it is never consumed.
func TestContinueIsValueless(t *testing.T) {
	if got := resumeform.Strip("grok", []string{"-c", "fix", "the", "bug"}); !reflect.DeepEqual(got, []string{"fix", "the", "bug"}) {
		t.Errorf("grok -c consumed a positional: %v", got)
	}
}

// The groups are per-agent like the resume spellings (BR-22): grok's `-s` and
// `-c` mean nothing for claude here (claude's selectors live in its fresh spec).
func TestContextSelectorIsPerAgent(t *testing.T) {
	if resumeform.ContextSelector("claude", "-s") || resumeform.ContextSelector("codex", "-c") {
		t.Error("context selector leaked across agents")
	}
	if got := resumeform.Strip("codex", []string{"-c", "k=v"}); !reflect.DeepEqual(got, []string{"-c", "k=v"}) {
		t.Errorf("codex -c (a config override) stripped: %v", got)
	}
	if resumeform.HasSessionID("codex", []string{"-s", "x"}) {
		t.Error("codex -s read as a session id")
	}
}

// Everything after `--` is the agent's prompt text, never a binding: every
// spelling of every agent placed there survives Strip and is not extracted or
// read as a session id.
func TestDoubleDashTailIsNeverABinding(t *testing.T) {
	for agent, form := range resumeform.Forms() {
		var spellings [][]string
		for _, s := range form.Space {
			spellings = append(spellings, []string{s, "sid"})
		}
		for _, s := range append(append([]string(nil), form.Inline...), form.Glued...) {
			spellings = append(spellings, []string{s + "sid"})
		}
		for _, s := range form.SessionID {
			spellings = append(spellings, []string{s, "sid"}, []string{s + "=sid"})
		}
		for _, s := range form.Continue {
			spellings = append(spellings, []string{s})
		}
		for _, tail := range spellings {
			args := append([]string{"--model", "m", "--"}, tail...)
			if got := resumeform.Strip(agent, args); !reflect.DeepEqual(got, args) {
				t.Errorf("%s stripped prompt text after --: %v -> %v", agent, args, got)
			}
			if got := resumeform.Extract(agent, args); got != "" {
				t.Errorf("%s extracted %q from prompt text after --: %v", agent, got, args)
			}
			if resumeform.HasSessionID(agent, args) {
				t.Errorf("%s read a session id from prompt text after --: %v", agent, args)
			}
		}
	}
}

// A single-letter SessionID spelling also takes its value glued (`-s<uuid>`),
// like the glued resume form, so the mint check and the strip both see it.
func TestGluedSessionIDShortForm(t *testing.T) {
	args := []string{"--model", "m", "-s12345678-1234-4234-8234-123456789abc"}
	if !resumeform.HasSessionID("grok", args) {
		t.Error("glued -s<uuid> not read as a session id")
	}
	if got := resumeform.Strip("grok", args); !reflect.DeepEqual(got, []string{"--model", "m"}) {
		t.Errorf("glued -s<uuid> stripped to %v", got)
	}
	if !resumeform.ContextSelector("grok", "-sabc") {
		t.Error("glued -s<uuid> not a context selector")
	}
	if resumeform.HasSessionID("grok", []string{"--session-idX"}) {
		t.Error("a long spelling must not take a glued value")
	}
}
