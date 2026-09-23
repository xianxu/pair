package couchcore

import (
	"errors"
	"os"
)

// PreviewSnapshot uses the regular decoder/routing without initializing stores,
// registering retention ownership or recovering an interrupted transaction.
func (s *ThreadStore) PreviewSnapshot() (ThreadSnapshot, error) {
	if s == nil {
		return ThreadSnapshot{}, errors.New("thread store unavailable")
	}
	view := *s
	view.readOnly = true
	return view.Snapshot()
}
func (s *ThreadStore) PreviewPathLaunchPreference(repoIdentity, path string) (PathLaunchPreference, bool, error) {
	if s == nil {
		return PathLaunchPreference{}, false, errors.New("thread store unavailable")
	}
	view := *s
	view.readOnly = true
	return view.GetPathLaunchPreference(repoIdentity, path)
}
func (s *ThreadStore) withPreviewLock(fn func() error) (err error) {
	if err := s.validateBackendPath(); err != nil {
		return err
	}
	if _, err := os.Lstat(s.root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	lock, err := s.retentionReadLock()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if _, err := os.Lstat(s.journalPath()); err == nil {
		return errors.New("thread store recovery pending; open the workspace before previewing")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return fn()
}
