package couchcore

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// orphanInputs is one candidate as the sampler hands it over: the kernel row plus
// every claim the pass proved about it.
type orphanInputs struct {
	row     procutil.Process
	args    []string
	claimed map[ResourceProcessKey]bool
	probe   ResourceOrphanProbe
	rss     *uint64
	roots   []string
	minAge  time.Duration
}

func orphanDefaults() orphanInputs {
	rss := uint64(22 << 30)
	return orphanInputs{
		row:     procutil.Process{PID: 4242, PPID: 1, Identity: "1760000000.5", Name: "/opt/homebrew/bin/nvim"},
		args:    []string{"nvim", "--embed"},
		claimed: map[ResourceProcessKey]bool{},
		probe: ResourceOrphanProbe{
			Cwd:      "/fleet/worktree/pair-slot1/pair/cmd",
			CwdKnown: true,
			Age:      13 * 24 * time.Hour,
			AgeKnown: true,
		},
		rss:    &rss,
		roots:  []string{"/fleet/pair", "/fleet/worktree/pair-slot1/pair"},
		minAge: 6 * time.Hour,
	}
}

func TestClassifyOrphanNvimFalsifiesOneClaimPerRow(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		falsify  func(*orphanInputs)
		want     bool
		wantRoot string
		noRSS    bool
	}{
		{name: "every claim intact", want: true, wantRoot: "/fleet/worktree/pair-slot1/pair"},
		{name: "exactly at the threshold", falsify: func(i *orphanInputs) { i.probe.Age = i.minAge }, want: true, wantRoot: "/fleet/worktree/pair-slot1/pair"},
		{name: "an unread RSS does not hide an orphan", falsify: func(i *orphanInputs) { i.rss = nil }, want: true, wantRoot: "/fleet/worktree/pair-slot1/pair", noRSS: true},
		{name: "the most specific checkout wins", falsify: func(i *orphanInputs) { i.roots = append(i.roots, "/fleet/worktree/pair-slot1/pair/cmd") }, want: true, wantRoot: "/fleet/worktree/pair-slot1/pair/cmd"},
		{name: "still has a parent", falsify: func(i *orphanInputs) { i.row.PPID = 99 }},
		{name: "a zombie", falsify: func(i *orphanInputs) { i.row.Zombie = true }},
		{name: "no generation to fence on", falsify: func(i *orphanInputs) { i.row.Identity = "" }},
		{name: "no pid", falsify: func(i *orphanInputs) { i.row.PID = 0 }},
		{name: "claimed by a slot", falsify: func(i *orphanInputs) { i.claimed[resourceKey(i.row)] = true }},
		{name: "a plain nvim is the operator's own", falsify: func(i *orphanInputs) { i.args = []string{"nvim", "notes.md"} }},
		{name: "argv not read", falsify: func(i *orphanInputs) { i.args = nil }},
		{name: "an unattributed kind that is not nvim", falsify: func(i *orphanInputs) { i.row.Name = "zellij" }},
		{name: "cwd not read", falsify: func(i *orphanInputs) { i.probe.Cwd, i.probe.CwdKnown = "", false }},
		{name: "cwd read as empty", falsify: func(i *orphanInputs) { i.probe.Cwd = "" }},
		{name: "cwd outside every managed checkout", falsify: func(i *orphanInputs) { i.probe.Cwd = "/home/operator/scratch" }},
		{name: "cwd shares a prefix with a managed checkout", falsify: func(i *orphanInputs) { i.probe.Cwd = "/fleet/pair-slot1/pair/cmd" }},
		{name: "no managed checkout known", falsify: func(i *orphanInputs) { i.roots = nil }},
		{name: "age unknown", falsify: func(i *orphanInputs) { i.probe.Age, i.probe.AgeKnown = 0, false }},
		{name: "younger than the threshold", falsify: func(i *orphanInputs) { i.probe.Age = i.minAge - time.Nanosecond }},
		{name: "a start stamp in the future is not an age", falsify: func(i *orphanInputs) { i.probe.Age, i.minAge = -time.Second, 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			in := orphanDefaults()
			if test.falsify != nil {
				test.falsify(&in)
			}
			got, ok := ClassifyOrphanNvim(in.row, in.args, in.claimed, in.probe, in.rss, in.roots, in.minAge)
			if ok != test.want {
				t.Fatalf("ClassifyOrphanNvim(%+v, %v, %+v) = %v; want %v", in.row, in.args, in.probe, ok, test.want)
			}
			if !ok {
				if got != (OrphanNvim{}) {
					t.Fatalf("a refused candidate carried %+v", got)
				}
				return
			}
			if got.Key != resourceKey(in.row) || got.Cwd != in.probe.Cwd || got.Age != in.probe.Age || got.Root != test.wantRoot || got.Reason == "" {
				t.Fatalf("classified %+v from %+v", got, in)
			}
			if (got.RSSBytes == nil) != test.noRSS || !test.noRSS && *got.RSSBytes != *in.rss {
				t.Fatalf("rss %v from %v", got.RSSBytes, in.rss)
			}
		})
	}
}

