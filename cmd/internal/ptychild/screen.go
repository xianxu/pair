package ptychild

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
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
	sgrMouse  bool
	// mouseObserved records that this Screen has SEEN a mouse DECSET or DECRST
	// at all. Without it, `mouse == false` conflates "the child asked for no
	// tracking" with "this Screen has never been in a position to know" -- and a
	// reattach mints a fresh Screen for a still-running child that will not
	// re-emit its startup DECSET, so the second case is common rather than
	// theoretical (pair#196).
	mouseObserved bool

	// Latched edge events, cleared by their Take* reader. The console acts
	// once per event, not once per poll.
	rowDirty bool
	// cursorSaved is whether the CHILD is currently holding a cursor save it has
	// not restored. See classify, and HoldsCursorSave.
	cursorSaved bool
	bell        bool

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

	outputParts       []OutputPart
	notifyCandidate   []byte
	notifyStart       uint64
	streamEnd         uint64
	replaySafeEnd     uint64
	notifyPassthrough skipKind
	notifyPassESC     bool
}

type NotificationObservation struct {
	Message string
	Raw     []byte
	Start   uint64
	End     uint64
}

type OutputPart struct {
	Bytes        []byte
	Notification *NotificationObservation
}

func (s *Screen) TakeOutputParts() []OutputPart {
	parts := s.outputParts
	s.outputParts = nil
	return parts
}

func (s *Screen) ReplaySafeEnd() uint64 { return s.replaySafeEnd }
func (s *Screen) StreamEnd() uint64     { return s.streamEnd }

type skipKind uint8

const (
	skipNone skipKind = iota
	skipCSI           // ends at a final byte, 0x40-0x7e
	skipOSC           // ends at BEL or ST
)

// AltScreen reports whether the child is currently on the alternate screen.
func (s *Screen) AltScreen() bool { return s.altScreen }

// Mouse reports whether the child has asked for mouse TRACKING (1000/1002/1003).
// It deliberately excludes 1006, which is an encoding rather than a request for
// events -- see the DECSET switch for what collapsing them cost.
func (s *Screen) Mouse() bool { return s.mouse }

// MouseObserved reports whether this Screen has seen the child say anything
// about mouse mode. False means UNKNOWN, not "no": a supervisor that writes a
// terminal-global mode on the strength of Mouse() being false must check this
// first, or it will overwrite a mode it simply never witnessed.
func (s *Screen) MouseObserved() bool { return s.mouseObserved }

// SGRMouse reports whether the child asked for SGR-encoded coordinates (1006).
// A supervisor forwarding reports to this child must send the encoding the child
// asked for: a child holding 1000 without 1006 cannot parse an SGR report.
func (s *Screen) SGRMouse() bool { return s.sgrMouse }

// TakeRowDirty reports and clears whether the child did something that may have
// destroyed a reserved row: dropped the scrolling region (DECSTBM, RIS, an
// alt-screen transition) or ERASED the display.
//
// The erase half is the one that cost an operator smoke. DECSTBM restricts
// SCROLLING, not erasing -- so a full-screen child clearing the display on
// startup, which every one of them does, wipes the reserved row while the
// region is still perfectly intact. Naming this "region lost" was the mistake
// behind missing it: the console does not care WHY the row is gone, only that
// it is, and one signal for "the row may be gone" is the honest concept.
func (s *Screen) TakeRowDirty() bool {
	dirty := s.rowDirty
	s.rowDirty = false
	return dirty
}

// HoldsCursorSave reports whether the child is between its own DECSC and DECRC.
//
// A console with a reserved row must not write while this is true. The save slot
// is SHARED -- one per terminal -- so a paint that saves and restores inside the
// child's pair leaves the slot holding the CONSOLE's position, and the child's
// restore lands there instead of where it meant to go.
//
// It is a defer condition exactly like MidSequence, and pairs with the same
// owe-and-flush machinery: the debt is paid on the chunk that carries the
// child's DECRC.
//
// A child that saves and never restores holds this forever, and the row then
// goes STALE rather than wrong -- the correct direction to fail. RIS and the
// alt-screen transitions clear it, and a console taking over the screen
// wholesale resets the whole Screen anyway.
func (s *Screen) HoldsCursorSave() bool { return s.cursorSaved }

