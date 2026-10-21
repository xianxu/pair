package operationalrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/durablefile"
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
	// The payloads were synced before their bundle's rename. Sync both parents
	// as well as the run directory to finish a lost publication/finalization
	// acknowledgement before reporting durable adoption.
	if err = s.fail("occurrence-sync"); err != nil {
		return record, output, false, err
	}
	err = errors.Join(durablefile.SyncDirectory(run), durablefile.SyncDirectory(filepath.Dir(run)), durablefile.SyncDirectory(task))
	return record, output, false, err
}
