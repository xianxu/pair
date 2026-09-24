package couchtty

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// fakeSlotGitProbe is the stateful stand-in for `git status` across checkouts
// (pair#317). While gated, a probe blocks until released or until its context
// ends, which is how the tests hold a pass in flight and prove the console is
// not waiting on it.
type fakeSlotGitProbe struct {
	mu        sync.Mutex
	gate      chan struct{}
	entered   chan string
	calls     map[string]int
	cancelled int
	reply     func(dir string, call int) (couchcore.SlotGitStatus, error)
}

func newFakeSlotGitProbe(reply func(dir string, call int) (couchcore.SlotGitStatus, error)) *fakeSlotGitProbe {
	return &fakeSlotGitProbe{calls: map[string]int{}, entered: make(chan string, 64), reply: reply}
}

func (f *fakeSlotGitProbe) probe(ctx context.Context, dir string) (couchcore.SlotGitStatus, error) {
	f.mu.Lock()
	f.calls[dir]++
	call, gate := f.calls[dir], f.gate
	f.mu.Unlock()
	select {
	case f.entered <- dir:
	default:
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			f.mu.Lock()
			f.cancelled++
			f.mu.Unlock()
			return couchcore.SlotGitStatus{}, ctx.Err()
		}
	}
	return f.reply(dir, call)
}

func (f *fakeSlotGitProbe) hold() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gate = make(chan struct{})
	return f.gate
}

func (f *fakeSlotGitProbe) callsFor(dir string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[dir]
}

// slotGitFixture runs a console whose attached child is slot :1 of pair, with
// the primary checkout in the same inventory, so both views have a slot group.
func slotGitFixture(t *testing.T, interval time.Duration, probe *fakeSlotGitProbe) (*consoleFixture, couchcore.ActionableThreadSummary, couchcore.ActionableThreadSummary) {
	t.Helper()
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) { c.slotGitInterval = interval })
	f.con.mu.Lock()
	address := f.con.panes[f.con.order[0]].thread
	f.con.mu.Unlock()
	primary, one := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one")
	one.Address = address
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{primary, one}, nil
	})
	waitFor(t, "slot inventory", func() bool { return len(f.con.menuSnapshot().Inventory) == 2 })
	f.con.SetSlotGitProbe(probe.probe)
	return f, primary, one
}

func (f *consoleFixture) slotGit() map[string]couchcore.SlotGitStatus {
	return f.con.menuSnapshot().SlotGit
}

func TestSlotGitRenderNeverBlocksOnSlowProbe(t *testing.T) {
	probe := newFakeSlotGitProbe(func(dir string, _ int) (couchcore.SlotGitStatus, error) {
		if strings.Contains(dir, "slot1") {
			return couchcore.SlotGitStatus{Branch: "main-slot1", Dirty: true}, nil
		}
		return couchcore.SlotGitStatus{Branch: "main", HasUpstream: true}, nil
	})
	gate := probe.hold()
	f, _, _ := slotGitFixture(t, time.Hour, probe)
	select {
	case <-probe.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("no probe started")
	}
	// A pass is stuck in git. The console must still render and take keys.
	start := time.Now()
	f.con.mu.Lock()
	row := RenderStatusRow(100, f.con.statusModelLocked()).Body
	f.con.mu.Unlock()
	_ = RenderMenu(f.con.menuSnapshot(), 100, 24, time.Now(), false)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("render took %v behind a hung probe", elapsed)
	}
	if strings.Contains(row, "*") {
		t.Fatalf("glyph before any observation: %q", row)
	}
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "switcher opens while git hangs", func() bool {
		return strings.Contains(f.screenText(), "threads")
	})
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "back to the child", func() bool { return !strings.Contains(f.screenText(), "threads") })
	close(gate)
	waitFor(t, "tab bar glyph after the pass lands", func() bool {
		return strings.Contains(f.screenText(), "pair:1*")
	})
}

