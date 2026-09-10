package ptychild

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Size is a terminal's dimensions. It exists so callers do not have to import
// creack/pty just to say how big a child should be -- the console reserves a
// row by subtracting from Rows, and that arithmetic should not require knowing
// what a Winsize is.
type Size struct {
	Rows, Cols uint16
}

type OutputBatch struct {
	Raw           []byte
	Parts         []OutputPart
	Bell          bool
	RowDirty      bool
	RingEnd       uint64
	ReplaySafeEnd uint64
}

// Options configures a child. Everything here is what the CALLER knows; nothing
// about switching policy belongs in it.
type Options struct {
	Dir        string
	Argv       []string
	Env        []string
	Size       Size
	ExtraFiles []*os.File

	// RingBytes is how much output to keep for a repaint. Zero means
	// DefaultRingBytes.
	RingBytes int

	// Sink receives every chunk the child writes, in order, from the pump
	// goroutine. The caller decides whether it reaches a screen -- that is
	// switching policy, and it stays with the caller.
	//
	// The ring and the screen are updated BEFORE Sink runs, so a caller that
	// switches away inside Sink still repaints a current screen.
	Sink func(OutputBatch)
}

// Child is one process on a pty, with the window of output needed to repaint it
// and the state its output implies.
type Child struct {
	cmd  *exec.Cmd
	ptmx *os.File
	sink func(OutputBatch)

	mu     sync.Mutex
	ring   *Ring
	screen *Screen
	// notificationSpans are absolute positions in the raw child stream. They
	// stay alongside the retained ring window so replay can remove a canonical
	// envelope even when the ring begins in its middle.
	notificationSpans []notificationSpan

	// done closes once the child has been reaped; code is written before the
	// close, so reading it after <-done needs no further synchronisation.
	// Same shape as couchcore's execHandle, and for the same reason: `kill -0`
	// succeeds for a zombie, so liveness must not be a syscall.
	done chan struct{}
	code int

	closeOnce sync.Once

	// geom guards the child's DIMENSIONS, and it is the whole of #209 C2's
	// answer: every size change takes it, and a repaint nudge holds it for its
	// entire shrink-settle-restore. Separate from mu because mu is held while
	// scanning output and a nudge holds this one for the 20 ms settle; sharing
	// them would stall the read pump on every switch.
	geom sync.Mutex
	// size is the last size successfully written to the pty. The nudge's
	// restore leg reads THIS rather than a value a caller passed in, so no
	// caller can hand it a size that a concurrent resize has already
	// superseded.
	size Size
	// nudging says a repaint request is already in flight, so a second is
	// dropped rather than queued behind the first one's settle.
	nudging bool
	// geomGen counts SIZE CHANGES THE CALLER ASKED FOR, so a nudge can tell
	// whether the world moved under it while it was settling (#209 I-1). The
	// nudge's own shrink does not bump it: the shrink is the nudge's business,
	// and counting it would make every nudge supersede itself.
	geomGen uint64

	// fake is non-nil only for NewFakeChild. Every method that would touch a
	// pty branches on it, so one type serves both paths and a test cannot be
	// exercising a different shape from production.
	fake *fakeState
}

// Start launches argv on a fresh pty sized to opts.Size.
func Start(opts Options) (*Child, error) {
	if len(opts.Argv) == 0 {
		return nil, fmt.Errorf("ptychild: empty argv")
	}
	capacity := opts.RingBytes
	if capacity <= 0 {
		capacity = DefaultRingBytes
	}

	cmd := exec.Command(opts.Argv[0], opts.Argv[1:]...)
	cmd.Dir = opts.Dir
	if opts.Env != nil {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	cmd.ExtraFiles = opts.ExtraFiles

	// Size at Start rather than start-then-resize: a child that draws its first
	// frame at 80x24 and reflows a moment later is a visible flash on every
	// spawn, and for a full-screen agent harness it is a whole redraw.
	ws := &pty.Winsize{Rows: opts.Size.Rows, Cols: opts.Size.Cols}
	ptmx, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		return nil, fmt.Errorf("ptychild: start %s: %w", opts.Argv[0], err)
	}

	c := &Child{
		cmd:    cmd,
		ptmx:   ptmx,
		sink:   opts.Sink,
		ring:   NewRing(capacity),
		screen: &Screen{},
		done:   make(chan struct{}),
		// The size it was STARTED at counts as a size change: a child that is
		// never resized still has geometry, and a nudge must be able to restore
		// it rather than decline for want of a remembered value.
		size: opts.Size,
	}
	go c.pump()
	return c, nil
}

