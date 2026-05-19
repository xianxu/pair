package main

import (
	"testing"
)

// TestIsActionableOSC pins which OSC sequences pair-wrap chooses to
// forward to the outer terminal and which it intentionally swallows.
// Misclassifying here either:
//   - leaks every claude title-update / iTerm progress tick to the
//     outer terminal, producing notification spam, or
//   - swallows real notification sequences, breaking the agent-
//     attention-needed signal pair was specifically built to emit.
//
// The table is the contract — preserve it.
func TestIsActionableOSC(t *testing.T) {
	cases := []struct {
		name string
		ps   string
		body string
		want bool
	}{
		// Forwarded:
		{"urxvt 777 forwarded", "777", "notify;Title;Body", true},
		{"iTerm 9 with text body forwarded", "9", "agent needs attention", true},
		{"iTerm 9 with empty body forwarded", "9", "", true},

		// Swallowed:
		{"OSC 0 title-set swallowed (claude updates every second)", "0", "Title", false},
		{"OSC 1 icon-name swallowed", "1", "Icon", false},
		{"OSC 2 window-title swallowed", "2", "Window Title", false},
		{"iTerm 9;4; progress swallowed", "9", "4;75", false},
		{"iTerm 9;4 (no semicolon body) still swallowed", "9", "4;", false},

		// Vendor / unknown:
		{"OSC 1337 iTerm vendor extension swallowed", "1337", "File=name=foo:base64", false},
		{"OSC 4 palette query swallowed", "4", "0;rgb:00/00/00", false},
		{"OSC 52 clipboard swallowed", "52", "c;base64", false},
		{"OSC 8 hyperlink swallowed", "8", ";http://example", false},
		{"unknown OSC number swallowed", "12345", "anything", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isActionableOSC([]byte(c.ps), []byte(c.body))
			if got != c.want {
				t.Errorf("isActionableOSC(%q, %q) = %v, want %v",
					c.ps, c.body, got, c.want)
			}
		})
	}
}

// TestOscRe_MatchesShape covers the regex itself — what counts as a
// "complete" OSC sequence that the dispatch loop will inspect. Two
// terminators (BEL = \x07, ST = \x1b\\) are both valid; partial
// sequences without a terminator must not match.
func TestOscRe_MatchesShape(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		matches   bool
		wantPs    string // captured first group, only checked on match
		wantBody  string // captured second group, only checked on match
	}{
		{
			name:     "OSC 9 BEL-terminated",
			input:    "\x1b]9;hello\x07",
			matches:  true,
			wantPs:   "9",
			wantBody: "hello",
		},
		{
			name:     "OSC 777 ST-terminated (\\x1b\\\\)",
			input:    "\x1b]777;notify;Title;Body\x1b\\",
			matches:  true,
			wantPs:   "777",
			wantBody: "notify;Title;Body",
		},
		{
			name:     "OSC 0 empty body BEL",
			input:    "\x1b]0;\x07",
			matches:  true,
			wantPs:   "0",
			wantBody: "",
		},
		{
			name:     "OSC inside a longer chunk",
			input:    "noise before \x1b]9;hi\x07 noise after",
			matches:  true,
			wantPs:   "9",
			wantBody: "hi",
		},
		{
			name:    "no terminator → no match",
			input:   "\x1b]9;hello",
			matches: false,
		},
		{
			name:    "not an OSC start (CSI) → no match",
			input:   "\x1b[9m",
			matches: false,
		},
		{
			name:    "OSC without ps digits → no match",
			input:   "\x1b];body\x07",
			matches: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := oscRe.FindStringSubmatch(c.input)
			if c.matches {
				if m == nil {
					t.Fatalf("no match in %q", c.input)
				}
				if m[1] != c.wantPs {
					t.Errorf("ps: got %q, want %q", m[1], c.wantPs)
				}
				if m[2] != c.wantBody {
					t.Errorf("body: got %q, want %q", m[2], c.wantBody)
				}
			} else {
				if m != nil {
					t.Errorf("unexpected match %v in %q", m, c.input)
				}
			}
		})
	}
}
