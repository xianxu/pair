package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdio "io"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

const (
	parkCommitSoftTarget = 100 * time.Millisecond
	parkCommitDeadline   = time.Second
)

type LifecycleIO interface {
	PublishRequest(artifactpath.LifecyclePaths, pairlifecycle.QuitRequest) error
	ObserveCompletion(artifactpath.LifecyclePaths, pairlifecycle.QuitRequest) (pairlifecycle.QuitCompletion, bool, error)
	CleanupAttempt(artifactpath.LifecyclePaths, pairlifecycle.QuitRequest) error
}

type PairLifecycleStoreIO struct{ Store pairlifecycle.Store }

func (io PairLifecycleStoreIO) PublishRequest(paths artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	return io.Store.PublishRequest(paths, request)
}

func (io PairLifecycleStoreIO) ObserveCompletion(paths artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) (pairlifecycle.QuitCompletion, bool, error) {
	if err := io.Store.Reconcile(paths, pairlifecycle.RecordCompletion, request.Attempt); err != nil {
		if pairlifecycle.PublicationOutcomeOf(err) == pairlifecycle.NotCommitted {
			return pairlifecycle.QuitCompletion{}, false, nil
		}
		return pairlifecycle.QuitCompletion{}, false, err
	}
	path, err := paths.Completion(request.Attempt)
	if err != nil {
		return pairlifecycle.QuitCompletion{}, false, err
	}
	raw, err := io.Store.Runtime.ReadFile(path)
	if err != nil {
		return pairlifecycle.QuitCompletion{}, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var completion pairlifecycle.QuitCompletion
	if err := decoder.Decode(&completion); err != nil {
		return pairlifecycle.QuitCompletion{}, false, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, stdio.EOF) {
		return pairlifecycle.QuitCompletion{}, false, errors.New("quit completion has trailing data")
	}
	return completion, true, nil
}

func (io PairLifecycleStoreIO) CleanupAttempt(paths artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	requestPath, err := paths.Request(request.Attempt)
	if err != nil {
		return err
	}
	completionPath, err := paths.Completion(request.Attempt)
	if err != nil {
		return err
	}
	remove := func(path string) error {
		err := io.Store.Runtime.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return errors.Join(remove(requestPath), remove(completionPath))
}

type PairLifecycleController struct {
	Threads   *ThreadStore
	DataDir   string
	Lifecycle LifecycleIO
	Sessions  PairSessionIO
	Clock     Clock
	Nonce     func() (string, error)
}

type ParkResult struct {
	Thread                  ThreadRecord
	RequestedCommitDuration time.Duration
	SoftTargetMissed        bool
	CleanupError            error
}

func (c *PairLifecycleController) Park(ctx context.Context, address ThreadAddress) (ParkResult, error) {
	if err := c.validate(); err != nil {
		return ParkResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ParkResult{}, err
	}
	current, err := c.Threads.GetThread(address)
	if err != nil {
		return ParkResult{}, err
	}
	if current.Park != nil {
		return c.Retry(ctx, address)
	}
	incarnation, err := soleParkableIncarnation(current)
	if err != nil {
		return ParkResult{}, err
	}
	binding, err := c.Sessions.PairSession(address)
	if err != nil {
		return ParkResult{}, err
	}
	nonce, err := c.Nonce()
	if err != nil {
		return ParkResult{}, err
	}
	identity := ParkIdentity{Nonce: nonce, Address: address, PID: incarnation.PID, ProcessIdentity: incarnation.Identity}
	started := c.Clock.Now()
	begun, err := c.Threads.BeginPark(address, current.Revision, identity)
	duration := c.Clock.Now().Sub(started)
	result := ParkResult{Thread: begun, RequestedCommitDuration: duration, SoftTargetMissed: duration >= parkCommitSoftTarget}
	if err != nil {
		return result, err
	}
	if duration >= parkCommitDeadline {
		failed, advanceErr := c.Threads.AdvancePark(address, begun.Revision, ParkEvent{
			Kind: ParkFailureObserved, Identity: identity, Attempt: 1,
			Failure: &ParkFailure{Code: pairlifecycle.FailureRequestPublishFailed, Diagnostic: "requested commit exceeded 1s deadline"},
		})
		if advanceErr == nil {
			result.Thread = failed
		}
		return result, errors.Join(errors.New("park requested commit exceeded 1s deadline"), advanceErr)
	}
	return c.runActiveAttempt(ctx, result, begun, binding)
}

func (c *PairLifecycleController) Retry(ctx context.Context, address ThreadAddress) (ParkResult, error) {
	if err := c.validate(); err != nil {
		return ParkResult{}, err
	}
	current, err := c.Threads.GetThread(address)
	if err != nil {
		return ParkResult{}, err
	}
	if current.Park == nil {
		return ParkResult{}, errors.New("thread has no active park transaction")
	}
	binding, err := c.Sessions.PairSession(address)
	if err != nil {
		return ParkResult{Thread: current}, err
	}
	result, completed, err := c.reconcileAttempts(ParkResult{Thread: current}, current, binding.Name)
	if err != nil || completed {
		return result, err
	}
	current = result.Thread
	last := current.Park.Attempts[len(current.Park.Attempts)-1]
	if current.Park.Phase == ParkUnknown && !last.Closed {
		return ParkResult{Thread: current}, errors.New("unknown park requires Recover")
	}
	if last.Closed {
		if !binding.Present {
			return ParkResult{Thread: current}, errors.New("closed park attempt requires Recover because exact session is absent")
		}
		current, err = c.Threads.AppendParkAttempt(address, current.Revision, current.Park.Identity)
		if err != nil {
			return ParkResult{Thread: current}, err
		}
	}
	return c.runActiveAttempt(ctx, ParkResult{Thread: current}, current, binding)
}

func (c *PairLifecycleController) Recover(ctx context.Context, address ThreadAddress) (ParkResult, error) {
	if err := c.validate(); err != nil {
		return ParkResult{}, err
	}
	current, err := c.Threads.GetThread(address)
	if err != nil {
		return ParkResult{}, err
	}
	if current.Park == nil {
		return ParkResult{}, errors.New("thread has no active park transaction")
	}
	binding, err := c.Sessions.PairSession(address)
	if err != nil {
		return ParkResult{Thread: current}, err
	}
	result, completed, err := c.reconcileAttempts(ParkResult{Thread: current}, current, binding.Name)
	if err != nil || completed {
		return result, err
	}
	current = result.Thread
	if !binding.Present {
		request, _, requestErr := c.requestFor(current, binding.Name)
		if requestErr != nil {
			return ParkResult{Thread: current}, requestErr
		}
		if current.Park.Phase != ParkUnknown {
			current, err = c.recordFailure(current, request.Attempt, pairlifecycle.FailureCompletionMissing, "exact session absent without completion")
		}
		return ParkResult{Thread: current}, errors.Join(errors.New("exact session absent without matching completion"), err)
	}
	last := current.Park.Attempts[len(current.Park.Attempts)-1]
	if current.Park.Phase == ParkUnknown || last.Closed {
		current, err = c.Threads.AppendParkAttempt(address, current.Revision, current.Park.Identity)
		if err != nil {
			return ParkResult{Thread: current}, err
		}
	}
	return c.runActiveAttempt(ctx, ParkResult{Thread: current}, current, binding)
}

func (c *PairLifecycleController) Abandon(_ context.Context, address ThreadAddress) (ParkResult, error) {
	if err := c.validate(); err != nil {
		return ParkResult{}, err
	}
	current, err := c.Threads.GetThread(address)
	if err != nil {
		return ParkResult{}, err
	}
	if current.Park == nil {
		return ParkResult{}, errors.New("thread has no active park transaction")
	}
	abandoned, err := c.Threads.AbandonPark(address, current.Revision, current.Park.Identity)
	return ParkResult{Thread: abandoned}, err
}

func (c *PairLifecycleController) ReconcileActive(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return err
	}
	var result error
	for _, record := range snapshot.Records {
		if record.Park == nil {
			continue
		}
		_, reconcileErr := c.Recover(ctx, record.Address)
		result = errors.Join(result, reconcileErr)
	}
	return result
}

func (c *PairLifecycleController) runActiveAttempt(ctx context.Context, result ParkResult, current ThreadRecord, binding PairSessionBinding) (ParkResult, error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}
	request, paths, err := c.requestFor(current, binding.Name)
	if err != nil {
		return result, err
	}
	if current.Park.Phase == ParkRequested {
		if err := c.Lifecycle.PublishRequest(paths, request); err != nil {
			code := pairlifecycle.FailureRequestPublishFailed
			if pairlifecycle.PublicationOutcomeOf(err) == pairlifecycle.Indeterminate {
				code = pairlifecycle.FailureCompletionMissing
			}
			failed, advanceErr := c.recordFailure(current, request.Attempt, code, err.Error())
			result.Thread = failed
			return result, errors.Join(err, advanceErr)
		}
		current, err = c.Threads.AdvancePark(current.Address, current.Revision, ParkEvent{
			Kind: ParkRequestCommitted, Identity: current.Park.Identity, Attempt: request.Attempt,
		})
		if err != nil {
			return result, err
		}
		result.Thread = current
	}
	result, completed, err := c.reconcileAttempts(result, current, binding.Name)
	if err != nil || completed {
		return result, err
	}
	current = result.Thread
	var triggerErr error
	if binding.Present {
		triggerErr = c.Sessions.TriggerQuit(binding.Name, launcher.QuitIntent{
			Version: launcher.QuitIntentVersion, Kind: launcher.QuitIntentCouch,
			Request: &launcher.QuitRequestReference{
				DataDir: c.DataDir, RepoScope: current.Address.RepoScope, Tag: string(current.Address.Tag),
				Nonce: current.Park.Identity.Nonce, Attempt: request.Attempt,
			},
		})
	}
	completion, found, observeErr := c.Lifecycle.ObserveCompletion(paths, request)
	if observeErr != nil {
		return result, errors.Join(triggerErr, observeErr)
	}
	if found {
		return c.applyCompletion(result, current, paths, request, completion)
	}
	latestBinding, sessionErr := c.Sessions.PairSession(current.Address)
	if sessionErr != nil {
		return result, errors.Join(triggerErr, sessionErr)
	}
	if !latestBinding.Present {
		failed, advanceErr := c.recordFailure(current, request.Attempt, pairlifecycle.FailureCompletionMissing, "exact session absent without completion")
		result.Thread = failed
		return result, errors.Join(triggerErr, errors.New("exact session absent without matching completion"), advanceErr)
	}
	return result, triggerErr
}

