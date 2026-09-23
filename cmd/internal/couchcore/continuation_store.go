package couchcore

import (
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"path/filepath"
)

func (s *ThreadStore) PublishContinuation(address ThreadAddress, revision uint64, request checkpoint.Request) (ThreadRecord, error) {
	return s.updateExistingThread(address, revision, func(record *ThreadRecord) error {
		if err := request.Validate(); err != nil {
			return err
		}
		if request.Phase != checkpoint.Pending {
			return errors.New("new continuation must be pending")
		}
		if old := record.Continuation; old != nil {
			if old.ID == request.ID {
				return nil
			}
			if old.Phase != checkpoint.Complete {
				return fmt.Errorf("continuation %s is %s; resolve it before publishing another: %s", old.ID, old.Phase, checkpoint.Exits(old.Phase, string(address.Tag)))
			}
			if request.Source.LaunchOrdinal <= old.Source.LaunchOrdinal {
				return errors.New("continuation source generation did not advance")
			}
		}
		copy := request.Clone()
		record.Continuation = &copy
		return nil
	})
}

// BeginContinuationFromRetiredIncarnations records a continuation request and
// drops the incarnations it supersedes, in ONE transition.
//
// One, not two, because the pair is the invariant: a record carrying this
// request must not also carry the helper it is recovering from, and a crash
// between two writes would leave exactly that. The caller's authority is that
// every incarnation was observed Dead by exact identity and the source session
// was verified absent.
func (s *ThreadStore) BeginContinuationFromRetiredIncarnations(address ThreadAddress, revision uint64, request checkpoint.Request) (ThreadRecord, error) {
	return s.updateExistingThread(address, revision, func(record *ThreadRecord) error {
		if err := request.Validate(); err != nil {
			return err
		}
		if err := noOpenStartClaim(*record); err != nil {
			return err
		}
		copy := request.Clone()
		record.Continuation = &copy
		record.Incarnations = nil
		return nil
	})
}

// DismissFailedContinuation retires a FAILED continuation by deleting it: the
// operator's decision that the thread has moved on (#280). It is Failed's only
// exit besides retry, and deletion rather than a terminal phase on purpose --
// records decode strictly, so a new phase would make every pre-change binary
// reject the whole record. It refuses, writing nothing, for anything but the
// exact failed request; an empty requestID means the retained one.
func (s *ThreadStore) DismissFailedContinuation(address ThreadAddress, revision uint64, requestID string) (ThreadRecord, error) {
	return s.updateExistingThread(address, revision, func(record *ThreadRecord) error {
		if err := checkpoint.CheckDismissible(record.Continuation, requestID); err != nil {
			return err
		}
		record.Continuation = nil
		return nil
	})
}

func (s *ThreadStore) AdvanceContinuation(address ThreadAddress, revision uint64, event checkpoint.Event) (ThreadRecord, error) {
	return s.updateExistingThread(address, revision, func(record *ThreadRecord) error {
		if record.Continuation == nil {
			return errors.New("thread has no continuation request")
		}
		next, err := checkpoint.Advance(*record.Continuation, event)
		if err != nil {
			return err
		}
		record.Continuation = &next
		return nil
	})
}

// The embedded snapshot is authoritative. One replaceable private file per
// address is derived only after the outgoing source is parked.
func (c *Couch) materializeContinuation(record ThreadRecord) (string, error) {
	if record.Continuation == nil {
		return "", errors.New("missing continuation snapshot")
	}
	cp := record.Continuation.Checkpoint
	if err := cp.Validate(); err != nil {
		return "", err
	}
	backend, err := c.Threads.storeForAddress(record.Address)
	if err != nil {
		return "", err
	}
	path := backend.continuationPath(record.Address)
	publish := func() error { return writeAtomicBytes(path, []byte(cp.Body)) }
	if backend.layout.Local {
		publish = func() error {
			return backend.withLock(func() error { return backend.writeStoreAtomicLocked(path, []byte(cp.Body)) })
		}
	}
	if err := publish(); err != nil {
		return "", err
	}
	return path, nil
}
func (c *Couch) continuationPath(address ThreadAddress) (string, error) {
	backend, err := c.Threads.storeForAddress(address)
	if err != nil {
		return "", err
	}
	return backend.continuationPath(address), nil
}

func (s *ThreadStore) continuationPath(address ThreadAddress) string {
	if s.layout.Local {
		return filepath.Join(s.root, "continuation.md")
	}
	return filepath.Join(s.root, "continuation", address.RepoScope, string(address.Tag)+".md")
}
