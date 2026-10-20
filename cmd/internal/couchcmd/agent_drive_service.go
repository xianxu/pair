package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

type agentDriveEndpoint interface {
	ObserveDrive(context.Context) (couchmessage.DriveObservation, error)
	StartDrive(context.Context, string, couchmessage.DriveStart) (agentdrive.Result, error)
	DriveStatus(context.Context, string) (agentdrive.Result, error)
	CancelDrive(context.Context, string) (agentdrive.Result, error)
}
type agentDriveService struct {
	enqueue func(string, string, time.Time, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error
}
type agentDrivePayload struct {
	endpoint      agentDriveEndpoint
	id            string
	start         couchmessage.DriveStart
	callerCurrent func(context.Context) error
}

func (s *messageService) installAgentDriveService(console *couchtty.Console) {
	s.drives.Store(&agentDriveService{enqueue: console.EnqueueRemoteOperationDeadline})
}
func (s *messageService) handleAgentDrive(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	refuse := func(err error) couchmessage.Response {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	service := s.drives.Load()
	if service == nil || s.slotOps == nil {
		return couchmessage.Response{Code: "unsupported", Error: "agent drive unavailable; relaunch Couch and the slot on the new build"}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return refuse(err)
	}
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return refuse(err)
	}
	if !s.authority.isHub(origin) {
		return refuse(errors.New("only a verified hub may drive another slot"))
	}
	if response, held := s.slotOps.existingDrive(caller, r); held {
		return response
	}
	var bindings []couchmessage.Binding
	if connected := s.connected.Load(); connected != nil {
		for b, live := range *connected {
			if live {
				bindings = append(bindings, b)
			}
		}
	}
	target, _, err := s.tailBinding(ctx, couchmessage.Request{Target: r.Target}, bindings)
	if err != nil {
		return refuse(err)
	}
	if target.Scope == caller.Scope && target.Tag == caller.Tag {
		return refuse(errors.New("a hub cannot drive its own slot"))
	}
	if err = s.authorizeHubBinding(ctx, caller, target); err != nil {
		return refuse(err)
	}
	endpoint, ok := s.authority.endpoint(target).(agentDriveEndpoint)
	if !ok {
		return refuse(errors.New("wrapper has no agent drive capability; relaunch the slot on the new build"))
	}
	callerLive, ok := s.LivenessForThread(caller.Scope, caller.Tag)
	if !ok || callerLive.Binding != caller {
		return refuse(errors.New("caller wrapper unavailable"))
	}
	callerToken := switchSourceToken(callerLive)
	// Recovery and interrupt drives skip the settled check: an occupied
	// composer or a working turn is exactly what they act on.
	recoveryOnly := r.Drive.RecoveryOnly() || r.Drive.InterruptOnly()
	const notIdle = "wait until the agent composer is empty and its turn finishes"
	targetLive, ok := s.LivenessForThread(target.Scope, target.Tag)
	if !ok || targetLive.Binding != target {
		return refuse(errors.New(notIdle))
	}
	if !recoveryOnly && (targetLive.Settled == nil || !*targetLive.Settled) {
		// Refused either way. One read-only observation lets a refusal on an
		// open panel name the key that would be admitted (pair#716).
		detail := notIdle
		if observation, err := endpoint.ObserveDrive(ctx); err == nil {
			detail = r.Drive.Refusal(observation.Admission(), notIdle)
		}
		return refuse(errors.New(detail))
	}
	targetToken := switchSourceToken(targetLive)
	observation, err := endpoint.ObserveDrive(ctx)
	if err != nil {
		return refuse(fmt.Errorf("observe agent drive: %w; relaunch the slot on the new build if unsupported", err))
	}
	admissible := func(observation couchmessage.DriveObservation) bool {
		return r.Drive.Admits(observation.Admission())
	}
	if !admissible(observation) || observation.Instance == "" {
		return refuse(errors.New(r.Drive.Refusal(observation.Admission(), notIdle)))
	}
	callerCurrent := func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		latest, ok := s.LivenessForThread(caller.Scope, caller.Tag)
		if !ok || switchSourceToken(latest) != callerToken || !s.isConnected(caller) {
			return errors.New("caller wrapper changed or disconnected")
		}
		if _, err := s.switchCaller(ctx, r); err != nil {
			return err
		}
		identity, err := s.connectedWorkspace(ctx, caller)
		if err != nil {
			return err
		}
		if !s.authority.isHub(identity) {
			return errors.New("caller is no longer a verified hub")
		}
		return nil
	}
	check := func(ctx context.Context) error {
		if err := callerCurrent(ctx); err != nil {
			return err
		}
		if err := s.authorizeHubBinding(ctx, caller, target); err != nil {
			return err
		}
		latest, ok := s.LivenessForThread(target.Scope, target.Tag)
		if !ok || switchSourceToken(latest) != targetToken || !s.isConnected(target) || !recoveryOnly && (latest.Settled == nil || !*latest.Settled) {
			return errors.New("target wrapper changed or is busy; wait until the agent composer is empty and its turn finishes")
		}
		if _, err := s.authority.current(ctx, target); err != nil {
			return err
		}
		fresh, err := endpoint.ObserveDrive(ctx)
		if err != nil {
			return err
		}
		if !admissible(fresh) || fresh.Instance != observation.Instance || fresh.Sequence != observation.Sequence {
			return errors.New("target composer changed; inspect with couch --peek before retrying")
		}
		return ctx.Err()
	}
	guard := func(ctx context.Context) error {
		if err := check(ctx); err != nil {
			return &couchcore.SlotOperationError{Code: "drive-refused", Detail: err.Error()}
		}
		return nil
	}
	deadline := time.Now().Add(30 * time.Second)
	payload := agentDrivePayload{endpoint: endpoint, id: r.ID, start: couchmessage.DriveStart{Request: *r.Drive, Deadline: deadline, Instance: observation.Instance, Sequence: observation.Sequence}, callerCurrent: callerCurrent}
	return s.slotOps.admit(ctx, caller, r, func(key string, started func(), finished func(any, error)) error {
		return service.enqueue(key, r.Op, deadline, func(ctx context.Context) (couchcore.OperationCall, error) {
			if err := guard(ctx); err != nil {
				return couchcore.OperationCall{}, err
			}
			return couchcore.OperationCall{Name: r.Op, TypedPayload: hubMutationPayload{original: payload, guard: guard}}, nil
		}, started, finished)
	})
}

