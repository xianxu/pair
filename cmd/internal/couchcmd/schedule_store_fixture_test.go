package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

// Compatibility fixture transactions acquire the same capability as production.
func updateScheduleFixture(store couchschedule.StateStore, ctx context.Context, change func(*couchschedule.Ledger) error) error {
	owner, err := store.Own(ctx)
	if err != nil {
		return err
	}
	defer owner.Close()
	return owner.Update(ctx, change)
}
func withScheduleFixtureAuthority(store couchschedule.StateStore, ctx context.Context, inspect func(couchschedule.Ledger) error) error {
	owner, err := store.Own(ctx)
	if err != nil {
		return err
	}
	defer owner.Close()
	ledger, err := owner.ViewDurable(ctx)
	if err != nil {
		return err
	}
	return inspect(ledger)
}
