package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// withEmptyStartupTarget checks retained history and publishes an empty-only
// reservation under the existing coordinator -> global -> local lock order.
// The callback must use lock-assuming store operations, never public writers.
func (s *ThreadStore) withEmptyStartupTarget(ctx context.Context, target *ThreadStore, scope string, publish func() error) error {
	if s == nil || target == nil || s.layout.Local || s.readOnly || target.readOnly || scope == "" || publish == nil {
		return errors.New("invalid empty startup target")
	}
	if target != s && (!target.layout.Local || target.namespace != s.namespace) {
		return errors.New("empty startup target belongs to another store")
	}
	// The checked lock calls this both before and after admission, before its
	// normal journal recovery. Startup has no authority to replay that journal.
	check := func(store *ThreadStore) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := os.Lstat(store.journalPath()); !errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return err
			}
			return errors.New("startup history recovery pending; use an explicit lifecycle action")
		}
		return nil
	}
	empty := func(store *ThreadStore) error {
		retained, err := store.startupHistoryLocked(ctx, scope)
		if err != nil {
			return err
		}
		if retained {
			return fmt.Errorf("startup target history changed; inspect the parked hub before starting: %s", store.root)
		}
		return nil
	}
	// Startup reservations are foreground work: previews may briefly hold either
	// store lock outside the coordinator. Keep the existing bounded wait policy,
	// within the caller's deadline (pair#854).
	run := func() error {
		return s.withStoreLockBounded(func() error {
			if err := empty(s); err != nil {
				return err
			}
			if target == s {
				return publish()
			}
			return target.withStoreLockBounded(func() error {
				if err := empty(target); err != nil {
					return err
				}
				return publish()
			}, func() error { return check(target) }, storeLockBound(ctx, storeReadLockWait))
		}, func() error { return check(s) }, storeLockBound(ctx, storeReadLockWait))
	}
	if s.coordinator == nil {
		return run()
	}
	return s.coordinator.WithLock(ctx, func(held *storagegc.Locked) error {
		if err := held.RegisterStore(s.namespace.Dir()); err != nil {
			return err
		}
		return run()
	})
}