// pump reads the child until the pty closes, then reaps it.
func (c *Child) pump() {
	defer close(c.done)
	buf := make([]byte, 4096)
	for {
		n, err := c.ptmx.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])

			c.mu.Lock()
			c.ring.Append(chunk)
			c.screen.Feed(chunk)
			batch := c.outputBatchLocked(chunk)
			sink := c.sink
			c.mu.Unlock()

			if sink != nil {
				sink(batch)
			}
		}
		if err != nil {
			break
		}
	}
	c.code = procutil.WaitCode(c.cmd)
}

func (c *Child) outputBatchLocked(raw []byte) OutputBatch {
	parts := c.screen.TakeOutputParts()
	for _, part := range parts {
		if part.Notification != nil {
			c.notificationSpans = append(c.notificationSpans, notificationSpan{
				start: part.Notification.Start,
				end:   part.Notification.End,
			})
		}
	}
	ringStart := c.screen.StreamEnd() - uint64(c.ring.Len())
	firstRetained := 0
	for firstRetained < len(c.notificationSpans) && c.notificationSpans[firstRetained].end <= ringStart {
		firstRetained++
	}
	if firstRetained > 0 {
		copy(c.notificationSpans, c.notificationSpans[firstRetained:])
		c.notificationSpans = c.notificationSpans[:len(c.notificationSpans)-firstRetained]
	}
	batch := OutputBatch{
		Raw:           raw,
		Parts:         parts,
		RingEnd:       c.screen.StreamEnd(),
		ReplaySafeEnd: c.screen.ReplaySafeEnd(),
	}
	if c.sink != nil {
		batch.Bell = c.screen.TakeBell()
		batch.RowDirty = c.screen.TakeRowDirty()
	}
	return batch
}

// Write sends bytes to the child's terminal.
func (c *Child) Write(p []byte) (int, error) {
	if c.fake != nil {
		if c.Done() {
			return 0, fmt.Errorf("ptychild: write to a child that has exited")
		}
		c.fake.mu.Lock()
		c.fake.writes = append(c.fake.writes, append([]byte(nil), p...))
		c.fake.mu.Unlock()
		return len(p), nil
	}
	return c.ptmx.Write(p)
}

