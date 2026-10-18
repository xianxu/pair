package launcher

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeListings struct {
	mu    sync.Mutex
	calls int
	err   error
	short string
}

func (f *fakeListings) list(context.Context) ([]byte, []byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, nil, f.err
	}
	return []byte(f.short), []byte(strings.TrimSuffix(f.short, "\n") + " [Created 1s ago]\n"), nil
}

func testPoller(f *fakeListings, now *time.Time) *ListingPoller {
	return &ListingPoller{Interval: time.Second, MaxStale: 5 * time.Second, now: func() time.Time { return *now }, list: f.list}
}

// Reads never run zellij; freshness and the stale refusal follow the clock; a
// failed listing keeps the last good pair until it ages out (pair#618).
func TestListingPollerServesTheLatestSnapshotUntilItIsStale(t *testing.T) {
	now := time.Unix(1000, 0)
	f := &fakeListings{short: "📁1-264\n"}
	p := testPoller(f, &now)
	if _, _, err := p.Listings(); err == nil || !strings.Contains(err.Error(), "not yet known") {
		t.Fatalf("before any listing: %v", err)
	}
	p.Refresh(context.Background())
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if short, _, err := p.Listings(); err != nil || string(short) != "📁1-264\n" {
				t.Errorf("read: %q %v", short, err)
			}
		}()
	}
	wg.Wait()
	if f.calls != 1 {
		t.Fatalf("reads ran zellij: %d listings", f.calls)
	}
	f.err = errors.New("zellij hung")
	now = now.Add(4 * time.Second)
	p.Refresh(context.Background())
	if short, _, err := p.Listings(); err != nil || string(short) != "📁1-264\n" {
		t.Fatalf("a failed listing must keep the last good pair while fresh: %q %v", short, err)
	}
	now = now.Add(2 * time.Second)
	_, _, err := p.Listings()
	var stale *ErrListingStale
	if !errors.As(err, &stale) || stale.Age != 6*time.Second || !strings.Contains(err.Error(), "zellij hung") {
		t.Fatalf("stale read: %v", err)
	}
	f.err = nil
	p.Refresh(context.Background())
	if _, _, err := p.Listings(); err != nil {
		t.Fatalf("after recovery: %v", err)
	}
}

// A source holding a poller answers snapshots from it; per-session queries and
// a nil poller still run zellij.
func TestZellijSourceReadsListingsFromItsPoller(t *testing.T) {
	now := time.Unix(1000, 0)
	f := &fakeListings{short: "📁1-264\n"}
	p := testPoller(f, &now)
	p.Refresh(context.Background())
	s := ZellijSource{Path: "/nonexistent/zellij", Listings: p}
	sessions, err := s.LivenessContext(context.Background())
	if err != nil || len(sessions) != 1 || sessions[0].Name != "📁1-264" || sessions[0].State != SessionLive {
		t.Fatalf("liveness from poller: %+v %v", sessions, err)
	}
	if _, err := (ZellijSource{Path: "/nonexistent/zellij"}).LivenessContext(context.Background()); err == nil {
		t.Fatal("a source without a poller must run zellij")
	}
}

func TestListingPollerRunStopsWithItsContext(t *testing.T) {
	f := &fakeListings{short: ""}
	p := &ListingPoller{Interval: time.Millisecond, MaxStale: time.Second, now: time.Now, list: f.list}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { p.Run(ctx); close(done) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls < 2 {
		t.Fatalf("Run listed %d times", f.calls)
	}
}
