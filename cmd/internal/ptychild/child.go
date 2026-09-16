package ptychild

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"golang.org/x/sys/unix"
)

// Size is a terminal's dimensions. It exists so callers do not have to import
// creack/pty just to say how big a child should be -- the console reserves a
// row by subtracting from Rows, and that arithmetic should not require knowing
// what a Winsize is.
type Size struct {
	Rows, Cols uint16
}

// OutputBatch carries endpoint publication plus optional raw diagnostic bytes.
// Raw must never be replayed to a parent terminal.
type OutputBatch struct {
	Terminal terminal.Output
	Err      error
	Raw      []byte
}

// Options configures a child. Everything here is what the CALLER knows; nothing
// about switching policy belongs in it.
type Options struct {
	EndpointID string
	Dir        string
	Argv       []string
	Env        []string
	Size       Size
	ExtraFiles []*os.File

	// RingBytes bounds the raw diagnostic tail. Zero means
	// DefaultRingBytes.
	RingBytes int

	// Sink acknowledges already-ingested output on a bounded publication worker.
	// The Endpoint and diagnostic ring are updated before delivery. Callbacks
	// must honor cancellation and must not synchronously Close this Child.
	Sink Sink
}

// Child owns one PTY process, its authoritative Endpoint, and bounded output delivery.
type Child struct {
	cmd         *exec.Cmd
	ptmx        *os.File
	sink        Sink
	endpoint    *terminal.Endpoint
	transport   *ttyio.File
	fd          int
	publication *publication
	feedMu      sync.Mutex

	mu   sync.Mutex
	ring *Ring

	// done closes once the child has been reaped; code is written before the
	// close, so reading it after <-done needs no further synchronisation.
	// Same shape as couchcore's execHandle, and for the same reason: `kill -0`
	// succeeds for a zombie, so liveness must not be a syscall.
	done chan struct{}
	code int

	closeOnce sync.Once
	closeErr  error

	// geom serializes PTY geometry changes independently of diagnostic output.
	geom sync.Mutex
	size Size

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
	if err := (terminal.Geometry{Cols: int(opts.Size.Cols), Rows: int(opts.Size.Rows)}).Validate(); err != nil {
		return nil, err
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
		cmd:  cmd,
		ptmx: ptmx,
		sink: opts.Sink,
		ring: NewRing(capacity),
		done: make(chan struct{}),
		// Endpoint and PTY begin with the same acknowledged geometry.
		size: opts.Size,
	}
	c.fd = int(ptmx.Fd())
	c.transport, err = ttyio.NewFile(ptmx, ptmx, true)
	if err == nil {
		err = c.initTerminal(opts.EndpointID)
	}
	if err != nil {
		_ = cmd.Process.Kill()
		if c.transport != nil {
			_ = c.transport.Close()
		} else {
			_ = ptmx.Close()
		}
		_ = cmd.Wait()
		return nil, err
	}
	go c.pump()
	return c, nil
}

// pump reads the child until the pty closes, then reaps it.
func (c *Child) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := c.transport.ReadContext(c.publication.ctx, buf)
		if n > 0 {
			if deliveryErr := c.ingest(buf[:n]); deliveryErr != nil {
				_ = c.cmd.Process.Kill()
				break
			}
		}
		if err != nil {
			if c.publication.ctx.Err() != nil {
				_ = c.cmd.Process.Kill()
			}
			break
		}
	}
	c.endpoint.EndInput()
	c.finish(procutil.WaitCode(c.cmd))
}

// Write sends bytes to the child's terminal.
func (c *Child) Write(p []byte) (int, error) {
	return c.endpoint.WriteInput(p)
}

// Resize changes the child's terminal dimensions. The child gets SIGWINCH.
//
// Endpoint serializes parser geometry and input with the acknowledged ioctl.
func (c *Child) Resize(s Size) error {
	return c.endpoint.Resize(terminal.Geometry{Cols: int(s.Cols), Rows: int(s.Rows)}, func(terminal.Geometry) error { return c.ResizePTY(s) })
}

// ResizePTY acknowledges only the OS geometry; Presenter already owns the
// endpoint resize in this path, so calling Resize here would recurse.
func (c *Child) ResizePTY(s Size) error {
	c.geom.Lock()
	defer c.geom.Unlock()
	return c.resizeLocked(s)
}

// resizeLocked commits geometry only after the ioctl succeeds. Caller holds geom.
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
	return unix.IoctlSetWinsize(c.fd, unix.TIOCSWINSZ, &unix.Winsize{Row: s.Rows, Col: s.Cols})
}

// Size returns the last acknowledged PTY geometry, including its launch size.
func (c *Child) Size() Size {
	c.geom.Lock()
	defer c.geom.Unlock()
	return c.size
}

// Snapshot returns the bounded raw diagnostic tail, never display state.
func (c *Child) Snapshot() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ring.Snapshot()
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
	c.closeOnce.Do(func() {
		c.publication.close()
		c.endpoint.EndInput()
		if c.fake != nil {
			c.Exit(0)
		} else {
			_ = c.cmd.Process.Kill()
			c.geom.Lock()
			c.closeErr = c.transport.Close()
			c.geom.Unlock()
		}
		<-c.done
		c.feedMu.Lock()
		c.endpoint.Close()
		c.feedMu.Unlock()
	})
	return c.closeErr
}
