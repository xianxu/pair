package couchcore

import (
	"encoding/base64"
	"errors"
	"io"
	"sync"
	"time"
)

const (
	startGrantCapacity          = 16
	startGrantTTL               = 5 * time.Minute
	startGrantTokenBytes        = 32
	startGrantCollisionAttempts = 3
)

var (
	ErrStartGrantCapacity    = errors.New("start grant capacity reached")
	ErrStartGrantUnavailable = errors.New("start grant unavailable")
	ErrStartGrantCollision   = errors.New("start grant token collision limit reached")
)

type StartGrantToken string

type startGrant[T any] struct {
	value     T
	issuedAt  time.Time
	consuming bool
}

// StartGrantStore is an owner-local, bounded one-attempt capability table.
// T is cloned at both boundaries so callers cannot mutate accepted authority.
type StartGrantStore[T any] struct {
	mu      sync.Mutex
	clock   Clock
	entropy io.Reader
	clone   func(T) T
	grants  map[StartGrantToken]startGrant[T]
}

func NewStartGrantStore[T any](clock Clock, entropy io.Reader, clone func(T) T) *StartGrantStore[T] {
	return &StartGrantStore[T]{
		clock: clock, entropy: entropy, clone: clone,
		grants: make(map[StartGrantToken]startGrant[T], startGrantCapacity),
	}
}

func (s *StartGrantStore[T]) Issue(value T) (StartGrantToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clock == nil || s.entropy == nil || s.clone == nil {
		return "", errors.New("start grant store is not configured")
	}
	now := s.clock.Now()
	s.pruneExpired(now)
	if len(s.grants) >= startGrantCapacity {
		return "", ErrStartGrantCapacity
	}
	for attempt := 0; attempt < startGrantCollisionAttempts; attempt++ {
		raw := make([]byte, startGrantTokenBytes)
		if _, err := io.ReadFull(s.entropy, raw); err != nil {
			return "", err
		}
		token := StartGrantToken(base64.RawURLEncoding.EncodeToString(raw))
		if _, exists := s.grants[token]; exists {
			continue
		}
		s.grants[token] = startGrant[T]{value: s.clone(value), issuedAt: now}
		return token, nil
	}
	return "", ErrStartGrantCollision
}

func (s *StartGrantStore[T]) Claim(token StartGrantToken) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var zero T
	if s.clock == nil || s.clone == nil {
		return zero, errors.New("start grant store is not configured")
	}
	s.pruneExpired(s.clock.Now())
	grant, exists := s.grants[token]
	if !exists || grant.consuming {
		return zero, ErrStartGrantUnavailable
	}
	grant.consuming = true
	s.grants[token] = grant
	return s.clone(grant.value), nil
}

func (s *StartGrantStore[T]) Finish(token StartGrantToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, exists := s.grants[token]
	if !exists || !grant.consuming {
		return ErrStartGrantUnavailable
	}
	delete(s.grants, token)
	return nil
}

func (s *StartGrantStore[T]) pruneExpired(now time.Time) {
	for token, grant := range s.grants {
		if !grant.consuming && !now.Before(grant.issuedAt.Add(startGrantTTL)) {
			delete(s.grants, token)
		}
	}
}
