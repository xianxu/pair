package couchcore

import (
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"path/filepath"
)

func (s *ThreadStore) PublishContinuation(address ThreadAddress, revision uint64, request checkpoint.Request) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, revision, func(record *ThreadRecord) error {
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
				return fmt.Errorf("continuation %s is %s; inspect or retry it before publishing another", old.ID, old.Phase)
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
func (s *ThreadStore) AdvanceContinuation(address ThreadAddress, revision uint64, event checkpoint.Event) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, revision, func(record *ThreadRecord) error {
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
	path := c.continuationPath(record.Address)
	if err := writeAtomicBytes(path, []byte(cp.Body)); err != nil {
		return "", err
	}
	return path, nil
}
func (c *Couch) continuationPath(address ThreadAddress) string {
	return c.Threads.continuationPath(address)
}

func (s *ThreadStore) continuationPath(address ThreadAddress) string {
	return filepath.Join(s.root, "continuation", address.RepoScope, string(address.Tag)+".md")
}
