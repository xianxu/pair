package terminal

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"golang.org/x/term"
)

// stalledParent is a presenter over a real PTY whose master plays the host
// terminal: output backs up until the test drains it (#383).
type stalledParent struct {
	out, host *ttyio.File
}

func newStalledParent(t *testing.T) stalledParent {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close(); slave.Close() })
	raw, err := term.MakeRaw(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Restore(int(slave.Fd()), raw) })
	out, err := ttyio.NewFile(nil, slave, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { out.Close() })
	host, err := ttyio.NewFile(master, master, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { host.Close() })
	return stalledParent{out, host}
}

// fill backs the host up until a write no longer progresses.
func (s stalledParent) fill(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := s.out.WriteContext(ctx, make([]byte, 1<<20)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("host did not stall: %v", err)
	}
}

// drain collects host output until it is quiet. It returns the bytes received
// even on error, and never touches t, so it is safe on a goroutine.
func (s stalledParent) drain(quiet time.Duration) drained {
	var got []byte
	buf := make([]byte, 64<<10)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), quiet)
		n, err := s.host.ReadContext(ctx, buf)
		cancel()
		got = append(got, buf[:n]...)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				err = nil
			}
			return drained{got, err}
		}
	}
}

type drained struct {
	bytes []byte
	err   error
}

func (d drained) check(t *testing.T) []byte {
	t.Helper()
	if d.err != nil {
		t.Fatalf("drain host: %v", d.err)
	}
	return d.bytes
}

func chrome(cols int) []Cell {
	cells := make([]Cell, cols)
	for i := range cells {
		cells[i] = uv.Cell{Content: string(rune('a' + i%23)), Width: 1}
	}
	return cells
}

// A host that stops reading for longer than two seconds but resumes inside
// WriteTimeout costs a delayed frame, not the presenter: the paint completes
// and release delivers the mode reset exactly once.
func TestPresenterSurvivesTransientParentStallAndRestoresModes(t *testing.T) {
	parent := newStalledParent(t)
	p := NewPresenter(parent.out, AnyMotion)
	t.Cleanup(func() { p.Release(context.Background()) })
	e, _ := newEndpointTest(t, "selected")
	parent.fill(t)
	result := make(chan drained, 1)
	go func() {
		time.Sleep(2500 * time.Millisecond)
		result <- parent.drain(500 * time.Millisecond)
	}()
	defer func() {
		if result != nil {
			<-result
		}
	}()
	start := time.Now()
	if err := p.Select(context.Background(), e, Geometry{8, 5}, chrome(8)); err != nil {
		t.Fatalf("paint failed after %s: %v", time.Since(start), err)
	}
	if elapsed := time.Since(start); elapsed < 2500*time.Millisecond {
		t.Fatalf("paint finished in %s, before the host resumed: stall not exercised", elapsed)
	}
	if err := p.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := (<-result).check(t)
	result = nil
	reset := parentReleaseControls(true)
	if !bytes.HasSuffix(got, reset) || bytes.Count(got, reset) != 1 {
		t.Fatalf("host output does not end with exactly one mode reset (%d bytes)", len(got))
	}
}

// Past WriteTimeout the presenter fails; when the host later resumes, release
// still restores its modes with a budget of its own.
func TestPresenterRestoresModesWhenHostResumesAfterFailure(t *testing.T) {
	parent := newStalledParent(t)
	p := NewPresenter(parent.out, AnyMotion)
	e, _ := newEndpointTest(t, "selected")
	parent.fill(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := p.Select(ctx, e, Geometry{8, 5}, chrome(8)); err == nil {
		t.Fatal("paint into a stalled host succeeded")
	}
	result := make(chan drained, 1)
	go func() { result <- parent.drain(500 * time.Millisecond) }()
	err := p.Release(context.Background())
	got := (<-result).check(t)
	if err != nil {
		t.Fatalf("release after host resumed: %v", err)
	}
	if !bytes.HasSuffix(got, parentReleaseControls(false)) {
		t.Fatalf("host output does not end with the mode reset (%d bytes)", len(got))
	}
}

// A host that never resumes leaves its modes enabled; release says so instead
// of reporting a restoration that no byte delivered.
func TestPresenterReleaseReportsModesNotRestoredWhenHostStaysStalled(t *testing.T) {
	parent := newStalledParent(t)
	p := NewPresenter(parent.out, AnyMotion)
	e, _ := newEndpointTest(t, "selected")
	parent.fill(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := p.Select(ctx, e, Geometry{8, 5}, chrome(8)); err == nil {
		t.Fatal("paint into a stalled host succeeded")
	}
	releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer releaseCancel()
	err := p.Release(releaseCtx)
	var failure *WriteFailure
	if !errors.Is(err, ErrModesNotRestored) || !errors.As(err, &failure) {
		t.Fatalf("release = %v, want ErrModesNotRestored with its write failure", err)
	}
	if failure.Accepted != 0 || failure.Total != len(parentReleaseControls(false)) {
		t.Fatalf("release write accepted %d/%d, want 0/%d", failure.Accepted, failure.Total, len(parentReleaseControls(false)))
	}
}

// Release's drag cancellation stays bounded when the caller passes no deadline
// and the child stops reading: the child's InputWriter owns that budget.
func TestPresenterReleaseBoundsDragCancellationWhenChildStalls(t *testing.T) {
	p, _, a, aw := presenterFixture(t, AnyMotion)
	a.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, a)
	if err := p.Input(context.Background(), uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	if err := a.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	aw.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- p.Release(context.Background()) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("release reported a drag cancellation the child never received")
		}
		if elapsed := time.Since(start); elapsed > WriteTimeout+time.Second {
			t.Fatalf("release took %s", elapsed)
		}
	case <-time.After(WriteTimeout + 2*time.Second):
		t.Fatal("release did not bound drag cancellation")
	}
}
