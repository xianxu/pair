package couchcore

import (
	"context"
	"sync"
	"time"
)

// FakeLaunchFailureStore retains evidence across Couch reconstructions. Tests
// may inject persistence failures and inspect ordered writes after cleanup.
type FakeLaunchFailureStore struct {
	mu        sync.Mutex
	History   LaunchFailureHistory
	Writes    []LaunchFailure
	ReadError error
	SaveError error
}

func NewFakeLaunchFailureStore() *FakeLaunchFailureStore { return &FakeLaunchFailureStore{} }
func (s *FakeLaunchFailureStore) Path() string           { return "fake/launch-diagnostics/failures.json" }
func (s *FakeLaunchFailureStore) Read() ([]LaunchFailure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ReadError != nil {
		return nil, s.ReadError
	}
	out := make([]LaunchFailure, len(s.History))
	for i, f := range s.History {
		out[i] = cloneLaunchFailure(f)
	}
	return out, nil
}
func (s *FakeLaunchFailureStore) Save(ctx context.Context, f LaunchFailure) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.Writes = append(s.Writes, cloneLaunchFailure(f))
	if s.SaveError != nil {
		return s.SaveError
	}
	next, err := s.History.Upsert(f)
	if err == nil {
		s.History = next
	}
	return err
}

func (s *FakeLaunchFailureStore) Supersede(ctx context.Context, t LifecycleFailureTarget, epoch string, order uint64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.ReadError != nil {
		return s.ReadError
	}
	next, changed := s.History.Supersede(t, epoch, order, at)
	if !changed {
		return nil
	}
	if s.SaveError != nil {
		return s.SaveError
	}
	next, err := fitLifecycleFailureHistory(next, launchFailureEncodedLimit, encodeLifecycleFailureRow)
	if err != nil {
		return err
	}
	s.History = next
	return nil
}
