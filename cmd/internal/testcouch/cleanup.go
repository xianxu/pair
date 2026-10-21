package testcouch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// captureOwned observes one complete table and retains only the private run's
// process generations. Reading this single marker never retains credentials.
func (f *Fleet) captureOwned(ctx context.Context) error {
	reader := procutil.NativeResourceReader{}
	rows, _, err := reader.Table(ctx)
	if err != nil {
		f.record(RunEvent{Kind: "unknown", Error: err.Error()})
		return err
	}
	marked := map[int]bool{}
	for _, p := range rows {
		if p.Zombie || p.Identity == "" {
			continue
		}
		// Native daemon reparenting makes ancestry alone insufficient. The
		// fresh run marker follows the actual inherited environment instead.
		env, _, e := reader.Environment(ctx, p, "PAIR_TESTCOUCH_RUN_ID")
		if e != nil {
			continue
		}
		if env["PAIR_TESTCOUCH_RUN_ID"] == f.RunID {
			marked[p.PID] = true
		}
	}
	f.mu.Lock()
	known := f.model
	f.mu.Unlock()
	owned, err := selectOwnedCohort(rows, known, marked)
	if err != nil {
		f.record(RunEvent{Kind: "unknown", Error: err.Error()})
		return err
	}
	for _, r := range owned {
		f.record(RunEvent{Kind: "own", Resource: r})
	}
	return nil
}

func (f *Fleet) stopAgents(ctx context.Context) error {
	pending := OwnedRun{}
	for n := 0; n <= f.Options.Workers; n++ {
		dir, dirErr := f.actorDirectory(n)
		if dirErr != nil {
			continue
		}
		ready, err := f.Agent(n)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil && (ready.RunID != f.RunID || ready.PID <= 0 || ready.Generation == "") {
			return err
		}
		pending = ReduceRun(pending, RunEvent{Kind: "own", Resource: Resource{ID: fmt.Sprint(ready.PID), Generation: ready.Generation, Kind: "agent"}})
		if err == nil {
			if err = agentfixture.WriteCommand(dir, agentfixture.Command{ID: "close-" + f.RunID, Kind: agentfixture.Stop}); err != nil {
				return err
			}
		}
	}
	return waitUntil(ctx, func() (bool, error) {
		next, err := stopOwned(ctx, nativeProcesses{}, pending, 0)
		pending = next
		return len(pending.Resources) == 0, err
	})
}

func (f *Fleet) reconcileProcesses(ctx context.Context) error {
	f.mu.Lock()
	state := f.model
	f.mu.Unlock()
	var err error
	previous := state
	state, err = stopOwned(ctx, nativeProcesses{}, state, syscall.SIGTERM)
	for key, r := range previous.Resources {
		if _, exists := state.Resources[key]; !exists {
			f.record(RunEvent{Kind: "gone", Resource: r})
		}
	}
	if err != nil {
		f.record(RunEvent{Kind: "unknown", Error: err.Error()})
		return err
	}
	termDeadline := time.Now().Add(1500 * time.Millisecond)
	for len(state.Resources) > 0 {
		if ctx.Err() != nil {
			f.record(RunEvent{Kind: "unknown", Error: ctx.Err().Error()})
			return ctx.Err()
		}
		signal := syscall.Signal(0)
		if time.Now().After(termDeadline) {
			signal = syscall.SIGKILL
		}
		previous := state
		state, err = stopOwned(ctx, nativeProcesses{}, state, signal)
		for key, r := range previous.Resources {
			if _, exists := state.Resources[key]; !exists {
				f.record(RunEvent{Kind: "gone", Resource: r})
			}
		}
		if err != nil {
			f.record(RunEvent{Kind: "unknown", Error: err.Error()})
			return err
		}
		if len(state.Resources) > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	f.mu.Lock()
	cleaned := f.model.Phase == "quiescent"
	f.mu.Unlock()
	if !cleaned {
		return errors.New("native ownership did not converge to cleaned")
	}
	return nil
}

func (f *Fleet) AssertCleaned() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.model.Phase != "cleaned" || len(f.model.Resources) != 0 {
		return fmt.Errorf("owned resources remain: phase=%s count=%d", f.model.Phase, len(f.model.Resources))
	}
	return nil
}
