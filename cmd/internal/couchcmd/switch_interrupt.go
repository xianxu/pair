package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// InterruptBeforeAgentSwitch runs only from the held lifecycle transaction.
// The existing drive owner proves and sends at most one interrupt; all later
// observations can only establish fresh idle or refuse, never send more input.
func (g *switchServiceGuard) InterruptBeforeAgentSwitch(base context.Context, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence) error {
	if evidence.Vacant() {
		return g.ConfirmAgentSwitch(base, address, source, evidence)
	}
	s := g.service.Load()
	if s == nil {
		return errors.New("switch-agent: supervisor unavailable")
	}
	ctx, cancel := context.WithTimeout(base, 30*time.Second)
	defer cancel()
	observeSource := func(ctx context.Context) (couchmessage.SlotLiveness, error) {
		live, err := inspectSwitchLiveSource(ctx, s, address, source, false)
		if err != nil {
			return live, err
		}
		if s.authority.record == nil {
			return live, errors.New("switch-agent: source record verification unavailable")
		}
		record, err := s.authority.record(ctx, address)
		if err != nil {
			return live, err
		}
		if !evidence.MatchesRecord(record) {
			return live, errors.New("switch-agent: source generation changed before interrupt")
		}
		return live, ctx.Err()
	}
	live, err := observeSource(ctx)
	if err != nil {
		return fmt.Errorf("switch-agent: interrupt source proof: %w", err)
	}
	// A form's empty token is pinned before any observation can yield to input.
	source = switchSourceToken(live)
	if live.Settled != nil && *live.Settled {
		return nil
	}
	detail := "wrapper has no authenticated interrupt capability"
	endpoint, capable := s.authority.endpoint(live.Binding).(agentDriveEndpoint)
	if capable {
		observation, err := endpoint.ObserveDrive(ctx)
		if err != nil {
			return fmt.Errorf("switch-agent: observe interrupt admission: %w", err)
		}
		request := agentdrive.Request{Interrupt: true}
		detail = request.Refusal(observation.Admission(), agentdrive.InterruptRefusal)
		if request.Admits(observation.Admission()) && observation.Instance != "" {
			id, err := newRequestID()
			if err != nil {
				return err
			}
			deadline, _ := ctx.Deadline()
			value, err := executeAgentDrive(couchcore.OperationCall{Context: ctx, TypedPayload: agentDrivePayload{
				endpoint: endpoint, id: id, start: couchmessage.DriveStart{Request: request, Deadline: deadline, Instance: observation.Instance, Sequence: observation.Sequence},
				callerCurrent: func(ctx context.Context) error { _, err := observeSource(ctx); return err },
			}})
			if err != nil {
				return fmt.Errorf("switch-agent: interrupt phase: %w", err)
			}
			result, ok := value.(agentdrive.Result)
			if !ok {
				return errors.New("switch-agent: interrupt outcome unconfirmed")
			}
			if result.Outcome != agentdrive.OutcomeConfirmed {
				return errors.Join(fmt.Errorf("switch-agent: interrupt phase %s: %s", result.Outcome, result.Detail), ctx.Err())
			}
			detail = "interrupt confirmed"
		} else if observation.Instance == "" {
			detail = "wrapper instance proof unavailable; nothing was sent"
		}
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		live, err := observeSource(ctx)
		if err != nil {
			return fmt.Errorf("switch-agent: waiting for fresh idle: %s; final idle observation: %w", detail, err)
		}
		if live.Settled != nil && *live.Settled {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("switch-agent: waiting for fresh idle: %s; source is not known idle: %w", detail, ctx.Err())
		case <-ticker.C:
		}
	}
}
