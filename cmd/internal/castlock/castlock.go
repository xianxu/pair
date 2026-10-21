// Package castlock is the second factor on a cast link (pair#495): a short
// numeric code, said aloud on the call, that a viewer enters before the link
// serves anything. One Lock guards one link token; the HTTP gate that consults
// it lives in package broadcast's Host.
//
// A Lock holds the code, its wrong-try budget and the sessions it issued. A
// session is a bearer for the life of one generation of sessions: an operator
// rotation revokes every session and closes the generation's Revoked channel,
// so streams admitted under it end. A rotation forced by guessing replaces only
// the code; viewers already in keep watching. The third forced rotation caps
// new unlocks until successful operator rotation; unlocking never resets it.
//
// Every state change happens under one mutex, so an unlock and a rotation are
// totally ordered: an unlock is judged against the code current when it takes
// the lock. Callbacks run after the mutex is released.
package castlock

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"
)

// Defaults for a view link; the drive link asks for more digits.
const (
	DefaultDigits      = 4
	DefaultBudget      = 8
	MaxForcedRotations = 3
	DefaultMaxSessions = 64
	// MaxCodeInput bounds what Unlock will look at; a code is a few digits.
	MaxCodeInput = 32
)

// Limiter admits one unlock attempt at now; broadcast.RateLimit is one. The
// Lock calls it under its own mutex.
type Limiter interface{ Allow(now time.Time) bool }

// Reason says why a code rotated.
type Reason uint8

const (
	// ReasonOperator: the operator rotated it; every session was revoked.
	ReasonOperator Reason = iota + 1
	// ReasonGuessing: the wrong-try budget ran out; sessions were kept.
	ReasonGuessing
)

// Rotation reports a new code. Gen numbers the lock's codes and only
// increases: callbacks run outside the mutex, so two rotations can be
// delivered in either order, and a receiver keeps the highest Gen.
type Rotation struct {
	Code   string
	Reason Reason
	Gen    uint64
	Capped bool // this rotation spent the link lifetime guess budget
}

// Config makes a Lock.
type Config struct {
	Digits      int
	Budget      int
	MaxSessions int
	// Limiter bounds the rate of unlock attempts; nil admits every attempt.
	Limiter Limiter
	// Rand supplies codes and session tokens; nil is crypto/rand.
	Rand io.Reader
	// OnRotated is told every rotation, after the Lock's mutex is released.
	OnRotated func(Rotation)
}

// Outcome is what an unlock attempt came to.
type Outcome uint8

const (
	Unlocked  Outcome = iota + 1
	Wrong             // well formed, not the code
	Malformed         // not a code at all; costs no budget
	Limited           // too many attempts too fast; not judged
	Ended             // the lock's link has ended
	Failed            // the lock could not mint a session; try again
)

// Lock guards one link.
type Lock struct {
	cfg Config

	mu       sync.Mutex
	code     string
	gen      uint64 // numbers code; bumped on every rotation
	wrong    int
	forced   int                          // only a successful operator rotation resets this
	sessions map[[sha256.Size]byte]uint64 // session hash → issue order
	issued   uint64
	revoked  chan struct{} // the current generation's; closed on revoke
	ended    bool
}

// New makes a Lock with a fresh random code.
func New(cfg Config) (*Lock, error) {
	if cfg.Digits <= 0 {
		cfg.Digits = DefaultDigits
	}
	if cfg.Digits > 9 {
		return nil, errors.New("castlock: a code has at most 9 digits")
	}
	if cfg.Budget <= 0 {
		cfg.Budget = DefaultBudget
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = DefaultMaxSessions
	}
	if cfg.Rand == nil {
		cfg.Rand = rand.Reader
	}
	l := &Lock{cfg: cfg, sessions: map[[sha256.Size]byte]uint64{}, revoked: make(chan struct{})}
	code, err := l.newCode("")
	if err != nil {
		return nil, err
	}
	l.code = code
	return l, nil
}

// Code is the current code.
func (l *Lock) Code() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.code
}

// Digits is how long the code is.
func (l *Lock) Digits() int { return l.cfg.Digits }

