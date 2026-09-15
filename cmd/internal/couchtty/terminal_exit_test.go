package couchtty

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
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
