package couchcore

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func (s *ThreadStore) withUpdateLockContext(ctx context.Context, run func() error) error {
	if ctx == nil {
		return s.withLock(run)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.readOnly {
		return errors.New("park recovery requires a writable store")
	}
	if s.coordinator != nil {
		return s.coordinator.WithLock(ctx, func(held *storagegc.Locked) error {
			if err := held.RegisterStore(s.namespace.Dir()); err != nil {
				return err
			}
			return s.withStoreLockChecked(run, held.CheckContext, storeReadLockWait)
		})
	}
	return s.withStoreLockChecked(run, ctx.Err, storeReadLockWait)
}

// RetireInterruptedPark closes failed bookkeeping and retires only the exact
// screened incarnation in one journal image. No cleanup success is invented.
func (s *ThreadStore) RetireInterruptedPark(ctx context.Context, address ThreadAddress, revision uint64, identity ParkIdentity, at time.Time) (ThreadRecord, error) {
	return s.updateExistingThreadContext(ctx, address, revision, func(next *ThreadRecord) error {
		if !parkRecoveryOwnerMatches(*next) || next.Park.Identity != identity {
			return errors.New("interrupted park owner was replaced; inspect before recovering")
		}
		if err := abandonParkRecord(next, identity); err != nil {
			return err
		}
		return retireIncarnationRecord(next, ProcessIdentity{PID: identity.PID, Identity: identity.ProcessIdentity}, at, next.Incarnations[0].State)
	})
}

func (s *ThreadStore) rollbackUnpublishedPark(ctx context.Context, current ThreadRecord) (ThreadRecord, error) {
	return s.updateExistingThreadContext(ctx, current.Address, current.Revision, func(next *ThreadRecord) error {
		if !parkRecoveryOwnerMatches(*next) || next.Park.Identity != current.Park.Identity {
			return errors.New("unpublished park owner changed")
		}
		return abandonParkRecord(next, current.Park.Identity)
	})
}

// recoveryBackend bounds both routing reads and the explicit journal replay
// needed after a lost mutation reply. It never waits on a nested busy store.
func (s *ThreadStore) recoveryBackend(ctx context.Context, address ThreadAddress) (*ThreadStore, error) {
	if err := validateThreadAddress(address); err != nil {
		return nil, err
	}
	if err := s.withUpdateLockContext(ctx, func() error { return nil }); err != nil {
		return nil, err
	}
	route := *s
	route.readOnly = true
	backend, err := route.storeForAddressContext(ctx, address)
	if err != nil {
		return nil, err
	}
	writable := *backend
	writable.readOnly = false
	return &writable, nil
}

func (s *ThreadStore) recoveryReadThread(ctx context.Context, address ThreadAddress) (ThreadRecord, error) {
	backend, err := s.recoveryBackend(ctx, address)
	if err != nil {
		return ThreadRecord{}, err
	}
	var record ThreadRecord
	err = backend.withUpdateLockContext(ctx, func() error {
		var readErr error
		record, readErr = backend.readThreadLockedContext(ctx, address)
		return readErr
	})
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && s.hooks.AfterGetThread != nil {
		err = s.hooks.AfterGetThread(address)
	}
	return record, err
}
