package testcouch

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
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
			f.record(RunEvent{Kind: "own", Resource: resourceFor(p)})
		}
	}
	return nil
}

func (f *Fleet) reconcileProcesses(ctx context.Context) error {
	f.mu.Lock()
	state := f.model
	f.mu.Unlock()
	var err error
	state, err = stopOwned(ctx, nativeProcesses{}, state, syscall.SIGTERM)
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
	// Adopt absence from the initial stop pass too, including already-exited
	// direct children and re-used numeric PIDs that must never be signalled.
	f.mu.Lock()
	remaining := f.model.Resources
	f.mu.Unlock()
	for _, r := range remaining {
		f.record(RunEvent{Kind: "gone", Resource: r})
	}
	f.mu.Lock()
	cleaned := f.model.Phase == "cleaned"
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
