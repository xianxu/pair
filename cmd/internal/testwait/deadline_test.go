package testwait

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDeadlineOnCue(t *testing.T) {
	type key struct{}
	parent := context.WithValue(context.Background(), key{}, "value")
	before := time.Now()
	ctx, expire, cancel := DeadlineOnCue(parent, time.Minute)
	defer cancel()
	if ctx.Value(key{}) != "value" {
		t.Fatal("lost parent value")
	}
	at, ok := ctx.Deadline()
	if !ok || at.Before(before.Add(time.Minute)) || at.After(time.Now().Add(time.Minute)) {
		t.Fatalf("deadline=%v, %v", at, ok)
	}
	select {
	case <-ctx.Done():
		t.Fatal("expired before cue")
	default:
	}
	child, stop := context.WithTimeout(ctx, 2*time.Minute)
	defer stop()
	expire()
	<-child.Done()
	if ctx.Err() != context.DeadlineExceeded || child.Err() != context.DeadlineExceeded || context.Cause(ctx) != context.DeadlineExceeded {
		t.Fatalf("expiry: ctx=%v child=%v cause=%v", ctx.Err(), child.Err(), context.Cause(ctx))
	}
	cancel()
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatal("cancel replaced expiry")
	}
}

func TestDeadlineOnCuePreservesFirstTerminalCause(t *testing.T) {
	for _, mode := range []string{"parent", "cancel", "expire"} {
		t.Run(mode, func(t *testing.T) {
			parent, stop := context.WithCancelCause(context.Background())
			defer stop(nil)
			ctx, expire, cancel := DeadlineOnCue(parent, time.Minute)
			defer cancel()
			cause := errors.New("parent cause")
			wantErr, wantCause := error(context.Canceled), error(context.Canceled)
			switch mode {
			case "parent":
				stop(cause)
				wantCause = cause
			case "cancel":
				cancel()
			case "expire":
				expire()
				wantErr = context.DeadlineExceeded
				wantCause = wantErr
			}
			<-ctx.Done()
			expire()
			cancel()
			stop(cause)
			if ctx.Err() != wantErr || context.Cause(ctx) != wantCause {
				t.Fatalf("err=%v cause=%v", ctx.Err(), context.Cause(ctx))
			}
		})
	}
}

func TestDeadlineOnCueParentAlreadyDoneAndRaces(t *testing.T) {
	parent, stop := context.WithCancelCause(context.Background())
	stop(context.DeadlineExceeded)
	ctx, expire, cancel := DeadlineOnCue(parent, time.Minute)
	defer cancel()
	if ctx.Err() != context.Canceled || context.Cause(ctx) != context.DeadlineExceeded {
		t.Fatalf("already canceled: %v, %v", ctx.Err(), context.Cause(ctx))
	}
	expire()
	for i := 0; i < 100; i++ {
		parent, stop := context.WithCancel(context.Background())
		ctx, expire, cancel := DeadlineOnCue(parent, time.Minute)
		var wg sync.WaitGroup
		for _, f := range []func(){stop, expire, cancel} {
			wg.Add(1)
			go func(f func()) { defer wg.Done(); f() }(f)
		}
		wg.Wait()
		<-ctx.Done()
		first := ctx.Err()
		if first != context.Canceled && first != context.DeadlineExceeded {
			t.Fatalf("terminal err=%v", first)
		}
		expire()
		cancel()
		if ctx.Err() != first {
			t.Fatal("terminal cause changed")
		}
	}
}

// notificationParent records registration ownership without a goroutine or clock.
// Its nonnil Done makes context use its AfterFunc registration seam.
type notificationParent struct {
	context.Context
	done   chan struct{}
	active int
}

func (p *notificationParent) Done() <-chan struct{} { return p.done }
func (p *notificationParent) AfterFunc(func()) func() bool {
	p.active++
	var once sync.Once
	return func() bool {
		stopped := false
		once.Do(func() { p.active--; stopped = true })
		return stopped
	}
}

func TestDeadlineOnCueReleasesParentNotification(t *testing.T) {
	for _, mode := range []string{"cancel", "expire"} {
		t.Run(mode, func(t *testing.T) {
			parent := &notificationParent{Context: context.Background(), done: make(chan struct{})}
			ctx, expire, cancel := DeadlineOnCue(parent, time.Minute)
			defer cancel()
			if parent.active != 1 {
				t.Fatalf("registrations=%d", parent.active)
			}
			if mode == "cancel" {
				cancel()
			} else {
				expire()
			}
			<-ctx.Done()
			if parent.active != 0 {
				t.Fatalf("retained parent registration=%d", parent.active)
			}
		})
	}
}

func TestDeadlineOnCuePreservesEarlierParentDeadline(t *testing.T) {
	parent, expireParent, cancelParent := DeadlineOnCue(context.Background(), time.Minute)
	defer cancelParent()
	ctx, _, cancel := DeadlineOnCue(parent, 2*time.Minute)
	defer cancel()
	want, _ := parent.Deadline()
	if got, ok := ctx.Deadline(); !ok || !got.Equal(want) {
		t.Fatalf("deadline=%v want=%v", got, want)
	}
	expireParent()
	<-ctx.Done()
	if ctx.Err() != context.DeadlineExceeded || context.Cause(ctx) != context.DeadlineExceeded {
		t.Fatalf("err=%v cause=%v", ctx.Err(), context.Cause(ctx))
	}
}

func TestDeadlineOnCueAlreadyCanceledParentWinsLaterCue(t *testing.T) {
	for i := 0; i < 100; i++ {
		parent, stop := context.WithCancelCause(context.Background())
		ctx, expire, cancel := DeadlineOnCue(parent, time.Minute)
		cause := errors.New("parent stopped first")
		stop(cause)
		expire()
		cancel()
		if ctx.Err() != context.Canceled || context.Cause(ctx) != cause {
			t.Fatalf("later cue replaced parent: err=%v cause=%v", ctx.Err(), context.Cause(ctx))
		}
	}
}
