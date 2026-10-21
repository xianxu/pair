package couchschedule

import (
	"context"
	"errors"
	"os"
)

// StateOwner is the sole namespace writer capability. It must not be copied.
// The channel only makes method/Close lifetimes safe; scheduler serialization
// belongs to its single owner loop. No operation reacquires the lifetime flock.
// The caller must join its workers before Close releases namespace authority.
type StateOwner struct {
	self    *StateOwner
	store   StateStore
	dir     *os.File
	release func()
	access  chan struct{}
	closed  bool
}

// Own acquires the existing permanent .state.lock for the returned lifetime.
// Contenders cannot inspect or recover ledger stages before acquiring it.
func (s StateStore) Own(ctx context.Context) (*StateOwner, error) {
	dir, err := openScheduleDir(ctx, s.Dir, true)
	if err != nil {
		return nil, err
	}
	release, err := lockSchedule(ctx, dir)
	if err != nil {
		dir.Close()
		return nil, err
	}
	owner := &StateOwner{store: s, dir: dir, release: release, access: make(chan struct{}, 1)}
	owner.self = owner
	owner.access <- struct{}{}
	if _, err = owner.ViewDurable(ctx); err != nil {
		owner.Close()
		return nil, err
	}
	return owner, nil
}

func (o *StateOwner) enter(ctx context.Context) error {
	if o == nil || o.self != o {
		return errors.New("schedule owner is nil or copied")
	}
	if ctx == nil {
		return errors.New("schedule owner requires context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-o.access:
	}
	if o.closed {
		o.leave()
		return errors.New("schedule owner is closed")
	}
	if err := ctx.Err(); err != nil {
		o.leave()
		return err
	}
	return nil
}
func (o *StateOwner) leave() { o.access <- struct{}{} }

// Update reads and publishes relative to the held namespace descriptor. A
// post-rename error never becomes an in-memory success; subsequent operations
// reread canonical state and ViewDurable explicitly confirms directory sync.
func (o *StateOwner) Update(ctx context.Context, change func(*Ledger) error) error {
	if change == nil {
		return errors.New("schedule update requires callback")
	}
	if err := o.enter(ctx); err != nil {
		return err
	}
	defer o.leave()
	return o.store.updateHeld(ctx, o.dir, change)
}

// ViewDurable validates canonical state and confirms namespace durability.
// It recovers only admitted staging residues under the retained owner fence.
func (o *StateOwner) ViewDurable(ctx context.Context) (Ledger, error) {
	if err := o.enter(ctx); err != nil {
		return Ledger{}, err
	}
	defer o.leave()
	state, err := readLedger(ctx, o.dir)
	if err != nil {
		return Ledger{}, err
	}
	if err = cleanupLedgerStages(ctx, o.dir, o.store.stateStep); err != nil {
		return Ledger{}, err
	}
	if err = ctx.Err(); err != nil {
		return Ledger{}, err
	}
	if o.store.BeforeDirectorySync != nil {
		if err = o.store.BeforeDirectorySync(); err != nil {
			return Ledger{}, err
		}
	}
	if err = o.dir.Sync(); err != nil {
		return Ledger{}, err
	}
	return state, nil
}

// Close is idempotent on the original handle and refuses copied handles.
func (o *StateOwner) Close() error {
	if o == nil || o.self != o {
		return errors.New("schedule owner is nil or copied")
	}
	<-o.access
	defer o.leave()
	if o.closed {
		return nil
	}
	o.closed = true
	o.release()
	return o.dir.Close()
}
