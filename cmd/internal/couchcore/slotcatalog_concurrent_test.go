package couchcore

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// latencyIO adds a fixed latency to every probe and records the command
// count and the peak number in flight (pair#726).
type latencyIO struct {
	inner         ProvisionIO
	delay         time.Duration
	mu            sync.Mutex
	count, active int
	peak          int
}

func (l *latencyIO) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	l.mu.Lock()
	l.count++
	l.active++
	if l.active > l.peak {
		l.peak = l.active
	}
	l.mu.Unlock()
	time.Sleep(l.delay)
	defer func() { l.mu.Lock(); l.active--; l.mu.Unlock() }()
	return l.inner.Run(ctx, c)
}

// TestDiscoverVerifiesSlotsConcurrently: per-slot verification runs in
// parallel, so wall time stays near the fixed prefix as slots double, while
// the command count and the result rows are what a serial pass produces.
func TestDiscoverVerifiesSlotsConcurrently(t *testing.T) {
	const delay = 40 * time.Millisecond
	measure := func(slots int) (count, peak int, wall time.Duration, rows int) {
		f := newProvisionFixture(t)
		for n := 1; n <= slots; n++ {
			f.git(f.Primary, "worktree", "add", "-b", fmt.Sprintf("main-slot%d", n), f.host(n), "main")
		}
		io := &latencyIO{inner: f, delay: delay}
		start := time.Now()
		repo, err := NewOSSlotCatalog(io).Discover(context.Background(), f.Primary)
		wall = time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range repo.Slots {
			if !row.Verified || row.Err != nil {
				t.Fatalf("slot %d unverified: %v", row.Identity.Number, row.Err)
			}
		}
		return io.count, io.peak, wall, len(repo.Slots)
	}
	c2, p2, w2, r2 := measure(2)
	c4, p4, w4, r4 := measure(4)
	if r2 != 2 || r4 != 4 {
		t.Fatalf("rows %d, %d", r2, r4)
	}
	perSlot := (c4 - c2) / 2
	if perSlot <= 0 || c4 != c2+2*perSlot {
		t.Fatalf("command count not linear in slots: %d at 2, %d at 4", c2, c4)
	}
	if p2 < 2 || p4 < 4 || p4 > 4 {
		t.Fatalf("peak concurrency %d at 2 slots, %d at 4 (want one per slot, bounded)", p2, p4)
	}
	if _, p10, _, r10 := measure(10); r10 != 10 || p10 > slotVerifyConcurrency {
		t.Fatalf("10 slots: rows %d, peak concurrency %d over the cap %d", r10, p10, slotVerifyConcurrency)
	}
	serial := time.Duration(c4) * delay
	if w4 >= serial*3/4 {
		t.Fatalf("4-slot discovery took %v, close to the serial %v (2 slots: %v)", w4, serial, w2)
	}
}

// TestDiscoverStillFailsWholeInventoryOnReplacedHost keeps the replacement
// fence under concurrency: a host swapped during verification fails the scan.
func TestDiscoverStillFailsWholeInventoryOnReplacedHost(t *testing.T) {
	f := newProvisionFixture(t)
	for n := 1; n <= 3; n++ {
		f.git(f.Primary, "worktree", "add", "-b", fmt.Sprintf("main-slot%d", n), f.host(n), "main")
	}
	host := f.host(2)
	var once sync.Once
	f.AfterGit = func(c ProvisionCommand, _ []byte) error {
		if c.Dir == host && len(c.Args) > 0 && c.Args[0] == "rev-parse" {
			once.Do(func() {
				moved := host + ".moved"
				if err := os.Rename(host, moved); err != nil {
					t.Error(err)
				}
				if err := os.Mkdir(host, 0700); err != nil {
					t.Error(err)
				}
				for _, name := range []string{".git"} {
					if raw, err := os.ReadFile(moved + "/" + name); err == nil {
						_ = os.WriteFile(host+"/"+name, raw, 0600)
					}
				}
			})
		}
		return nil
	}
	if _, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary); err == nil {
		t.Fatal("a host replaced during discovery did not fail the inventory")
	}
}
