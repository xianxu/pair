package couchcore

import (
	"context"
	"time"

	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

func (c *Couch) trackedStartupBudget(ctx context.Context, resume bool) (context.Context, time.Time, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, time.Time{}, func() {}, err
	}
	now := time.Now()
	selected, _ := ctx.Deadline()
	fallback := startupbudget.Default
	// The explicit failure-fixture policy remains independently injectable.
	if resume && c.resumeRegistrationTimeout > 0 && c.resumeRegistrationTimeout != startupbudget.Default {
		fallback = c.resumeRegistrationTimeout
		if d := now.Add(fallback); selected.IsZero() || d.Before(selected) {
			selected = d
		}
	}
	selected, err := startupbudget.Select(now, selected, fallback)
	if err != nil {
		return nil, time.Time{}, func() {}, err
	}
	bounded, stop := context.WithDeadline(ctx, selected)
	return bounded, selected, stop, nil
}
