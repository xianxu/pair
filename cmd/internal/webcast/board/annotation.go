package board

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

// AnnotationCleanup acknowledges an ordered clear. id is stable across retries.
type AnnotationCleanup func(context.Context, uint64) error

// AnnotationAdmission captures authority before reading a request body.
// Its mutations recheck the permit at dispatch, including across Off/On cycles.
type AnnotationAdmission struct {
	board     *Board
	authority annotationAuthority
}

// annotationAuthority is issued only by this Board. A zero value grants nothing.
type annotationAuthority struct {
	kind       annotationAuthorityKind
	permit     capability.Permit
	generation uint64
}
type annotationAuthorityKind uint8

const (
	viewerAnnotation annotationAuthorityKind = iota
	ownerAnnotation
	ownerSketch
)

// AdmitOperatorAnnotationGeneration is for authenticated operator ingress only.
// Unlike viewer permission, settled OFF still admits the cast owner's tools.
func (b *Board) AdmitOperatorAnnotationGeneration(expected uint64) (AnnotationAdmission, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a := AnnotationAdmission{board: b, authority: annotationAuthority{kind: ownerAnnotation, generation: expected}}
	if err := b.admissionLocked(a.authority); err != nil {
		return AnnotationAdmission{}, err
	}
	return a, nil
}

func (b *Board) admitOperatorAnnotation() (AnnotationAdmission, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a := AnnotationAdmission{board: b, authority: annotationAuthority{kind: ownerAnnotation, generation: b.annotation.Generation}}
	if err := b.admissionLocked(a.authority); err != nil {
		return AnnotationAdmission{}, err
	}
	return a, nil
}

type annotationAttempt struct {
	done chan struct{}
	err  error
}

// WithAnnotationCleanup configures cleanup before the Board is used.
func (b *Board) WithAnnotationCleanup(fn AnnotationCleanup) *Board {
	b.mu.Lock()
	b.annotationCleanup = fn
	b.mu.Unlock()
	return b
}

func (b *Board) AdmitAnnotation() (AnnotationAdmission, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.admitAnnotationLocked()
}

// AdmitAnnotationGeneration binds externally originated work to the authority
// observed by its sender, rather than granting fresh authority on arrival.
func (b *Board) AdmitAnnotationGeneration(expected uint64) (AnnotationAdmission, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if expected != b.annotation.Generation {
		return AnnotationAdmission{}, ErrDisabled
	}
	return b.admitAnnotationLocked()
}

// AdmitOperatorSketchGeneration is issued only to the trusted console sketch
// ingress. Viewer annotation admission remains gated by its capability phase.
func (b *Board) AdmitOperatorSketchGeneration(expected uint64) (AnnotationAdmission, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return AnnotationAdmission{}, ErrClosed
	}
	if expected != b.annotation.Generation {
		return AnnotationAdmission{}, ErrDisabled
	}
	return AnnotationAdmission{board: b, authority: annotationAuthority{kind: ownerSketch, generation: expected}}, nil
}

func (b *Board) admitAnnotationLocked() (AnnotationAdmission, error) {
	if b.closed {
		return AnnotationAdmission{}, ErrClosed
	}
	p, ok := b.annotation.Permit()
	if !ok {
		return AnnotationAdmission{}, ErrDisabled
	}
	return AnnotationAdmission{board: b, authority: annotationAuthority{permit: p}}, nil
}

// Check rechecks captured authority without acquiring the dispatch lock. The
// recorder uses it at its ordered write boundary: taking command there would
// deadlock cleanup, which holds command while awaiting the recorder.
func (a AnnotationAdmission) Check() error {
	if a.board == nil {
		return ErrDisabled
	}
	a.board.mu.Lock()
	defer a.board.mu.Unlock()
	return a.board.authorityLocked(a.authority, true)
}

func (b *Board) admissionLocked(a annotationAuthority) error {
	return b.authorityLocked(a, false)
}

