package termcmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"golang.org/x/term"
)

// This child is a real PTY peer. Its receipt includes exact input bytes; the
// parent test never substitutes a fake sink or reads an unacknowledged frame.
func TestTerminalSoakChild(t *testing.T) {
	if os.Getenv("PAIR_TERMINAL_SOAK_CHILD") != "1" {
		t.Skip("subprocess helper")
	}
	restore, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		os.Exit(2)
	}
	defer term.Restore(int(os.Stdin.Fd()), restore)
	fmt.Print("\x1b[?1006h\x1b[2J\x1b[HREADY")
	var line []byte
	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return
		}
		for _, b := range buf[:n] {
			if b != '\r' && b != '\n' {
				line = append(line, b)
				if len(line) > 4096 {
					os.Exit(3)
				}
				continue
			}
			if bytes.HasPrefix(line, []byte("app")) {
				fmt.Print("\x1b[?1002h")
			}
			if bytes.HasPrefix(line, []byte("shell")) {
				fmt.Print("\x1b[?1002l")
			}
			fmt.Printf("\x1b[?2026h\x1b[2J\x1b[H%s\x1b[2;1H界 é\x1b]2;soak\a\x1b[?2026l", hex.EncodeToString(line))
			line = line[:0]
		}
	}
}

type soakParent struct {
	mu            sync.Mutex
	screen        *vt.Emulator
	bytes, writes uint64
}

func (p *soakParent) WriteContext(ctx context.Context, data []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	n, err := p.screen.Write(data)
	p.bytes += uint64(n)
	p.writes++
	return n, err
}
func (p *soakParent) text() string { p.mu.Lock(); defer p.mu.Unlock(); return p.screen.String() }

// Default is a short deterministic integration run. Opt in with
// PAIR_TERMINAL_SOAK_DURATION=30m; this alone does not qualify native Zellij or
// Couch detach/reattach. Captures are bounded to current screens on failure.
func TestTerminalProductionSoak(t *testing.T) {
	duration := time.Duration(0)
	if value := os.Getenv("PAIR_TERMINAL_SOAK_DURATION"); value != "" {
		var err error
		duration, err = time.ParseDuration(value)
		if err != nil || duration <= 0 {
			t.Fatalf("invalid soak duration %q", value)
		}
	}
	parent := &soakParent{screen: vt.NewEmulator(80, 8)}
	t.Cleanup(func() { parent.screen.Close() })
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	m := newTerminalMux(binary, []string{"-test.run=^TestTerminalSoakChild$"}, parent, &fakeRuntime{})
	m.rows, m.cols = 8, 80
	m.shellEnv = []string{"PAIR_TERMINAL_SOAK_CHILD=1"}
	t.Cleanup(m.closeAll)
	for i := 0; i < 2; i++ {
		if err := m.newTab(); err != nil {
			t.Fatal(err)
		}
	}
	wait := func(child *ptychild.Child, marker string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			frame, err := child.Endpoint().Snapshot(time.Now())
			if err != nil {
				t.Fatal(err)
			}
			var text strings.Builder
			for _, cell := range frame.Cells {
				text.WriteString(cell.Content)
			}
			if strings.Contains(text.String(), marker) {
				if err := child.FlushOutput(context.Background()); err != nil {
					t.Fatal(err)
				}
				if err := m.presenter.Flush(context.Background()); err != nil {
					t.Fatal(err)
				}
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("missing receipt %q; bounded parent=%q", marker, parent.text())
	}
	for _, tab := range m.tabs {
		wait(tab.child, "READY")
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 8, Cols: 80})
	send := func(raw string) { pumpStdin(strings.NewReader(raw), m, m.rt, io.Discard) }
	start := time.Now()
	iterations := 0
	var maxLatency time.Duration
	for iterations < 12 || time.Since(start) < duration {
		m.nextTab()
		m.mu.Lock()
		tab := m.activeTabLocked()
		m.mu.Unlock()
		app := iterations%2 == 0
		mode := fmt.Sprintf("shell%d", iterations)
		if app {
			mode = fmt.Sprintf("app%d", iterations)
		}
		send(mode + "\r")
		wait(tab.child, hex.EncodeToString([]byte(mode)))
		mouse := "\x1b[<0;2;2M\x1b[<32;3;2M\x1b[<0;3;2m"
		receipt := fmt.Sprintf("r%d", iterations)
		want := receipt
		if app {
			want = mouse + receipt
		}
		sent := time.Now()
		send(mouse + receipt + "\r")
		wait(tab.child, hex.EncodeToString([]byte(want)))
		if elapsed := time.Since(sent); elapsed > maxLatency {
			maxLatency = elapsed
		}
		if !strings.Contains(parent.text(), hex.EncodeToString([]byte(want))) {
			t.Fatalf("acknowledged receipt not visible: %q", parent.text())
		}
		rows := uint16(8)
		if iterations%3 == 0 {
			rows = 6
		}
		parent.mu.Lock()
		parent.screen.Resize(80, int(rows))
		parent.mu.Unlock()
		host.SetSize(ptychild.Size{Rows: rows, Cols: 80})
		m.inheritSize(host)
		m.mu.Lock()
		failure := m.failure
		m.mu.Unlock()
		if failure != nil {
			t.Fatal(failure)
		}
		iterations++
	}
	m.closeAll()
	if m.closeErr != nil {
		t.Fatal(m.closeErr)
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	t.Logf("term soak seed=0 duration=%s iterations=%d parent_bytes=%d writes=%d max_input_visible=%s heap_alloc=%d goroutines=%d", time.Since(start), iterations, parent.bytes, parent.writes, maxLatency, stats.HeapAlloc, runtime.NumGoroutine())
}
