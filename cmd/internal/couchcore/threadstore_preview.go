package couchcore

import (
	"context"
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
func (s *ThreadStore) PreviewPathLaunchPreference(repoIdentity, path, scope string) (PathLaunchPreference, bool, error) {
	if s == nil {
		return PathLaunchPreference{}, false, errors.New("thread store unavailable")
	}
	view := *s
	view.readOnly = true
	return view.getPathLaunchPreference(repoIdentity, path, scope)
}
func (s *ThreadStore) withPreviewLock(fn func() error) error {
	return s.withPreviewLockContext(context.Background(), fn)
}

// withPreviewLockContext is a read under store.lock that waits a busy store
// out for up to storeReadLockWait, bounded by ctx.
func (s *ThreadStore) withPreviewLockContext(ctx context.Context, fn func() error) (err error) {
	if s.inspection != nil {
		return s.inspection.withRoot(s, fn)
	}
	if err := provisionSafePath(s.root); err != nil {
		return err
	}
	if _, err := os.Lstat(s.root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := s.validateBackendPath(); err != nil {
		return err
	}
	lock, err := s.retentionReadLock(ctx, storeReadLockWait)
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
