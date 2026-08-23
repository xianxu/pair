package ptychild

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

// maxPending bounds the partial sequence held across reads.
//
// Holding a prefix is correct; holding an unbounded one is not. Child output is
// arbitrary bytes, so `\x1b[` followed by a megabyte of parameter bytes is a
// reachable input, and "wait for a final byte" would buffer all of it.
//
// 64 KiB, not something tight. The first version used 256 and was WRONG in an
// everyday case: an OSC 52 clipboard write is kilobytes and always crosses a
// 4096-byte pty read boundary, so it blew the bound, got abandoned mid-sequence,
// and its terminating BEL was then counted by the plain-run scan -- a false
// "the agent wants you" on every clipboard copy. A real prefix has to be able to
// be as long as the protocol allows; the bound is a memory guard, not a
// plausibility judgement.
const maxPending = 64 * 1024

// Screen is what a child's own output says about the screen it thinks it is
// drawing on. One scanner, four answers.
//
// It absorbs termcmd's updateMouseMode, which scanned each read independently
// and therefore could not see a sequence split across two pty reads. Framing is
// delegated to cmd/internal/ansi -- there is exactly one place in this package
// that decides where a sequence ends (see frame), because two sites making that
// decision independently is how #127's dead keyboard happened.
//
// Not safe for concurrent use: it is fed only by its child's read pump.
type Screen struct {
	pending []byte

	altScreen bool
	mouse     bool

	// Latched edge events, cleared by their Take* reader. The console acts
	// once per event, not once per poll.
	regionLost bool
	bell       bool

	// skipping says we are inside a sequence too long to buffer, and what
	// terminator ends it. Bytes are consumed and discarded until then.
	//
	// This is the third shape of the BR-1 fix, and the first two are why it is
	// worth spelling out. Raising maxPending was an instance fix. Discarding to
	// the next ESC restored invariance for UNTERMINATED runs but broke it for
	// terminated ones: a 70 KiB OSC fed whole frames fine (its terminator is in
	// the buffer) while the same bytes fed in 4096-byte chunks blew the bound,
	// got abandoned, and DROPPED a real BEL that followed. The bound was the
	// asymmetry.
	//
	// So the rule is not "give up", it is "stop BUFFERING, keep FRAMING":
	// memory stays O(1) while the sequence is still consumed to its real
	// terminator. Whole and split then agree at every length -- neither counts
	// the sequence's own terminator, both count a bell after it.
	skipping skipKind
}

type skipKind uint8

const (
	skipNone skipKind = iota
	skipCSI           // ends at a final byte, 0x40-0x7e
	skipOSC           // ends at BEL or ST
)

// AltScreen reports whether the child is currently on the alternate screen.
func (s *Screen) AltScreen() bool { return s.altScreen }

// Mouse reports whether the child has asked for mouse reporting.
func (s *Screen) Mouse() bool { return s.mouse }

// TakeRegionLost reports and clears whether the child did something that can
// drop the host's scrolling region -- DECSTBM, a full reset, or an alt-screen
// transition. The console re-asserts its reserved row on each.
func (s *Screen) TakeRegionLost() bool {
	lost := s.regionLost
	s.regionLost = false
	return lost
}

// TakeBell reports and clears whether the child rang the terminal bell. This is
// the one "the agent wants you" signal available before #147's transport, so a
// false positive matters: every title change ends in BEL, which is why BEL is
// only counted outside a sequence.
func (s *Screen) TakeBell() bool {
	bell := s.bell
	s.bell = false
	return bell
}

// Pending reports how many bytes are held waiting for a terminator. Exported
// for the tests that pin the bound -- an unbounded scanner is invisible from
// the state accessors alone.
func (s *Screen) Pending() int { return len(s.pending) }

