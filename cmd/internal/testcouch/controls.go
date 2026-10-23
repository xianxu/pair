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
func (f *Fleet) SetState(ctx context.Context, n int, phase agentfixture.Phase) (outcome error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	command := agentfixture.Command{ID: fmt.Sprintf("state-%d", nextCommand.Add(1)), Kind: agentfixture.SetState, State: phase}
	if err := agentfixture.ValidateCommand(command); err != nil {
		return err
	}
	ready, err := f.Agent(n)
	if err != nil {
		return err
	}
	command.PID, command.RunID, command.Generation = ready.PID, ready.RunID, ready.Generation
	control, err := f.actorDirectory(n)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := agentfixture.WriteCommand(control, command); err != nil {
		return err
	}
	retired := false
	defer func() {
		if !retired {
			outcome = errors.Join(agentfixture.ErrCleanupUncertain, outcome)
			f.record(RunEvent{Kind: "unknown", Error: outcome.Error()})
		}
	}()
	var result agentfixture.Result
	observe := func() (bool, error) {
		var err error
		result, err = agentfixture.ReadResult(control, command.ID)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return err == nil, err
	}
	if err := waitUntil(ctx, observe); err != nil {
		outcome = err
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := agentfixture.CancelCommand(control, command.ID); err != nil {
			return errors.Join(outcome, err)
		}
		if err := waitUntil(cleanup, observe); err != nil {
			return errors.Join(outcome, err)
		}
	}
	if err := agentfixture.RetireResult(control, command.ID); err != nil {
		return errors.Join(outcome, err)
	}
	retired = true
	if result.Error != "" || result.ExitCode != 0 {
		return errors.Join(outcome, fmt.Errorf("fixture state refused: %s (exit %d)", result.Error, result.ExitCode))
	}
	return outcome
}