// Unlock judges one attempt at now. On Unlocked it returns a new session.
func (l *Lock) Unlock(now time.Time, input string) (session string, out Outcome) {
	var rotated *Rotation
	defer func() {
		if rotated != nil && l.cfg.OnRotated != nil {
			l.cfg.OnRotated(*rotated)
		}
	}()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ended {
		return "", Ended
	}
	if l.cfg.Limiter != nil && !l.cfg.Limiter.Allow(now) {
		return "", Limited
	}
	code, ok := ParseCode(input, l.cfg.Digits)
	if !ok {
		return "", Malformed
	}
	if l.forced >= MaxForcedRotations || l.wrong >= l.cfg.Budget {
		// A capped link, or a spent code whose rotation failed: refuse every attempt, right
		// or wrong, until the operator rotates it (fail closed).
		return "", Wrong
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(l.code)) != 1 {
		l.wrong++
		if l.wrong >= l.cfg.Budget {
			// The code is spent, not the sessions: a guesser can't evict viewers.
			if next, err := l.newCode(l.code); err == nil {
				l.code, l.wrong = next, 0
				l.gen++
				l.forced++
				rotated = &Rotation{Code: next, Reason: ReasonGuessing, Gen: l.gen, Capped: l.forced == MaxForcedRotations}
			}
		}
		return "", Wrong
	}
	token, err := l.token()
	if err != nil {
		return "", Failed
	}
	l.issued++
	l.sessions[sha256.Sum256([]byte(token))] = l.issued
	l.evictLocked()
	return token, Unlocked
}

// Valid reports whether session was issued by this Lock and not revoked. When
// it is, revoked closes when the session's generation is revoked or the link
// ends.
func (l *Lock) Valid(session string) (revoked <-chan struct{}, ok bool) {
	if session == "" || len(session) > 128 {
		return nil, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ended {
		return nil, false
	}
	if _, ok := l.sessions[sha256.Sum256([]byte(session))]; !ok {
		return nil, false
	}
	return l.revoked, true
}

// Rotate replaces the code and, by the operator's hand, revokes every session.
// It returns the new code.
func (l *Lock) Rotate() (string, error) {
	l.mu.Lock()
	if l.ended {
		l.mu.Unlock()
		return "", errors.New("castlock: the link has ended")
	}
	next, err := l.newCode(l.code)
	if err != nil {
		l.mu.Unlock()
		return "", err
	}
	l.code, l.wrong, l.forced = next, 0, 0
	l.gen++
	r := Rotation{Code: next, Reason: ReasonOperator, Gen: l.gen}
	l.revokeLocked()
	l.mu.Unlock()
	if l.cfg.OnRotated != nil {
		l.cfg.OnRotated(r)
	}
	return next, nil
}

// End retires the lock with its link: every session is revoked and no unlock
// succeeds again. Idempotent.
func (l *Lock) End() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ended {
		return
	}
	l.ended = true
	l.revokeLocked()
}

// Sessions is how many sessions are live.
func (l *Lock) Sessions() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.sessions)
}

func (l *Lock) revokeLocked() {
	clear(l.sessions)
	close(l.revoked)
	if !l.ended {
		l.revoked = make(chan struct{})
	}
}

// evictLocked drops the oldest sessions above the cap.
func (l *Lock) evictLocked() {
	for len(l.sessions) > l.cfg.MaxSessions {
		var oldest [sha256.Size]byte
		first := true
		var at uint64
		for k, n := range l.sessions {
			if first || n < at {
				oldest, at, first = k, n, false
			}
		}
		delete(l.sessions, oldest)
	}
}

// newCode draws a code different from previous.
func (l *Lock) newCode(previous string) (string, error) {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(l.cfg.Digits)), nil)
	for range 16 {
		n, err := rand.Int(l.cfg.Rand, limit)
		if err != nil {
			return "", fmt.Errorf("castlock: code: %w", err)
		}
		code := fmt.Sprintf("%0*d", l.cfg.Digits, n)
		if code != previous {
			return code, nil
		}
	}
	return "", errors.New("castlock: no fresh code")
}

func (l *Lock) token() (string, error) {
	var b [32]byte
	if _, err := io.ReadFull(l.cfg.Rand, b[:]); err != nil {
		return "", fmt.Errorf("castlock: session: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// ParseCode normalizes what a viewer typed: spaces and dashes are ignored, and
// what remains must be exactly digits ASCII digits.
func ParseCode(input string, digits int) (string, bool) {
	if len(input) > MaxCodeInput {
		return "", false
	}
	var b strings.Builder
	for _, r := range input {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-':
		default:
			return "", false
		}
	}
	if b.Len() != digits {
		return "", false
	}
	return b.String(), true
}
