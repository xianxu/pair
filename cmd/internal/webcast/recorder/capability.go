package recorder

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ClearAnnotations acknowledges a clear only after its ordered sink write.
// id is the authority's disable generation, unchanged by a retry.
func (r *Recorder) ClearAnnotations(ctx context.Context, id uint64) error {
	return r.applyCapabilityCleanup(ctx, id, &r.annotationCleanup,
		&rec.AnnotationClear{Who: rec.Author{ID: rec.OperatorID}})
}

// ResetControl atomically removes driver and pending requests in the stream.
// Held input release is a separate, acknowledged controller operation.
func (r *Recorder) ResetControl(ctx context.Context, id uint64) error {
	return r.applyCapabilityCleanup(ctx, id, &r.driveCleanup,
		&rec.Ctl{Op: rec.CtlReset, Who: rec.Author{ID: rec.OperatorID}})
}

func (r *Recorder) applyCapabilityCleanup(ctx context.Context, id uint64, completed *uint64, record rec.Record) error {
	if id == 0 {
		return fmt.Errorf("recorder: cleanup requires an operation identity")
	}
	if err := ctx.Err(); err != nil {
		return &capability.Failure{Outcome: capability.Retryable, Cause: err}
	}
	var result error
	err := r.ask(ctx, func() {
		// The queue owns completed, reducer, sink and sticky recording error.
		if err := ctx.Err(); err != nil {
			result = &capability.Failure{Outcome: capability.Retryable, Cause: err}
			return
		}
		if r.err != nil {
			result = &capability.Failure{Outcome: capability.Broken, Cause: r.err}
			return
		}
		if r.m.State().Ended {
			result = errClosed
			return
		}
		if id < *completed {
			result = fmt.Errorf("recorder: stale cleanup generation")
			return
		}
		if id == *completed {
			return
		}
		if !r.opened {
			// No record has been admitted; do not create an empty recording.
			*completed = id
			return
		}
		if !r.m.Emit(r.t(r.opts.Now()), record) {
			result = fmt.Errorf("recorder: %s refused", record.Kind())
			return
		}
		r.write()
		if r.err != nil {
			// A sink may have accepted part of a write. Never retry that as
			// a fresh effect or claim success from the advanced reducer.
			result = &capability.Failure{Outcome: capability.Broken, Cause: r.err}
			return
		}
		*completed = id
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// Unlike CDP input, this local operation is safe to reconcile:
			// a committed ID is remembered even if its acknowledgement is lost.
			return &capability.Failure{Outcome: capability.Retryable, Cause: err}
		}
		return err
	}
	return result
}
