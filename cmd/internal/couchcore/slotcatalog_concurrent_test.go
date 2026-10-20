package couchcore

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// rendezvousIO holds each slot host's verification probe (rev-parse
// --absolute-git-dir in the slot's directory, one per slot; the primary's
// own probe does not count) until `peers` of them are in flight at
// once. Concurrent verification reaches the rendezvous whatever the host
// load; a serial pass never can, and is released by the safety bound and
// reported. It also counts commands and the peak number in flight (pair#726).
type rendezvousIO struct {
	inner                ProvisionIO
	primary              string
	peers                int
	mu                   sync.Mutex
	cond                 *sync.Cond
	arrived, count, peak int
	active               int
	met, gaveUp          bool
}

func newRendezvousIO(inner ProvisionIO, primary string, peers int) *rendezvousIO {
	r := &rendezvousIO{inner: inner, primary: primary, peers: peers}
	r.cond = sync.NewCond(&r.mu)
	return r
}

func (r *rendezvousIO) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	r.mu.Lock()
	r.count++
	r.active++
	if r.active > r.peak {
		r.peak = r.active
	}
	if c.Program == "git" && c.Dir != r.primary && len(c.Args) == 2 && c.Args[0] == "rev-parse" && c.Args[1] == "--absolute-git-dir" {
		r.arrived++ // probes held right now, not a running total
		if r.arrived == r.peers {
			r.met = true
			r.cond.Broadcast()
		}
		safety := time.AfterFunc(10*time.Second, func() { r.mu.Lock(); r.cond.Broadcast(); r.mu.Unlock() })
		start := time.Now()
		for !r.met && !r.gaveUp && time.Since(start) < 10*time.Second {
			r.cond.Wait()
		}
		if !r.met {
			r.gaveUp = true // serial: stop holding later probes
		}
		r.arrived--
		safety.Stop()
	}
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.active--; r.mu.Unlock() }()
	return r.inner.Run(ctx, c)
}

// TestDiscoverVerifiesSlotsConcurrently: every slot's verification is in
// flight at once (up to the cap), the command count stays linear in slots,
// and the rows are what a serial pass produces. No wall-clock assertion.
func TestDiscoverVerifiesSlotsConcurrently(t *testing.T) {
	measure := func(slots int) (count, peak int, met bool, rows int) {
		f := newProvisionFixture(t)
		for n := 1; n <= slots; n++ {
			f.git(f.Primary, "worktree", "add", "-b", fmt.Sprintf("main-slot%d", n), f.host(n), "main")
		}
		io := newRendezvousIO(f, f.Primary, min(slots, slotVerifyConcurrency))
		repo, err := NewOSSlotCatalog(io).Discover(context.Background(), f.Primary)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range repo.Slots {
			if !row.Verified || row.Err != nil {
				t.Fatalf("slot %d unverified: %v", row.Identity.Number, row.Err)
			}
		}
		return io.count, io.peak, io.met, len(repo.Slots)
	}
	c2, _, met2, r2 := measure(2)
	c4, _, met4, r4 := measure(4)
	if r2 != 2 || r4 != 4 || !met2 || !met4 {
		t.Fatalf("rows %d, %d; all slots in flight together: %v, %v", r2, r4, met2, met4)
	}
	perSlot := (c4 - c2) / 2
	if perSlot <= 0 || c4 != c2+2*perSlot {
		t.Fatalf("command count not linear in slots: %d at 2, %d at 4", c2, c4)
	}
	if _, p10, met10, r10 := measure(10); r10 != 10 || !met10 || p10 > slotVerifyConcurrency {
		t.Fatalf("10 slots: rows %d, rendezvous %v, peak concurrency %d (cap %d)", r10, met10, p10, slotVerifyConcurrency)
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
