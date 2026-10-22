package couchtty

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"golang.org/x/term"
)

// This subprocess reports the kernel's PTY winsize and paints its last child
// row. The agent names in the parent identify the two bounded attach scenarios;
// no installed agent, multiplexer, or live fleet participates in this test.
func TestConsoleStatusWrapNativeChild(t *testing.T) {
	if os.Getenv("PAIR_STATUS_WRAP_NATIVE_CHILD") != "1" {
		t.Skip("PTY subprocess helper")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	fmt.Print("\x1b[2J\x1b[HREADY")
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("\x1b[2J\x1b[%d;1H%dx%d:%s", rows, cols, rows, input.Text())
	}
}

func TestConsoleStatusWrapNativePTY(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			host := &couchSoakHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 8, Cols: 80}), em: vt.NewEmulator(80, 8)}
			drained := make(chan struct{})
			go func() { _, _ = io.Copy(io.Discard, host.em); close(drained) }()
			t.Cleanup(func() { _ = host.em.InputPipe().(io.Closer).Close(); <-drained; host.em.Close() })
			reader, input := io.Pipe()
			con := New(host, reader)
			done := make(chan int, 1)
			children := map[string]*ptychild.Child{}
			t.Cleanup(func() {
				con.Stop()
				_ = input.Close()
				select {
				case <-done:
				case <-testwait.After(t):
					t.Error("native console did not join")
				}
				for _, child := range children {
					_ = child.Close()
				}
				_ = reader.Close()
			})
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			attach := func(id, label string) *ptychild.Child {
				t.Helper()
				ready := make(chan struct{})
				child, err := ptychild.Start(ptychild.Options{
					Argv: []string{binary, "-test.run=^TestConsoleStatusWrapNativeChild$"},
					Env:  []string{"PAIR_STATUS_WRAP_NATIVE_CHILD=1"}, Size: ptychild.Size{Rows: 7, Cols: 80},
					Sink: func(ctx context.Context, b ptychild.OutputBatch) error {
						select {
						case <-ready:
						case <-ctx.Done():
							return ctx.Err()
						}
						return con.Deliver(ctx, id, b)
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				children[id] = child
				con.Attach(id, label, child)
				close(ready)
				return child
			}
			selected, background := attach("selected", agent+"-long"), attach("background", "other-long")
			go func() { done <- con.Run() }()
			text := func(f terminal.Frame) string {
				var b strings.Builder
				for _, c := range f.Cells {
					b.WriteString(c.Content)
				}
				return b.String()
			}
			snapshot := func(child *ptychild.Child) terminal.Frame {
				t.Helper()
				f, err := child.Endpoint().Snapshot(time.Now())
				if err != nil {
					t.Fatal(err)
				}
				return f
			}
			for _, child := range []*ptychild.Child{selected, background} {
				waitFor(t, "native child ready", func() bool { return strings.Contains(text(snapshot(child)), "READY") })
			}
			con.Switch("selected")
			for step, cols := range []uint16{80, 18, 80} {
				chromeRows := 1
				if cols == 18 {
					chromeRows = 2
				}
				host.SetSize(ptychild.Size{Rows: 8, Cols: cols})
				wantSize := ptychild.Size{Rows: uint16(8 - chromeRows), Cols: cols}
				waitFor(t, "selected and background resize", func() bool {
					return selected.Size() == wantSize && background.Size() == wantSize && con.chromeConfirmed()
				})
				state, err := con.presenter.ChromeState(context.Background())
				if err != nil || state.Rows != chromeRows {
					t.Fatalf("chrome admission: %+v %v", state, err)
				}
				for index, child := range []*ptychild.Child{selected, background} {
					token := fmt.Sprintf("s%dc%d", step, index)
					want := fmt.Sprintf("%dx%d:%s", cols, wantSize.Rows, token)
					if _, err := child.Write([]byte(token + "\n")); err != nil {
						t.Fatal(err)
					}
					waitFor(t, "kernel winsize and bottom output "+want, func() bool { return strings.Contains(text(snapshot(child)), want) })
					frame := snapshot(child)
					var bottom strings.Builder
					start := (frame.Geometry.Rows - 1) * frame.Geometry.Cols
					for _, cell := range frame.Cells[start:] {
						bottom.WriteString(cell.Content)
					}
					if !strings.Contains(bottom.String(), want) {
						t.Fatalf("last child row lost: %q, want %q", bottom.String(), want)
					}
					id := "selected"
					if index == 1 {
						id = "background"
					}
					con.Switch(id)
					waitFor(t, "visible bottom output "+want, func() bool {
						ctx, cancel := context.WithTimeout(context.Background(), testwait.Budget(t))
						defer cancel()
						if err := child.FlushOutput(ctx); err != nil {
							return false
						}
						if err := con.presenter.Flush(ctx); err != nil {
							return false
						}
						return con.presenter.View().Admitted == child.Endpoint().ID() && strings.Contains(host.text(), want)
					})
					host.mu.Lock()
					var physicalBottom strings.Builder
					for x := 0; x < int(cols); x++ {
						if cell := host.em.CellAt(x, int(wantSize.Rows)-1); cell != nil {
							physicalBottom.WriteString(cell.Content)
						}
					}
					host.mu.Unlock()
					if !strings.Contains(physicalBottom.String(), want) {
						t.Fatalf("physical bottom row %q, want %q", physicalBottom.String(), want)
					}
				}
				con.Switch("selected")
			}
		})
	}
}
