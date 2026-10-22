package drive

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var ErrDriveUnavailable = errors.New("drive: authority unavailable")

// DriveAdmission cannot be constructed by a viewer. It fences delayed bodies
// and queued dispatch against the issuing controller's authority generation.
type DriveAdmission struct {
	owner  *MemController
	permit capability.Permit
}

func (f *MemController) AdmitDrive() (DriveAdmission, error) {
	return f.admitDrive(nil)
}

// AdmitDriveGeneration binds a browser operation to its originating authority.
func (f *MemController) AdmitDriveGeneration(generation uint64) (DriveAdmission, error) {
	return f.admitDrive(&generation)
}
func (f *MemController) admitDrive(generation *uint64) (DriveAdmission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.authority.Permit()
	if !ok || f.closed || f.busy != nil || (generation != nil && *generation != f.authority.Generation) {
		return DriveAdmission{}, ErrDriveUnavailable
	}
	return DriveAdmission{f, p}, nil
}
func (a DriveAdmission) Request(key string) error { return a.viewerOp(key, recording.CtlRequest) }

// Release withdraws only control admitted in this authority generation.
func (a DriveAdmission) Release(key string) error { return a.viewerOp(key, recording.CtlRelease) }
func (a DriveAdmission) viewerOp(key, op string) error {
	if a.owner == nil {
		return ErrDriveUnavailable
	}
	f := a.owner
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authority.Admits(a.permit) || f.busy != nil {
		return ErrDriveUnavailable
	}
	s, err := f.bySeat(key)
	if err != nil {
		return err
	}
	return f.changeLocked(op, s.Viewer)
}

// Operator actions retain the admission captured on the terminal input loop.
// Validation and changeLocked share mu, so scheduling cannot refresh authority.
func (a DriveAdmission) Grant(viewer string) error { return a.operatorOp(viewer, recording.CtlGrant) }
func (a DriveAdmission) Deny(viewer string) error  { return a.operatorOp(viewer, recording.CtlDeny) }
func (a DriveAdmission) Revoke() error             { return a.operatorOp("", recording.CtlRevoke) }
func (a DriveAdmission) operatorOp(viewer, op string) error {
	if a.owner == nil {
		return ErrDriveUnavailable
	}
	f := a.owner
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed || !f.authority.Admits(a.permit) || f.busy != nil {
		return ErrDriveUnavailable
	}
	if op == recording.CtlRevoke {
		if f.control.Driver == nil {
			return nil
		}
		return f.changeLocked(op, *f.control.Driver)
	}
	s := f.seats[viewer]
	if s == nil {
		return ErrNoSeat
	}
	return f.changeLocked(op, s.Viewer)
}

func (a DriveAdmission) Input(ctx context.Context, key string, b Batch) error {
	if a.owner == nil {
		return ErrDriveUnavailable
	}
	f := a.owner
	f.mu.Lock()
	s, err := f.bySeat(key)
	if err == nil && (!f.authority.Admits(a.permit) || f.busy != nil) {
		err = ErrDriveUnavailable
	}
	if err == nil && f.control.Driving() != s.Viewer.ID {
		err = ErrNotDriver
	}
	if err == nil {
		f.work.Add(1)
	}
	f.mu.Unlock()
	if err != nil {
		return err
	}
	// Registration and Close's end fence share mu: every admitted dispatch is
	// canceled by controller lifetime and joined through its settled outcome.
	defer f.work.Done()
	inputCtx, cancel := context.WithCancel(ctx)
	stopLife := context.AfterFunc(f.life, cancel)
	defer stopLife()
	defer cancel()
	b.Admit = func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return !f.closed && f.busy == nil && f.authority.Admits(a.permit) && f.control.Driving() == s.Viewer.ID
	}
	err = f.inj.Inject(inputCtx, s.Viewer.ID, b)
	var releaseErr *ReleaseFailure
	if errors.As(err, &releaseErr) {
		f.mu.Lock()
		if f.authority.Admits(a.permit) {
			f.freezeLocked(err, s.Viewer.ID)
		}
		f.mu.Unlock()
	}
	return err
}

