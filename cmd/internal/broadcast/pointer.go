package broadcast

import (
	"crypto/subtle"
	"sync"
	"time"
)

// PointerState is a broadcast's pointer capability (#412): a token minted on
// first use and kept for the broadcast's life, and whether pointing is on.
// Turning pointing off downgrades the link to view-only; turning it on again
// restores the same link. Its sticky notes (#447) are the same
// collaboration layer: every flip to off empties the board in the same
// critical section, so "off" and "no notes" are one state. mu is a leaf lock:
// nothing called while holding it takes mu or waits on another goroutine.
type PointerState struct {
	mu    sync.Mutex
	token string
	// mintLock makes the pointer link's code lock (pair#495) in the same
	// step as its token, so the link is never routable unlocked.
	mintLock func(token string) error
	on       bool
	gen      uint64 // bumped on every flip; a late watch report names the gen it saw
	stopped  bool   // the broadcast ended: pointing can't be turned on again
	limit    *RateLimit
	changed  chan struct{} // closed and replaced on every flip
	// inFlight is a semaphore: a request holds a slot while it is read and
	// handled.
	inFlight chan struct{}
	// held counts the pointer streams holding each palette color (#446).
	held [PointerColors]int
	// The note board (#447). notesVersion only increases, across flips too,
	// so a snapshot from before a clear is always older than the clear.
	board        NoteBoard
	notesVersion uint64
	notesChanged chan struct{} // closed and replaced on every board change
	noteLimit    *RateLimit
}

func newPointerState(mintLock func(token string) error) *PointerState {
	return &PointerState{mintLock: mintLock, limit: NewRateLimit(PointRatePerSec, PointRatePerSec), changed: make(chan struct{}),
		inFlight: make(chan struct{}, MaxPointInFlight), notesChanged: make(chan struct{}),
		noteLimit: NewRateLimit(NoteRatePerSec, NoteBurst)}
}

// enter takes an in-flight slot without waiting; leave returns it.
func (p *PointerState) enter() bool {
	select {
	case p.inFlight <- struct{}{}:
		return true
	default:
		return false
	}
}

func (p *PointerState) leave() { <-p.inFlight }

// Match reports whether segment is the pointer token, in constant time. An
// unminted token never matches.
func (p *PointerState) Match(segment string) bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	token := p.token
	p.mu.Unlock()
	return token != "" && subtle.ConstantTimeCompare([]byte(segment), []byte(token)) == 1
}

// On reports whether pointing is on, and a channel closed at the next flip.
func (p *PointerState) On() (bool, <-chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.on, p.changed
}

// allow takes one request from the link's rate budget.
func (p *PointerState) allow(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.limit.Allow(now)
}

// set turns pointing on or off, minting the token on first use. It reports
// the token and whether anything changed.
func (p *PointerState) set(on bool, mint func() (string, error)) (string, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if on && p.stopped {
		return "", false, ErrHubClosed
	}
	if on {
		if err := p.ensureLocked(mint); err != nil {
			return "", false, err
		}
	}
	if p.on == on {
		return p.token, false, nil
	}
	p.on = on
	p.gen++
	close(p.changed)
	p.changed = make(chan struct{})
	if !on {
		p.clearNotesLocked()
	}
	return p.token, true, nil
}

// ensure mints a locked link without changing admission or clearing marks.
func (p *PointerState) ensure(mint func() (string, error)) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return "", ErrHubClosed
	}
	if err := p.ensureLocked(mint); err != nil {
		return "", err
	}
	return p.token, nil
}
func (p *PointerState) ensureLocked(mint func() (string, error)) error {
	if p.token != "" {
		return nil
	}
	token, err := mint()
	if err != nil {
		return err
	}
	if p.mintLock != nil {
		if err := p.mintLock(token); err != nil {
			return err
		}
	}
	p.token = token
	return nil
}

func (p *PointerState) link() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token
}

// generation is the current flip count.
func (p *PointerState) generation() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gen
}

// offIfGeneration turns pointing off only if nothing flipped since gen,
// reporting whether it did. A watch report that arrives after the operator
// turned pointing off and on again must not turn the new pointing off.
func (p *PointerState) offIfGeneration(gen uint64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.gen != gen || !p.on {
		return false
	}
	p.on = false
	p.gen++
	close(p.changed)
	p.changed = make(chan struct{})
	p.clearNotesLocked()
	return true
}

// claimColor gives a new pointer stream the palette color fewest live
// streams hold, the lowest on a tie, so two viewers that haven't picked
// rarely share one. The stream returns it with releaseColor when it ends.
func (p *PointerState) claimColor() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	best := 0
	for i, n := range p.held {
		if n < p.held[best] {
			best = i
		}
	}
	p.held[best]++
	return best
}

func (p *PointerState) releaseColor(c int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.held[c]--
}

// stop turns pointing off for good.
func (p *PointerState) stop() {
	_, _, _ = p.set(false, nil)
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
}
