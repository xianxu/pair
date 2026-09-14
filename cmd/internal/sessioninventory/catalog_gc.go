package sessioninventory

import (
	"errors"
	"os"
	"path/filepath"
)

// Invalidate removes only the derived catalog under its publisher's stable
// lock. Native agent stores and other files are never part of this operation.
func (s CatalogStore) Invalidate(path string) (err error) {
	if path == "" || s.Runtime == nil {
		return errors.New("catalog invalidation needs path and runtime")
	}
	lock, err := s.Runtime.Lock(path + ".lock")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err := s.Runtime.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.Runtime.SyncDirectory(filepath.Dir(path))
}
