package ptychild

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// Size is a terminal's dimensions. It exists so callers do not have to import
// creack/pty just to say how big a child should be -- the console reserves a
// row by subtracting from Rows, and that arithmetic should not require knowing
// what a Winsize is.
type Size struct {
	Rows, Cols uint16
}

// Options configures a child. Everything here is what the CALLER knows; nothing
// about switching policy belongs in it.
type Options struct {
	Dir  string
	Argv []string
	Env  []string
	Size Size

	// RingBytes is how much output to keep for a repaint. Zero means
	// DefaultRingBytes.
	RingBytes int

	// Sink receives every chunk the child writes, in order, from the pump
	// goroutine. The caller decides whether it reaches a screen -- that is
	// switching policy, and it stays with the caller.
	//
	// The ring and the screen are updated BEFORE Sink runs, so a caller that
	// switches away inside Sink still repaints a current screen.
	Sink func([]byte)
}

// Child is one process on a pty, with the window of output needed to repaint it
// and the state its output implies.
type Child struct {
	cmd  *exec.Cmd
	ptmx *os.File
	sink func([]byte)

	mu     sync.Mutex
	ring   *Ring
	screen *Screen

	// done closes once the child has been reaped; code is written before the
	// close, so reading it after <-done needs no further synchronisation.
	// Same shape as couchcore's execHandle, and for the same reason: `kill -0`
	// succeeds for a zombie, so liveness must not be a syscall.
	done chan struct{}
	code int

	closeOnce sync.Once
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
			c.mu.Unlock()

			if c.sink != nil {
				c.sink(chunk)
			}
		}
		if err != nil {
			break
		}
	}
	c.code = waitCode(c.cmd)
}

func waitCode(cmd *exec.Cmd) int {
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if ok := asExitError(err, &ee); ok {
		return ee.ExitCode()
	}
	return -1
}

// Write sends bytes to the child's terminal.
func (c *Child) Write(p []byte) (int, error) { return c.ptmx.Write(p) }

// Resize changes the child's terminal dimensions. The child gets SIGWINCH.
func (c *Child) Resize(s Size) error {
	return pty.Setsize(c.ptmx, &pty.Winsize{Rows: s.Rows, Cols: s.Cols})
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
	return StripQueries(c.Snapshot())
}

func (c *Child) AltScreen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.AltScreen()
}

func (c *Child) Mouse() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.Mouse()
}

// TakeRegionLost reports and clears whether this child did something that can
// drop the host's scrolling region.
func (c *Child) TakeRegionLost() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.TakeRegionLost()
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

// Wait blocks until the child exits and returns its code.
func (c *Child) Wait() int {
	<-c.done
	return c.code
}

func (c *Child) PID() int {
	if c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}

// Signal sends sig to the child.
func (c *Child) Signal(sig os.Signal) error {
	if c.cmd.Process == nil {
		return fmt.Errorf("ptychild: no process")
	}
	return c.cmd.Process.Signal(sig)
}

// Close tears the child down: closing the pty ends the pump, which reaps.
func (c *Child) Close() error {
	var err error
	c.closeOnce.Do(func() {
		err = c.ptmx.Close()
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
	})
	return err
}
