package launcher

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

func TestProductionTerminalStartupHonorsCallerDeadlinePastBirthProbe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		caller, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		w := newHandoffWorld()
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			<-w.started
			// Observe this timer in testing/synctest virtual time; no wall-clock guess.
			<-time.After(12 * time.Second)
			w.admission <- nil
			<-w.released
			w.exit <- HandoffExit{Code: 0}
		}()
		_, err := runTerminalHandoffAttempt(caller, nil, caller, w, true)
		<-finished
		if err != nil {
			t.Fatalf("caller has 18s left but readiness refused: %v", err)
		}
	})
}

type startupProbeRuntime struct {
	*fakeRuntime
	query context.Context
}

func (r *startupProbeRuntime) SessionLivenessContext(ctx context.Context) ([]Session, error) {
	r.query = ctx
	<-ctx.Done()
	return nil, ctx.Err()
}
func TestBirthDiagnosticProbeUsesRemainingStartupBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r := &startupProbeRuntime{fakeRuntime: newFakeRuntime()}
		aborted := false
		got := watchBirth(ctx, r, panebirth.WallClock{}, "/fixture/no-pane", watchedSession, birthBound, func() { aborted = true })
		if got != birthStopped || aborted || r.query == nil {
			t.Fatalf("verdict=%v aborted=%v query=%v", got, aborted, r.query)
		}
	})
}

func TestProductionTerminalStartupNoDeadlineUsesExistingCeiling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		at := time.Now()
		_, err := runTerminalHandoffAttempt(context.Background(), nil, context.Background(), newHandoffWorld(), true)
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(at) != startupbudget.Default {
			t.Fatalf("elapsed=%s error=%v", time.Since(at), err)
		}
	})
}

func TestProductionTerminalStartupEarlierCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		selected, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		life, stop := context.WithCancel(context.Background())
		defer stop()
		w := newHandoffWorld()
		// Observe this timer in testing/synctest virtual time; no wall-clock guess.
		go func() { <-w.started; <-time.After(2 * time.Second); stop() }()
		at := time.Now()
		_, err := runTerminalHandoffAttempt(life, selected, selected, w, true)
		if !errors.Is(err, context.Canceled) || time.Since(at) != 2*time.Second {
			t.Fatalf("elapsed=%s error=%v", time.Since(at), err)
		}
	})
}

func TestProductionTerminalStartupAdmissionDoesNotOwnInteractiveLifetime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		selected, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		w := newHandoffWorld()
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			<-w.started
			// Observe this timer in testing/synctest virtual time; no wall-clock guess.
			<-time.After(12 * time.Second)
			w.admission <- nil
			<-w.released
			// Observe this timer in testing/synctest virtual time; no wall-clock guess.
			<-time.After(30 * time.Second)
			w.exit <- HandoffExit{Code: 0}
		}()
		at := time.Now()
		_, err := runTerminalHandoffAttempt(context.Background(), selected, selected, w, true)
		<-finished
		if err != nil || time.Since(at) != 42*time.Second {
			t.Fatalf("admitted interactive lifetime truncated: elapsed=%s error=%v", time.Since(at), err)
		}
	})
}

type polledStartupWorld struct {
	*handoffWorld
	born           time.Time
	polls, queries int
}

type expiringStartupQueryWorld struct {
	*handoffWorld
	readyAt time.Time
	queries int
}

func (w *expiringStartupQueryWorld) Admit(ctx context.Context) error {
	return admitTerminalAfterReady(ctx, ctx, func(context.Context) error {
		if time.Now().Before(w.readyAt) {
			return errors.New("not ready")
		}
		return nil
	}, func(query context.Context) error { w.queries++; <-query.Done(); return query.Err() })
}
func TestProductionTerminalStartupCapsPostbirthObservationAtRemainingDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		at := time.Now()
		selected, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		w := &expiringStartupQueryWorld{handoffWorld: newHandoffWorld(), readyAt: at.Add(29 * time.Second)}
		_, err := runTerminalHandoffAttempt(context.Background(), selected, selected, w, true)
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(at) != 30*time.Second || w.queries != 1 {
			t.Fatalf("elapsed=%s queries=%d error=%v", time.Since(at), w.queries, err)
		}
	})
}

func TestProductionTerminalStartupRechecksCancellationAfterReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		selected, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		w := cancelOnAdmissionWorld{handoffWorld: newHandoffWorld(), cancel: cancel}
		_, err := runTerminalHandoffAttempt(context.Background(), selected, selected, w, true)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled admission accepted: %v", err)
		}
	})
}

func (w *polledStartupWorld) Admit(ctx context.Context) error {
	return admitTerminalAfterReady(ctx, ctx, func(context.Context) error {
		w.polls++
		if time.Now().Before(w.born) {
			return errors.New("not ready")
		}
		return nil
	}, func(context.Context) error { w.queries++; return nil })
}
func TestProductionTerminalStartupPollingCountsAtNAnd2N(t *testing.T) {
	for _, n := range []int{50, 100} {
		t.Run(time.Duration(n).String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := &polledStartupWorld{handoffWorld: newHandoffWorld(), born: time.Now().Add(time.Duration(n) * birthPoll)}
				selected, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				query, stop := context.WithTimeout(selected, 5*time.Second)
				defer stop()
				go func() { <-w.released; w.exit <- HandoffExit{Code: 0} }()
				_, err := runTerminalHandoffAttempt(context.Background(), selected, query, w, true)
				if err != nil || w.polls > n+2 || w.polls < n || w.queries != 1 {
					t.Fatalf("n=%d polls=%d queries=%d error=%v", n, w.polls, w.queries, err)
				}
			})
		})
	}
}

func TestProductionTerminalStartupAbsentReadinessSpendsCallerBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := time.Now()
		caller, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := runTerminalHandoffAttempt(caller, nil, caller, newHandoffWorld(), true)
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) != 30*time.Second {
			t.Fatalf("absent readiness: elapsed=%s error=%v", time.Since(started), err)
		}
	})
}
