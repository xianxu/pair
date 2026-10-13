package broadcast

import (
	"fmt"
	"sync"

	"github.com/xianxu/pair/cmd/internal/castlock"
)

// LinkLocks is one cast's code locks, one per link token (pair#495): the
// screen broadcast's view and pointer links, the web cast's view, pointer and
// drive links. A link gets its lock with its token (Mint) before the token is
// routable, so a mount's LockFor never finds an owned token unlocked, and every
// lock ends with the cast (End). A new kind of link is one Mint. Its mutex is
// a leaf: no call holds it across a lock's own.
type LinkLocks struct {
	mu    sync.Mutex
	links []linkLock
	ended bool
}

type linkLock struct {
	label, token string
	lock         *castlock.Lock
}

// Mint makes the lock for label's link, under token, with digits-long codes;
// onRotated hears its rotations with the label.
func (ls *LinkLocks) Mint(label, token string, digits int, onRotated func(label string, r castlock.Rotation)) error {
	lock, err := NewLinkLock(digits, func(r castlock.Rotation) {
		if onRotated != nil {
			onRotated(label, r)
		}
	})
	if err != nil {
		return err
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.ended {
		lock.End()
		return fmt.Errorf("broadcast: the cast has ended")
	}
	ls.links = append(ls.links, linkLock{label: label, token: token, lock: lock})
	return nil
}

// LockFor is the lock guarding segment, or nil when no link here has it.
func (ls *LinkLocks) LockFor(segment string) *castlock.Lock {
	if ls == nil {
		return nil
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	var found *castlock.Lock
	for _, l := range ls.links {
		// Every token is compared, so the time taken says nothing of which.
		if OwnsToken(segment, l.token) {
			found = l.lock
		}
	}
	return found
}

// Codes is each link's current code, in the order the links were minted. A
// code is shown to the operator only, never on a broadcast frame.
func (ls *LinkLocks) Codes() []LinkCode {
	if ls == nil {
		return nil
	}
	ls.mu.Lock()
	links := append([]linkLock(nil), ls.links...)
	ls.mu.Unlock()
	codes := make([]LinkCode, 0, len(links))
	for _, l := range links {
		codes = append(codes, LinkCode{Link: l.label, Code: l.lock.Code()})
	}
	return codes
}

// Rotate gives label's link a new code and disconnects everyone who unlocked
// it with the old one.
func (ls *LinkLocks) Rotate(label string) error {
	if ls == nil {
		return fmt.Errorf("broadcast: no %s link", label)
	}
	ls.mu.Lock()
	var lock *castlock.Lock
	for _, l := range ls.links {
		if l.label == label {
			lock = l.lock
		}
	}
	ls.mu.Unlock()
	if lock == nil {
		return fmt.Errorf("broadcast: no %s link", label)
	}
	_, err := lock.Rotate()
	return err
}

// End retires every lock with the cast; no link mints after it. Idempotent.
func (ls *LinkLocks) End() {
	if ls == nil {
		return
	}
	ls.mu.Lock()
	links := ls.links
	ls.ended = true
	ls.mu.Unlock()
	for _, l := range links {
		l.lock.End()
	}
}