// Feed consumes a chunk of the child's output.
func (s *Screen) Feed(p []byte) {
	if len(p) == 0 {
		return
	}
	buf := p
	if len(s.pending) > 0 {
		buf = append(s.pending, p...)
		s.pending = nil
	}

	for len(buf) > 0 {
		if s.skipping != skipNone {
			n, done := s.skipTerminator(buf)
			buf = buf[n:]
			if !done {
				return // whole chunk was sequence body; nothing derivable
			}
			s.skipping = skipNone
			continue
		}
		if buf[0] != 0x1b {
			// Bulk-scan the plain run. BEL only counts here: inside an OSC it
			// is the terminator, not a bell.
			next := bytes.IndexByte(buf, 0x1b)
			plain := buf
			if next >= 0 {
				plain = buf[:next]
			}
			if bytes.IndexByte(plain, 0x07) >= 0 {
				s.bell = true
			}
			if next < 0 {
				return
			}
			buf = buf[next:]
			continue
		}

		size, ok := frame(buf)
		if !ok {
			// A real prefix -- hold it for the next read, unless it has grown
			// past the memory guard.
			if len(buf) > maxPending {
				// Too long to hold -- stop buffering, keep framing. The state
				// this sequence would have set is lost (it is a control we
				// could not read), but the STREAM stays in sync, so what
				// follows is still interpreted correctly. Ring still holds the
				// raw bytes for the repaint.
				s.skipping = skipCSI
				if len(buf) > 1 && buf[1] == ']' {
					s.skipping = skipOSC
				}
				buf = buf[2:]
				continue
			}
			s.pending = append([]byte(nil), buf...)
			return
		}
		s.classify(buf[:size])
		buf = buf[size:]
	}
}

// skipTerminator consumes buf while inside an over-long sequence, returning how
// many bytes it took and whether the terminator was found.
//
// It uses the SAME terminator predicates as the framing in ansi -- a second
// opinion about where a sequence ends is the bug this repo has already paid for
// twice (#127's dead keyboard, and the paired-terminator lesson).
func (s *Screen) skipTerminator(buf []byte) (n int, done bool) {
	switch s.skipping {
	case skipOSC:
		for i := 0; i < len(buf); i++ {
			if buf[i] == 0x07 {
				return i + 1, true
			}
			if buf[i] == 0x1b {
				if i+1 < len(buf) {
					if buf[i+1] == '\\' {
						return i + 2, true
					}
					continue
				}
				// ESC at the boundary: hold it so ST is not split in half.
				return i, false
			}
		}
		return len(buf), false
	default: // skipCSI
		for i := 0; i < len(buf); i++ {
			if ansi.IsFinalByte(buf[i]) {
				return i + 1, true
			}
		}
		return len(buf), false
	}
}

// classify applies the event rules to one complete sequence.
//
// Deliberately a best-effort table, like the replay deny-list: a control we do
// not recognise degrades to "no event", which costs a repaint at worst. The
// rules that must not fire wrongly are the negatives -- an 'r' final behind the
// private introducer is DECRSTR, not a margin change.
func (s *Screen) classify(seq []byte) {
	if len(seq) < 2 {
		return
	}
	// RIS resets everything the console set, margins included.
	if seq[1] == 'c' {
		s.regionLost = true
		return
	}
	if seq[1] != '[' || len(seq) < 3 {
		return
	}
	final := seq[len(seq)-1]
	params := seq[2 : len(seq)-1]

	if len(params) > 0 && params[0] == '?' {
		if final != 'h' && final != 'l' {
			return
		}
		on := final == 'h'
		for _, mode := range splitParams(params[1:]) {
			switch mode {
			case "1049", "1047", "47":
				s.altScreen = on
				// An alt-screen transition is exactly when a child redraws
				// from scratch and the region can go with it.
				s.regionLost = true
			case "1000", "1002", "1003", "1006":
				s.mouse = on
			}
		}
		return
	}
	// DECSTBM: `\x1b[r` or `\x1b[<top>;<bottom>r`, no private introducer.
	if final == 'r' {
		s.regionLost = true
	}
}

// splitParams splits a CSI parameter run on both separators the protocol allows.
func splitParams(p []byte) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(p); i++ {
		if p[i] == ';' || p[i] == ':' {
			out = append(out, string(p[start:i]))
			start = i + 1
		}
	}
	return append(out, string(p[start:]))
}

// frame returns the length of the escape sequence at buf[0] and whether it is
// terminated within buf. It is the ONLY place in this package that decides
// where a sequence ends; both the replay strip and the screen scanner go
// through it.
func frame(buf []byte) (int, bool) {
	if len(buf) < 2 {
		return 0, false
	}
	switch buf[1] {
	case '[':
		end := ansi.TerminatorScan(buf)
		if end < 0 {
			return 0, false
		}
		return end, true
	case ']':
		return ansi.OSCEnd(buf, ansi.Lenient)
	default:
		// A two-byte escape (ESC c, ESC M, a charset designation). Complete by
		// construction, so it is consumed rather than held.
		return 2, true
	}
}
