package pairlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// logEffects observes the store's actual publication under its existing lock.
// A retry that only syncs existing bytes is not a new authored-content effect.
type logEffects struct {
	Runtime
	changed bool
}

func (r *logEffects) Rename(from, to string) error {
	err := r.Runtime.Rename(from, to)
	if err == nil {
		r.changed = true
	}
	return err
}

func managedLogWrite(getenv func(string) string, path string, write func(SessionLogStore) error) error {
	if getenv("PAIR_DATA_DIR") == "" && getenv("PAIR_TAG") == "" {
		return write(SessionLogStore{Runtime: OSRuntime{}})
	}
	owner, err := storagegc.SelectedOwner(getenv("PAIR_DATA_DIR"), getenv("PAIR_SCOPE_KEY"), getenv("PAIR_TAG"))
	if err != nil {
		return fmt.Errorf("retention owner: %w", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("retention log directory: %w", err)
	}
	if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("retention: unsafe log symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	path = filepath.Join(parent, filepath.Base(path))
	paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if path != paths.Log() {
		return fmt.Errorf("retention: log path does not match selected owner")
	}
	lease, err := storagegc.AcquireSelectedProcess(context.Background(), getenv, "prompt-log-writer")
	if err != nil {
		return fmt.Errorf("retention registration: %w", err)
	}
	defer lease.Close()
	process, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		return err
	}
	id, err := lease.Coordinator.BeginUse(context.Background(), owner, process, path)
	if err != nil {
		return fmt.Errorf("retention intent: %w", err)
	}
	effects := &logEffects{Runtime: OSRuntime{}}
	writeErr := write(SessionLogStore{Runtime: effects})
	if writeErr != nil && commitoutcome.Of(writeErr) != commitoutcome.Committed {
		return writeErr
	}
	if effects.changed {
		err = lease.Coordinator.CompleteUse(context.Background(), owner, id)
	} else {
		err = lease.Coordinator.CancelUnchangedUse(context.Background(), owner, id)
	}
	if err != nil {
		return fmt.Errorf("retention completion: %w", err)
	}
	return writeErr
}
