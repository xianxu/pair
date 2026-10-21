package operationalrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// PublishOccurrence records the first immutable result for a run key. An
// existing same-key bundle is canonical, including its original execution facts,
// output and delivery metadata. Publish retains its stricter equality contract.
func (s Store) PublishOccurrence(ctx context.Context, key RunKey, proposed Record, output []byte) (Record, []byte, error) {
	record, canonical, absent, err := s.readOccurrence(ctx, key)
	if !absent {
		return record, canonical, err
	}
	if err = s.fail("occurrence-absent"); err != nil {
		return Record{}, nil, err
	}
	publishErr := s.Publish(ctx, key, proposed, output)
	// A competing writer, or a publication whose directory sync failed after
	// rename, may have installed the immutable winner. Re-read and sync it rather
	// than comparing regenerated timestamps or overwriting its facts.
	record, canonical, _, err = s.readOccurrence(ctx, key)
	if err != nil {
		return Record{}, nil, errors.Join(publishErr, err)
	}
	return record, canonical, nil
}

// ReadDurable reads an existing immutable bundle under the task lock and
// confirms its directory durability before permitting consumption. It never
// creates missing history or repairs damaged payloads. Read is inspection-only.
func (s Store) ReadDurable(ctx context.Context, key RunKey) (Record, []byte, error) {
	record, output, _, err := s.readOccurrence(ctx, key)
	return record, output, err
}

// absent distinguishes a wholly missing bundle from damage or a failure while
// syncing an existing one. Only the former permits a publication attempt.
func (s Store) readOccurrence(ctx context.Context, key RunKey) (record Record, output []byte, absent bool, err error) {
	task, err := s.task(key)
	if err != nil {
		return record, nil, false, err
	}
	run := filepath.Join(task, "runs", key.ID)
	unlock, err := taskLock(ctx, task, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_, probe := os.Lstat(run)
			absent = errors.Is(probe, os.ErrNotExist)
		}
		return record, nil, absent, err
	}
	defer unlock()
	record, output, err = s.read(key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_, probe := os.Lstat(run)
			absent = errors.Is(probe, os.ErrNotExist)
		}
		return record, nil, absent, err
	}
	if err = ctx.Err(); err != nil {
		return record, output, false, err
	}
	// The payloads were synced before their bundle's rename. Confirm the run
	// and every ancestor link through the namespace before reporting durable
	// adoption of a lost publication/finalization acknowledgement.
	if err = s.fail("occurrence-sync"); err != nil {
		return record, output, false, err
	}
	err = s.syncRunDirectories(run)
	return record, output, false, err
}
