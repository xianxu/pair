package couchcore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// LifecycleFailureRecorder appends terminal failure evidence. It owns no
// lifecycle identity, admission ordering or successful-action state.
type LifecycleFailureRecorder struct {
	store    LaunchFailureStorage
	now      func() time.Time
	nonce    func() (string, error)
	mu       sync.Mutex // serializes only the injected nonce source
	snapshot lifecycleFailureSnapshotCache
}

// OperationAttempt belongs to one completion boundary; nested calls borrow it.
type OperationAttempt struct {
	recorder *LifecycleFailureRecorder
	nonce    string
	op       string
	target   LifecycleFailureTarget
	once     sync.Once
	mu       sync.Mutex
	value    any
	err      error
	warning  string
	failure  bool
}

func NewLifecycleFailureRecorder(store LaunchFailureStorage, now func() time.Time, nonce func() (string, error)) *LifecycleFailureRecorder {
	return &LifecycleFailureRecorder{store: store, now: now, nonce: nonce}
}

// Begin copies diagnostic evidence only. It performs no IO and cannot refuse a
// lifecycle action. Entropy and publication are needed only after failure.
func (r *LifecycleFailureRecorder) Begin(op string, target LifecycleFailureTarget) (*OperationAttempt, error) {
	return &OperationAttempt{recorder: r, op: op, target: target}, nil
}

// ObserveCall enriches diagnostic evidence with the call actually prepared,
// including an error return. It grants no effect or identity authority.
func (a *OperationAttempt) ObserveCall(call OperationCall) {
	t := lifecycleFailureTargetForCall(call)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.target.Address = completeFailureAddress(t.Address, a.target.Address)
	if t.StartingPath != "" {
		a.target.StartingPath = t.StartingPath
	}
	if t.ParkNonce != "" {
		a.target.ParkNonce = t.ParkNonce
	}
	if t.Slot != "" {
		a.target.Slot = t.Slot
	}
	if call.FailureTarget != nil && call.FailureTarget.Target != "" {
		a.target.Target = t.Target
	}
}

func (a *OperationAttempt) Nonce() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.nonce
}
func (a *OperationAttempt) DiagnosticWarning() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.warning
}
func (a *OperationAttempt) RecordedFailureNonce() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failure {
		return a.nonce
	}
	return ""
}

// Finish is the only publisher. Success reads and writes no diagnostics;
// duplicate completion returns the original result without another append.
func (a *OperationAttempt) Finish(value any, cause error) (any, error) {
	return a.FinishContext(context.Background(), value, cause)
}