// RequestRepaint asks the child to repaint from its OWN state — the only
// authority for a frame the bounded replay ring no longer holds (#209).
//
// SIGWINCH is the mechanism: zellij 0.44.3 has no repaint action (`clear`
// destroys buffers, `dump-screen` writes to a file), and cmd/probes/zellijrepaint
// confirms against the real binary that zellij re-renders its pane from its own
// buffer on this exact sequence. It is the same thing the operator's mouse
// click achieved, issued deliberately.
//
// IT TAKES NO SIZE, and that is the fix rather than an omission (#209 C2). The
// first version took the size to restore, which made a stale restore
// EXPRESSIBLE: whichever goroutine happened to call it sampled a size, and a
// host resize landing inside the settle window was then overwritten by the
// restore leg — the child left permanently mis-sized with no event to correct
// it. The first answer to that was a rule ("callers must stay on the goroutine
// that serializes their other resizes") written in this comment, and a comment
// is not a mechanism: couch broke it immediately, because `switchTo` is reached
// from the operationQueue goroutine as well as the Run loop, and that path is
// the operator's primary switch gesture.
//
// So the Child owns its geometry, and there is no other owner to disagree with.
// Every size change goes through `geom`, the nudge holds it for its whole
// shrink-settle-restore, and the size it restores is the one IT read under that
// lock. A resize arriving during a nudge waits and then applies — the child
// ends at the newest size either way, from any goroutine, with no ordering rule
// for a caller to remember.
//
// Asynchronous, because the settle is 20 ms and a caller is an event loop. The
// serialization above is what makes that safe; it was not safe when ordering
// depended on the caller's goroutine, which is why the first version blocked.
// A second request while one is in flight is DROPPED rather than queued: two
// nudges produce one repaint, and holding a key on tab-switch would otherwise
// spend the settle once per keystroke.
//
// And it RELEASES the lock while it settles, so an ordinary Resize never waits
// on it (#209 I-1). Holding it was the first version and it inverted the
// priority: a real resize is mandatory and a nudge is optional, but a SIGWINCH
// landing inside a switch blocked for a measured 20.8 ms — on termcmd's writer
// goroutine, which is the sole writer of the pane, so all output stalled with
// it. The generation counter is what keeps that safe: if a caller resized while
// this nudge slept, the world has moved and the restore leg SKIPS rather than
// writing back a size nobody asked for. Optional work must never gate the
// mandatory kind, and must never win a race against it.
//
// Rows, not columns: a column change reflows wrapped lines, which is an edit
// rather than a repaint.
//
// Fire and forget. The replay has already painted, so a nudge that fails
// degrades to the old behaviour rather than to a blank screen — a switch must
// never abort because a child would not resize.
//
// One home, not one per consumer (#209 BR-5).
func (c *Child) RequestRepaint() {
	if c == nil {
		return
	}
	c.geom.Lock()
	if c.nudging {
		c.geom.Unlock()
		return
	}
	c.nudging = true
	c.geom.Unlock()
	go c.nudge()
}

func (c *Child) nudge() {
	c.geom.Lock()
	size := c.size
	gen := c.geomGen
	if size.Rows < 2 {
		// Below two rows a "shrink" is a resize to zero, which is a different
		// event with different consequences. (c.size is already the size the
		// console gives the CHILD, with any reserved row subtracted, so this
		// floor is about the child's own rows and nothing else.)
		c.nudging = false
		c.geom.Unlock()
		return
	}
	shrunk := size
	shrunk.Rows--
	// setSizeLocked, not resizeLocked: the shrink is a TRANSIENT poke at the
	// pty, not a new intent, so it must not become the remembered size — that
	// would make Size() report a row less than the truth for the length of a
	// settle, which is exactly the window a debugger would ask in.
	err := c.setSizeLocked(shrunk)
	c.geom.Unlock()
	if err != nil {
		c.geom.Lock()
		c.nudging = false
		c.geom.Unlock()
		return
	}

	// Cancellable, not a bare Sleep. A nudge is a goroutine the caller does not
	// join, so a bare sleep left it holding a stale intent to resize for a full
	// settle after the child was closed — and `-race` caught the consequence:
	// pty.Setsize reading the fd while the pump's teardown destroyed it. Optional
	// work must not outlive the thing it is optional about.
	select {
	case <-time.After(RepaintSettle):
	case <-c.done:
		c.geom.Lock()
		c.nudging = false
		c.geom.Unlock()
		return
	}

	c.geom.Lock()
	defer func() {
		c.nudging = false
		c.geom.Unlock()
	}()
	if c.geomGen != gen {
		// A caller resized while we settled. Its size is the current one and
		// ours is stale, so restoring would undo it — the exact "permanently
		// mis-sized" failure this mechanism exists to prevent, arriving from
		// the other direction. Its own resizeLocked already put the pty there,
		// so there is nothing to put back.
		return
	}
	_ = c.setSizeLocked(size)
}

