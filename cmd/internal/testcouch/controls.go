package testcouch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// Press sends one physical key using the console's negotiated keyboard modes.
func (f *Fleet) Press(legacy string, code, modifier int) error {
	_, done, err := f.begin(context.Background())
	if err != nil {
		return err
	}
	defer done()
	f.mu.Lock()
	p := f.console
	f.mu.Unlock()
	if p == nil {
		return errors.New("no active console")
	}
	return p.press(legacy, code, modifier)
}

// SetState asks the existing fixture owner to change its screen. It pins the
// current process generation and joins/retire its existing command receipt.
func (f *Fleet) SetState(ctx context.Context, n int, phase agentfixture.Phase) error {
	_, err := f.setState(ctx, n, phase, nil)
	return err
}

// ConfigureWorkload changes bounded redraw activity through the same correlated
// request owner as SetState. Its observation is copied before receipt retirement.
func (f *Fleet) ConfigureWorkload(ctx context.Context, n int, request agentfixture.WorkloadRequest) (agentfixture.WorkloadObservation, error) {
	result, err := f.setState(ctx, n, agentfixture.Working, &request)
	if err != nil {
		return agentfixture.WorkloadObservation{}, err
	}
	if result.Workload == nil {
		return agentfixture.WorkloadObservation{}, errors.New("fixture workload response lacks observation")
	}
	return *result.Workload, nil
}

func (f *Fleet) setState(ctx context.Context, n int, phase agentfixture.Phase, workload *agentfixture.WorkloadRequest) (result agentfixture.Result, outcome error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return result, err
	}
	defer done()
	command := agentfixture.Command{ID: fmt.Sprintf("state-%d", nextCommand.Add(1)), Kind: agentfixture.SetState, State: phase, Workload: workload}
	if err := agentfixture.ValidateCommand(command); err != nil {
		return result, err
	}
	ready, err := f.Agent(n)
	if err != nil {
		return result, err
	}
	command.PID, command.RunID, command.Generation = ready.PID, ready.RunID, ready.Generation
	control, err := f.actorDirectory(n)
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := agentfixture.WriteCommand(control, command); err != nil {
		return result, err
	}
	retired := false
	defer func() {
		if !retired {
			outcome = errors.Join(agentfixture.ErrCleanupUncertain, outcome)
			f.record(RunEvent{Kind: "unknown", Error: outcome.Error()})
		}
	}()
	observe := func() (bool, error) {
		var err error
		result, err = agentfixture.ReadResult(control, command.ID)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err == nil && result.Workload != nil {
			observation := *result.Workload
			result.Workload = &observation
		}
		return err == nil, err
	}
	if err := waitUntil(ctx, observe); err != nil {
		outcome = err
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := agentfixture.CancelCommand(control, command.ID); err != nil {
			return result, errors.Join(outcome, err)
		}
		if err := waitUntil(cleanup, observe); err != nil {
			return result, errors.Join(outcome, err)
		}
	}
	if err := agentfixture.RetireResult(control, command.ID); err != nil {
		return result, errors.Join(outcome, err)
	}
	retired = true
	if result.Error != "" || result.ExitCode != 0 {
		return result, errors.Join(outcome, fmt.Errorf("fixture state refused: %s (exit %d)", result.Error, result.ExitCode))
	}
	return result, outcome
}
