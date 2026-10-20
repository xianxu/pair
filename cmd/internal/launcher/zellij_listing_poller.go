package launcher

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ListingPoller keeps the latest validated full `list-sessions` reply in memory so a
// request path reads zellij liveness without running zellij (pair#618). The
// supervisor's message authority checked ownership on every admission and every
// queued-message retry, each check running native listings; under load that
// exceeded the 2 s admission budget and woke every session server several
// times a second. One loop now lists on a fixed cadence and readers take its
// latest answer. A snapshot older than MaxStale is not evidence: the read
// refuses with its age, so a hung zellij fails closed, visibly.
type ListingPoller struct {
	Interval, MaxStale time.Duration

	now  func() time.Time
	list func(context.Context) ([]byte, error)

	mu      sync.Mutex
	raw     []byte
	at      time.Time
	known   bool
	failing error
}

// NewListingPoller lists through source, which must not itself read a poller.
func NewListingPoller(source ZellijSource, interval, maxStale time.Duration) *ListingPoller {
	source.Listings = nil
	return &ListingPoller{Interval: interval, MaxStale: maxStale, now: time.Now, list: source.fullListing}
}

// ErrListingStale is the read of a poller whose last good listing is too old
// (or that has never listed).
type ErrListingStale struct {
	Age   time.Duration
	Never bool
	Last  error
}

func (e *ErrListingStale) Error() string {
	msg := fmt.Sprintf("zellij liveness is stale (last listing %s ago)", e.Age.Round(time.Second))
	if e.Never {
		msg = "zellij liveness is not yet known (no listing has completed)"
	}
	if e.Last != nil {
		msg += ": " + e.Last.Error()
	}
	return msg
}

// Run lists at once and then every Interval until ctx ends. One caller owns it.
func (p *ListingPoller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()
	for {
		p.Refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Refresh runs one full listing. A failure keeps the previous good reply, whose
// age keeps growing until reads refuse it.
func (p *ListingPoller) Refresh(ctx context.Context) {
	raw, err := p.list(ctx)
	if ctx.Err() != nil {
		return
	}
	if err == nil {
		err = validateSessionListing(raw, false)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.failing = err
		return
	}
	p.raw, p.at, p.known, p.failing = raw, p.now(), true, nil
}

// Listings returns the latest validated full reply when it is at most MaxStale old.
func (p *ListingPoller) Listings() (raw []byte, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.known {
		return nil, &ErrListingStale{Never: true, Last: p.failing}
	}
	if age := p.now().Sub(p.at); age > p.MaxStale {
		return nil, &ErrListingStale{Age: age, Last: p.failing}
	}
	return p.raw, nil
}
