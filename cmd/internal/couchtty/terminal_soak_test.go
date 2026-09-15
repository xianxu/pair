package couchtty

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"golang.org/x/term"
)

func TestCouchSoakPTYChild(t *testing.T) {
	if os.Getenv("PAIR_COUCH_SOAK_CHILD") != "1" {
		t.Skip("PTY subprocess helper")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		os.Exit(2)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	fmt.Print("\x1b[?1006h\x1b[2J\x1b[HREADY")
	line := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return
		}
		fmt.Print("\x1b[3;1Houtput界")
		for _, b := range buf[:n] {
			if b != '\r' && b != '\n' {
				line = append(line, b)
				if len(line) > 4096 {
					os.Exit(3)
				}
				continue
			}
			if bytes.HasPrefix(line, []byte("app")) {
				fmt.Print("\x1b[?1002h\x1b[?1049h")
			}
			if bytes.HasPrefix(line, []byte("shell")) {
				fmt.Print("\x1b[?1002l\x1b[?1049l")
			}
			fmt.Printf("\x1b[?2026h\x1b[0m\x1b[2J\x1b[H%s\x1b[2;1H\x1b[1;38;2;1;2;3m界 é\x1b[0m\x1b[2;5H\x1b[?25h\x1b[?2026l", hex.EncodeToString(line))
			line = line[:0]
		}
	}
}

// FakeHost supplies geometry/event ownership only: emitted bytes go solely to
// this bounded interpreter, never FakeHost's ever-growing transcript.
type couchSoakHost struct {
	*hostty.FakeHost
	mu            sync.Mutex
	em            *vt.Emulator
	bytes, writes uint64
}

func (h *couchSoakHost) Write(p []byte) (int, error) { return h.WriteContext(context.Background(), p) }
func (h *couchSoakHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	size, _ := h.Size()
	if h.em.Width() != int(size.Cols) || h.em.Height() != int(size.Rows) {
		h.em.Resize(int(size.Cols), int(size.Rows))
	}
	n, err := h.em.Write(p)
	h.bytes += uint64(n)
	h.writes++
	return n, err
}
func (h *couchSoakHost) compare(frame terminal.Frame) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for y := 0; y < frame.Geometry.Rows; y++ {
		for x := 0; x < frame.Geometry.Cols; x++ {
			want := frame.Cells[y*frame.Geometry.Cols+x]
			got := h.em.CellAt(x, y)
			if got == nil {
				return fmt.Errorf("missing physical cell%d,%d", x, y)
			}
			// Blank paint may be an explicit space while the endpoint stores empty.
			content := func(s string) string {
				if s == "" {
					return " "
				}
				return s
			}
			if content(got.Content) != content(want.Content) || got.Width != want.Width || !reflect.DeepEqual(got.Style, want.Style) || !reflect.DeepEqual(got.Link, want.Link) {
				return fmt.Errorf("cell%d,%d got%+v want%+v", x, y, *got, want)
			}
		}
	}
	cursor := h.em.CursorPosition()
	if cursor.X != frame.Cursor.X || cursor.Y != frame.Cursor.Y {
		return fmt.Errorf("cursor%v want%+v", cursor, frame.Cursor)
	}
	return nil
}

