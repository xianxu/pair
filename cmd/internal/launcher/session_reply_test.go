package launcher

import "testing"

func TestValidateSessionListing(t *testing.T) {
	tests := []struct {
		name, raw    string
		short, valid bool
	}{
		{"verified absence full", "", false, true},
		{"verified absence short", "", true, true},
		{"full running", "pair-demo [Created 1s ago] \n", false, true},
		{"full current", "pair-demo [Created 0s ago] (current)\n", false, true},
		{"full exited", "pair-demo [Created 2days 4h 3ms ago] (EXITED - attach to resurrect)\n", false, true},
		{"native duration units", "foreign [Created 1year 2months 3days 4h 5m 6s 7ms 8us 9ns ago] \n", false, true},
		{"foreign spaces", "a foreign session [Created 1month ago] \npair-demo [Created 1day ago] \n", false, true},
		{"foreign tabs", "a\tforeign [Created 1s ago] \n", false, true},
		{"foreign surrounding spaces", " foreign session  [Created 1s ago] \n", false, true},
		{"foreign metadata in name", "foreign [Created 1s ago] session [Created 2s ago] \n", false, true},
		{"foreign JSON-like full name", "{} [Created 1s ago] \n", false, true},
		{"short names", "pair-demo\nforeign session\n会话\n", true, true},
		{"short foreign tabs", "foreign\tsession\n", true, true},
		{"short punctuation", "my-session_2:dev\n", true, true},
		{"bare full", "pair-demo\n", false, false},
		{"truncated metadata", "pair-demo [Created 1s ago\n", false, false},
		{"unknown duration", "pair-demo [Created yesterday ago] \n", false, false},
		{"unknown suffix", "pair-demo [Created 1s ago] (unknown)\n", false, false},
		{"repeated suffix", "pair-demo [Created 1s ago] (current) (EXITED - attach to resurrect)\n", false, false},
		{"duplicate full", "pair-demo [Created 1s ago] \npair-demo [Created 2s ago] (current)\n", false, false},
		{"duplicate short", "pair-demo\npair-demo\n", true, false},
		{"blank full row", "pair-demo [Created 1s ago] \n\n", false, false},
		{"blank short row", "pair-demo\n\n", true, false},
		{"whitespace only", " \n", true, false},
		{"client table full", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_2 bash\n", false, false},
		{"client table short", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_2 bash\n", true, false},
		{"pane json full", `[{"id":1,"is_plugin":false}]`, false, false},
		{"pane json short", `[{"id":1,"is_plugin":false}]`, true, false},
		{"tab json short", `{"tabs":[]}`, true, false},
		{"full sent to short", "pair-demo [Created 1s ago] \n", true, false},
		{"error diagnostic", "No active zellij sessions found.\n", true, false},
		{"invalid name", "..\n", true, false},
		{"slash name", "a/b\n", true, false},
		{"control character", "pair-demo\x00\n", true, false},
		{"invalid utf8", "pair-\xff\n", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSessionListing([]byte(tt.raw), tt.short); (err == nil) != tt.valid {
				t.Fatalf("validateSessionListing(%q, %v) = %v; valid = %v", tt.raw, tt.short, err, tt.valid)
			}
		})
	}
}

func FuzzValidateSessionListing(f *testing.F) {
	for _, seed := range []string{"", "pair-demo\n", "pair-demo [Created 1s ago] \n", `[]`, "a foreign session\n"} {
		f.Add([]byte(seed), true)
		f.Add([]byte(seed), false)
	}
	f.Fuzz(func(t *testing.T, raw []byte, short bool) {
		if len(raw) == 0 || validateSessionListing(raw, short) != nil {
			return
		}
		// A malformed additional row must poison otherwise accepted evidence.
		poisoned := append(append([]byte(nil), raw...), []byte("\n\x00invalid\n")...)
		if validateSessionListing(poisoned, short) == nil {
			t.Fatal("accepted listing containing a malformed additional row")
		}
	})
}
