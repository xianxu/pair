package testpair

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
)

type couchIdentitySequence struct {
	mu   sync.Mutex
	last uint64
}

func (s *couchIdentitySequence) next() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last >= math.MaxUint32 {
		return 0, errors.New("test Couch identity sequence exhausted")
	}
	s.last++
	return s.last, nil
}

var couchIdentitySeeds couchIdentitySequence

func couchIdentitySeed(pid int64, sequence uint64) (uint64, error) {
	if pid <= 0 || pid > math.MaxInt32 || sequence == 0 || sequence > math.MaxUint32 {
		return 0, errors.New("test Couch identity requires a positive 31-bit PID and positive 32-bit sequence")
	}
	// The allocator must increment NextC when enrolling the fixture's store.
	if pid == math.MaxInt32 && sequence == math.MaxUint32 {
		return 0, errors.New("test Couch identity seed exhausted")
	}
	return 1<<63 | uint64(pid)<<32 | sequence, nil
}

// SeedCouchIdentity initializes a new private host authority under an owned
// fixture root. Each fixture must enroll exactly one store through the real
// couchidentity allocator; this helper reserves only its initial C.
//
// The high bit, live parent PID and process-local sequence keep concurrent
// fixtures distinct, including fixtures from concurrent parent processes on
// the same host. This is not durable uniqueness across PID reuse: crashed native
// residue retains the existing fail-closed collision handling. Never use a live
// host directory or reuse an authority; an existing file is always refused.
func SeedCouchIdentity(hostDir string) (uint64, error) {
	if hostDir == "" {
		return 0, errors.New("test Couch host directory is required")
	}
	sequence, err := couchIdentitySeeds.next()
	if err != nil {
		return 0, err
	}
	c, err := couchIdentitySeed(int64(os.Getpid()), sequence)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(hostDir, 0700); err != nil {
		return 0, fmt.Errorf("create test Couch host directory: %w", err)
	}
	path := filepath.Join(hostDir, "couch-identities.json")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return 0, fmt.Errorf("create test Couch identity authority: %w", err)
	}
	registry := struct {
		SchemaVersion int    `json:"schema_version"`
		NextC         uint64 `json:"next_c"`
		Stores        []any  `json:"stores"`
	}{SchemaVersion: 1, NextC: c, Stores: []any{}}
	writeErr := json.NewEncoder(f).Encode(registry)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return 0, fmt.Errorf("write test Couch identity authority: %w", err)
	}
	return c, nil
}
