package resumeform_test

import (
	"reflect"
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