// SafeToPaint reports whether a console may write to the terminal right now.
//
// THE SHARED DOOR. Both reserved-row consumers must ask the same question, and
// an earlier version answered it only inside termcmd's private predicate -- so
// `pair term` was guarded and couch, running the identical primitive, was not.
// couch's child is a full-screen TUI that repaints continuously, which MASKS
// the damage rather than preventing it; two bugs latent in this primitive since
// #146 were invisible for exactly that reason, so masking has already proven
// not to be safety here.
//
// Two conditions, clearing on different bytes:
//
//   - MID-SEQUENCE: a write between two of the child's escape bytes lands
//     inside its sequence.
//   - THE CHILD HOLDS THE CURSOR SAVE: the slot is shared, one per terminal, so
//     a console save/restore inside the child's pair leaves the slot holding the
//     CONSOLE's position and the child's restore lands there.
//
// The cursor-save half applies only OUTSIDE the alt screen, and that is not a
// weakening -- it is what the hazard actually is.
//
// The danger is a child that saves, expects the slot intact, and restores
// SOON: a line-oriented child drawing a prompt, where the window is
// microseconds and a clobber lands the shell's cursor in the strip. A
// full-screen child in the alt screen is the opposite case. `?1049h` takes the
// slot for its ENTIRE lifetime -- minutes of nvim -- and gating on that would
// freeze the row for the whole session, trading a rare one-frame cursor glitch
// for a strip that is permanently stale, which is the common case and strictly
// worse. It is also the couch situation: a full-screen child repaints from its
// own model every frame and repositions absolutely, so a disturbed cursor does
// not survive to be seen.
//
// The residual is bounded and known: at `?1049l` the terminal restores what was
// saved, so a clobbered slot puts the shell's cursor one prompt out of place
// after the child exits. The operator ran exactly this configuration through
// #199 M3's smoke test with nvim and reported no such symptom.
func (s *Screen) SafeToPaint() bool {
	return !s.MidSequence() && !(s.cursorSaved && !s.altScreen)
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
//
// It is NOT the "is it safe to interleave" question: it reads 0 while an
// over-long sequence is being skipped, because those bytes are consumed rather
// than held. Use MidSequence for that.
func (s *Screen) Pending() int { return len(s.pending) }

// MidSequence reports whether the stream fed so far ends INSIDE an escape
// sequence -- either holding a partial, or consuming an over-long one.
//
// This is what a caller needs before writing its own bytes into the same
// stream: anything injected here lands in the middle of a sequence and
// corrupts it. Pending() alone answers the wrong question, which is how the
// first version of couch's fix shipped a hole (M2 BR-21).
func (s *Screen) MidSequence() bool {
	return len(s.pending) > 0 || s.skipping != skipNone
}

// Feed consumes a chunk of the child's output.
func (s *Screen) Feed(p []byte) {
	if len(p) == 0 {
		return
	}
	s.observeNotifications(p)
	s.feedFraming(p)
}

// FeedFraming consumes a chunk only for terminal state and sequence-boundary
// tracking. Unlike Feed, it does not retain output for notification observers.
// Use it when the caller already owns the bytes and only needs MidSequence and
// the other screen-state answers.
func (s *Screen) FeedFraming(p []byte) {
	if len(p) == 0 {
		return
	}
	s.feedFraming(p)
}

func (s *Screen) feedFraming(p []byte) {
	buf := p
	if len(s.pending) > 0 {
		buf = append(s.pending, p...)
		s.pending = nil
	}

	for len(buf) > 0 {
		if s.skipping != skipNone {
			n, done := s.skipTerminator(buf)
			if !done {
				// HOLD the unconsumed remainder. The OSC scan stops before a
				// trailing ESC so a two-byte ST is not split in half -- but the
				// first cut of this dropped those bytes instead of keeping
				// them, so a chunk boundary falling inside an ST swallowed the
				// next real bell. Measured at 1 of 70,550 cut positions, which
				// is exactly the kind of residual a fuzzer finds and a reader
				// does not.
				s.pending = append([]byte(nil), buf[n:]...)
				return
			}
			buf = buf[n:]
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
				if len(buf) > 1 {
					switch buf[1] {
					case ']', 'P', '_', '^', 'X':
						s.skipping = skipOSC // string-terminated
					}
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

func (s *Screen) observeNotifications(p []byte) {
	for _, b := range p {
		pos := s.streamEnd
		s.streamEnd++
		if s.notifyPassthrough != skipNone {
			s.appendOutputByte(b)
			s.replaySafeEnd = s.streamEnd
			done := s.notifyPassthrough == skipCSI && b >= 0x40 && b <= 0x7e
			if s.notifyPassthrough == skipOSC && (b == 0x07 || s.notifyPassESC && b == '\\') {
				done = true
			}
			if done {
				s.notifyPassthrough = skipNone
			}
			s.notifyPassESC = b == 0x1b
			continue
		}
		if len(s.notifyCandidate) == 0 {
			if b == 0x1b {
				s.notifyStart = pos
				s.notifyCandidate = append(s.notifyCandidate, b)
				continue
			}
			s.appendOutputByte(b)
			s.replaySafeEnd = s.streamEnd
			continue
		}

		s.notifyCandidate = append(s.notifyCandidate, b)
		if len(s.notifyCandidate) <= len(notifyosc.Prefix) {
			if !bytes.Equal(s.notifyCandidate, []byte(notifyosc.Prefix[:len(s.notifyCandidate)])) {
				candidate := append([]byte(nil), s.notifyCandidate...)
				s.flushNotifyCandidate()
				s.beginNotifyPassthrough(candidate)
			}
			continue
		}

		if b == 0x07 || len(s.notifyCandidate) >= 2 && s.notifyCandidate[len(s.notifyCandidate)-2] == 0x1b && b == '\\' {
			raw := append([]byte(nil), s.notifyCandidate...)
			if notification, ok := notifyosc.DecodeOSC(raw); ok {
				obs := &NotificationObservation{Message: notification.Message, Raw: raw, Start: s.notifyStart, End: s.streamEnd}
				s.outputParts = append(s.outputParts, OutputPart{Notification: obs})
				s.notifyCandidate = nil
				s.replaySafeEnd = s.streamEnd
				continue
			}
			s.flushNotifyCandidate()
			continue
		}
		if len(s.notifyCandidate) >= 2 && s.notifyCandidate[len(s.notifyCandidate)-2] == 0x1b {
			s.flushNotifyCandidate()
			s.notifyPassthrough = skipOSC
			s.notifyPassESC = b == 0x1b
			continue
		}
		messageBytes := len(s.notifyCandidate) - len(notifyosc.Prefix)
		if b == 0x1b {
			messageBytes-- // possible first byte of ST
		}
		if messageBytes > notifyosc.MaxMessageBytes {
			s.flushNotifyCandidate()
			s.notifyPassthrough = skipOSC
			s.notifyPassESC = b == 0x1b
		}
	}
}

func (s *Screen) beginNotifyPassthrough(candidate []byte) {
	if _, ok := frame(candidate); ok || len(candidate) < 2 {
		return
	}
	switch candidate[1] {
	case '[', 'O':
		s.notifyPassthrough = skipCSI
	case ']', 'P', '_', '^', 'X':
		s.notifyPassthrough = skipOSC
		s.notifyPassESC = candidate[len(candidate)-1] == 0x1b
	}
}

func (s *Screen) flushNotifyCandidate() {
	if len(s.notifyCandidate) == 0 {
		return
	}
	s.appendOutputBytes(s.notifyCandidate)
	s.notifyCandidate = nil
	s.replaySafeEnd = s.streamEnd
}

func (s *Screen) appendOutputByte(b byte) { s.appendOutputBytes([]byte{b}) }

func (s *Screen) appendOutputBytes(p []byte) {
	if len(p) == 0 {
		return
	}
	if n := len(s.outputParts); n > 0 && s.outputParts[n-1].Notification == nil {
		s.outputParts[n-1].Bytes = append(s.outputParts[n-1].Bytes, p...)
		return
	}
	s.outputParts = append(s.outputParts, OutputPart{Bytes: append([]byte(nil), p...)})
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
	// RIS resets everything the console set, margins included -- including any
	// cursor save the child was holding.
	if seq[1] == 'c' {
		s.rowDirty = true
		s.cursorSaved = false
		return
	}
	// DECSC / DECRC. Tracked because the cursor save slot is SHARED: a console
	// that paints between the child's ESC 7 and its ESC 8 clobbers what the
	// child saved, and the child's restore then recovers the CONSOLE's position.
	// Measured 2026-09-08 in pair#199: zsh draws its right-hand prompt with
	// terminfo sc/rc (which are these), so the operator's cursor ended up in the
	// tab strip and the right-prompt was drawn on the strip's row.
	//
	// One bit, not a stack: terminals keep ONE slot, so a second DECSC simply
	// overwrites the first and a DECRC clears the debt either way.
	if seq[1] == '7' {
		s.cursorSaved = true
		return
	}
	if seq[1] == '8' {
		s.cursorSaved = false
		return
	}
	if seq[1] != '[' || len(seq) < 3 {
		return
	}
	final := seq[len(seq)-1]
	params := seq[2 : len(seq)-1]

	// SCOSC / SCORC -- the CSI spelling of the same save slot, `CSI s` and
	// `CSI u` with NO parameters.
	//
	// Tracked alongside DECSC because this repo's own probe
	// (probes/cursorsaveslots) established that zellij does not give them a
	// SECOND slot: `CSI u` failed to restore where it was told while `ESC 8`
	// succeeded under the identical harness. So a child using the CSI form is
	// holding the SAME slot our paint would clobber, and a gate that watched
	// only `ESC 7` would let the collision straight back in for that child.
	//
	// Parameter-free is load-bearing: `CSI <n> s` is DECSLRM (set left/right
	// margins), a different operation entirely, and treating it as a save would
	// defer paints forever against a child that never restores.
	if len(params) == 0 {
		switch final {
		case 's':
			s.cursorSaved = true
			return
		case 'u':
			s.cursorSaved = false
			return
		}
	}

	if len(params) > 0 && params[0] == '?' {
		if final != 'h' && final != 'l' {
			return
		}
		on := final == 'h'
		for _, mode := range splitParams(params[1:]) {
			switch mode {
			// THE SAVE-SLOT ENUMERATION, one arm per spelling, because a
			// missing arm is a silent hole in the paint gate rather than a
			// visible failure. `?1049h` is DEFINED as DECSC-then-switch: it
			// SAVES the cursor, and an earlier version cleared the flag here --
			// exactly backwards for how nearly every full-screen child takes
			// the slot.
			//
			//   ESC 7 / ESC 8          DECSC / DECRC          take / release
			//   CSI s / CSI u          SCOSC / SCORC          take / release
			//   CSI ?1048h / ?1048l    save / restore cursor  take / release
			//   CSI ?1049h / ?1049l    ?1048 + ?1047          take / release
			//   CSI ?1047h / ?1047l    alt screen ONLY        no effect
			//   CSI ?47h   / ?47l      alt screen ONLY        no effect
			//   ESC c                  RIS                    release
			case "1049":
				s.altScreen = on
				// 1049 = 1048 + 1047: it saves on entry and restores on exit.
				s.cursorSaved = on
				s.rowDirty = true
			case "1047", "47":
				// Screen switch WITHOUT the save half. The slot is untouched,
				// so the flag must not move in either direction -- clearing it
				// here would hand a console permission to paint inside a save
				// the child still holds.
				s.altScreen = on
				s.rowDirty = true
			case "1048":
				// The save half on its own.
				s.cursorSaved = on
			case "1000", "1002", "1003":
				s.mouse = on
				s.mouseObserved = true
				// A mouse-mode change is an EVENT, not just a fact to read
				// later. Mouse reporting is terminal-global, so a supervisor
				// deciding whether to hold its own mode has to re-evaluate when
				// this changes -- and without a latch nothing tells it: a bare
				// `?1000l` left couch's clicks off until some unrelated paint
				// happened to run (pair#172 I1).
				s.rowDirty = true
			case "1006":
				s.sgrMouse = on
				s.mouseObserved = true
				s.rowDirty = true
			}
		}
		return
	}
	switch final {
	case 'r':
		// DECSTBM: `\x1b[r` or `\x1b[<top>;<bottom>r`, no private introducer.
		s.rowDirty = true
	case 'J':
		// Treat every ED form conservatively as possible reserved-row damage.
		// Some forms erase only part of the display, but repainting one status
		// row is cheaper and safer than duplicating cursor-aware ED semantics.
		s.rowDirty = true
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
	case ']', 'P', '_', '^', 'X':
		// The STRING-terminated classes: OSC, DCS, APC, PM, SOS. All end at ST
		// (or BEL, which xterm accepts), so one scan serves them.
		//
		// Only ']' was covered at first, so a DCS/APC/PM/SOS payload fell
		// through to the two-byte case below and was scanned as plain text --
		// `\x1bP+q616263\x07\x1b\\` rang a false bell, and a tmux passthrough
		// `\x1bPtmux;\x1b[?1049h\x1b\\` set alt-screen from INSIDE a sequence.
		// Reachability is low today (kitty-graphics APC and XTGETTCAP DCS carry
		// base64/hex), but TakeBell's doc and atlas/architecture.md both state
		// "outside a sequence" as a property, and an invariant has to hold over
		// every class it is claimed over.
		return ansi.OSCEnd(buf, ansi.Lenient)
	default:
		// A two-byte escape (ESC c, ESC M, a charset designation). Complete by
		// construction, so it is consumed rather than held.
		return 2, true
	}
}