// WithControlReset installs an acknowledged, ordered, idempotent reset. Configure
// before use. A missing reset adapter deliberately fails closed.
func (f *MemController) WithControlReset(reset func(context.Context, uint64) error) *MemController {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reset = reset
	return f
}
func (f *MemController) DriveCapability() capability.Snapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authority
}
func (f *MemController) SubscribeDriveCapability() (<-chan capability.Snapshot, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan capability.Snapshot, 1)
	ch <- f.authority
	if f.closed {
		close(ch)
		return ch, func() {}
	}
	f.capSubs[ch] = struct{}{}
	return ch, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if _, ok := f.capSubs[ch]; ok {
			delete(f.capSubs, ch)
			close(ch)
		}
	}
}
func (f *MemController) publishCapabilityLocked() {
	for ch := range f.capSubs {
		select {
		case <-ch:
		default:
		}
		ch <- f.authority
	}
}
func (f *MemController) SetDriveAllowed(ctx context.Context, allowed bool) error {
	kind := capability.Disable
	if allowed {
		kind = capability.Enable
	}
	return f.driveTransition(ctx, kind)
}
func (f *MemController) RetryDrive(ctx context.Context) error {
	return f.driveTransition(ctx, capability.Retry)
}
func (f *MemController) driveTransition(ctx context.Context, kind capability.EventKind) error {
	f.mu.Lock()
	next, err := capability.NextCapability(f.authority.CapabilityState, capability.Event{Kind: kind})
	if err != nil {
		f.mu.Unlock()
		return err
	}
	if next == f.authority.CapabilityState {
		f.mu.Unlock()
		return nil
	}
	if next.Phase == capability.Disabling {
		f.beginDrawingTransitionLocked()
	}
	f.authority = capability.Snapshot{CapabilityState: next}
	f.publishCapabilityLocked()
	if next.Phase != capability.Disabling {
		f.mu.Unlock()
		return nil
	}
	if kind == capability.Disable {
		f.cleanupBy = f.control.Driving()
		f.resetDone = false
	}
	done := make(chan error, 1)
	wait := f.busy
	f.work.Add(1)
	cleanupCtx, cancel := context.WithTimeout(f.life, releaseWait)
	stopCaller := context.AfterFunc(ctx, cancel)
	go func() {
		defer f.work.Done()
		defer cancel()
		defer stopCaller()
		f.cleanup(cleanupCtx, next, wait, done)
	}()
	f.mu.Unlock()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (f *MemController) cleanup(ctx context.Context, state capability.CapabilityState, wait <-chan struct{}, done chan error) {
	var err error
	if wait != nil {
		select {
		case <-wait:
		case <-ctx.Done():
			err = &capability.Failure{Outcome: capability.Retryable, Cause: ctx.Err()}
		}
	}
	f.mu.Lock()
	reset, resetDone, by := f.reset, f.resetDone, f.cleanupBy
	// Control can settle after a canceled cleanup wait. Its unobserved
	// outcome survives the busy channel and must be consumed before reset,
	// including by a later retry that no longer has a channel to wait on.
	if f.busy == nil && f.busyErr != nil {
		err = f.busyErr
		f.busyErr = nil
	}
	f.mu.Unlock()
	if err == nil && !resetDone {
		if reset == nil {
			err = errors.New("drive: acknowledged control reset is unavailable")
		} else {
			err = reset(ctx, state.Generation)
		}
		if err == nil {
			f.mu.Lock()
			f.resetDone = true
			f.control = recording.Control{}
			for ch := range f.subs {
				offer(ch, f.control)
			}
			f.mu.Unlock()
		}
	}
	if err == nil && by != "" {
		if rel, ok := f.inj.(Releaser); ok {
			err = rel.Release(ctx, by)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	event := capability.Event{Kind: capability.CleanupSucceeded, Attempt: state.Attempt}
	if err != nil {
		event.Kind = capability.CleanupFailed
		event.Outcome = capability.FailureOutcome(err)
	}
	f.authority.CapabilityState, _ = capability.NextCapability(f.authority.CapabilityState, event)
	if err == nil {
		f.renewDrawingLocked()
	}
	if f.authority.Phase != capability.Ended {
		f.authority.Problem = ""
		if err != nil {
			f.authority.Problem = err.Error()
		}
		f.publishCapabilityLocked()
	}
	done <- err
	close(done)
}
func (f *MemController) freezeLocked(err error, by string) {
	f.freezeDrawingLocked()
	if f.authority.Phase != capability.On {
		return
	}
	// This failure is being surfaced directly in authority; cleanup need
	// not report that same settled control outcome again on explicit retry.
	f.busyErr = nil
	s, _ := capability.NextCapability(f.authority.CapabilityState, capability.Event{Kind: capability.Disable})
	s, _ = capability.NextCapability(s, capability.Event{Kind: capability.CleanupFailed, Attempt: s.Attempt, Outcome: capability.FailureOutcome(err)})
	f.authority = capability.Snapshot{CapabilityState: s, Problem: err.Error()}
	f.cleanupBy = by
	f.resetDone = false
	f.publishCapabilityLocked()
}
