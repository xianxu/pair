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
	return s.PreviewSnapshotContext(context.Background())
}
func (s *ThreadStore) PreviewSnapshotContext(ctx context.Context) (ThreadSnapshot, error) {
	if s == nil {
		return ThreadSnapshot{}, errors.New("thread store unavailable")
	}
	view := *s
	view.readOnly = true
	return view.snapshotContext(ctx)
}

// PreviewThreadContext reads through the regular decoder and backend router,
// without initializing stores, enrolling retention or replaying pending journals.
func (s *ThreadStore) PreviewThreadContext(ctx context.Context, address ThreadAddress) (ThreadRecord, error) {
	if s == nil {
		return ThreadRecord{}, errors.New("thread store unavailable")
	}
	view := *s
	view.readOnly = true
	record, _, err := view.getThreadStampedContext(ctx, address)
	return record, err
}

// withReadLockContext retains lifecycle recovery for ordinary getters while
// preview views carry the caller's cancellation through every routed read.
func (s *ThreadStore) withReadLockContext(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.readOnly {
		return s.withPreviewLockContext(ctx, fn)
	}
	return s.withLock(fn)
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
		return errors.Join(err, ctx.Err())
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if _, err := os.Lstat(s.journalPath()); err == nil {
		return errors.New("thread store recovery pending; an explicit lifecycle action is required")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return fn()
}
