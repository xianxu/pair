package launcher

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type budgetHandoffWorld struct {
	*handoffWorld
	acquireCtx, checkCtx, startCtx context.Context
	onCheck                        func()
	admitContext                   chan context.Context
}

type cancelOnAdmissionWorld struct {
	*handoffWorld
	cancel context.CancelFunc
}

func (w cancelOnAdmissionWorld) Admit(context.Context) error {
	w.cancel()
	return nil
}

func TestTerminalHandoffWarmRechecksBudgetAfterAdmission(t *testing.T) {
	query, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := cancelOnAdmissionWorld{handoffWorld: newHandoffWorld(), cancel: cancel}
	// Correct refusal cancels and reaps the client before releasing. The
	// release barrier also lets the old acceptance path finish for a RED result.
	finished := make(chan struct{})
	go func() { <-w.released; w.exit <- HandoffExit{Code: 0}; close(finished) }()
	_, err := coordinateTerminalHandoff(context.Background(), query, w, time.Minute, false)
	<-finished
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expired admission accepted: %v", err)
	}
}

func (w *budgetHandoffWorld) Acquire(ctx context.Context) (io.Closer, error) {
	w.acquireCtx = ctx
	return w.handoffWorld.Acquire(ctx)
}
func (w *budgetHandoffWorld) Check(ctx context.Context) error {
	w.checkCtx = ctx
	if w.onCheck != nil {
		w.onCheck()
	}
	return w.handoffWorld.Check(ctx)
}
func (w *budgetHandoffWorld) Start(ctx context.Context) (HandoffClient, error) {
	w.startCtx = ctx
	return w.handoffWorld.Start(ctx)
}
func (w *budgetHandoffWorld) Admit(ctx context.Context) error {
	w.admitContext <- ctx
	return w.handoffWorld.Admit(ctx)
}

func TestTerminalHandoffWarmSharesQueryBudgetAndPreservesClientLifetime(t *testing.T) {
	w := &budgetHandoffWorld{handoffWorld: newHandoffWorld(), admitContext: make(chan context.Context, 1)}
	query, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := coordinateTerminalHandoff(context.Background(), query, w, time.Minute, false)
		done <- err
	}()
	admit := <-w.admitContext
	want, _ := query.Deadline()
	for _, ctx := range []context.Context{w.acquireCtx, w.checkCtx, admit} {
		if got, ok := ctx.Deadline(); !ok || !got.Equal(want) {
			t.Errorf("query deadline=%v want=%v", got, want)
		}
	}
	w.admission <- nil
	<-w.released
	cancel()
	if w.startCtx.Err() != nil {
		t.Errorf("query cancellation canceled interactive client: %v", w.startCtx.Err())
	}
	if _, bounded := w.startCtx.Deadline(); bounded {
		t.Error("interactive client inherited query deadline")
	}
	w.exit <- HandoffExit{Code: 0}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestTerminalHandoffQueryCancellationDuringCheckPreventsStart(t *testing.T) {
	query, cancel := context.WithCancel(context.Background())
	w := &budgetHandoffWorld{handoffWorld: newHandoffWorld(), onCheck: cancel}
	_, err := coordinateTerminalHandoff(context.Background(), query, w, time.Minute, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	select {
	case <-w.started:
		t.Fatal("started after admission canceled")
	default:
	}
	<-w.released
}

func TestTerminalHandoffColdReadinessDoesNotInheritPrecreateBudget(t *testing.T) {
	w := &budgetHandoffWorld{handoffWorld: newHandoffWorld(), admitContext: make(chan context.Context, 1)}
	query, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := coordinateTerminalHandoff(context.Background(), query, w, time.Minute, true)
		done <- err
	}()
	admit := <-w.admitContext
	cancel()
	if admit.Err() != nil {
		t.Errorf("precreate cancellation canceled readiness: %v", admit.Err())
	}
	if deadline, ok := admit.Deadline(); !ok || time.Until(deadline) < 30*time.Second {
		t.Errorf("birth readiness bound changed: %v", deadline)
	}
	w.admission <- nil
	<-w.released
	w.exit <- HandoffExit{Code: 0}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestTerminalHandoffPostbirthQueryHasFreshCallerCappedBudget(t *testing.T) {
	for _, capped := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "caller-capped"}[capped], func(t *testing.T) {
			lifetime := context.Background()
			if capped {
				var cancel context.CancelFunc
				lifetime, cancel = context.WithTimeout(lifetime, 3*time.Second)
				defer cancel()
			}
			readiness, stopReadiness := context.WithCancel(lifetime)
			defer stopReadiness()
			queried := false
			err := admitTerminalAfterReady(readiness, lifetime, func(context.Context) error { stopReadiness(); return nil }, func(ctx context.Context) error {
				queried = true
				if ctx.Err() != nil {
					t.Errorf("postbirth query inherited readiness cancellation: %v", ctx.Err())
				}
				got, ok := ctx.Deadline()
				if !ok || time.Until(got) <= 0 || time.Until(got) > 5*time.Second {
					t.Errorf("query budget=%v", got)
				}
				if want, ok := lifetime.Deadline(); ok && !got.Equal(want) {
					t.Errorf("lost caller deadline=%v want=%v", got, want)
				}
				return nil
			})
			if err != nil || !queried {
				t.Fatalf("queried=%v error=%v", queried, err)
			}
		})
	}
}

func TestTerminalHandoffPostbirthQueryKeepsLifetimeCancellation(t *testing.T) {
	lifetime, cancel := context.WithCancel(context.Background())
	defer cancel()
	queried := false
	err := admitTerminalAfterReady(lifetime, lifetime, func(context.Context) error {
		cancel()
		return nil
	}, func(context.Context) error {
		queried = true
		return nil
	})
	if !errors.Is(err, context.Canceled) || queried {
		t.Fatalf("query=%v error=%v", queried, err)
	}
}

type postbirthExitWorld struct {
	*handoffWorld
	lifetime context.Context
	queried  chan struct{}
}

func (w *postbirthExitWorld) Start(ctx context.Context) (HandoffClient, error) {
	w.lifetime = ctx
	return w.handoffWorld.Start(ctx)
}
func (w *postbirthExitWorld) Admit(readiness context.Context) error {
	return admitTerminalAfterReady(readiness, w.lifetime, func(context.Context) error { return nil }, func(ctx context.Context) error {
		close(w.queried)
		<-ctx.Done()
		w.event("postbirth-query-ended")
		return ctx.Err()
	})
}
func TestTerminalHandoffClientExitCancelsPostbirthQuery(t *testing.T) {
	for _, callerCancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "client-exit", true: "caller-cancel"}[callerCancel], func(t *testing.T) {
			lifetime, cancel := context.WithCancel(context.Background())
			defer cancel()
			w := &postbirthExitWorld{handoffWorld: newHandoffWorld(), queried: make(chan struct{})}
			done := make(chan error, 1)
			go func() { _, err := coordinateTerminalHandoff(lifetime, lifetime, w, time.Minute, true); done <- err }()
			<-w.queried
			if callerCancel {
				cancel()
			} else {
				w.exit <- HandoffExit{Code: 0}
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("client exit before query admission reported success")
				}
			case <-time.After(time.Second):
				cancel()
				<-done
				t.Fatal("client exit did not cancel and join the postbirth query")
			}
			ended, released := -1, -1
			for i, e := range w.events {
				if e == "postbirth-query-ended" {
					ended = i
				}
				if e == "release" {
					released = i
				}
			}
			if ended < 0 || released <= ended {
				t.Fatalf("query not joined before lease release: %v", w.events)
			}
		})
	}
}
