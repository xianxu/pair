// Package ansi owns Pair's byte-level knowledge of terminal escape sequences:
// where one starts, where it ends, and whether it is complete (#128).
//
// It replaced two independent framings in one binary — wrapcmd's `otherEscRe`
// regex and termcmd's csiEnd/oscEnd scanners. What is shared here is the
// STRUCTURE of a sequence. What deliberately stays with each caller is POLICY:
// wrapcmd strips `\x1b[>7u` so codex stops pushing Kitty flags, while termcmd
// requires `\x1b[>1u` to survive a replay. Those tables are opposed on purpose;
// merging them would be the bug, not the cleanup.
//
// The exports are shaped by what the callers actually need, which is not the same
// entry point:
//
//   - Frame — the sequence at buf[0], across all four classes wrapcmd recognises.
//   - TerminatorScan — introducer-INDEPENDENT scan to a final byte. termcmd's
//     csiEnd is this, and rename_input.go routes SS3 (`\x1bO…`) through it, so a
//     dispatch on buf[1] would frame `\x1bOX` as a two-byte escape and leak the X
//     into a tab name.
//   - OSCEnd — the one place with a genuine strictness split between callers.
package ansi

// Status distinguishes the three outcomes callers branch on. They are NOT
// interchangeable: collapsing them into a single int is what made an earlier
// design return 0 into `input = input[size:]`, a zero-advance infinite loop.
type Status int

const (
	// None: buf does not start an escape sequence this package frames.
	None Status = iota
	// Incomplete: buf starts one, but it is truncated. The caller must carry the
	// tail rather than consume a partial sequence — stripping it would corrupt
	// every chunk boundary in a stream.
	Incomplete
	// Complete: fully framed; the returned size is its length.
	Complete
)

// Mode selects how tolerant OSC framing is of a bare ESC in the payload.
type Mode int

const (
	// Strict stops at a bare ESC, matching the regex wrapcmd used
	// (`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`).
	Strict Mode = iota
	// Lenient scans past a bare ESC looking for BEL or ST, matching termcmd's
	// oscEnd. Kept because termcmd's decoder consumes what this returns, and
	// tightening it would change how malformed input is eaten mid-rename.
	Lenient
)

// IsFinalByte reports whether c terminates a parameterised escape sequence.
func IsFinalByte(c byte) bool { return c >= 0x40 && c <= 0x7e }

func isParamByte(c byte) bool        { return c >= 0x30 && c <= 0x3f }
func isIntermediateByte(c byte) bool { return c >= 0x20 && c <= 0x2f }

// TerminatorScan returns the length of the parameterised sequence at buf[0] by
// scanning from index 2 for the first final byte, or -1 if there is none in buf.
//
// Introducer-independent BY DESIGN: it does not look at buf[1], so the same scan
// serves CSI (`\x1b[`) and SS3 (`\x1bO`). termcmd.malformedEscapeSize depends on
// that — it routes both through here, and its result feeds `input = input[size:]`.
//
// PRECONDITION: buf must start an escape sequence (buf[0] == 0x1b). It does not
// check, matching the csiEnd it replaced — TerminatorScan([]byte("abc")) returns 3,
// because 'c' is a final byte. Callers that have not already established the ESC
// should use Frame instead.
func TerminatorScan(buf []byte) int {
	for i := 2; i < len(buf); i++ {
		if IsFinalByte(buf[i]) {
			return i + 1
		}
	}
	return -1
}

// OSCEnd returns the length of the OSC sequence at buf[0], terminated by BEL or
// ST (ESC \). See Mode for the bare-ESC difference between callers.
func OSCEnd(buf []byte, mode Mode) (int, bool) {
	for i := 2; i < len(buf); i++ {
		switch {
		case buf[i] == 0x07:
			return i + 1, true
		case buf[i] == 0x1b:
			if i+1 < len(buf) && buf[i+1] == '\\' {
				return i + 2, true
			}
			if mode == Strict {
				// The regex's [^\x07\x1b]* class cannot cross a bare ESC.
				return 0, false
			}
		}
	}
	return 0, false
}

