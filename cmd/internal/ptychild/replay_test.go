package ptychild

import "testing"

// Every row of the deny-list is stripped out of a replay.
func TestStripTerminalQueriesRemovesEachRow(t *testing.T) {
	rows := []struct {
		name  string
		query string
	}{
		{"DA1", "\x1b[c"},
		{"DA1 explicit zero", "\x1b[0c"},
		{"DA2", "\x1b[>c"},
		{"XTVERSION", "\x1b[>q"},
		{"kitty flags query", "\x1b[?u"},
		{"DSR cursor position", "\x1b[6n"},
		{"OSC 10 foreground", "\x1b]10;?\x07"},
		{"OSC 11 background", "\x1b]11;?\x07"},
		{"OSC 11 ST terminated", "\x1b]11;?\x1b\\"},
		{"DECRQM 2026", "\x1b[?2026$p"},
		{"DECRQM 2031", "\x1b[?2031$p"},
		{"OSC 4 colour", "\x1b]4;12;?\x07"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			in := "before" + row.query + "after"
			if got := string(StripQueries([]byte(in))); got != "beforeafter" {
				t.Fatalf("strip(%q) = %q, want %q", in, got, "beforeafter")
			}
		})
	}
}

// The backstop for the deny-list: for every final byte the table matches, a
// legitimate sequence sharing that final must SURVIVE. A greedy rule here is
// silent — it breaks mouse mode, key encoding, or the cursor shape with nothing
// failing.
func TestStripTerminalQueriesPreservesLegitimateSequences(t *testing.T) {
	keep := []struct {
		name string
		seq  string
		why  string
	}{
		{"DECSET SGR mouse", "\x1b[?1006h", "shares \\x1b[? with DECRQM; updateMouseMode parses it"},
		{"DECSET button mouse", "\x1b[?1002h", "same"},
		{"DECSET bracketed paste", "\x1b[?2004h", "same"},
		{"DECRST mouse off", "\x1b[?1006l", "same, reset form"},
		{"kitty flags push", "\x1b[>1u", "shares the u final with the \\x1b[?u query"},
		{"kitty flags pop", "\x1b[<u", "same; stripping it drops the app to legacy keys"},
		{"kitty key chord", "\x1b[110;3u", "Alt+n as a live chord, u final"},
		{"DECSCUSR cursor shape", "\x1b[5 q", "nvim emits per mode change; q final like XTVERSION"},
		{"SGR reset", "\x1b[0m", "ordinary styling"},
		{"OSC 0 title", "\x1b]0;title\x07", "non-query OSC"},
		{"DA1 reply", "\x1b[?62;4;52c", "shell echo puts replies in the buffer; c final"},
		{"DECRPM report", "\x1b[?2026;2$y", "reply terminates $y, not $p"},
		{"kitty flags reply", "\x1b[?0u", "reply, not the \\x1b[?u query literal"},
		{"DSR cursor report", "\x1b[24;1R", "the reply to 6n"},
		{"DSR status request", "\x1b[0n", "n final, but not the 6n cursor-position query"},
		// Malformed OSC 4: the 4-byte prefix and 2-byte suffix checks OVERLAP
		// here, which used to invert a slice bound and panic (found at close
		// review). Not a query — must pass through.
		{"malformed OSC 4 BEL", "\x1b]4;?\x07", "prefix/suffix overlap; must not panic or match"},
		{"malformed OSC 4 ST", "\x1b]4;?\x1b\\", "same, ST terminated"},
		{"malformed OSC 4 no index", "\x1b]4;;?\x07", "empty index is not a colour query"},
	}
	for _, k := range keep {
		t.Run(k.name, func(t *testing.T) {
			in := "before" + k.seq + "after"
			if got := string(StripQueries([]byte(in))); got != in {
				t.Fatalf("strip(%q) = %q, want unchanged (%s)", in, got, k.why)
			}
		})
	}
}

// The Ring keeps only the last DefaultRingBytes, so it can begin or end
// mid-sequence. Never drop-to-end: that would swallow the visible screen.
func TestStripTerminalQueriesHandlesTruncatedSequences(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"unterminated tail", "visible text\x1b[?100"},
		{"bare escape at end", "visible text\x1b"},
		{"unterminated OSC tail", "visible text\x1b]11;?"},
		{"buffer starts mid-sequence", "6;4;52c\x1b[?1006hvisible"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(StripQueries([]byte(tt.in))); got != tt.in {
				t.Fatalf("strip(%q) = %q, want unchanged", tt.in, got)
			}
		})
	}
}

// The structural guard C1 was missing: strip must never panic and never grow
// the buffer, for ANY input. Child output is arbitrary bytes (`cat` a binary and
// a malformed OSC 4 arrives), and every other strip test feeds a syntactically
// valid sequence — which is exactly how the \x1b]4;? slice-bound panic shipped.
func FuzzStripQueries(f *testing.F) {
	seeds := []string{
		"", "\x1b", "\x1b[", "\x1b]", "\x1b]4;?\x07", "\x1b]4;\x07",
		"\x1b[?$p", "\x1b[?u", "\x1b]11;?", "\x1b[c\x1b]4;12;?\x07plain",
		"\x1b]4;?\x1b\\", "\x1bP+q\x1b\\", "\x1b[?1006h\x1b[?2026$p",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		out := StripQueries(in) // must not panic
		if len(out) > len(in) {
			t.Fatalf("strip grew the buffer: %d > %d", len(out), len(in))
		}
	})
}
