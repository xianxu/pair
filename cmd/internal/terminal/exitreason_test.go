package terminal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestExitReason(t *testing.T) {
	stalled := &WriteFailure{Op: "parent output", Accepted: 1024, Total: 17424, Err: context.DeadlineExceeded}
	for _, c := range []struct {
		name string
		err  error
		want string
		ok   bool
	}{
		{"stall", stalled, "terminal stopped accepting output for " + WriteTimeout.String() + " (wrote 1024 of 17424 bytes)", true},
		{"joined, as teardown reports it", errors.Join(fmt.Errorf("present: %w", stalled), errors.New("restore failed")), "terminal stopped accepting output for " + WriteTimeout.String(), true},
		{"broken pipe", &WriteFailure{Op: "parent output", Accepted: 0, Total: 9, Err: syscall.EPIPE}, "terminal output failed after 0 of 9 bytes: broken pipe", true},
		{"child input is not the terminal", &WriteFailure{Op: "child input", Err: context.DeadlineExceeded}, "", false},
		{"other error", errors.New("x"), "", false},
		{"nil", nil, "", false},
	} {
		got, ok := ExitReason(c.err)
		if ok != c.ok || !strings.HasPrefix(got, c.want) {
			t.Errorf("%s: ExitReason = %q, %v; want prefix %q, %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

// TestAStalledParentReadsAsTheTerminalStopping: a frame painted into a parent
// that stopped reading fails as a parent-output write past its deadline, which
// is exactly what ExitReason words as the terminal having stopped (pair#409).
// A caller deadline stands in for WriteTimeout; both end in DeadlineExceeded.
func TestAStalledParentReadsAsTheTerminalStopping(t *testing.T) {
	parent := newStalledParent(t)
	p := NewPresenter(parent.out, AnyMotion)
	e, _ := newEndpointTest(t, "selected")
	parent.fill(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := p.Select(ctx, e, Geometry{8, 5}, chrome(8))
	reason, ok := ExitReason(err)
	if !ok || !strings.HasPrefix(reason, "terminal stopped accepting output for ") {
		t.Fatalf("stalled paint = %v; ExitReason = %q, %v", err, reason, ok)
	}
}
