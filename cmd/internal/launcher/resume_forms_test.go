package launcher

import (
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/resumeform"
)

// BR-15: every resume spelling the create path honors must also be stripped
// from the persisted config and refused on fresh launches. These rows pin the
// two divergences the M1 boundary review measured; the table-range test below
// keeps the whole family mechanical.
func TestResumeSpellingDivergences(t *testing.T) {
	if got := extractExplicitResume("qoder", []string{"-rabc"}); got != "abc" {
		t.Errorf("glued qoder resume not extracted: %q", got)
	}
	if got := extractExplicitResume("claude", []string{"-rabc"}); got != "abc" {
		t.Errorf("glued claude resume not extracted: %q", got)
	}
	if got := extractExplicitResume("qoder", []string{"-r=abc"}); got != "abc" {
		t.Errorf("inline short resume not extracted: %q", got)
	}
	if got := persistedConfigArgs([]string{"--model", "m", "-rabc"}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
		t.Errorf("glued resume survives persist: %v", got)
	}
	if err := ValidateFreshAgentArgs("qoder", []string{"-rabc"}); err == nil {
		t.Error("glued resume accepted on fresh launch")
	}
	// A valueless space form must not swallow the flag that follows it.
	if got := persistedConfigArgs([]string{"--resume", "--model", "m"}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
		t.Errorf("valueless resume ate the next flag: %v", got)
	}
	if got := persistedConfigArgs([]string{"-r", "--model", "m"}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
		t.Errorf("valueless short resume ate the next flag: %v", got)
	}
}

// The table (resumeform.Forms) is the contract for the whole family: every
// spelling must extract, strip from the persisted config, and be refused by
// the fresh-arg validator — so a form added to the table cannot be honored at
// one site and lost at another (BR-15).
func TestResumeFormTableRoundTrip(t *testing.T) {
	for agent, form := range resumeform.Forms {
		for _, spelling := range form.Space {
			pinned := []string{"--model", "m", spelling, "sid"}
			if got := extractExplicitResume(agent, pinned); got != "sid" {
				t.Errorf("%s space %q extracted %q", agent, spelling, got)
			}
			if got := persistedConfigArgs(pinned); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s space %q survives persist: %v", agent, spelling, got)
			}
			if err := ValidateFreshAgentArgs(agent, []string{spelling, "sid"}); err == nil {
				t.Errorf("%s space %q accepted on fresh launch", agent, spelling)
			}
		}
		for _, prefix := range form.Inline {
			tok := prefix + "sid"
			if got := extractExplicitResume(agent, []string{"--model", "m", tok}); got != "sid" {
				t.Errorf("%s inline %q extracted %q", agent, tok, got)
			}
			if got := persistedConfigArgs([]string{"--model", "m", tok}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s inline %q survives persist: %v", agent, tok, got)
			}
			if err := ValidateFreshAgentArgs(agent, []string{tok}); err == nil {
				t.Errorf("%s inline %q accepted on fresh launch", agent, tok)
			}
		}
		for _, spelling := range form.Glued {
			tok := spelling + "sid"
			if got := extractExplicitResume(agent, []string{"--model", "m", tok}); got != "sid" {
				t.Errorf("%s glued %q extracted %q", agent, tok, got)
			}
			if got := persistedConfigArgs([]string{"--model", "m", tok}); !reflect.DeepEqual(got, []string{"--model", "m"}) {
				t.Errorf("%s glued %q survives persist: %v", agent, tok, got)
			}
			if err := ValidateFreshAgentArgs(agent, []string{tok}); err == nil {
				t.Errorf("%s glued %q accepted on fresh launch", agent, tok)
			}
		}
	}
}
