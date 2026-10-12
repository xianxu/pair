package testwait

import (
	"testing"
	"time"
)

func TestGiveUpAtLeavesMarginBeforeDeadline(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		remaining time.Duration
		want      time.Duration // after now
	}{
		{"ample time keeps the full margin", 10 * time.Minute, 10*time.Minute - margin},
		{"short time gives up halfway", 8 * time.Second, 4 * time.Second},
		{"past deadline gives up now-ish", -2 * time.Second, -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := giveUpAt(now.Add(tc.remaining), now).Sub(now); got != tc.want {
				t.Fatalf("give up %s after now, want %s", got, tc.want)
			}
		})
	}
}

func TestWaitReturnsOnceTheEventHolds(t *testing.T) {
	calls := 0
	if !Wait(t, func() bool { calls++; return calls == 3 }) {
		t.Fatal("Wait gave up on an event that arrived")
	}
	if calls != 3 {
		t.Fatalf("cond polled %d times, want 3", calls)
	}
	ready := make(chan struct{})
	close(ready)
	select {
	case <-ready:
	case <-After(t):
		t.Fatal("After fired before a ready event")
	}
}

func TestGiveUpPathsFailAndUnboundedNeverFires(t *testing.T) {
	calls := 0
	if waitUntil(time.Now().Add(-time.Second), true, func() bool { calls++; return false }) {
		t.Fatal("waitUntil held past its give-up time")
	}
	if calls != 1 {
		t.Fatalf("cond polled %d times past the give-up time, want 1", calls)
	}
	if ch := afterAt(time.Time{}, false); ch != nil {
		t.Fatal("After without a test deadline returned a channel that can fire")
	}
	select {
	case <-afterAt(time.Now().Add(-time.Second), true):
	case <-time.After(time.Minute):
		t.Fatal("After did not fire at a past give-up time")
	}
}
