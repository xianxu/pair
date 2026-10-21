package remotesession

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
)

// Store is the single supervisor's writer for its private authority latch.
// Independent concurrent writers and intentional rollback are unsupported.
// A failed publication blocks this owner until reopening: a rename may have
// succeeded even when the final directory sync reported an error.
type Store struct {
	mu        sync.Mutex
	path      string
	record    latchRecord
	uncertain bool
	publish   func(string, []byte) error
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path, record: latchRecord{Version: 1, Phase: latchOff}, publish: writeLatch}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := reclaimRemoteStage(path, maxLatchBytes); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read remote latch: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxLatchBytes {
		return nil, errors.New("unsafe remote latch file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Check the opened descriptor too; do not read a replaced directory/device.
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.New("remote latch changed during open")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxLatchBytes+1))
	if err != nil {
		return nil, err
	}
	record, err := decodeLatch(raw)
	if err != nil {
		return nil, fmt.Errorf("decode remote latch: %w", err)
	}
	if record.Phase == latchArmed {
		record.Phase = latchKilled
	}
	s.record = record
	if err := reclaimRemoteStage(path, maxLatchBytes); err != nil {
		return nil, err
	}
	return s, nil
}

func writeLatch(path string, raw []byte) error {
	return writeRemoteAtomic(path, raw, maxLatchBytes)
}

func (s *Store) Snapshot() (latchPhase, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.record.Phase, s.record.Generation
}

func (s *Store) Arm() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uncertain || s.record.Phase != latchOff {
		return 0, errors.New("remote latch is blocked")
	}
	if s.record.Generation == math.MaxUint64 {
		return 0, errors.New("remote latch generation exhausted")
	}
	next := s.record
	next.Phase = latchArmed
	next.Generation++
	if err := s.commit(next); err != nil {
		return 0, err
	}
	return next.Generation, nil
}

func (s *Store) Kill() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uncertain {
		return errors.New("remote latch publication uncertain; reopen required")
	}
	next := s.record
	next.Phase = latchKilled
	return s.commit(next)
}

// consoleReset must only be captured by the owned host-console key path.
// A reset does not arm; starting is a separate explicit operation.
func (s *Store) consoleReset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uncertain || s.record.Phase != latchKilled {
		return errors.New("remote latch cannot reset")
	}
	next := s.record
	next.Phase = latchOff
	return s.commit(next)
}

// stop is called only after ordinary exposure teardown was confirmed. The
// generation fences delayed stop completions, and a kill always wins.
func (s *Store) stop(generation uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uncertain || s.record.Phase != latchArmed || s.record.Generation != generation {
		return errors.New("stale or blocked remote stop")
	}
	next := s.record
	next.Phase = latchOff
	return s.commit(next)
}

// Caller holds mu. Authority is granted only after successful publication.
func (s *Store) commit(next latchRecord) error {
	raw, err := encodeLatch(next)
	if err == nil {
		err = s.publish(s.path, raw)
	}
	if err != nil {
		s.uncertain = true
		s.record.Phase = latchKilled
		// Retain the attempted generation: it may have reached disk before error.
		s.record.Generation = next.Generation
		return fmt.Errorf("remote latch publication uncertain: %w", err)
	}
	s.record = next
	return nil
}
