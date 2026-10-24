package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// startFreshPrimary consumes the pre-launch observation. It must not re-observe
// after launching: its own child would turn a fresh start into a normal restart.
func startFreshPrimary(ctx context.Context, c *couchcore.Couch, startup couchcore.StartupObservation, root string, start couchcore.StartResult) (couchcore.StartResult, error) {
	if startup.Kind != couchcore.StartupFresh || root == "" || start.Handle != nil {
		return start, nil
	}
	return c.StartFreshHub(ctx, root)
}
