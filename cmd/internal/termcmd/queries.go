package termcmd

import "bytes"

// Terminal capability queries, and why they must not be replayed (#127).
//
// `redrawTab` repaints a tab by writing its stored raw output back to the real
// terminal. That output still contains whatever CAPABILITY QUERIES the app in
// that tab emitted at startup — DA1, DECRQM, the Kitty keyboard flags probe.
// Replaying them re-ASKS the host terminal, which answers on our stdin; the
// pump then hands that answer to the currently active tab's shell, which tries
// to run it as a command. Observed as a shell line reading
// `execute: …\x1b[?62;4;52c\x1b[?2026;2$y…`.
//
// The fix is to strip queries out of the REPLAY only. The live path is
// untouched (`copyActiveOutput` writes each chunk to stdout separately), so an
// app's first, real query still reaches the terminal and still gets its answer —
// capability negotiation is not disturbed. We deliberately do NOT filter replies
// on the input path: a reply arriving while its own tab is active is solicited
// and correct, and dropping it would silently break that negotiation.
//
// This table is a best-effort DENY-LIST of what nvim / zsh / fzf actually emit.
// It does not need to be exhaustive: a query we miss simply degrades to the old
// behavior. There is no live conformance check behind it.
//
// Note replies DO reach this buffer — the pump writes one into the child's PTY,
// the shell's line discipline echoes it, and it returns through readPTY. So the
// rows are deliberately shaped so that no reply form matches: DECRPM replies
// terminate `$y` not `$p`, the Kitty reply `\x1b[?0u` is not the `\x1b[?u`
// literal, and `\x1b[?62;4;52c` is neither `\x1b[c` nor `\x1b[0c`.
//
// Params are kept OUT of the matched-variable part wherever possible. The
// private-parameter introducers (`?`, `>`, `<`, `=`) are what discriminate these
// rows from legitimate output sharing the same final byte, so a broad "any
// params" class would make a `u` rule eat the Kitty push `\x1b[>1u` and an
// `h`/`l` rule eat DECSET `\x1b[?1006h` — killing `updateMouseMode`.
//
// A separate policy table lives in `wrapcmd` (`codexKKPMarkers`,
// `codexSyncOutputMarkers`). The two are intentionally distinct and in one case
// OPPOSED: wrapcmd strips `\x1b[>7u` so codex stops pushing Kitty flags, while
// here `\x1b[>1u` must SURVIVE. Two policy tables is deliberate; sharing the
// *framing* code is tracked separately.
var terminalQueryLiterals = [][]byte{
	[]byte("\x1b[c"),    // DA1 — primary device attributes
	[]byte("\x1b[0c"),   // DA1, explicit-zero form
	[]byte("\x1b[>c"),   // DA2 — secondary device attributes
	[]byte("\x1b[>q"),   // XTVERSION
	[]byte("\x1b[?u"),   // Kitty keyboard — query current flags
	[]byte("\x1b[6n"),   // DSR — cursor position report
	[]byte("\x1b]10;?"), // OSC 10 — foreground colour
	[]byte("\x1b]11;?"), // OSC 11 — background colour
}

// stripTerminalQueries removes capability queries from a tab's stored output so
// a redraw cannot re-issue them. Pure: no IO, no state.
//
// An unterminated escape at end-of-buffer is emitted VERBATIM. The buffer can
// legitimately begin or end mid-sequence — `appendBuffer` re-slices to the last
// 128 KiB, which bisects whatever spans that boundary — and a "no final byte
// found, drop the rest" rule would silently swallow the tail of the replay, i.e.
// the visible screen.
func stripTerminalQueries(buf []byte) []byte {
	if len(buf) == 0 {
		return buf
	}
	out := make([]byte, 0, len(buf))
	for i := 0; i < len(buf); {
		if buf[i] != 0x1b {
			// Bulk-copy the run up to the next escape rather than byte-at-a-time:
			// this path is ~all of a 128 KiB replay.
			next := bytes.IndexByte(buf[i:], 0x1b)
			if next < 0 {
				out = append(out, buf[i:]...)
				break
			}
			out = append(out, buf[i:i+next]...)
			i += next
			continue
		}
		size, isQuery, ok := terminalSequenceAt(buf[i:])
		if !ok {
			// Unterminated (or not an escape we frame) — emit the rest as-is.
			out = append(out, buf[i:]...)
			break
		}
		if !isQuery {
			out = append(out, buf[i:i+size]...)
		}
		i += size
	}
	return out
}

