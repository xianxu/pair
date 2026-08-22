package ptychild

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

// maxPending bounds the partial sequence held across reads.
//
// Holding a prefix is correct; holding an unbounded one is not. Child output is
// arbitrary bytes, so `\x1b[` followed by a megabyte of parameter bytes is a
// reachable input, and "wait for a final byte" would buffer all of it. Past this
// many bytes the run is not a real prefix any more and is consumed as text --
// the same prefix-vs-complete rule the rename decoder learned in #118.
const maxPending = 256

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
}

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
			// past the point where "prefix" is a plausible reading.
			if len(buf) > maxPending {
				// Consume the ESC and rescan; the run behind it is text.
				buf = buf[1:]
				continue
			}
			s.pending = append([]byte(nil), buf...)
			return
		}
		s.classify(buf[:size])
		buf = buf[size:]
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

func splitParams(p []byte) []string {
	return splitAny(string(p), ";:")
}

func splitAny(s, seps string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if bytes.IndexByte([]byte(seps), s[i]) >= 0 {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
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
