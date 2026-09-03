package wrapcmd

import (
	"bytes"
	"testing"
)

// extractFG is the SGR-parameter foreground-color resolver feeding
// updateAgentOutput's span capture. Wrong output here cascades into
// every downstream colored-span heuristic (nvim word completion pool,
// claude end-of-turn marker → OSC9 notify), so the surface deserves
// table-driven coverage of the cases we actually see in agent streams.
func TestExtractFG(t *testing.T) {
	cases := []struct {
		name    string
		params  string // contents of CSI <params> m, without the m
		current []byte // FG before this SGR
		want    []byte // FG after
	}{
		{
			name:    "empty params == full reset",
			params:  "",
			current: []byte("34"),
			want:    nil,
		},
		{
			name:    "code 0 resets",
			params:  "0",
			current: []byte("31"),
			want:    nil,
		},
		{
			name:    "code 39 resets FG only",
			params:  "39",
			current: []byte("31"),
			want:    nil,
		},
		{
			name:   "16-color FG (30-37)",
			params: "31",
			want:   []byte("31"),
		},
		{
			name:   "16-color bright FG (90-97)",
			params: "92",
			want:   []byte("92"),
		},
		{
			name:   "256-color indexed 38;5;N",
			params: "38;5;208",
			want:   []byte("5;208"),
		},
		{
			name:   "truecolor 38;2;R;G;B",
			params: "38;2;177;185;249",
			want:   []byte("2;177;185;249"),
		},
		{
			name:    "later FG wins (red then default → default)",
			params:  "31;39",
			current: nil,
			want:    nil,
		},
		{
			name:    "earlier bold leaves FG untouched when current is set",
			params:  "1",
			current: []byte("31"),
			want:    []byte("31"),
		},
		{
			name:    "background code (40-47) leaves FG untouched",
			params:  "44",
			current: []byte("31"),
			want:    []byte("31"),
		},
		{
			name:   "compound: reset then 256-color",
			params: "0;38;5;177",
			want:   []byte("5;177"),
		},
		{
			name:   "compound: bold + truecolor",
			params: "1;38;2;255;0;0",
			want:   []byte("2;255;0;0"),
		},
		{
			name:   "empty parameter slot == 0 (reset)",
			params: ";1",       // ECMA-48: omitted == 0
			current: []byte("31"),
			want:   nil,
		},
		{
			name:   "malformed 38 (missing mode/index) is a no-op for FG",
			params: "38",
			current: []byte("36"),
			want:   []byte("36"),
		},
		{
			name:   "malformed 38;5 (missing index) is a no-op for FG",
			params: "38;5",
			current: []byte("36"),
			want:   []byte("36"),
		},
		{
			name:   "malformed 38;2;R;G (missing B) is a no-op for FG",
			params: "38;2;1;2",
			current: []byte("36"),
			want:   []byte("36"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractFG([]byte(c.params), c.current)
			if !bytes.Equal(got, c.want) {
				t.Errorf("extractFG(%q, %q) = %q, want %q",
					c.params, c.current, got, c.want)
			}
		})
	}
}

func TestSplitBytes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{""}},
		{"1", []string{"1"}},
		{"1;2", []string{"1", "2"}},
		{"1;;3", []string{"1", "", "3"}}, // empty middle slot
		{";1", []string{"", "1"}},        // leading empty
		{"1;", []string{"1", ""}},        // trailing empty
		{"38;2;1;2;3", []string{"38", "2", "1", "2", "3"}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := splitBytes([]byte(c.in), ';')
			if len(got) != len(c.want) {
				t.Fatalf("len: got %d, want %d (got=%q)", len(got), len(c.want), got)
			}
			for i := range got {
				if string(got[i]) != c.want[i] {
					t.Errorf("[%d]: got %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