// RepaintSettle is how long the shrink is left standing before the restore
// erases it, and it is what makes the nudge WORK rather than usually work.
//
// MEASURED, not chosen (cmd/probes/zellijrepaint, zellij 0.44.3 / macOS).
// Standard signals do not queue: issue both TIOCSWINSZ ioctls back-to-back and
// zellij can take a single SIGWINCH, read a winsize already back at 24 rows,
// and re-render nothing. That is not theoretical — it is what the first version
// shipped, and the probe caught it:
//
//	settle   repainted
//	none     6 of 12   <- the sequence that shipped: a coin flip
//	1 ms     5 of 5
//	2 ms     5 of 5
//	5 ms     8 of 8
//	20 ms    3 of 3
//
// The floor is under a millisecond and the margin is what is being bought:
// #204 measured this host's process wake-up delay at 4.92 ms, so 20 ms is about
// four wake-ups of headroom.
//
// EXPORTED so the probe reads this value instead of restating it. A probe that
// hard-codes the sequence it is meant to verify measures itself: the default
// was 0 for one commit, which is exactly the sequence this table condemns, and
// `make test-smoke` would have run it unattended (#209 C1).
const RepaintSettle = 20 * time.Millisecond

// Resize changes the child's terminal dimensions. The child gets SIGWINCH.
//
// Serialized with every other size change, including a repaint nudge's pair, so
// a caller needs no knowledge of which goroutine any other caller is on.
func (c *Child) Resize(s Size) error {
	c.geom.Lock()
	defer c.geom.Unlock()
	// Bumped even when the ioctl fails: the caller ASKED, and an in-flight
	// nudge restoring a pre-request size afterwards would be answering a
	// question nobody is still asking.
	c.geomGen++
	return c.resizeLocked(s)
}

// resizeLocked records a new INTENT and applies it. Caller holds geom.
//
// Split from setSizeLocked because the nudge's two legs are not intents: they
// poke the pty to provoke a repaint and put it straight back, and recording
// them would make c.size — the answer to "how big is this child meant to be" —
// briefly wrong.
func (c *Child) resizeLocked(s Size) error {
	if err := c.setSizeLocked(s); err != nil {
		return err
	}
	c.size = s
	return nil
}

// setSizeLocked is the one place a size reaches the pty. Caller holds geom.
func (c *Child) setSizeLocked(s Size) error {
	if c.Done() {
		return fmt.Errorf("ptychild: resize a child that has exited")
	}
	if c.fake != nil {
		c.fake.mu.Lock()
		c.fake.resizes = append(c.fake.resizes, s)
		c.fake.mu.Unlock()
		return nil
	}
	return pty.Setsize(c.ptmx, &pty.Winsize{Rows: s.Rows, Cols: s.Cols})
}

// Size is how big this child is MEANT to be: the last size a caller asked for,
// including the one it was started at.
//
// Not "the last size written to the pty" — a repaint nudge shrinks by a row and
// puts it back, and reporting that would mean this lies for the length of a
// settle.
func (c *Child) Size() Size {
	c.geom.Lock()
	defer c.geom.Unlock()
	return c.size
}

// Snapshot is the raw replay window. Prefer Replay for repainting a screen --
// this is for tests and for callers that need the bytes unfiltered.
func (c *Child) Snapshot() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ring.Snapshot()
}

// Replay is what a repaint should write: the window with capability queries
// removed, so landing on this child cannot re-ask the host terminal and have
// the answer arrive as another child's input (#127).
func (c *Child) Replay() []byte {
	c.mu.Lock()
	cutoff := c.screen.ReplaySafeEnd()
	c.mu.Unlock()
	return c.ReplayThrough(cutoff)
}

// ReplaySafeEnd is the newest absolute stream offset that does not bisect a
// withheld canonical notification candidate.
func (c *Child) ReplaySafeEnd() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.ReplaySafeEnd()
}

type notificationSpan struct{ start, end uint64 }