func (c *PairLifecycleController) applyCompletion(result ParkResult, current ThreadRecord, paths artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest, completion pairlifecycle.QuitCompletion) (ParkResult, error) {
	if err := pairlifecycle.MatchQuitCompletion(request, completion); err != nil {
		failed, advanceErr := c.recordFailure(current, request.Attempt, pairlifecycle.FailureStaleCompletion, err.Error())
		result.Thread = failed
		return result, errors.Join(err, advanceErr)
	}
	if completion.Outcome == pairlifecycle.CompletionFailure {
		failed, err := c.recordFailure(current, request.Attempt, completion.FailureCode, "Pair cleanup failed")
		result.Thread = failed
		return result, errors.Join(errors.New("Pair cleanup failed"), err)
	}
	finalized, err := c.Threads.FinalizePark(current.Address, current.Revision, current.Park.Identity, request.Attempt, completion.CompletedAt)
	if err != nil {
		latest, getErr := c.Threads.GetThread(current.Address)
		if getErr != nil {
			return result, errors.Join(err, getErr)
		}
		result.Thread = latest
		var revisionErr *ThreadRevisionError
		if !errors.As(err, &revisionErr) || latest.Park == nil || latest.Park.Identity != current.Park.Identity {
			return result, err
		}
		code := pairlifecycle.FailureRevisionConflict
		if !hasExactParkIncarnation(latest, current.Park.Identity) {
			code = pairlifecycle.FailureReplacementIncarnation
		}
		failed, advanceErr := c.recordFailure(latest, request.Attempt, code, err.Error())
		if advanceErr == nil {
			result.Thread = failed
		}
		return result, errors.Join(err, advanceErr)
	}
	result.Thread = finalized
	result.CleanupError = c.Lifecycle.CleanupAttempt(paths, request)
	return result, nil
}

