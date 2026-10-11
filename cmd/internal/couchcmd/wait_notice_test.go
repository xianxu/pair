package couchcmd

import (
	"strings"
	"testing"
	"time"
)

// Tails below are shaped on live `couch --peek` captures (2026-10-11).
var (
	claudePermissionTail = []string{
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
	claudeSelectionTail = []string{
		"  Which layout should the notice use?",
		"",
		"❯ 1. Compact",
		"  2. Verbose",
		"",
		"Enter to select · ↑/↓ to navigate · Esc to cancel",
	}
	claudeQuestionTail = []string{
		"  The fix is on the branch and the tests pass.",
		"  Should I merge it now?",
		"",
		"✻ Cooked for 15m 32s · done 9:15 AM",
		"",
		"※ recap: I'm landing #318; next, you run the commit.",
		"  Then tell me to merge?",
		"",
		"──────────────────────────────────────────────",
		"❯ ‹cursor›",
		"──────────────────────────────────────────────",
		"  ⏵⏵ auto mode on (shift+tab to cycle)",
	}
	claudeDoneTail = []string{
		"  Review: the first close review approved it.",
		"  Merged as PR #247; pair:2 is back on its resting branch.",
		"",
		"✻ Baked for 20m 24s · done 1:01 PM",
		"",
		"──────────────────────────────────────────────",
		"❯ ‹cursor›‹dim›move this branch to :0‹/dim›",
		"──────────────────────────────────────────────",
	}
	codexDoneTail = []string{
		"  ‹dim›│‹/dim› plans/000426-review-pane",
		"  ‹dim›└    Passed seven targeted Lua suites‹/dim›",
		"",
		"‹dim›•‹/dim› Implemented #426: compact comment chains.",
		"",
		"  Not merged. Uses ‹‹markup›› in text.",
		"",
		"  ‹dim›Worked for 28m 1s • 12:46 PM‹/dim›",
		"",
		"",
		"› ‹cursor›‹dim›Ask Codex to do anything‹/dim›",
		"",
		"  GPT-6-Astra medium · ~/workspace/pair",
	}
)

func boolp(b bool) *bool { return &b }

func TestClassifyWait(t *testing.T) {
	const threshold = 10 * time.Minute
	quiet := 11 * time.Minute
	cases := []struct {
		name    string
		f       waitFacts
		tail    []string
		reason  waitReason
		summary string
		waiting bool
	}{
		{"permission dialog names the dialog", waitFacts{Quiet: quiet, Settled: boolp(false), Resting: false}, claudePermissionTail, waitPrompt, "Do you want to proceed?", true},
		{"selection dialog names its question", waitFacts{Quiet: quiet, Settled: boolp(false), Resting: true}, claudeSelectionTail, waitPrompt, "Which layout should the notice use?", true},
		{"unsettled without a dialog quotes the prose", waitFacts{Quiet: quiet, Settled: boolp(false), Resting: true}, claudeDoneTail, waitPrompt, "Merged as PR #247; pair:2 is back on its resting branch.", true},
		{"turn ended with a question", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: true}, claudeQuestionTail, waitQuestion, "Should I merge it now?", true},
		{"normal end on resting branch", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: true}, claudeDoneTail, "", "", false},
		{"codex normal end on resting branch", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: true}, codexDoneTail, "", "", false},
		{"idle off resting branch", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: false}, claudeDoneTail, waitIdle, "Merged as PR #247; pair:2 is back on its resting branch.", true},
		{"idle off resting with a message in flight", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: false, Pending: true}, claudeDoneTail, "", "", false},
		{"not quiet long enough", waitFacts{Quiet: 9 * time.Minute, Settled: boolp(false)}, claudePermissionTail, "", "", false},
		{"no Settled claim is unknown, never waiting", waitFacts{Quiet: quiet}, claudePermissionTail, "", "", false},
	}
	for _, c := range cases {
		reason, summary, waiting := classifyWait(c.f, threshold, c.tail)
		if reason != c.reason || summary != c.summary || waiting != c.waiting {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q, %v)", c.name, reason, summary, waiting, c.reason, c.summary, c.waiting)
		}
	}
}

func TestLastProseLineSkipsChrome(t *testing.T) {
	for _, c := range []struct {
		tail []string
		want string
	}{
		{claudeQuestionTail, "Should I merge it now?"},
		{codexDoneTail, "Not merged. Uses ‹markup›› in text."},
		{[]string{"plain", "", "last line"}, "last line"},
		{nil, ""},
	} {
		if got := lastProseLine(c.tail); got != c.want {
			t.Errorf("lastProseLine(%q) = %q, want %q", c.tail, got, c.want)
		}
	}
}

func TestIsQuestionToleratesTrailingMarkdown(t *testing.T) {
	for line, want := range map[string]bool{"Merge now?": true, "**Merge now?**": true, "(merge now?)": true, "Merged.": false, "": false} {
		if got := isQuestion(line); got != want {
			t.Errorf("isQuestion(%q) = %v", line, got)
		}
	}
}

func TestWaitNoticeBodyPutsSlotFirstAndBoundsTheSummary(t *testing.T) {
	body := waitNoticeBody("ariadne:3", 12*time.Minute+40*time.Second, waitPrompt, strings.Repeat("é", 300), "/store/wait-notice-off")
	if !strings.HasPrefix(body, "slot ariadne:3 waiting 12 min on a prompt: ") {
		t.Fatalf("body head %q", body[:60])
	}
	if !strings.Contains(body, "é…") || strings.Count(body, "é") != 100 {
		t.Fatalf("summary not bounded on a rune boundary: %q", body)
	}
	if !strings.HasSuffix(body, "(couch --peek ariadne:3; mute: add ariadne:3 to /store/wait-notice-off)") {
		t.Fatalf("body tail %q", body)
	}
}

func TestParseWaitOptOut(t *testing.T) {
	got := parseWaitOptOut("# muted\nariadne:2\n  pair:4  # noisy\n\n")
	if len(got) != 2 || !got["ariadne:2"] || !got["pair:4"] {
		t.Fatalf("opt-out %v", got)
	}
}