// Only a console-local capability can perform input effects. Start is never replayed.
func executeAgentDrive(call couchcore.OperationCall) (any, error) {
	base := call.Context
	if base == nil {
		base = context.Background()
	}
	held, err := unwrapHubMutation(base, call.TypedPayload)
	if err != nil {
		return nil, err
	}
	payload, ok := held.(agentDrivePayload)
	if !ok {
		return nil, errors.New("agent drive requires the authenticated supervisor capability")
	}
	ctx, cancel := context.WithDeadline(base, payload.start.Deadline)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, &couchcore.SlotOperationError{Code: "drive-refused", Detail: "drive deadline expired before admission"}
	}
	if err := payload.callerCurrent(ctx); err != nil {
		return nil, &couchcore.SlotOperationError{Code: "drive-refused", Detail: err.Error()}
	}
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if payload.callerCurrent(ctx) != nil {
					cancel()
					return
				}
			}
		}
	}()
	defer func() { cancel(); <-joined }()
	result, err := payload.endpoint.StartDrive(ctx, payload.id, payload.start)
	if err != nil {
		cancelAgentDrive(payload)
		return agentdrive.Result{Outcome: agentdrive.OutcomeUnconfirmed, Detail: "drive admission reply lost; inspect with couch --peek before sending anything else"}, nil
	}
	for !result.Terminal() {
		select {
		case <-ctx.Done():
			cancelAgentDrive(payload)
			return agentdrive.Result{Outcome: agentdrive.OutcomeUnconfirmed, Detail: "drive cancelled or timed out; inspect with couch --peek before sending anything else", Step: result.Step, Tail: result.Tail}, nil
		case <-time.After(50 * time.Millisecond):
		}
		result, err = payload.endpoint.DriveStatus(ctx, payload.id)
		if err != nil {
			cancelAgentDrive(payload)
			return agentdrive.Result{Outcome: agentdrive.OutcomeUnconfirmed, Detail: "drive status unavailable; inspect with couch --peek before sending anything else"}, nil
		}
	}
	return result, nil
}
func cancelAgentDrive(p agentDrivePayload) {
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	_, _ = p.endpoint.CancelDrive(ctx, p.id)
}
func (s *slotOperations) existingDrive(caller couchmessage.Binding, r couchmessage.Request) (couchmessage.Response, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	old, ok := s.receipts[receiptKey{caller.Scope, caller.Tag, r.ID}]
	if !ok {
		return couchmessage.Response{}, false
	}
	if old.Op != r.Op || old.Target != r.Target || old.IntentDigest != driveIntentDigest(r) {
		return couchmessage.Response{Code: "id-conflict", Error: "request ID already names another intent"}, true
	}
	return couchmessage.Response{Code: "accepted", Operation: &old}, true
}
