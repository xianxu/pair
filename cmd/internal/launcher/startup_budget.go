package launcher

import (
	"context"
	"time"

	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

// Lifetime owns the interactive client; startup owns admission only.
func terminalStartupContext(lifetime, selected context.Context) (context.Context, context.CancelFunc, error) {
	if err := lifetime.Err(); err != nil {
		return nil, func() {}, err
	}
	parent := lifetime
	deadline, _ := lifetime.Deadline()
	if selected != nil {
		if err := selected.Err(); err != nil {
			return nil, func() {}, err
		}
		parent = selected
		if d, ok := selected.Deadline(); ok && (deadline.IsZero() || d.Before(deadline)) {
			deadline = d
		}
	}
	deadline, err := startupbudget.Select(time.Now(), deadline, startupbudget.Default)
	if err != nil {
		return nil, func() {}, err
	}
	ctx, cancel := context.WithDeadline(parent, deadline)
	stop := context.AfterFunc(lifetime, cancel)
	return ctx, func() { stop(); cancel() }, nil
}
