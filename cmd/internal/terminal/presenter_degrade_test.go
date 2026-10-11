package terminal

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// replayScreen feeds a parent's byte stream into a fresh emulator: what an
// outer terminal would show after parsing all of it.
func replayScreen(t *testing.T, g Geometry, stream []byte) Frame {
	t.Helper()
	e, err := NewEndpoint("replay", g, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Feed(stream, time.Now())
	f, err := e.Snapshot(time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func screenText(f Frame) string {
	var b strings.Builder
	for y := 0; y < f.Geometry.Rows; y++ {
		for x := 0; x < f.Geometry.Cols; x++ {
			c := f.Cells[y*f.Geometry.Cols+x]
			if c.Content == "" {
				b.WriteByte(' ')
			} else {
				b.WriteString(c.Content)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func presenterFailed(p *Presenter) bool {
	select {
	case <-p.Failed():
		return true
	default:
		return false
	}
}

// A parent that stops reading for longer than WriteTimeout stalls the
// presenter instead of failing it (#430): the paint returns, frames published
// during the stall are coalesced rather than queued, effects are dropped, and
// once the parent drains one full repaint leaves it showing exactly what a
// fresh presentation of the final frame would.
func TestPresenterStallPastWriteTimeoutDegradesThenRepaints(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, AnyMotion)
	host := Geometry{8, 5}
	chromeRow := chrome(8)
	if err := p.Select(context.Background(), e, host, chromeRow); err != nil {
		t.Fatal(err)
	}
	gate := make(chan struct{})
	for i := 0; i < 200; i++ {
		parent.Enqueue(ttyio.WriteStep{Block: gate})
	}
	e.Feed([]byte("\x1b[Hone"), time.Now())
	start := time.Now()
	if err := p.Present(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(context.Background()); err != nil {
		t.Fatalf("stalled paint failed: %v", err)
	}
	if elapsed := time.Since(start); elapsed < WriteTimeout {
		t.Fatalf("paint returned after %s, before WriteTimeout: stall not exercised", elapsed)
	}
	if presenterFailed(p) || p.View().State != Ready {
		t.Fatalf("stall failed the presenter: %+v", p.View())
	}
	if got := p.Stalls(); got != 1 {
		t.Fatalf("stalls = %d, want 1", got)
	}
	stalledBytes := len(parent.Bytes())
	for _, text := range []string{"\x1b[Htwo", "\x1b[Hthree"} {
		e.Feed([]byte(text), time.Now())
		if err := p.Present(context.Background(), e); err != nil {
			t.Fatal(err)
		}
		if err := p.Flush(context.Background()); err != nil {
			t.Fatalf("paint during stall: %v", err)
		}
	}
	if err := p.Copy(context.Background(), []byte("dropped")); err != nil {
		t.Fatalf("copy during stall: %v", err)
	}
	if got := len(parent.Bytes()); got != stalledBytes {
		t.Fatalf("parent accepted %d bytes while stalled", got-stalledBytes)
	}
	close(gate)
	waitFor(t, "repaint after drain", func() bool { return strings.Contains(string(parent.Bytes()), "three") })
	if presenterFailed(p) {
		t.Fatal("presenter failed after recovery")
	}
	recovered := parent.Bytes()[stalledBytes:]
	if !bytes.HasPrefix(recovered, []byte(stallProbe)) {
		t.Fatalf("recovery does not begin with the probe: %q", recovered)
	}
	if strings.Contains(string(recovered), "two") {
		t.Fatal("a frame published during the stall was queued, not coalesced")
	}
	if got := bytes.Count(parent.Bytes(), []byte(keyboardPush)); got != 1 {
		t.Fatalf("keyboard pushes = %d, want 1: recovery must not push the stack again", got)
	}

	fresh, freshParent, freshEndpoint, _ := presenterFixture(t, AnyMotion)
	freshEndpoint.Feed([]byte("\x1b[Hthree"), time.Now())
	if err := fresh.Select(context.Background(), freshEndpoint, host, chromeRow); err != nil {
		t.Fatal(err)
	}
	got, want := screenText(replayScreen(t, host, parent.Bytes())), screenText(replayScreen(t, host, freshParent.Bytes()))
	if got != want {
		t.Fatalf("recovered screen differs from a fresh presentation:\n%s\nwant:\n%s", got, want)
	}
	if err := p.Release(context.Background()); err != nil {
		t.Fatalf("release after recovery: %v", err)
	}
	if !bytes.HasSuffix(parent.Bytes(), parentReleaseControls(true)) {
		t.Fatal("release after recovery did not restore modes")
	}
}

// A terminal that is gone, not slow, still ends the presentation with its
// reason (#409's path).
func TestPresenterFailsWhenTheParentIsGone(t *testing.T) {
	for _, gone := range []error{syscall.EPIPE, syscall.EIO} {
		p, parent, e, _ := presenterFixture(t, AnyMotion)
		parent.Enqueue(ttyio.WriteStep{Limit: 3, Err: gone})
		err := p.Select(context.Background(), e, Geometry{8, 5}, chrome(8))
		if !errors.Is(err, gone) || !presenterFailed(p) {
			t.Fatalf("%v: paint = %v, failed = %v", gone, err, presenterFailed(p))
		}
		if reason, ok := ExitReason(p.Failure()); !ok || !strings.HasPrefix(reason, "terminal output failed after 3 of ") {
			t.Fatalf("%v: ExitReason = %q, %v", gone, reason, ok)
		}
		if p.Stalls() != 0 {
			t.Fatalf("%v: a gone terminal counted as a stall", gone)
		}
	}
}

// Over a real PTY: a host that stops reading well past WriteTimeout and then
// drains costs a stall, not the presenter, and it ends showing the frame.
func TestPresenterSurvivesARealStallPastWriteTimeout(t *testing.T) {
	parent := newStalledParent(t)
	p := NewPresenter(parent.out, AnyMotion)
	t.Cleanup(func() { p.Release(context.Background()) })
	e, _ := newEndpointTest(t, "selected")
	e.Feed([]byte("\x1b[Hframe"), time.Now())
	parent.fill(t)
	result := make(chan drained, 1)
	go func() {
		time.Sleep(WriteTimeout + 2*time.Second)
		result <- parent.drain(time.Second)
	}()
	if err := p.Select(context.Background(), e, Geometry{8, 5}, chrome(8)); err != nil {
		t.Fatalf("paint into a stalled host: %v", err)
	}
	got := (<-result).check(t)
	if presenterFailed(p) || p.Stalls() != 1 {
		t.Fatalf("failed = %v, stalls = %d", presenterFailed(p), p.Stalls())
	}
	if !strings.Contains(string(got), "frame") {
		t.Fatal("the frame never reached the host after it drained")
	}
}