// Frame returns the length and status of the escape sequence at buf[0].
//
// The alternatives are tried in the SAME ORDER as the regex this replaced, and
// the order is load-bearing: `]` (0x5D) falls inside the two-byte class
// [0x5C-0x5F], so an unterminated OSC like "\x1b]0;title" is not "incomplete" —
// the regex matches `\x1b]` as a two-byte escape and leaves "0;title" as text.
// Reproducing that exactly is why the differential fuzzers exist.
func Frame(buf []byte) (int, Status) {
	if len(buf) == 0 || buf[0] != 0x1b {
		return 0, None
	}
	if len(buf) == 1 {
		return 0, Incomplete
	}

	switch {
	case buf[1] == '[':
		// No two-byte fallback for '[' (0x5B sits outside [0x40-0x5A] and
		// [0x5C-0x5F]), so frameCSI's verdict stands — including its distinction
		// between "malformed" (None) and "truncated" (Incomplete). Falling through
		// here would report a malformed CSI as merely incomplete and pin it in a
		// caller's pending buffer forever.
		return frameCSI(buf)
	case buf[1] == ']':
		if n, ok := OSCEnd(buf, Strict); ok {
			return n, Complete
		}
		// Falls through to the two-byte class below, exactly as the regex does.
	case buf[1] == '(' || buf[1] == ')' || buf[1] == '*' || buf[1] == '+':
		if len(buf) < 3 {
			return 0, Incomplete
		}
		if IsFinalByte(buf[2]) {
			return 3, Complete
		}
		return 0, None
	}

	// Two-byte escapes: ESC followed by 0x40-0x5A or 0x5C-0x5F.
	if (buf[1] >= 0x40 && buf[1] <= 0x5a) || (buf[1] >= 0x5c && buf[1] <= 0x5f) {
		return 2, Complete
	}
	return 0, None
}

// frameCSI applies the strict CSI grammar: params, then intermediates, then a
// final byte. Anything out of range aborts, which is what keeps wrapcmd from
// over-stripping — the asymmetric danger, since an over-strip silently removes
// mouse mode, Kitty encoding or the cursor shape.
func frameCSI(buf []byte) (int, Status) {
	i := 2
	for i < len(buf) && isParamByte(buf[i]) {
		i++
	}
	for i < len(buf) && isIntermediateByte(buf[i]) {
		i++
	}
	if i >= len(buf) {
		return 0, Incomplete
	}
	if IsFinalByte(buf[i]) {
		return i + 1, Complete
	}
	return 0, None
}

// SequenceLen is Frame reduced to the answer wrapcmd's call sites want: the
// length of a COMPLETE sequence at buf[0], else 0.
//
// Incomplete returning 0 is load-bearing for the wrapper's tail carry
// (wrapcmd's p.stdoutPending): a scanner that consumed a partial sequence would
// corrupt every chunk edge.
func SequenceLen(buf []byte) int {
	if n, st := Frame(buf); st == Complete {
		return n
	}
	return 0
}

// Strip removes every complete escape sequence from buf, preserving an incomplete
// trailing one for the caller to carry.
//
// CONTRACT: the result NEVER aliases buf, even when nothing is stripped.
//
// Not a style preference — an "obvious" fast path returning buf on ESC-free input is
// a silent corruption bug. Both wrapcmd callers pipe the result straight into
// bytesReplaceAll, which compacts IN PLACE (`out := b[:0:len(b)]`). With an aliased
// return that writes through to the caller's own buffer: at wrap.go:1152 it rewrites
// p.captureBuffer's backing array while leaving its length unchanged, so the tail
// keeps stale duplicated bytes ("hello\r\nworld" becomes "hello\nworldd") and the
// corrupted capture is what nvim reads for Alt+i image paste — and it does so OUTSIDE
// p.captureMu, racing the SIGUSR1 handler's appends. PTY output is \r\n-terminated
// under ONLCR, so "plain chunk with \r and no ESC" is the common case.
//
// The regex this replaced allocated unconditionally (Go's replaceAll appends into a
// nil buf even on the no-match path), which is why the invariant held before.
func Strip(buf []byte) []byte {
	out := make([]byte, 0, len(buf))
	for i := 0; i < len(buf); {
		if n := SequenceLen(buf[i:]); n > 0 {
			i += n
			continue
		}
		out = append(out, buf[i])
		i++
	}
	return out
}
