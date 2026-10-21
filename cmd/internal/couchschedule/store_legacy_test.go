package couchschedule

import (
	"context"
	"errors"
)

func (s StateStore) WithAuthority(ctx context.Context, inspect func(Ledger) error) error {
	if inspect == nil {
		return errors.New("schedule authority requires callback")
	}
	dir, err := openScheduleDir(ctx, s.Dir, true)
	if err != nil {
		return err
	}
	defer dir.Close()
	release, err := lockSchedule(ctx, dir)
	if err != nil {
		return err
	}
	defer release()
	state, err := readLedger(ctx, dir)
	if err != nil {
		return err
	}
	return inspect(state)
}

func (s StateStore) Update(ctx context.Context, change func(*Ledger) error) error {
	if change == nil {
		return errors.New("schedule update requires callback")
	}
	dir, err := openScheduleDir(ctx, s.Dir, true)
	if err != nil {
		return err
	}
	defer dir.Close()
	release, err := lockSchedule(ctx, dir)
	if err != nil {
		return err
	}
	defer release()
	return s.updateHeld(ctx, dir, change)
}
