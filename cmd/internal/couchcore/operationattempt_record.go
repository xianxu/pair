package couchcore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// LifecycleFailureRecorder owns terminal publication and the ordering evidence
// retained while attempts on the same slot overlap.
type LifecycleFailureRecorder struct {
	store    LaunchFailureStorage
	now      func() time.Time
	nonce    func() (string, error)
	mu       sync.Mutex
	epoch    string
	next     uint64
	slots    map[string]*operationAttemptGroup
	snapshot lifecycleFailureSnapshotCache
}

type operationAttemptGroup struct {
	mu    sync.Mutex
	state OperationAttemptState
	refs  int // protected by the recorder mutex, never by the group mutex
}

// OperationAttempt owns one terminal result. Nested calls borrow it through
// context; only its admitting boundary calls Finish.
type OperationAttempt struct {
	recorder *LifecycleFailureRecorder
	group    *operationAttemptGroup
	key      string
	order    uint64
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

// NewLifecycleFailureRecorder injects storage, clock and nonce allocation.
func NewLifecycleFailureRecorder(store LaunchFailureStorage, now func() time.Time, nonce func() (string, error)) *LifecycleFailureRecorder {
	return &LifecycleFailureRecorder{store: store, now: now, nonce: nonce, slots: make(map[string]*operationAttemptGroup)}
}

// Begin freezes the exact target and admission order without inspecting a
// replacement thread or performing a lifecycle effect.
func (r *LifecycleFailureRecorder) Begin(op string, target LifecycleFailureTarget) (*OperationAttempt, error) {
	key := LifecycleFailureSlotKey(target)
	if key == "" {
		if target.Address.RepoScope == "" || target.Address.Tag == "" {
			return nil, errors.New("lifecycle diagnostics need an exact slot or thread target")
		}
		key = "thread:" + target.Address.RepoScope + "/" + string(target.Address.Tag)
	}
	r.mu.Lock()
	if r.epoch == "" {
		var err error
		r.epoch, err = r.nonce()
		if err != nil {
			r.mu.Unlock()
			return nil, err
		}
	}
	nonce, err := r.nonce()
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	r.next++
	order := r.next
	group := r.slots[key]
	if group == nil {
		group = &operationAttemptGroup{}
		r.slots[key] = group
	}
	group.refs++
	r.mu.Unlock()
	group.mu.Lock()
	group.state, _, err = AdvanceOperationAttempt(group.state, OperationAttemptEvent{Kind: AttemptAdmitted, Order: order})
	group.mu.Unlock()
	if err != nil {
		r.release(key, group)
		return nil, err
	}
	return &OperationAttempt{recorder: r, group: group, key: key, order: order, nonce: nonce, op: op, target: target}, nil
}

func (r *LifecycleFailureRecorder) release(key string, group *operationAttemptGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()
	group.refs--
	if group.refs == 0 && r.slots[key] == group {
		delete(r.slots, key)
	}
}

// ActiveSlots counts only slots with admitted attempts still to finish.
func (r *LifecycleFailureRecorder) ActiveSlots() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.slots)
}

// Nonce is the immutable operation identity carried by the failure and receipt.
func (a *OperationAttempt) Nonce() string { return a.nonce }

// DiagnosticWarning reports a failed optional publication without changing a
// successful lifecycle effect into a reason to repeat it.
func (a *OperationAttempt) DiagnosticWarning() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.warning
}

// RecordedFailureNonce includes partial cleanup evidence on successful effects.
func (a *OperationAttempt) RecordedFailureNonce() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failure {
		return a.nonce
	}
	return ""
}

// Finish is the sole terminal recorder. Repeated or concurrent completion
// returns the first answer and cannot publish or execute an effect again.
func (a *OperationAttempt) Finish(value any, cause error) (any, error) {
	a.once.Do(func() {
		evidenceCause := cause
		if parked, ok := value.(ParkResult); ok && evidenceCause == nil && parked.CleanupError != nil {
			evidenceCause = parked.CleanupError
		}
		err, warning := a.finish(cause, evidenceCause)
		a.mu.Lock()
		a.value, a.err, a.warning = value, err, warning
		a.failure = evidenceCause != nil
		a.mu.Unlock()
	})
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.value, a.err
}

func (a *OperationAttempt) finish(cause, evidenceCause error) (error, string) {
	r := a.recorder
	defer r.release(a.key, a.group)
	a.group.mu.Lock()
	defer a.group.mu.Unlock()
	at := r.now()
	kind := AttemptSucceeded
	if evidenceCause != nil {
		kind = AttemptFailed
	}
	next, decision, transitionErr := AdvanceOperationAttempt(a.group.state, OperationAttemptEvent{Kind: kind, Order: a.order, At: at})
	if transitionErr != nil {
		return errors.Join(cause, transitionErr), transitionErr.Error()
	}
	a.group.state = next
	ctx, cancel := context.WithTimeout(context.Background(), launchDiagnosticsAdmissionTimeout)
	defer cancel()
	if evidenceCause == nil {
		if r.store == nil {
			return nil, "lifecycle diagnostics unavailable; success was not repeated"
		}
		history, err := r.Snapshot()
		if err == nil {
			_, changed := LifecycleFailureHistory(history).Supersede(a.target, r.epoch, a.order, at)
			if !changed {
				return nil, ""
			}
			if store, ok := r.store.(LifecycleFailureStorage); ok {
				err = store.Supersede(ctx, a.target, r.epoch, a.order, at)
			} else {
				err = errors.New("lifecycle diagnostics do not support successful supersession")
			}
		}
		if err != nil {
			return nil, "lifecycle succeeded; diagnostic supersession failed: " + err.Error()
		}
		return nil, ""
	}
	f := lifecycleFailureForAttempt(a, evidenceCause, at)
	if cause == nil {
		f.Outcome = "unproven"
	}
	f.SupersededAt = decision.SupersededAt
	var saved error
	if r.store == nil {
		saved = errors.New("lifecycle diagnostic store unavailable")
	} else {
		saved = r.store.Save(ctx, f)
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
	f := LifecycleFailure{FailedAt: at, Outcome: "failed"}
	var launch *launchFailureError
	if errors.As(cause, &launch) {
		f = cloneLaunchFailure(launch.failure)
		f.LaunchNonce = f.Nonce
		f.Phase = "launch"
	}
	f.Nonce, f.Operation, f.AttemptEpoch, f.AttemptOrder = a.nonce, a.op, a.recorder.epoch, a.order
	if a.target.Address != (ThreadAddress{}) {
		f.Address = a.target.Address
	}
	if a.target.StartingPath != "" {
		f.StartingPath = a.target.StartingPath
	}
	f.Slot, f.ParkNonce, f.FinalError = a.target.Slot, a.target.ParkNonce, cause.Error()
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
