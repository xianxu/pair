package remotesession

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var errSession = errors.New("remote session refused; handshake required")
var errCapacity = errors.New("remote session capacity reached; log out another session")

type sessionCredentials struct {
	Cookie, CSRF string
	Expires      time.Time
}
type sessionRef struct {
	key   [32]byte
	epoch uint64
	done  <-chan struct{}
}
type sessionEntry struct {
	identity AccessIdentity
	csrf     [32]byte
	done     chan struct{}
}

// Sessions owns at most four cookie hashes and their stream revocation signals.
// Time is a value supplied by the owner; there is no independent idle timer.
type Sessions struct {
	credential func() (string, error)
	mu         sync.Mutex
	epoch      uint64
	revoked    bool
	entries    map[[32]byte]*sessionEntry
}

func newSessions(epoch uint64) *Sessions {
	return &Sessions{credential: randomCredential, epoch: epoch, revoked: epoch == 0, entries: make(map[[32]byte]*sessionEntry)}
}
func randomCredential() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
func credentialHash(raw string) ([32]byte, bool) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || len(decoded) != 32 || len(raw) != 43 {
		return [32]byte{}, false
	}
	return sha256.Sum256(decoded), true
}
func (s *Sessions) retireLocked(key [32]byte) {
	if entry := s.entries[key]; entry != nil {
		close(entry.done)
		delete(s.entries, key)
	}
}
func (s *Sessions) Handshake(id AccessIdentity, now time.Time) (sessionCredentials, error) {
	if !id.validAt(now) {
		return sessionCredentials{}, errSession
	}
	cookie, err := s.credential()
	if err != nil {
		return sessionCredentials{}, err
	}
	csrf, err := s.credential()
	if err != nil {
		return sessionCredentials{}, err
	}
	key, _ := credentialHash(cookie)
	csrfHash, _ := credentialHash(csrf)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		return sessionCredentials{}, errSession
	}
	for k, e := range s.entries {
		if !e.identity.validAt(now) {
			s.retireLocked(k)
		}
	}
	if len(s.entries) >= 4 {
		return sessionCredentials{}, errCapacity
	}
	if s.entries[key] != nil {
		return sessionCredentials{}, errSession
	}
	s.entries[key] = &sessionEntry{identity: id, csrf: csrfHash, done: make(chan struct{})}
	return sessionCredentials{Cookie: cookie, CSRF: csrf, Expires: id.expires}, nil
}
func (s *Sessions) Admit(cookie, csrf string, id AccessIdentity, now time.Time, mutation bool) (sessionRef, error) {
	key, ok := credentialHash(cookie)
	if !ok {
		return sessionRef{}, errSession
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[key]
	if s.revoked || entry == nil {
		return sessionRef{}, errSession
	}
	if !entry.identity.validAt(now) {
		s.retireLocked(key)
		return sessionRef{}, errSession
	}
	if !id.validAt(now) || !entry.identity.samePrincipal(id) {
		return sessionRef{}, errSession
	}
	if mutation {
		hash, ok := credentialHash(csrf)
		if !ok || subtle.ConstantTimeCompare(hash[:], entry.csrf[:]) != 1 {
			return sessionRef{}, errSession
		}
	}
	return sessionRef{key: key, epoch: s.epoch, done: entry.done}, nil
}
func (s *Sessions) Logout(cookie string) {
	key, ok := credentialHash(cookie)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retireLocked(key)
}
func (s *Sessions) Revoke() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked = true
	for key := range s.entries {
		s.retireLocked(key)
	}
}
