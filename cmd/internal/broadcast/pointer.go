package broadcast

import (
	"crypto/subtle"
	"sync"
	"time"
)

// PointerState is a broadcast's pointer capability (#412): a token minted on
// first use and kept for the broadcast's life, and whether pointing is on.
// Turning pointing off downgrades the link to view-only; turning it on again
// restores the same link. mu is a leaf lock: nothing is called while holding
// it.
type PointerState struct {
	mu      sync.Mutex
	token   string
	on      bool
	gen     uint64 // bumped on every flip; a late watch report names the gen it saw
	stopped bool   // the broadcast ended: pointing can't be turned on again
	limit   *RateLimit
	changed chan struct{} // closed and replaced on every flip
	// inFlight is a semaphore: a request holds a slot while it is read and
	// handled.
	inFlight chan struct{}
}

func newPointerState() *PointerState {
	return &PointerState{limit: NewRateLimit(PointRatePerSec, PointRatePerSec), changed: make(chan struct{}),
		inFlight: make(chan struct{}, MaxPointInFlight)}
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
	if on && p.token == "" {
		token, err := mint()
		if err != nil {
			return "", false, err
		}
		p.token = token
	}
	if p.on == on {
		return p.token, false, nil
	}
	p.on = on
	p.gen++
	close(p.changed)
	p.changed = make(chan struct{})
	return p.token, true, nil
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
	return true
}

// stop turns pointing off for good.
func (p *PointerState) stop() {
	_, _, _ = p.set(false, nil)
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
}
