package couchschedule

import (
	"context"
	"encoding/json"
	"errors"
	"os"
)

const maxScheduleStateBytes = 2 * 1024 * 1024

// Ledger owns the scheduler snapshot. original retains future key/spec members
// which have no storage field in the pure routing/declaration value types.
type Ledger struct {
	Version  int
	Tasks    map[string]TaskState
	Unknown  map[string]json.RawMessage
	original map[string]json.RawMessage
}

type StateStore struct {
	Dir           string
	BeforePersist func() error
	// BeforeDirectorySync injects a post-rename durability fault in store tests.
	BeforeDirectorySync func() error
}

func emptyLedger() Ledger { return Ledger{Version: 1, Tasks: map[string]TaskState{}} }

// View is read-only, including when the namespace has never been created.
func (s StateStore) View(ctx context.Context) (Ledger, error) {
	dir, err := openScheduleDir(ctx, s.Dir, false)
	if errors.Is(err, os.ErrNotExist) {
		return emptyLedger(), nil
	}
	if err != nil {
		return Ledger{}, err
	}
	defer dir.Close()
	return readLedger(ctx, dir)
}

func readLedger(ctx context.Context, dir *os.File) (Ledger, error) {
	raw, err := readScheduleFile(ctx, dir, "state.json", maxScheduleStateBytes)
	if errors.Is(err, os.ErrNotExist) {
		return emptyLedger(), nil
	}
	if err != nil {
		return Ledger{}, err
	}
	return DecodeLedger(raw)
}

// Update serializes fresh read/reduce/publish under one permanent lock inode.
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
	state, err := readLedger(ctx, dir)
	if err != nil {
		return err
	}
	if err = change(&state); err != nil {
		return err
	}
	raw, err := EncodeLedger(state)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if s.BeforePersist != nil {
		if err = s.BeforePersist(); err != nil {
			return err
		}
	}
	return publishSchedule(ctx, dir, raw, s.BeforeDirectorySync)
}