func TestManagedRootsCollectsEverySlotCheckout(t *testing.T) {
	t.Parallel()
	if got := ManagedRoots(nil); len(got) != 0 {
		t.Fatalf("an empty inventory named %v", got)
	}
	slot := func(root string) ResourceSlot { return ResourceSlot{Workspace: WorkspaceIdentity{WorktreeRoot: root}} }
	got := ManagedRoots([]ResourceSlot{slot("/b"), slot(""), slot("/a"), slot("/b"), {}})
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("ManagedRoots = %v; want the checkouts deduplicated and sorted", got)
	}
}

func TestWithinManagedRootIsSegmentSafe(t *testing.T) {
	t.Parallel()
	roots := []string{"/fleet/pair", "/fleet/worktree/pair-slot1/pair", "/fleet/worktree/pair-slot1/pair/nested"}
	for _, test := range []struct{ cwd, want string }{
		{cwd: "/fleet/pair", want: "/fleet/pair"},
		{cwd: "/fleet/pair/cmd", want: "/fleet/pair"},
		{cwd: "/fleet/worktree/pair-slot1/pair/nested/deep", want: "/fleet/worktree/pair-slot1/pair/nested"},
		{cwd: "/fleet/pair-slot1/pair/cmd"},
		{cwd: "/fleet/pai"},
		{cwd: "/fleet"},
		{cwd: "cmd/internal"},
		{cwd: "/fleet/pair/../pair-slot2"},
	} {
		got, ok := WithinManagedRoot(test.cwd, roots)
		if ok != (test.want != "") || got != test.want {
			t.Fatalf("WithinManagedRoot(%q) = %q, %v; want %q", test.cwd, got, ok, test.want)
		}
	}
	if got, ok := WithinManagedRoot("/fleet/pair/cmd", nil); ok || got != "" {
		t.Fatalf("no managed checkout contains %q, %v", got, ok)
	}
}

// The view is the operator's only window on the sweep, so a pass that probed
// nothing must report nothing: silence there means "not proven", never "clean".
func TestAdvanceResourcesReportsTheOrphanNvimSweep(t *testing.T) {
	t.Parallel()
	p, err := ParseResourcePolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	key := ResourceProcessKey{PID: 4242, Identity: "1760000000.5"}
	probe := ResourceOrphanProbe{Cwd: "/fleet/worktree/pair-slot1/pair/cmd", CwdKnown: true, Age: 13 * 24 * time.Hour, AgeKnown: true}
	s := ResourceSample{
		At: time.Unix(1760100000, 0), TableKnown: true, InventoryComplete: true,
		Table:        []procutil.Process{{PID: key.PID, PPID: 1, Identity: key.Identity, Name: "/opt/homebrew/bin/nvim"}},
		Slots:        []ResourceSlot{{Slot: "pair:1", OwnershipKnown: true, Workspace: WorkspaceIdentity{WorktreeRoot: "/fleet/worktree/pair-slot1/pair"}}},
		Observations: []ResourceUsageObservation{{Key: key, RSSKnown: true, RSSBytes: 22 << 30}},
		Commands:     map[ResourceProcessKey][]string{key: {"nvim", "--embed"}},
		OrphanProbes: map[ResourceProcessKey]ResourceOrphanProbe{key: probe},
	}
	_, v, _ := AdvanceResources(ResourceState{}, s, p)
	if len(v.OrphanNvim) != 1 {
		t.Fatalf("orphans %+v; errors %v", v.OrphanNvim, v.Errors)
	}
	got := v.OrphanNvim[0]
	if got.Key != key || got.Root != "/fleet/worktree/pair-slot1/pair" || got.Cwd != probe.Cwd || got.Age != probe.Age || got.RSSBytes == nil || *got.RSSBytes != 22<<30 || got.Reason == "" {
		t.Fatalf("orphan %+v", got)
	}
	for _, row := range v.Unattributed {
		if row.Key == key {
			t.Fatalf("the specific sweep must subsume the generic row: %+v", v.Unattributed)
		}
	}
	s.OrphanProbes = nil
	if _, v, _ = AdvanceResources(ResourceState{}, s, p); len(v.OrphanNvim) != 0 {
		t.Fatalf("an unprobed pass reported %+v", v.OrphanNvim)
	}
	s.OrphanProbes = map[ResourceProcessKey]ResourceOrphanProbe{key: probe}
	wide := p
	wide.OrphanNvimAge = 30 * 24 * time.Hour
	if _, v, _ = AdvanceResources(ResourceState{}, s, wide); len(v.OrphanNvim) != 0 {
		t.Fatalf("an age below the threshold was reported: %+v", v.OrphanNvim)
	}
}