// terminalSequenceAt frames the escape sequence at the start of buf and reports
// whether it is a capability query. ok is false when the sequence is not
// terminated within buf (a truncated tail), which the caller emits verbatim.
func terminalSequenceAt(buf []byte) (size int, isQuery bool, ok bool) {
	for _, lit := range terminalQueryLiterals {
		if bytes.HasPrefix(buf, lit) {
			// OSC literals still need their terminator consumed.
			if lit[1] == ']' {
				if end, found := oscEnd(buf); found {
					return end, true, true
				}
				return 0, false, false
			}
			return len(lit), true, true
		}
	}
	if len(buf) < 2 {
		return 0, false, false
	}
	switch buf[1] {
	case '[':
		end := csiEnd(buf)
		if end < 0 {
			return 0, false, false
		}
		return end, isParameterizedCSIQuery(buf[:end]), true
	case ']':
		end, found := oscEnd(buf)
		if !found {
			return 0, false, false
		}
		return end, isParameterizedOSCQuery(buf[:end]), true
	default:
		return 2, false, true
	}
}

// csiEnd returns the length of the CSI sequence at the start of buf, or -1 when
// it is not terminated within buf. The one final-byte scan in this package —
// escapeSequenceIncomplete and malformedEscapeSize both derive from it, since
// three independent copies of "where does this CSI end" is the divergence class
// that caused this issue's first defect.
func csiEnd(buf []byte) int {
	for i := 2; i < len(buf); i++ {
		if isTerminalFinalByte(buf[i]) {
			return i + 1
		}
	}
	return -1
}

// oscEnd returns the length of the OSC sequence at the start of buf, terminated
// by BEL or ST (ESC \).
func oscEnd(buf []byte) (int, bool) {
	for i := 2; i < len(buf); i++ {
		if buf[i] == 0x07 {
			return i + 1, true
		}
		if buf[i] == 0x1b && i+1 < len(buf) && buf[i+1] == '\\' {
			return i + 2, true
		}
	}
	return 0, false
}

// isParameterizedCSIQuery matches DECRQM — `\x1b[?<digits>$p`. Deliberately
// narrow: DECSET/DECRST (`\x1b[?1006h`) share the `\x1b[?` prefix and must
// survive, and DECRPM *replies* end `$y`, not `$p`.
func isParameterizedCSIQuery(seq []byte) bool {
	if !bytes.HasPrefix(seq, []byte("\x1b[?")) || !bytes.HasSuffix(seq, []byte("$p")) {
		return false
	}
	digits := seq[3 : len(seq)-2]
	if len(digits) == 0 {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isParameterizedOSCQuery matches OSC 4 colour queries — `\x1b]4;<digits>;?`.
func isParameterizedOSCQuery(seq []byte) bool {
	if !bytes.HasPrefix(seq, []byte("\x1b]4;")) {
		return false
	}
	body := bytes.TrimRight(seq, "\x07")
	body = bytes.TrimSuffix(body, []byte("\x1b\\"))
	// len >= 6 is load-bearing, not defensive: the 4-byte prefix and the 2-byte
	// suffix can OVERLAP on a short body. `\x1b]4;?` satisfies both (index 3 is
	// both the prefix's ';' and the suffix's ';'), and the slice below would
	// then invert into a panic — which, on the tab-switch path, would take down
	// pair term and every shell in the pane.
	if len(body) < 6 || !bytes.HasSuffix(body, []byte(";?")) {
		return false
	}
	digits := body[4 : len(body)-2]
	if len(digits) == 0 {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