// FinishContext bounds optional publication by its joined worker owner's lifetime.
func (a *OperationAttempt) FinishContext(owner context.Context, value any, cause error) (any, error) {
	if owner == nil {
		owner = context.Background()
	}
	a.once.Do(func() {
		evidenceCause := cause
		if parked, ok := value.(ParkResult); ok && evidenceCause == nil && parked.CleanupError != nil {
			evidenceCause = parked.CleanupError
		}
		err, warning := a.finish(owner, cause, evidenceCause)
		a.mu.Lock()
		a.value, a.err, a.warning = value, err, warning
		a.failure = evidenceCause != nil
		a.mu.Unlock()
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.value, a.err
}

func (a *OperationAttempt) finish(owner context.Context, cause, evidenceCause error) (error, string) {
	if evidenceCause == nil {
		return nil, ""
	}
	r := a.recorder
	var saved error
	if a.nonce == "" {
		r.mu.Lock()
		nonce, err := r.nonce()
		r.mu.Unlock()
		a.mu.Lock()
		a.nonce = nonce
		a.mu.Unlock()
		saved = err
	}
	if saved == nil {
		if r.store == nil {
			saved = errors.New("lifecycle diagnostic store unavailable")
		} else {
			f := lifecycleFailureForAttempt(a, evidenceCause, r.now())
			if cause == nil {
				f.Outcome = "unproven"
			}
			ctx, cancel := context.WithTimeout(owner, launchDiagnosticsAdmissionTimeout)
			saved = r.store.Save(ctx, f)
			cancel()
		}
	}
	warning := ""
	if saved != nil {
		warning = "lifecycle diagnostics could not be saved: " + saved.Error()
	}
	if cause == nil {
		return nil, "lifecycle completed with unproven cleanup [failure " + a.nonce + "]" + optionalDiagnosticWarning(warning)
	}
	path := "unavailable"
	if r.store != nil {
		path = r.store.Path()
	}
	return &lifecycleFailureError{cause: cause, nonce: a.nonce, path: path, saveError: saved}, warning
}

func optionalDiagnosticWarning(warning string) string {
	if warning == "" {
		return ""
	}
	return "; " + warning
}

func lifecycleFailureForAttempt(a *OperationAttempt, cause error, at time.Time) LifecycleFailure {
	a.mu.Lock()
	target := a.target
	a.mu.Unlock()
	f := LifecycleFailure{FailedAt: at, Outcome: "failed"}
	var launch *launchFailureError
	if errors.As(cause, &launch) {
		f = cloneLaunchFailure(launch.failure)
		f.LaunchNonce = f.Nonce
		f.Phase = "launch"
	}
	f.Nonce, f.Operation, f.FailedAt = a.nonce, a.op, at
	f.Target = target.Target
	f.Address = completeFailureAddress(target.Address, f.Address)
	if target.StartingPath != "" {
		f.StartingPath = target.StartingPath
	}
	f.Slot, f.ParkNonce, f.FinalError = target.Slot, target.ParkNonce, cause.Error()
	f.Outcome = lifecycleFailureOutcome(cause)
	var details LifecycleFailureDetails
	if errors.As(cause, &details) {
		f.Phase, f.Holders, f.Outcome = details.FailurePhase(), details.FailureHolders(), details.FailureOutcome()
	}
	return f
}

func lifecycleFailureOutcome(cause error) string {
	var recovery *RecoveryLaunchUnconfirmed
	var reload *ReloadUnconfirmed
	var readiness *RestartUnready
	if errors.As(cause, &recovery) || errors.As(cause, &reload) || errors.As(cause, &readiness) {
		return "unproven"
	}
	var slot *SlotOperationError
	var recoverRefusal *RecoverRefusal
	var reap *ReapRefusal
	var resume *ResumeRefusal
	var confirmation *RecoveryConfirmationRequired
	if errors.As(cause, &slot) || errors.As(cause, &recoverRefusal) || errors.As(cause, &reap) || errors.As(cause, &resume) || errors.As(cause, &confirmation) {
		return "refused"
	}
	return "failed"
}

type lifecycleFailureError struct {
	cause     error
	nonce     string
	path      string
	saveError error
}

func (e *lifecycleFailureError) Unwrap() error { return errors.Join(e.cause, e.saveError) }
func (e *lifecycleFailureError) Error() string {
	text := fmt.Sprintf("%s [failure %s; diagnostics %s]", e.cause.Error(), e.nonce, e.path)
	if e.saveError != nil {
		text += "; diagnostics could not be saved: " + e.saveError.Error()
	}
	return text
}

// FailureNonce recovers diagnostic identity through ordinary wrapped errors.
func FailureNonce(err error) string {
	var failed *lifecycleFailureError
	if errors.As(err, &failed) {
		return failed.nonce
	}
	return ""
}

type operationAttemptContextKey struct{}

// WithOperationAttempt makes nested operations borrow one terminal owner.
func WithOperationAttempt(ctx context.Context, attempt *OperationAttempt) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, operationAttemptContextKey{}, attempt)
}

// OperationAttemptFromContext returns the existing terminal owner, if any.
func OperationAttemptFromContext(ctx context.Context) *OperationAttempt {
	if ctx == nil {
		return nil
	}
	attempt, _ := ctx.Value(operationAttemptContextKey{}).(*OperationAttempt)
	return attempt
}
