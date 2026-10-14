package couchtty

import "context"

// NotifyDesktop writes a supervisor-owned notification, separate from child
// effects. The broker proves hub authority before calling this method.
func (c *Console) NotifyDesktop(ctx context.Context, title, body string) error {
	if err := c.lifetime.Err(); err != nil {
		return err
	}
	return c.presenter.Notify(ctx, title, body)
}