// ReplayThrough returns retained terminal history whose absolute position is
// below cutoff. Pair-owned notification envelopes are excluded: Couch already
// consumed their meaning and replaying them would notify again (or expose a
// bisected private control sequence after ring eviction).
func (c *Child) ReplayThrough(cutoff uint64) []byte {
	c.mu.Lock()
	snapshot := c.ring.Snapshot()
	streamEnd := c.screen.StreamEnd()
	ringStart := streamEnd - uint64(len(snapshot))
	spans := append([]notificationSpan(nil), c.notificationSpans...)
	c.mu.Unlock()

	if cutoff < ringStart {
		return nil
	}
	if cutoff > streamEnd {
		cutoff = streamEnd
	}
	snapshot = snapshot[:int(cutoff-ringStart)]
	out := make([]byte, 0, len(snapshot))
	position := ringStart
	for _, span := range spans {
		if span.end <= position || span.start >= cutoff {
			continue
		}
		keepEnd := min(span.start, cutoff)
		if keepEnd > position {
			out = append(out, snapshot[position-ringStart:keepEnd-ringStart]...)
		}
		if span.end > position {
			position = min(span.end, cutoff)
		}
	}
	if position < cutoff {
		out = append(out, snapshot[position-ringStart:cutoff-ringStart]...)
	}
	return StripQueries(out)
}

func (c *Child) AltScreen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.AltScreen()
}

// RepaintModes is what hostty.RepaintFor needs from this child, taken in one
// locked read so the two fields cannot disagree.
//
// The pair, not two accessors. `observed` is what stops a repaint asserting a
// buffer this child never witnessed (#196's shape, #209's field); reading it
// apart from the state it qualifies is how the two come to disagree, so there
// is deliberately no exported AltScreenObserved to reach for (#209 BR-12).
func (c *Child) RepaintModes() (altScreen, observed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.AltScreen(), c.screen.AltScreenObserved()
}

func (c *Child) Mouse() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.Mouse()
}

// MouseObserved reports whether this child's output has said anything about
// mouse mode. False means unknown rather than "no" -- see Screen.MouseObserved.
func (c *Child) MouseObserved() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.MouseObserved()
}

// SGRMouse reports whether the child asked for SGR-encoded mouse coordinates.
func (c *Child) SGRMouse() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.SGRMouse()
}

// TakeRowDirty reports and clears whether this child did something that may
// have destroyed a reserved row -- dropping the scrolling region, or erasing
// the display.
func (c *Child) TakeRowDirty() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.TakeRowDirty()
}

// TakeBell reports and clears whether this child rang the bell.
func (c *Child) TakeBell() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.TakeBell()
}

// Done reports whether the child has exited AND been reaped. It is a closed
// channel rather than a signal probe, because `kill -0` succeeds for a zombie.
func (c *Child) Done() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

// Exited closes when the child has exited and been reaped. Consumers that
// supervise several children select on this signal, then call Wait for the
// already-published exit code; exposing the signal avoids one permanently
// blocked waiter goroutine per warm detached child.
func (c *Child) Exited() <-chan struct{} { return c.done }

// Wait blocks until the child exits and returns its code.
func (c *Child) Wait() int {
	<-c.done
	return c.code
}

func (c *Child) PID() int {
	if c.cmd == nil || c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}

// Signal sends sig to the child.
func (c *Child) Signal(sig os.Signal) error {
	if c.fake != nil {
		return c.fakeSignal(sig)
	}
	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("ptychild: no process")
	}
	return c.cmd.Process.Signal(sig)
}

// Close tears the child down: closing the pty ends the pump, which reaps.
func (c *Child) Close() error {
	var err error
	c.closeOnce.Do(func() {
		if c.fake != nil {
			c.Exit(0)
			return
		}
		// Under geom, so an in-flight Setsize finishes before the fd goes.
		// The nudge releases this lock while it settles, so the wait here is a
		// syscall long, not a settle long.
		c.geom.Lock()
		err = c.ptmx.Close()
		c.geom.Unlock()
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
	})
	return err
}
