package ptychild

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

// Terminal capability queries, and why they must not be replayed (#127).
//
// A repaint replays a child by writing its stored raw output back to the real
// terminal. That output still contains whatever CAPABILITY QUERIES the app in
// that tab emitted at startup — DA1, DECRQM, the Kitty keyboard flags probe.
// Replaying them re-ASKS the host terminal, which answers on our stdin; the
// pump then hands that answer to the currently active child, which tries
// to run it as a command. Observed as a shell line reading
// `execute: …\x1b[?62;4;52c\x1b[?2026;2$y…`.
//
// The fix is to strip queries out of the REPLAY only. The live path is
// untouched (a live chunk goes to the terminal unmodified), so an
// app's first, real query still reaches the terminal and still gets its answer —
// capability negotiation is not disturbed. We deliberately do NOT filter replies
// on the input path: a reply arriving while its own tab is active is solicited
// and correct, and dropping it would silently break that negotiation.
//
// Moved out of termcmd for #146: `couch`'s repaint-on-attach is the same
// operation `redrawTab` performs, so this deny-list is ONE policy with two
// sites rather than two policies. Contrast wrapcmd's table, which is opposed to
// this one (it strips `\x1b[>7u`; here `\x1b[>1u` must survive) and correctly
// stays where it is.
//
// This table is a best-effort DENY-LIST of what nvim / zsh / fzf actually emit.
// It does not need to be exhaustive: a query we miss simply degrades to the old
// behavior. There is no live conformance check behind it.
//
// Note replies DO reach this buffer — the pump writes one into the child's PTY,
// the shell's line discipline echoes it, and the pump reads it back. So the
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

// StripQueries removes capability queries from a child's stored output so a
// repaint cannot re-issue them. Pure: no IO, no state.
//
// An unterminated escape at end-of-buffer is emitted VERBATIM. The buffer can
// legitimately begin or end mid-sequence — `Ring` keeps only the last `DefaultRingBytes`, which bisects whatever spans that boundary — and a "no final byte
// found, drop the rest" rule would silently swallow the tail of the replay, i.e.
// the visible screen.
func StripQueries(buf []byte) []byte {
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
		size, isQuery, ok := sequenceAt(buf[i:])
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

// sequenceAt frames the escape sequence at the start of buf and reports whether
// it is a capability query. ok is false when the sequence is not terminated
// within buf (a truncated tail), which the caller emits verbatim.
//
// Framing itself is `frame` (screen.go) -- one site per package decides where a
// sequence ends. This function is only the query POLICY over that framing.
func sequenceAt(buf []byte) (size int, isQuery bool, ok bool) {
	for _, lit := range terminalQueryLiterals {
		if bytes.HasPrefix(buf, lit) {
			// OSC literals still need their terminator consumed.
			if lit[1] == ']' {
				if end, found := ansi.OSCEnd(buf, ansi.Lenient); found {
					return end, true, true
				}
				return 0, false, false
			}
			return len(lit), true, true
		}
	}
	size, ok = frame(buf)
	if !ok {
		return 0, false, false
	}
	switch buf[1] {
	case '[':
		return size, isParameterizedCSIQuery(buf[:size]), true
	case ']':
		return size, isParameterizedOSCQuery(buf[:size]), true
	default:
		return size, false, true
	}
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
