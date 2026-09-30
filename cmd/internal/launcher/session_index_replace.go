package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// ReplaceSessionNameIndex publishes the current Couch terminal association.
// Scope/tag is the replacement key; another scope's identical tag is untouched.
// The same root coordinator serializes ordinary append and retention cleanup.
func (r OSRuntime) ReplaceSessionNameIndex(entry SessionNameEntry) error {
	if err := validateSessionNameEntry(entry); err != nil {
		return err
	}
	owner, err := storagegc.SelectedOwner(r.DataDir, "", entry.Tag)
	if err != nil {
		return err
	}
	coordinator, err := storagegc.NewCoordinator(owner.DataDir)
	if err != nil {
		return err
	}
	return coordinator.WithLock(context.Background(), func(held *storagegc.Locked) error {
		scope, err := artifactpath.ResolveSelectedScope(r.DataDir)
		if err != nil {
			return err
		}
		path := scope.SessionBindings()
		var raw string
		existing, err := r.ReadFile(path)
		if err == nil {
			raw = existing
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		next, err := replaceSessionNameAssociation(raw, entry)
		if err != nil {
			return err
		}
		if err := r.WriteAtomic(path, next); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := errors.Join(file.Sync(), file.Close()); err != nil {
			return err
		}
		dir, err := os.Open(r.DataDir)
		if err != nil {
			return err
		}
		return errors.Join(dir.Sync(), dir.Close())
	})
}

func replaceSessionNameAssociation(raw string, replacement SessionNameEntry) (string, error) {
	index, err := DecodeSessionNameIndex(raw)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, entry := range index.Entries {
		if entry.ScopeKey == replacement.ScopeKey && entry.Tag == replacement.Tag {
			continue
		}
		if entry.SessionName == replacement.SessionName {
			return "", fmt.Errorf("couch session name %q belongs to another address", replacement.SessionName)
		}
		line, err := BuildSessionNameIndexLine(entry)
		if err != nil {
			return "", err
		}
		lines = append(lines, line)
	}
	line, err := BuildSessionNameIndexLine(replacement)
	if err != nil {
		return "", err
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n") + "\n", nil
}