// A leak that pages every cadence is noise and one that never pages is the
// 13-day silence #583 exists to end, so this walks both edges of the episode:
// accumulate, fire once, stay quiet, clear when the process goes, and re-arm
// under a recycled pid as the different process it is.
func TestOrphanNvimAlertsTheHubOncePerEpisode(t *testing.T) {
	t.Parallel()
	p, err := ParseResourcePolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	const root = "/fleet/worktree/pair-slot1/pair"
	sample := func(at time.Time, identity string) ResourceSample {
		s := ResourceSample{
			At: at, TableKnown: true, InventoryComplete: true,
			Slots: []ResourceSlot{{Slot: "pair:1", OwnershipKnown: true, Workspace: WorkspaceIdentity{WorktreeRoot: root}}},
		}
		if identity == "" {
			return s
		}
		key := ResourceProcessKey{PID: 4242, Identity: identity}
		s.Table = []procutil.Process{{PID: key.PID, PPID: 1, Identity: identity, Name: "/opt/homebrew/bin/nvim"}}
		s.Observations = []ResourceUsageObservation{{Key: key, RSSKnown: true, RSSBytes: 22 << 30}}
		s.Commands = map[ResourceProcessKey][]string{key: {"nvim", "--embed"}}
		s.OrphanProbes = map[ResourceProcessKey]ResourceOrphanProbe{key: {Cwd: root + "/cmd", CwdKnown: true, Age: 13 * 24 * time.Hour, AgeKnown: true}}
		return s
	}
	at := time.Unix(1760100000, 0)
	key := ResourceProcessKey{PID: 4242, Identity: "1760000000.5"}
	state, v, effects := AdvanceResources(ResourceState{}, sample(at, key.Identity), p)
	if len(effects) != 0 || len(v.OrphanNvim) != 1 {
		t.Fatalf("one sample alerted %+v; reported %d", effects, len(v.OrphanNvim))
	}
	state, _, effects = AdvanceResources(state, sample(at.Add(time.Minute), key.Identity), p)
	if len(effects) != 1 {
		t.Fatalf("effects %+v; episodes %+v", effects, state.Episodes)
	}
	effect := effects[0]
	if effect.Metric != "orphan-nvim" || effect.Process != key || effect.Name != "/opt/homebrew/bin/nvim" ||
		effect.Root != root || effect.Age != 13*24*time.Hour || effect.Workspace.WorktreeRoot != root || effect.ID == "" {
		t.Fatalf("effect %+v", effect)
	}
	for i := 2; i < 5; i++ {
		state, _, effects = AdvanceResources(state, sample(at.Add(time.Duration(i)*time.Minute), key.Identity), p)
		if len(effects) != 0 {
			t.Fatalf("sample %d repeated the alert: %+v", i, effects)
		}
	}
	// Reaped: the episode must clear rather than linger as a permanent row.
	state, _, effects = AdvanceResources(state, sample(at.Add(5*time.Minute), ""), p)
	if len(effects) != 0 {
		t.Fatalf("a gone orphan alerted %+v", effects)
	}
	for _, e := range state.Episodes {
		if strings.HasPrefix(e.Key, "nvim/") {
			t.Fatalf("episode survived the reap: %+v", e)
		}
	}
	recycled := ResourceProcessKey{PID: key.PID, Identity: "1760200000.25"}
	state, _, effects = AdvanceResources(state, sample(at.Add(6*time.Minute), recycled.Identity), p)
	if len(effects) != 0 {
		t.Fatalf("a recycled pid alerted on one sample: %+v", effects)
	}
	state, _, effects = AdvanceResources(state, sample(at.Add(7*time.Minute), recycled.Identity), p)
	if len(effects) != 1 || effects[0].ID == effect.ID || effects[0].Process != recycled {
		t.Fatalf("recycled pid did not re-arm: %+v (first %s)", effects, effect.ID)
	}
}
