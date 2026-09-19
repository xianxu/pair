package panebirth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// fakeClock stands still until slept on: Sleep advances it by exactly the
// requested duration, so a grace of N polls is N sleeps.
type fakeClock struct {
	now    time.Time
	sleeps int
}

func (c *fakeClock) Now() time.Time { return c.now }
func (c *fakeClock) Sleep(_ context.Context, d time.Duration) {
	c.sleeps++
	c.now = c.now.Add(d)
}

func bornOnRound(n int, calls *int) func() (bool, error) {
	return func() (bool, error) {
		*calls++
		return *calls >= n, nil
	}
}

func TestAwaitReturnsAtOnceWhenAlreadyBorn(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	calls := 0
	if err := Await(context.Background(), clock, time.Second, 5*time.Second, bornOnRound(1, &calls)); err != nil {
		t.Fatalf("Await = %v, want nil", err)
	}
	if clock.sleeps != 0 {
		t.Fatalf("sleeps = %d, want 0", clock.sleeps)
	}
}

func TestAwaitPollsUntilBirth(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	calls := 0
	if err := Await(context.Background(), clock, time.Second, 5*time.Second, bornOnRound(3, &calls)); err != nil {
		t.Fatalf("Await = %v, want nil", err)
	}
	if clock.sleeps != 2 || calls != 3 {
		t.Fatalf("sleeps = %d, observations = %d; want 2 and 3", clock.sleeps, calls)
	}
}

// The grace is a deadline, so passing it reads as one. Couch's registration
// context races this grace, and its diagnosis keys on DeadlineExceeded.
func TestAwaitReportsAPassedGraceAsADeadline(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	calls := 0
	err := Await(context.Background(), clock, time.Second, 5*time.Second, bornOnRound(1000, &calls))
	if !errors.Is(err, ErrUnborn) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Await = %v, want ErrUnborn and DeadlineExceeded", err)
	}
	if clock.sleeps != 5 {
		t.Fatalf("sleeps = %d, want 5 (grace / poll)", clock.sleeps)
	}
}

func TestAwaitStopsWhenTheContextEnds(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	born := func() (bool, error) {
		calls++
		if calls == 2 {
			cancel()
		}
		return false, nil
	}
	err := Await(ctx, clock, time.Second, time.Hour, born)
	if !errors.Is(err, ErrUnborn) || !errors.Is(err, context.Canceled) {
		t.Fatalf("Await = %v, want ErrUnborn and Canceled", err)
	}
	if calls != 2 {
		t.Fatalf("observations = %d, want none after the cancel was seen", calls)
	}
}

// A failed observation is "not yet", never birth, and it does not end the
// wait: a transient stat error after a good create must not read as death.
func TestAwaitTreatsAFailedObservationAsNotYet(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	statErr := errors.New("stat: interrupted")
	calls := 0
	born := func() (bool, error) {
		calls++
		if calls < 3 {
			return true, statErr // "true" alongside an error is not birth either
		}
		return true, nil
	}
	if err := Await(context.Background(), clock, time.Second, 5*time.Second, born); err != nil {
		t.Fatalf("Await = %v, want nil once an observation succeeds", err)
	}

	clock = &fakeClock{now: time.Unix(1_700_000_000, 0)}
	err := Await(context.Background(), clock, time.Second, 2*time.Second, func() (bool, error) { return false, statErr })
	if !errors.Is(err, ErrUnborn) || !errors.Is(err, statErr) {
		t.Fatalf("Await = %v, want ErrUnborn carrying the last observation error", err)
	}
}

func TestWallClockSleepReturnsWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	WallClock{}.Sleep(ctx, time.Hour)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Sleep on a cancelled context took %s", elapsed)
	}
}

func TestEvidenceIsTheAgentPaneSidecar(t *testing.T) {
	dir := t.TempDir()
	got, err := Evidence(dir, "work", "claude")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	want, err := paths.PaneChecked("claude")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Evidence = %q, want %q", got, want)
	}
	if _, err := Evidence(dir, "../escape", "claude"); err == nil {
		t.Fatal("Evidence accepted an invalid tag")
	}
}
