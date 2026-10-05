package couchcore

import (
	"context"
	"errors"
	"os"
	"time"
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

// withPreviewLockContext is a foreground read under store.lock: it holds no
// other lock, so it waits a busy store out for up to storeReadLockWait,
// bounded by ctx.
func (s *ThreadStore) withPreviewLockContext(ctx context.Context, fn func() error) error {
	return s.withPreviewLockWithin(ctx, storeReadLockWait, fn)
}

// withNestedPreviewLock is a read taken while another store lock is held: it
// never waits under that lock (lock ordering), so a busy store fails at once.
func (s *ThreadStore) withNestedPreviewLock(fn func() error) error {
	return s.withPreviewLockWithin(context.Background(), 0, fn)
}

func (s *ThreadStore) withPreviewLockWithin(ctx context.Context, wait time.Duration, fn func() error) (err error) {
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
	lock, err := s.retentionReadLock(ctx, wait)
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