func TestSlotGitFailureKeepsLastGlyph(t *testing.T) {
	failing := errors.New("git: index.lock exists")
	probe := newFakeSlotGitProbe(func(dir string, call int) (couchcore.SlotGitStatus, error) {
		if strings.Contains(dir, "slot1") {
			if call == 1 {
				return couchcore.SlotGitStatus{Branch: "main-slot1", Dirty: true}, nil
			}
			return couchcore.SlotGitStatus{}, failing
		}
		// The primary changes on every pass: its new value is the positive
		// proof that the failing pass has landed.
		return couchcore.SlotGitStatus{Branch: "main", HasUpstream: true, Ahead: call}, nil
	})
	f, primary, one := slotGitFixture(t, time.Hour, probe)
	waitFor(t, "first pass", func() bool { return f.slotGit()[one.StartingPath].Dirty })
	passes := probe.callsFor(primary.StartingPath)
	f.con.requestSlotGit()
	waitFor(t, "failing pass lands", func() bool {
		return f.slotGit()[primary.StartingPath].Ahead > passes
	})
	if got := f.slotGit()[one.StartingPath]; !got.Dirty || got.Branch != "main-slot1" {
		t.Fatalf("failed probe replaced the last observation: %+v", got)
	}
	waitFor(t, "tab bar keeps the glyph", func() bool { return strings.Contains(f.screenText(), "pair:1*") })
	if strings.Contains(f.screenText(), "index.lock") {
		t.Fatalf("git error reached chrome: %q", f.screenText())
	}
}

func TestSlotGitTriggers(t *testing.T) {
	clean := func(dir string, _ int) (couchcore.SlotGitStatus, error) {
		if strings.Contains(dir, "slot1") {
			return couchcore.SlotGitStatus{Branch: "main-slot1"}, nil
		}
		return couchcore.SlotGitStatus{Branch: "main"}, nil
	}
	probe := newFakeSlotGitProbe(clean)
	f, primary, _ := slotGitFixture(t, time.Hour, probe)
	settle := func(what string) int {
		t.Helper()
		last := -1
		waitFor(t, what, func() bool {
			n := probe.callsFor(primary.StartingPath)
			stable := n == last && n > 0
			last = n
			time.Sleep(20 * time.Millisecond)
			return stable
		})
		return last
	}
	before := settle("startup passes")
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher open probes", func() bool { return probe.callsFor(primary.StartingPath) > before })
	_, _ = f.stdin.Write([]byte{0})
	before = settle("switcher passes")
	f.con.mu.Lock()
	id := f.con.order[0]
	f.con.mu.Unlock()
	f.con.Switch(id)
	waitFor(t, "switch probes", func() bool { return probe.callsFor(primary.StartingPath) > before })
	before = settle("switch passes")

	// Requests during a running pass collapse into one follow-up.
	gate := probe.hold()
	f.con.requestSlotGit()
	select {
	case <-probe.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("gated pass did not start")
	}
	for i := 0; i < 5; i++ {
		f.con.requestSlotGit()
		time.Sleep(5 * time.Millisecond)
	}
	close(gate)
	if got := settle("collapsed passes") - before; got != 2 {
		t.Fatalf("passes after 6 overlapping requests = %d, want 2 (running + one follow-up)", got)
	}
}

func TestSlotGitTickerRefreshes(t *testing.T) {
	probe := newFakeSlotGitProbe(func(string, int) (couchcore.SlotGitStatus, error) {
		return couchcore.SlotGitStatus{Branch: "main"}, nil
	})
	_, primary, _ := slotGitFixture(t, 20*time.Millisecond, probe)
	waitFor(t, "ticker passes", func() bool { return probe.callsFor(primary.StartingPath) >= 5 })
}

func TestSlotGitStopsWithConsole(t *testing.T) {
	probe := newFakeSlotGitProbe(func(string, int) (couchcore.SlotGitStatus, error) {
		return couchcore.SlotGitStatus{}, nil
	})
	probe.hold()
	f, _, _ := slotGitFixture(t, time.Hour, probe)
	select {
	case <-probe.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("no probe started")
	}
	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("console did not stop behind a hung probe")
	}
	probe.mu.Lock()
	defer probe.mu.Unlock()
	if probe.cancelled == 0 {
		t.Fatal("hung probe's context was not cancelled")
	}
}

func TestSlotGitProbePaths(t *testing.T) {
	primary, one, two := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")
	broken := groupedRow("/workspace/pair", 3, "three")
	broken.Target.Slot.WorktreeRoot = "relative/path"
	got := slotGitProbePaths([]couchcore.ActionableThreadSummary{two, primary, one, broken, one})
	want := []string{"/workspace/pair", one.StartingPath, two.StartingPath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
}
