// Package testwait bounds a test's wait for an event by the test's own
// deadline, not by a guess at how fast the event arrives (#479). A short
// window (250ms) is a performance assertion that a loaded -race suite misses;
// a lost event still fails here, margin before the binary's timeout, with a
// message naming what the test waited for instead of a goroutine dump.
package testwait

import (
	"testing"
	"time"
)

// margin is how long before the binary's deadline a wait gives up, so the
// failure is the test's own message and the remaining tests still report.
const margin = 10 * time.Second

// After returns a channel that fires when a test should stop waiting: margin
// before its deadline (half the remaining time when less than 2×margin is
// left). With no deadline (-timeout 0) it never fires. Use it as the guard
// case of a select on the awaited event.
func After(t *testing.T) <-chan time.Time {
	return afterAt(giveUp(t))
}

func afterAt(at time.Time, bounded bool) <-chan time.Time {
	if !bounded {
		return nil
	}
	return time.After(time.Until(at))
}

// Wait polls cond until it holds (true) or the test should stop waiting
// (false), for callers that report their own failure detail.
func Wait(t *testing.T, cond func() bool) bool {
	t.Helper()
	at, bounded := giveUp(t)
	return waitUntil(at, bounded, cond)
}

func waitUntil(at time.Time, bounded bool, cond func() bool) bool {
	for !cond() {
		if bounded && !time.Now().Before(at) {
			return false
		}
		time.Sleep(2 * time.Millisecond)
	}
	return true
}

// Until is Wait that fails the test naming what it waited for.
func Until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	if !Wait(t, cond) {
		t.Fatalf("timed out waiting for %s", what)
	}
}

func giveUp(t *testing.T) (time.Time, bool) {
	deadline, ok := t.Deadline()
	if !ok {
		return time.Time{}, false
	}
	return giveUpAt(deadline, time.Now()), true
}

func giveUpAt(deadline, now time.Time) time.Time {
	m := margin
	if half := deadline.Sub(now) / 2; half < m {
		m = half
	}
	return deadline.Add(-m)
}