func hasExactParkIncarnation(record ThreadRecord, identity ParkIdentity) bool {
	matches := 0
	for _, incarnation := range record.Incarnations {
		if incarnation.PID == identity.PID && incarnation.Identity == identity.ProcessIdentity &&
			(incarnation.State == IncarnationLive || incarnation.State == IncarnationUnknown) {
			matches++
		}
	}
	return matches == 1
}

func (c *PairLifecycleController) recordFailure(current ThreadRecord, attempt uint64, code pairlifecycle.FailureCode, diagnostic string) (ThreadRecord, error) {
	return c.Threads.AdvancePark(current.Address, current.Revision, ParkEvent{
		Kind: ParkFailureObserved, Identity: current.Park.Identity, Attempt: attempt,
		Failure: &ParkFailure{Code: code, Diagnostic: diagnostic},
	})
}

func (c *PairLifecycleController) requestFor(current ThreadRecord, session string) (pairlifecycle.QuitRequest, artifactpath.LifecyclePaths, error) {
	if current.Park == nil || len(current.Park.Attempts) == 0 {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, errors.New("thread has no active park attempt")
	}
	return c.requestForAttempt(current, session, current.Park.Attempts[len(current.Park.Attempts)-1].Number)
}

func (c *PairLifecycleController) requestForAttempt(current ThreadRecord, session string, attempt uint64) (pairlifecycle.QuitRequest, artifactpath.LifecyclePaths, error) {
	if current.Park == nil || len(current.Park.Attempts) == 0 {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, errors.New("thread has no active park attempt")
	}
	found := false
	for _, candidate := range current.Park.Attempts {
		if candidate.Number == attempt {
			found = true
			break
		}
	}
	if !found {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, errors.New("park attempt is not part of the active transaction")
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: c.DataDir, RepoScope: current.Address.RepoScope, Tag: string(current.Address.Tag),
	})
	if err != nil {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, err
	}
	lifecyclePaths, err := paths.Lifecycle(current.Park.Identity.Nonce)
	if err != nil {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, err
	}
	completionKey, err := lifecyclePaths.CompletionKey(attempt)
	if err != nil {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, err
	}
	request := pairlifecycle.QuitRequest{
		SchemaVersion: pairlifecycle.SchemaVersion,
		Identity: pairlifecycle.Identity{
			Nonce: current.Park.Identity.Nonce, RepoScope: current.Address.RepoScope, Tag: string(current.Address.Tag),
			PID: current.Park.Identity.PID, ProcessIdentity: current.Park.Identity.ProcessIdentity,
		},
		Attempt: attempt, Session: session, Mode: pairlifecycle.CleanupPreserveScrollback,
		CompletionKey: completionKey,
	}
	if err := pairlifecycle.ValidateQuitRequest(request); err != nil {
		return pairlifecycle.QuitRequest{}, artifactpath.LifecyclePaths{}, err
	}
	return request, lifecyclePaths, nil
}

