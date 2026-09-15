package couchtty

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func requireDisposed(t *testing.T, child *ptychild.Child) {
	t.Helper()
	if _, err := child.Endpoint().Snapshot(time.Now()); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("removed endpoint is still readable: %v", err)
	}
}

func TestConsoleDisposesExitedOriginsAcrossRepeatedReplacement(t *testing.T) {
	f := newFixture(t, 12, 70)
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("replacement-%d", i)
		child := ptychild.NewFakeChild(nil)
		t.Cleanup(func() { _ = child.Close() })
		child.SetSink(func(ctx context.Context, b ptychild.OutputBatch) error { return f.con.Deliver(ctx, id, b) })
		f.con.attachThreadActor(id, couchcore.ActorID(id), couchcore.ThreadAddress{RepoScope: "repo", Tag: "same-thread"}, "/same-thread", "background", child)
		child.Feed([]byte("last publication"))
		child.Exit(0)
		waitFor(t, "exited pane removed", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); _, ok := f.con.panes[id]; return !ok })
		if err := f.con.runTerminalCommand(context.Background(), func() error { return nil }); err != nil {
			t.Fatal(err)
		}
		requireDisposed(t, child)
	}
}

func TestConsoleDisposesLastChildAfterPresentingItsFinalFrame(t *testing.T) {
	f := newFixture(t, 12, 70)
	f.child.Feed([]byte("LAST-FRAME-BEFORE-DISPOSAL"))
	f.child.Exit(0)
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("last child exit did not finish")
	}
	if !strings.Contains(f.host.Written(), "LAST-FRAME-BEFORE-DISPOSAL") {
		t.Fatal("disposal lost final frame")
	}
	requireDisposed(t, f.child)
}

func TestConsoleReleaseDisposesOwnedChildrenWithPendingDelivery(t *testing.T) {
	host := newVTHost(12, 70)
	inputR, inputW := io.Pipe()
	defer inputW.Close()
	con := New(host, inputR)
	child := ptychild.NewFakeChild(nil)
	defer child.Close()
	child.SetSink(func(ctx context.Context, b ptychild.OutputBatch) error { return con.Deliver(ctx, "a", b) })
	con.Attach("a", "actor", child)
	child.Feed([]byte("delivery before Run"))
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	waitFor(t, "console presentation", func() bool { con.mu.Lock(); defer con.mu.Unlock(); return con.framePainted })
	con.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pending delivery blocked teardown")
	}
	requireDisposed(t, child)
}

func TestConsoleCompatibilityAttachRefusalDisposesOnlyUnacceptedChild(t *testing.T) {
	f := newFixture(t, 12, 70)
	rejected := ptychild.NewFakeChild(nil)
	defer rejected.Close()
	f.con.Attach("c1", "duplicate", rejected)
	requireDisposed(t, rejected)
	// A repeated reference to a child already owned by this console must never
	// dispose that accepted lifetime when the duplicate handle is refused.
	f.con.Attach("c1", "duplicate", f.child)
	if _, err := f.child.Endpoint().Snapshot(time.Now()); err != nil {
		t.Fatalf("duplicate attach closed accepted endpoint: %v", err)
	}
}

type refusingOwnedHost struct{ *releaseFailureHost }

func (h *refusingOwnedHost) MakeRaw() (func() error, error) {
	return nil, errors.New("raw acquisition refused")
}
func TestConsoleFailedTerminalAcquisitionDisposesAcceptedChildren(t *testing.T) {
	host := &refusingOwnedHost{releaseFailureHost: &releaseFailureHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 12, Cols: 70})}}
	con := New(host, strings.NewReader(""))
	child := ptychild.NewFakeChild(nil)
	defer child.Close()
	con.Attach("a", "actor", child)
	diagnostic := &restoredDiagnostic{host: host.releaseFailureHost}
	con.SetErrorWriter(diagnostic)
	if code := con.Run(); code != 1 {
		t.Fatalf("Run=%d", code)
	}
	requireDisposed(t, child)
	if host.Written() != "" {
		t.Fatalf("failed raw acquisition changed unowned terminal: %q", host.Written())
	}
	diagnostic.mu.Lock()
	defer diagnostic.mu.Unlock()
	if diagnostic.early {
		t.Fatal("failed acquisition reported before closing host")
	}
}
