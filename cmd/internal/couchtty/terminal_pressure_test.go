package couchtty

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"golang.org/x/term"
)

const pressureWindow = 2 * time.Second

// The real helper reports receipt on fd 4 BEFORE enqueueing terminal output.
// fd 3 starts the finite workload only after all twelve children are attached.
func TestCouchPressurePTYChild(t *testing.T) {
	if os.Getenv("PAIR_COUCH_PRESSURE_CHILD") != "1" {
		t.Skip("PTY helper")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	start := os.NewFile(3, "start")
	defer start.Close()
	receipt := os.NewFile(4, "receipt")
	defer receipt.Close()
	ack := make(chan struct{}, 1)
	go func() {
		var b [1]byte
		for {
			if _, err := os.Stdin.Read(b[:]); err != nil {
				return
			}
			if b[0] == 'p' {
				fmt.Fprintf(receipt, "I %d\n", time.Now().UnixNano())
				select {
				case ack <- struct{}{}:
				default:
				}
			}
		}
	}()
	fmt.Print("\x1b[2J\x1b[HREADY")
	fmt.Fprintln(receipt, "R 0")
	var command [1]byte
	if _, err := io.ReadFull(start, command[:]); err != nil {
		return
	}
	burst := command[0] == 'B'
	emitted := pressureProduce(context.Background(), burst, func(p []byte) { _, _ = os.Stdout.Write(p) }, ack)
	fmt.Fprintf(receipt, "D %d\n", emitted)
	if os.Getenv("PAIR_COUCH_PRESSURE_TRAILING") == "true" {
		if _, err := io.ReadFull(start, command[:]); err != nil {
			return
		}
	}
	fmt.Print(pressureCompletion(emitted))
	// Stay alive until the parent closes the control pipe, avoiding an exit paint
	// in the recovery window. Process teardown joins the stdin reader too.
	_, _ = start.Read(command[:])
}

// At most 200 chunks per child, 4096 bytes per chunk (9.375 MiB/trial).
// Ticker skips missed deadlines: this models bounded offered load, not an
// unbounded catch-up queue. ACK and progress rows survive every pressure chunk.
func pressureProduce(ctx context.Context, burst bool, write func([]byte), ack <-chan struct{}) (emitted uint64) {
	emit := func(p []byte) { write(p); emitted += uint64(len(p)) }
	interval, chunk := 100*time.Millisecond, 128
	if burst {
		interval, chunk = 10*time.Millisecond, 4096
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(pressureWindow)
	defer timer.Stop()
	for i := 0; i < 200; {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			return
		case <-ack:
			emit([]byte("\x1b[1;1HACK"))
		case <-ticker.C:
			i++
			prefix := fmt.Sprintf("\x1b[2;1HPROGRESS%06d\x1b[3;1H", i)
			// Repeated cursor-positioned lines cause parsing/render work without
			// scrolling the independent ACK row off the screen.
			body := strings.Repeat("\x1b[3;1H"+strings.Repeat("x", 70), (chunk-len(prefix))/76)
			emit([]byte(prefix + body))
		}
	}
	return
}

type pressureHost struct {
	*couchSoakHost
	delay   time.Duration
	delayed atomic.Bool
}

func (h *pressureHost) Write(p []byte) (int, error) { return h.WriteContext(context.Background(), p) }
func (h *pressureHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if h.delayed.Load() && h.delay > 0 {
		timer := time.NewTimer(h.delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return h.couchSoakHost.WriteContext(ctx, p)
}

func TestCouchOutputPressure(t *testing.T) {
	if os.Getenv("PAIR_COUCH_PRESSURE") != "1" {
		t.Skip("set PAIR_COUCH_PRESSURE=1 for bounded isolated experiment")
	}
	for _, mode := range []string{"fake", "pty"} {
		for _, trial := range []string{"baseline", "burst", "burst-one-cpu", "slow-host"} {
			for rep := 1; rep <= 3; rep++ {
				t.Run(fmt.Sprintf("%s/%s/%d", mode, trial, rep), func(t *testing.T) { runPressureTrial(t, mode, trial) })
			}
		}
	}
}

// Routine CI exercises the same real-helper lifecycle and independent receipt
// channel without the twelve-child pressure matrix.
func TestCouchPressureControl(t *testing.T) { runPressureTrial(t, "pty", "control") }

// pressureAwait gives even context-free observations a deadline. The caller
// owns cancellation of the resources used by work and joins every worker.
func pressureAwait(ctx context.Context, workers *sync.WaitGroup, work func() error) error {
	done := make(chan error, 1)
	workers.Add(1)
	go func() { defer workers.Done(); done <- work() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func pressureCompletion(emitted uint64) string { return fmt.Sprintf("\x1b[4;1HCOMPLETE%012d", emitted) }

func TestCouchPressureStalledOperation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	var workers sync.WaitGroup
	start := time.Now()
	err := pressureAwait(ctx, &workers, func() error { _, err := writer.Write([]byte("blocked")); return err })
	if err != context.DeadlineExceeded {
		t.Fatalf("stalled write returned %v, want deadline", err)
	}
	reader.CloseWithError(ctx.Err())
	joined := make(chan struct{})
	go func() { workers.Wait(); close(joined) }()
	select {
	case <-joined:
	case <-time.After(time.Second):
		t.Fatal("stalled operation did not join")
	}
	if time.Since(start) > time.Second {
		t.Fatal("deadline did not bound stalled operation")
	}
}

func TestCouchPressureTrailingPTYOutput(t *testing.T) { runPressureTrial(t, "pty", "trailing-control") }

func runPressureTrial(t *testing.T, mode, trial string) {
	n := 12
	if strings.HasSuffix(trial, "control") {
		n = 1
	}
	burst := strings.HasPrefix(trial, "burst")
	cpus := runtime.GOMAXPROCS(0)
	if trial == "burst-one-cpu" {
		runtime.GOMAXPROCS(1)
		defer runtime.GOMAXPROCS(cpus)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	host := &pressureHost{couchSoakHost: &couchSoakHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 54, Cols: 191}), em: vt.NewEmulator(191, 54)}}
	if trial == "slow-host" {
		host.delay = 50 * time.Millisecond
	}
	drain := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, host.em); close(drain) }()
	reader, input := io.Pipe()
	con := New(host, reader)
	con.SetErrorWriter(io.Discard)
	con.SetOperationDispatcher(con.ExecuteConsoleOperation)
	done := make(chan int, 1)
	var children []*ptychild.Child
	var inventory []couchcore.ActionableThreadSummary
	var files []*os.File
	var workers, operations sync.WaitGroup
	var bytesIn, publications, emitted atomic.Uint64
	receipts := make([]atomic.Int64, n)
	ready := make([]atomic.Bool, n)
	finished := make([]atomic.Bool, n)
	expected := make([]atomic.Uint64, n)
	var releaseTrailing func()
	starts := make([]func(), n)
	var runStarted bool
	t.Cleanup(func() {
		cancel()
		con.Stop()
		input.Close()
		reader.Close()
		// Close the pipe, not Emulator.closed, while Read is still running.
		_ = host.em.InputPipe().(io.Closer).Close()
		for _, f := range files {
			f.Close()
		}
		joined := make(chan struct{})
		go func() {
			var closes sync.WaitGroup
			for _, child := range children {
				closes.Add(1)
				go func(c *ptychild.Child) { defer closes.Done(); c.Close(); c.Wait() }(child)
			}
			closes.Wait()
			workers.Wait()
			operations.Wait()
			if runStarted {
				<-done
			}
			<-drain
			// All writers and the reader are joined before touching Emulator.closed.
			host.em.Close()
			close(joined)
		}()
		select {
		case <-joined:
		case <-time.After(5 * time.Second):
			t.Error("pressure teardown exceeded shared five-second deadline")
		}
	})
	do := func(label string, work func() error) {
		t.Helper()
		if err := pressureAwait(ctx, &operations, work); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}

	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("pressure-%02d", i)
		attached := make(chan struct{})
		sink := func(ctx context.Context, b ptychild.OutputBatch) error {
			select {
			case <-attached:
			case <-ctx.Done():
				return ctx.Err()
			}
			bytesIn.Add(uint64(len(b.Raw)))
			publications.Add(1)
			return con.Deliver(ctx, id, b)
		}
		var child *ptychild.Child
		if mode == "pty" {
			cr, cw, e := os.Pipe()
			if e != nil {
				t.Fatal(e)
			}
			rr, rw, e := os.Pipe()
			if e != nil {
				cr.Close()
				cw.Close()
				t.Fatal(e)
			}
			files = append(files, cw, rr)
			child, err = ptychild.Start(ptychild.Options{Argv: []string{binary, "-test.run=^TestCouchPressurePTYChild$"}, Env: []string{"PAIR_COUCH_PRESSURE_CHILD=1", fmt.Sprintf("PAIR_COUCH_PRESSURE_TRAILING=%t", trial == "trailing-control")}, Size: ptychild.Size{Rows: 53, Cols: 191}, ExtraFiles: []*os.File{cr, rw}, Sink: sink})
			cr.Close()
			rw.Close()
			if err != nil {
				t.Fatal(err)
			}
			workers.Add(1)
			go func(index int) {
				defer workers.Done()
				scanner := bufio.NewScanner(rr)
				scanner.Buffer(make([]byte, 128), 128)
				for scanner.Scan() {
					fields := strings.Fields(scanner.Text())
					if len(fields) != 2 {
						continue
					}
					switch fields[0] {
					case "R":
						ready[index].Store(true)
					case "I":
						stamp, _ := strconv.ParseInt(fields[1], 10, 64)
						receipts[index].Store(stamp)
					case "D":
						count, _ := strconv.ParseUint(fields[1], 10, 64)
						emitted.Add(count)
						expected[index].Store(count)
						finished[index].Store(true)
					}
				}
			}(i)
			if trial == "trailing-control" {
				releaseTrailing = func() {
					do("release trailing PTY output", func() error { _, err := cw.Write([]byte("T")); return err })
				}
			}
			starts[i] = func() {
				command := "N"
				if burst {
					command = "B"
				}
				if _, e := cw.Write([]byte(command)); e != nil {
					t.Error(e)
				}
			}
		} else {
			child = ptychild.NewFakeChild(nil)
			child.SetSink(sink)
			ack := make(chan struct{}, 1)
			start := make(chan struct{})
			starts[i] = func() { close(start) }
			workers.Add(2)
			go func(index int, c *ptychild.Child) {
				defer workers.Done()
				select {
				case <-start:
				case <-ctx.Done():
					return
				}
				count := pressureProduce(ctx, burst, c.Feed, ack)
				emitted.Add(count)
				expected[index].Store(count)
				c.Feed([]byte(pressureCompletion(count)))
				finished[index].Store(true)
			}(i, child)
			go func(index int, c *ptychild.Child) {
				defer workers.Done()
				ticker := time.NewTicker(time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						for _, p := range c.Writes() {
							if strings.Contains(string(p), "p") {
								receipts[index].Store(time.Now().UnixNano())
								ack <- struct{}{}
								return
							}
						}
					}
				}
			}(i, child)
		}
		children = append(children, child)
		do("attach child", func() error { con.Attach(id, id, child); return nil })
		inventory = append(inventory, couchcore.ActionableThreadSummary{
			Address: couchcore.ThreadAddress{RepoScope: "legacy", Tag: couchcore.ThreadTag(id)},
			Name:    id, WorkingPath: id, State: couchcore.ThreadLive,
		})
		close(attached)
		if mode == "fake" {
			do("fake readiness", func() error { child.Feed([]byte("\x1b[2J\x1b[HREADY")); return nil })
			ready[i].Store(true)
		}
	}
	con.mu.Lock()
	con.menu = NewMenuState(inventory, inventory[0].Address)
	con.menuReady = true
	con.mu.Unlock()
	go func() { done <- con.Run() }()
	runStarted = true
	wait := func(label string, predicate func() bool) time.Time {
		t.Helper()
		for ctx.Err() == nil {
			var matched bool
			do(label, func() error { matched = predicate(); return nil })
			if matched {
				return time.Now()
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("timeout %s: %v", label, ctx.Err())
		return time.Time{}
	}
	endpointText := func(index int) string {
		f, e := children[index].Endpoint().Snapshot(time.Now())
		if e != nil {
			return ""
		}
		var b strings.Builder
		for _, cell := range f.Cells {
			b.WriteString(cell.Content)
		}
		return b.String()
	}
	wait("all ready", func() bool {
		for i := range ready {
			if !ready[i].Load() {
				return false
			}
		}
		return strings.Contains(host.text(), "READY")
	})
	host.delayed.Store(true)
	startAt := time.Now()
	recoveryTimer := time.AfterFunc(pressureWindow+5*time.Second, cancel)
	defer recoveryTimer.Stop()
	for _, start := range starts {
		do("start child", func() error { start(); return nil })
	}
	time.Sleep(250 * time.Millisecond)
	sent := time.Now()
	do("pane input", func() error { _, err := input.Write([]byte("p")); return err })
	// Observe the three paths together: sequential waits would inflate later
	// measurements and can misclassify a fast endpoint behind slow delivery.
	var receiptAt, endpointAt, displayAt time.Time
	probe := func() {
		now := time.Now()
		if stamp := receipts[0].Load(); stamp != 0 && receiptAt.IsZero() {
			receiptAt = time.Unix(0, stamp)
		}
		if endpointAt.IsZero() && strings.Contains(endpointText(0), "ACK") {
			endpointAt = now
		}
		if displayAt.IsZero() && strings.Contains(host.text(), "ACK") {
			displayAt = time.Now()
		}
	}
	// Preserve the menu probe inside the same output window even if the pane
	// stalls. Missing ACKs are right-censored before the menu covers the pane.
	probeDeadline := startAt.Add(1500 * time.Millisecond)
	for time.Now().Before(probeDeadline) {
		do("ACK observation", func() error { probe(); return nil })
		time.Sleep(10 * time.Millisecond)
	}
	do("ACK observation", func() error { probe(); return nil })
	receiptCensored, endpointCensored, displayCensored := receiptAt.IsZero(), endpointAt.IsZero(), displayAt.IsZero()
	if receiptAt.IsZero() {
		receiptAt = probeDeadline
	}
	if endpointAt.IsZero() {
		endpointAt = probeDeadline
	}
	if displayAt.IsZero() {
		displayAt = probeDeadline
	}
	menuSent := time.Now()
	if menuSent.Sub(startAt) >= pressureWindow {
		t.Fatal("menu probe missed pressure window; trial invalid")
	}
	do("menu input", func() error { _, err := input.Write([]byte{0}); return err })
	menuAt := wait("rendered menu", func() bool { return strings.Contains(host.text(), "threads") })
	do("restore input", func() error { _, err := input.Write([]byte{27}); return err })
	restoredAt := wait("pane restored after menu", func() bool { return strings.Contains(host.text(), "ACK") && !strings.Contains(host.text(), "threads") })
	wait("finite output windows", func() bool {
		for i := range finished {
			if !finished[i].Load() {
				return false
			}
		}
		return true
	})
	// The regression invokes this same recovery operation before releasing the
	// final PTY bytes; removing completion checks must make that test fail.
	recoverOutput := func(recoveryCtx context.Context) error {
		until := func(predicate func() bool) error {
			for {
				var matched bool
				if err := pressureAwait(recoveryCtx, &operations, func() error { matched = predicate(); return nil }); err != nil {
					return err
				}
				if matched {
					return nil
				}
				select {
				case <-recoveryCtx.Done():
					return recoveryCtx.Err()
				case <-time.After(time.Millisecond):
				}
			}
		}
		if err := until(func() bool {
			for i := range children {
				if !strings.Contains(endpointText(i), strings.TrimPrefix(pressureCompletion(expected[i].Load()), "\x1b[4;1H")) {
					return false
				}
			}
			return true
		}); err != nil {
			return err
		}
		// Snapshot may see the marker just before publication enqueue.
		wantBytes := emitted.Load() + uint64(n*len("\x1b[2J\x1b[HREADY"))
		for i := range children {
			wantBytes += uint64(len(pressureCompletion(expected[i].Load())))
		}
		if err := until(func() bool { return bytesIn.Load() >= wantBytes }); err != nil {
			return err
		}
		for _, c := range children {
			if err := c.FlushOutput(recoveryCtx); err != nil {
				return err
			}
		}
		if err := con.presenter.Flush(recoveryCtx); err != nil {
			return err
		}
		if err := until(func() bool {
			return strings.Contains(host.text(), strings.TrimPrefix(pressureCompletion(expected[0].Load()), "\x1b[4;1H"))
		}); err != nil {
			return err
		}
		if bytesIn.Load() != wantBytes {
			return fmt.Errorf("ingested %d bytes, want emitted+framing %d", bytesIn.Load(), wantBytes)
		}
		return nil
	}
	if releaseTrailing != nil {
		do("flush before trailing write", func() error { return children[0].FlushOutput(ctx) })
		heldCtx, heldCancel := context.WithTimeout(ctx, 30*time.Millisecond)
		err := recoverOutput(heldCtx)
		heldCancel()
		if err != context.DeadlineExceeded {
			t.Fatalf("recovery while trailing PTY output withheld: got %v, want deadline exceeded", err)
		}
		releaseTrailing()
	}
	do("terminal recovery", func() error { return recoverOutput(ctx) })

	recovery := time.Since(startAt.Add(pressureWindow))
	if recovery < 0 {
		recovery = 0
	}
	var frame string
	do("final endpoint", func() error { frame = endpointText(0); return nil })
	progress := "missing"
	if at := strings.Index(frame, "PROGRESS"); at >= 0 && at+14 <= len(frame) {
		progress = frame[at : at+14]
	}
	if progress == "missing" {
		t.Fatal("child output did not reach endpoint")
	}
	var hostBytes, hostWrites uint64
	do("host counters", func() error {
		host.mu.Lock()
		defer host.mu.Unlock()
		hostBytes, hostWrites = host.bytes, host.writes
		return nil
	})
	selective := (displayAt.Sub(sent) > time.Second || receiptAt.Sub(sent) > time.Second) && menuAt.Sub(menuSent) < 100*time.Millisecond
	t.Logf("mode=%s trial=%s children=%d geometry=191x54 cpus=%d window=%s elapsed=%s byte_cap=9830400 emitted_window_bytes=%d ingested_raw=%d publications=%d host_bytes=%d host_writes=%d receipt=%s endpoint_ack=%s displayed_ack=%s menu=%s final=%s receipt_censored=%t endpoint_censored=%t display_censored=%t restored_after_menu=%s recovery=%s ack_poll=10ms selective_stall=%t", mode, trial, n, runtime.GOMAXPROCS(0), pressureWindow, time.Since(startAt), emitted.Load(), bytesIn.Load(), publications.Load(), hostBytes, hostWrites, receiptAt.Sub(sent), endpointAt.Sub(sent), displayAt.Sub(sent), menuAt.Sub(menuSent), progress, receiptCensored, endpointCensored, displayCensored, restoredAt.Sub(menuAt), recovery, selective)
}