// This exercises hosted attachment replacement, not persistent-agent Zellij
// survival: the independent native harness owns that distinct obligation.
func TestCouchProductionSoak(t *testing.T) {
	duration := time.Duration(0)
	if raw := os.Getenv("PAIR_TERMINAL_SOAK_DURATION"); raw != "" {
		var err error
		duration, err = time.ParseDuration(raw)
		if err != nil || duration <= 0 {
			t.Fatalf("invalid soak duration%q", raw)
		}
	}
	host := &couchSoakHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 8, Cols: 160}), em: vt.NewEmulator(160, 8)}
	t.Cleanup(func() { host.em.Close() })
	reader, input := io.Pipe()
	con := New(host, reader)
	con.SetErrorWriter(io.Discard)
	done := make(chan int, 1)
	children := map[string]*ptychild.Child{}
	t.Cleanup(func() {
		con.Stop()
		input.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("console did not join")
		}
		for _, c := range children {
			c.Close()
		}
		reader.Close()
	})
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	attach := func(id string) *ptychild.Child {
		t.Helper()
		size, _ := host.Size()
		ready := make(chan struct{})
		child, err := ptychild.Start(ptychild.Options{Argv: []string{binary, "-test.run=^TestCouchSoakPTYChild$"}, Env: []string{"PAIR_COUCH_SOAK_CHILD=1"}, Size: ptychild.Size{Rows: size.Rows - 1, Cols: size.Cols}, Sink: func(ctx context.Context, b ptychild.OutputBatch) error {
			select {
			case <-ready:
			case <-ctx.Done():
				return ctx.Err()
			}
			return con.Deliver(ctx, id, b)
		}})
		if err != nil {
			t.Fatal(err)
		}
		children[id] = child
		con.Attach(id, id, child)
		close(ready)
		return child
	}
	a, b := attach("one"), attach("two")
	go func() { done <- con.Run() }()
	send := func(raw string) {
		t.Helper()
		if _, err := input.Write([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	wait := func(label string, predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if predicate() {
				return
			}
			select {
			case code := <-done:
				done <- code
				t.Fatalf("console exited%d waiting%s", code, label)
			default:
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("timeout waiting%s", label)
	}
	receipt := func(child *ptychild.Child, want string) {
		t.Helper()
		wait("receipt "+want, func() bool {
			f, err := child.Endpoint().Snapshot(time.Now())
			if err != nil {
				t.Fatal(err)
			}
			var s strings.Builder
			for _, c := range f.Cells {
				s.WriteString(c.Content)
			}
			return strings.Contains(s.String(), want)
		})
		if err := child.FlushOutput(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := con.presenter.Flush(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	receipt(a, "READY")
	receipt(b, "READY")
	start := time.Now()
	lastProgress := start
	iterations, replacements := 0, 0
	var maxLatency time.Duration
	for iterations < 8 || time.Since(start) < duration {
		id, child := "one", a
		if iterations%2 == 1 {
			id, child = "two", b
		}
		con.Switch(id)
		wait("selected actor", func() bool { return con.presenter.View().Admitted == child.Endpoint().ID() })
		mode := fmt.Sprintf("app%d", iterations)
		if iterations%3 == 0 {
			mode = fmt.Sprintf("shell%d", iterations)
		}
		send(mode + "\r")
		receipt(child, hex.EncodeToString([]byte(mode)))
		// A chrome-origin gesture cannot acquire the child after crossing its edge.
		size, _ := host.Size()
		send(fmt.Sprintf("\x1b[<0;159;%dM\x1b[<32;2;2M\x1b[<0;2;2m", size.Rows))
		token := fmt.Sprintf("r%d", iterations)
		sent := time.Now()
		send(token + "\r")
		receipt(child, hex.EncodeToString([]byte(token)))
		if elapsed := time.Since(sent); elapsed > maxLatency {
			maxLatency = elapsed
		}
		frame, err := child.Endpoint().Snapshot(time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err := host.compare(frame); err != nil {
			t.Fatal(err)
		}
		// Physical reports reach exactly the selected mouse-aware actor; normal
		// screen applications that did not request tracking receive only text.
		drag := "\x1b[<0;2;2M\x1b[<32;3;2M\x1b[<0;3;2m"
		dragToken := fmt.Sprintf("drag%d", iterations)
		expected := dragToken
		if strings.HasPrefix(mode, "app") {
			expected = drag + dragToken
		}
		send(drag + dragToken + "\r")
		receipt(child, hex.EncodeToString([]byte(expected)))
		// A panel-owned press remains parent-owned even after a fresh actor landing.
		send("\x00")
		wait("panel", func() bool { return con.presenter.View().Selected == "" })
		send("\x1b[<0;159;1M")
		con.Switch(id)
		wait("actor after panel", func() bool { return con.presenter.View().Admitted == child.Endpoint().ID() })
		send("\x1b[<32;2;2M\x1b[<0;2;2mpanel\r")
		receipt(child, hex.EncodeToString([]byte("panel")))
		rows := uint16(8)
		if iterations%2 == 0 {
			rows = 10
		}
		host.SetSize(ptychild.Size{Rows: rows, Cols: 160})
		wait("both resized", func() bool { return a.Size().Rows == rows-1 && b.Size().Rows == rows-1 })
		if iterations%4 == 3 {
			send("\x00")
			wait("replacement panel", func() bool { return con.presenter.View().Selected == "" })
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
			wait("old attachment removed", func() bool { con.mu.Lock(); defer con.mu.Unlock(); return con.panes["two"] == nil })
			b = attach("two")
			receipt(b, "READY")
			replacements++
			if con.presenter.View().Selected != "" {
				t.Fatal("background attachment stole panel focus")
			}
		}
		iterations++
		if now := time.Now(); now.Sub(lastProgress) >= time.Minute {
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			host.mu.Lock()
			written, writes := host.bytes, host.writes
			host.mu.Unlock()
			t.Logf("couch soak progress elapsed=%s iterations=%d attachment_replacements=%d parent_bytes=%d writes=%d max_input_visible=%s heap_alloc=%d goroutines=%d", now.Sub(start), iterations, replacements, written, writes, maxLatency, stats.HeapAlloc, runtime.NumGoroutine())
			lastProgress = now
		}
	}
	con.Stop()
	input.Close()
	select {
	case code := <-done:
		done <- code
		if code != 0 {
			t.Fatalf("console exit%d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("console shutdown timeout")
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	t.Logf("couch soak seed=0 duration=%s iterations=%d attachment_replacements=%d max_input_visible=%s parent_bytes=%d writes=%d heap_alloc=%d", time.Since(start), iterations, replacements, maxLatency, host.bytes, host.writes, memory.HeapAlloc)
}
