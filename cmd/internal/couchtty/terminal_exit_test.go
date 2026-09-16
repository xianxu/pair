package couchtty

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

type releaseFailureHost struct {
	*hostty.FakeHost
	fail   atomic.Bool
	closed atomic.Bool
}

func (h *releaseFailureHost) WriteContext(ctx context.Context, b []byte) (int, error) {
	if h.fail.Load() {
		return 0, errors.New("release write failed")
	}
	return h.FakeHost.WriteContext(ctx, b)
}
func (h *releaseFailureHost) Close() error { h.closed.Store(true); return h.FakeHost.Close() }

type restoredDiagnostic struct {
	host  *releaseFailureHost
	mu    sync.Mutex
	early bool
	text  strings.Builder
}

func (w *restoredDiagnostic) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.host.RawDepth() != 0 || !w.host.closed.Load() {
		w.early = true
	}
	return w.text.Write(p)
}

func TestConsoleReleaseFailureIsReportedAfterRestoreAndFailsRun(t *testing.T) {
	for _, duringRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "release", true: "runtime"}[duringRun], func(t *testing.T) {
			host := &releaseFailureHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 8, Cols: 50})}
			inputR, inputW := io.Pipe()
			defer inputW.Close()
			con := New(host, inputR)
			child := ptychild.NewFakeChild(nil)
			defer child.Close()
			con.Attach("a", "actor", child)
			diagnostic := &restoredDiagnostic{host: host}
			con.SetErrorWriter(diagnostic)
			done := make(chan int, 1)
			go func() { done <- con.Run() }()
			waitFor(t, "initial terminal presentation", func() bool { con.mu.Lock(); defer con.mu.Unlock(); return con.framePainted })
			if err := con.presenter.Flush(context.Background()); err != nil {
				t.Fatal(err)
			}
			if duringRun {
				con.terminalError(errors.New("runtime presentation failed"))
			} else {
				host.fail.Store(true)
				con.Stop()
			}
			select {
			case code := <-done:
				if code != 1 {
					t.Errorf("Run=%d, want failure", code)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Run did not join")
			}
			diagnostic.mu.Lock()
			defer diagnostic.mu.Unlock()
			if diagnostic.early {
				t.Error("diagnostic wrote while presenter or raw terminal still owned host")
			}
			want := "release write failed"
			if duringRun {
				want = "runtime presentation failed"
			}
			if !strings.Contains(diagnostic.text.String(), want) {
				t.Errorf("diagnostic %q lacks %q", diagnostic.text.String(), want)
			}
		})
	}
}

func TestShutdownCancellationRequiresOnlyCanceledLeaves(t *testing.T) {
	broken := errors.New("physical write failed")
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"cancel", context.Canceled, true},
		{"wrapped", fmt.Errorf("write: %w", context.Canceled), true},
		{"joined cancellations", errors.Join(context.Canceled, fmt.Errorf("write: %w", context.Canceled)), true},
		{"deadline", context.DeadlineExceeded, false},
		{"mixed", errors.Join(context.Canceled, broken), false},
		{"nested mixed", fmt.Errorf("write: %w", errors.Join(context.Canceled, broken)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := shutdownCancellation(tc.err); got != tc.want {
				t.Fatalf("cancellation-only=%v want%v for %v", got, tc.want, tc.err)
			}
		})
	}
}

type stoppedPaintHost struct {
	*hostty.FakeHost
	armed   atomic.Bool
	entered chan struct{}
	extra   error
	partial bool
}

func (h *stoppedPaintHost) WriteContext(ctx context.Context, b []byte) (int, error) {
	if h.armed.CompareAndSwap(true, false) {
		accepted := 0
		if h.partial {
			// End inside the first control: release must abort the prefix before
			// resetting modes, margins, rendition and cursor visibility.
			accepted, _ = h.FakeHost.WriteContext(ctx, b[:3])
		}
		close(h.entered)
		<-ctx.Done()
		return accepted, errors.Join(ctx.Err(), h.extra)
	}
	return h.FakeHost.WriteContext(ctx, b)
}

func TestConsoleStopDuringPaintClassifiesJoinedFailure(t *testing.T) {
	for _, partial := range []bool{false, true} {
		for _, realFailure := range []bool{false, true} {
			t.Run(fmt.Sprintf("partial=%t/host-error=%t", partial, realFailure), func(t *testing.T) {
				h := &stoppedPaintHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 8, Cols: 50}), entered: make(chan struct{}), partial: partial}
				if realFailure {
					h.extra = errors.New("physical write failed")
				}
				r, w := io.Pipe()
				defer w.Close()
				c := New(h, r)
				child := ptychild.NewFakeChild(nil)
				c.Attach("a", "actor", child)
				var diagnostic bytes.Buffer
				c.SetErrorWriter(&diagnostic)
				done := make(chan int, 1)
				go func() { done <- c.Run() }()
				t.Cleanup(func() { c.Stop(); w.Close(); child.Close() })
				waitFor(t, "console startup", func() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.started })
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := c.runTerminalCommand(ctx, func() error { return nil }); err != nil {
					t.Fatal(err)
				}
				h.armed.Store(true)
				commandDone := make(chan error, 1)
				go func() {
					commandDone <- c.runTerminalCommand(ctx, func() error {
						cells, err := terminal.StyledRows("changed", 50, 1)
						if err != nil {
							return err
						}
						return c.presenter.UpdateChrome(c.lifetime, cells)
					})
				}()
				select {
				case <-h.entered:
				case <-ctx.Done():
					t.Fatal("paint did not enter host")
				}
				c.Stop()
				select {
				case <-commandDone:
				case <-ctx.Done():
					t.Fatal("paint command did not return")
				}
				select {
				case code := <-done:
					want := 0
					if realFailure {
						want = 1
					}
					if code != want {
						t.Errorf("Run=%d want%d; diagnostics=%q presenter=%v", code, want, diagnostic.String(), c.presenter.Failure())
					}
					if realFailure && !strings.Contains(diagnostic.String(), "physical write failed") {
						t.Errorf("lost host failure: %q", diagnostic.String())
					}
					if !realFailure && diagnostic.Len() != 0 {
						t.Errorf("shutdown reported failure: %q", diagnostic.String())
					}
					if h.RawDepth() != 0 || !h.Closed() {
						t.Errorf("host cleanup raw=%d closed=%v", h.RawDepth(), h.Closed())
					}
					wire := h.Written()
					if !strings.Contains(wire, "\x18\x1b\\") || !strings.HasSuffix(wire, "\x1b[?6l\x1b[r\x1b[?7h\x1b[0m\x1b]8;;\x1b\\\x1b[0 q\x1b[?25h") {
						t.Errorf("missing parent repair after interrupted paint: %q", wire)
					}
				case <-ctx.Done():
					t.Fatal("Run did not join")
				}
			})
		}
	}

}

func TestConsoleLiveCancellationFailureRemainsLatched(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 8, Cols: 50}), nil)
	var diagnostic bytes.Buffer
	c.SetErrorWriter(&diagnostic)
	c.terminalError(context.Canceled)
	if err := c.teardown(func() error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("lost live cancellation failure: %v", err)
	}
	if !strings.Contains(diagnostic.String(), "context canceled") {
		t.Fatal("missing live failure diagnostic")
	}
}