// reconcileAttempts checks every durable attempt before a newer trigger is
// delivered. A late success from an older attempt closes the transaction and
// suppresses all newer work.
func (c *PairLifecycleController) reconcileAttempts(result ParkResult, current ThreadRecord, session string) (ParkResult, bool, error) {
	if current.Park == nil {
		return result, true, nil
	}
	attempts := append([]ParkAttempt(nil), current.Park.Attempts...)
	for _, attempt := range attempts {
		if attempt.Closed {
			continue
		}
		request, paths, err := c.requestForAttempt(current, session, attempt.Number)
		if err != nil {
			return result, false, err
		}
		completion, found, err := c.Lifecycle.ObserveCompletion(paths, request)
		if err != nil {
			if pairlifecycle.PublicationOutcomeOf(err) == pairlifecycle.Indeterminate {
				failed, advanceErr := c.recordFailure(current, attempt.Number, pairlifecycle.FailureCompletionMissing, err.Error())
				if advanceErr == nil {
					result.Thread = failed
				}
				return result, false, errors.Join(err, advanceErr)
			}
			return result, false, err
		}
		if !found {
			continue
		}
		next, applyErr := c.applyCompletion(result, current, paths, request, completion)
		result = next
		if next.Thread.Park == nil {
			return result, true, applyErr
		}
		current = next.Thread
		if applyErr != nil {
			return result, false, applyErr
		}
	}
	return result, false, nil
}

func (c *PairLifecycleController) validate() error {
	if c == nil || c.Threads == nil || c.Lifecycle == nil || c.Sessions == nil || c.Clock == nil || c.Nonce == nil || c.DataDir == "" {
		return errors.New("Pair lifecycle controller is incomplete")
	}
	return nil
}

func soleParkableIncarnation(record ThreadRecord) (ThreadIncarnation, error) {
	var found *ThreadIncarnation
	for i := range record.Incarnations {
		incarnation := &record.Incarnations[i]
		if incarnation.State != IncarnationLive && incarnation.State != IncarnationUnknown {
			continue
		}
		if found != nil {
			return ThreadIncarnation{}, errors.New("park requires exactly one live or unknown incarnation")
		}
		found = incarnation
	}
	if found == nil || found.PID <= 0 || found.Identity == "" {
		return ThreadIncarnation{}, errors.New("park requires exactly one identified live or unknown incarnation")
	}
	return *found, nil
}

var _ LifecycleIO = PairLifecycleStoreIO{}
var _ PairSessionIO = ScopedThreadArtifactCollisionChecker{}
