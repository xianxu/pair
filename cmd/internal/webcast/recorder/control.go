package recorder

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// WriteControl commits one ordinary control record on the recorder's writer.
// Success means reducer acceptance and completed sink writes. Once started,
// cancellation cannot turn an uncertain write into a retryable enqueue refusal.
func (r *Recorder) WriteControl(ctx context.Context, ctl *rec.Ctl) error {
	return r.acknowledged(ctx, func() error {
		if r.err != nil {
			return &capability.Failure{Outcome: capability.Broken, Cause: r.err}
		}
		if r.m.State().Ended {
			return drive.ErrClosed
		}
		if !r.opened {
			return drive.ErrNotLive
		}
		if ctl == nil {
			return &capability.Failure{Outcome: capability.Retryable, Cause: fmt.Errorf("recorder: nil control record")}
		}
		if !r.m.Emit(r.t(r.opts.Now()), ctl) {
			return &capability.Failure{Outcome: capability.Retryable, Cause: fmt.Errorf("%w: recorder refused control", rec.ErrControl)}
		}
		r.write()
		if r.err != nil {
			return &capability.Failure{Outcome: capability.Broken, Cause: r.err}
		}
		return nil
	})
}