// surfaceAdmissionLocked prevents trusted owner authority from being used by
// ordinary viewer entry points, and viewer grants from entering owner surfaces.
func (b *Board) surfaceAdmissionLocked(a annotationAuthority, operator bool) error {
	if (operator && a.kind != ownerAnnotation) || (!operator && a.kind != viewerAnnotation) {
		return ErrDisabled
	}
	return b.admissionLocked(a)
}

func (b *Board) authorityLocked(a annotationAuthority, sketch bool) error {
	if b.closed {
		return ErrClosed
	}
	switch a.kind {
	case ownerAnnotation:
		if a.generation == b.annotation.Generation && (b.annotation.Phase == capability.Off || b.annotation.Phase == capability.On) {
			return nil
		}
	case ownerSketch:
		// Console sketches historically use generation fencing independently of
		// viewer cleanup. Do not extend this policy to owner Note or Pointer.
		if sketch && a.generation == b.annotation.Generation {
			return nil
		}
	case viewerAnnotation:
		if b.annotation.Admits(a.permit) {
			return nil
		}
	}
	return ErrDisabled
}

func (b *Board) AnnotationCapability() capability.Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.annotationSnapshotLocked()
}
func (b *Board) annotationSnapshotLocked() capability.Snapshot {
	return capability.Snapshot{CapabilityState: b.annotation, Problem: b.annotationProblem}
}
func (b *Board) publishAnnotationLocked() {
	for ch := range b.annotationSubs {
		select {
		case <-ch:
		default:
		}
		ch <- b.annotationSnapshotLocked()
	}
}

// SubscribeAnnotationCapability retains only the latest state for slow readers.
func (b *Board) SubscribeAnnotationCapability() (<-chan capability.Snapshot, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan capability.Snapshot, 1)
	ch <- b.annotationSnapshotLocked()
	if b.closed {
		close(ch)
		return ch, func() {}
	}
	b.annotationSubs[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.annotationSubs[ch]; ok {
			delete(b.annotationSubs, ch)
			close(ch)
		}
	}
}

func (b *Board) SetAnnotationAllowed(ctx context.Context, allowed bool) error {
	kind := capability.Disable
	if allowed {
		kind = capability.Enable
	}
	return b.changeAnnotation(ctx, kind)
}
func (b *Board) RetryAnnotations(ctx context.Context) error {
	return b.changeAnnotation(ctx, capability.Retry)
}

func (b *Board) changeAnnotation(ctx context.Context, kind capability.EventKind) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	if kind != capability.Enable && b.annotation.Phase == capability.Disabling && b.annotation.Outcome == capability.Running {
		attempt := b.annotationAttempt
		b.mu.Unlock()
		select {
		case <-attempt.done:
			return attempt.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	next, err := capability.NextCapability(b.annotation, capability.Event{Kind: kind})
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if next == b.annotation {
		b.mu.Unlock()
		return nil
	}
	b.annotation = next
	b.publishAnnotationLocked()
	if next.Phase != capability.Disabling {
		b.mu.Unlock()
		return nil
	}
	attempt := &annotationAttempt{done: make(chan struct{})}
	b.annotationAttempt = attempt
	cleanup := b.annotationCleanup
	b.mu.Unlock()

	// Admission was fenced before waiting for a mutation already dispatching.
	// Put takes only the leaf mutex, so synchronous writers may call it here.
	b.command.Lock()
	if cleanup == nil {
		err = &capability.Failure{Outcome: capability.Broken, Cause: errors.New("board: annotation cleanup is unavailable")}
	} else {
		err = cleanup(ctx, next.Generation)
	}
	b.command.Unlock()

	b.mu.Lock()
	event := capability.Event{Kind: capability.CleanupSucceeded, Attempt: next.Attempt}
	if err != nil {
		event.Kind = capability.CleanupFailed
		event.Outcome = capability.FailureOutcome(err)
	}
	settled, _ := capability.NextCapability(b.annotation, event)
	if settled != b.annotation {
		b.annotation = settled
		b.annotationProblem = ""
		if err != nil {
			b.annotationProblem = err.Error()
		}
		b.publishAnnotationLocked()
	}
	attempt.err = err
	close(attempt.done)
	b.mu.Unlock()
	return err
}
