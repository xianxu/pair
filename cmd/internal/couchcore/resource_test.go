package couchcore

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"reflect"
	"testing"
	"time"
)

func TestResourceSelectDeduplicatesAndRetainsProvenance(t *testing.T) {
	root := ResourceRoot{Key: ResourceProcessKey{1, "a"}, Slot: "r:1", Thread: ThreadAddress{Tag: "old"}}
	slots := []ResourceSlot{{Slot: "r:1", OwnershipKnown: true, Roots: []ResourceRoot{root, root}}}
	table := []procutil.Process{{PID: 1, Identity: "a"}, {PID: 2, PPID: 1, Identity: "b"}}
	got := SelectResourceProcesses(table, slots, nil, true)
	if len(got.Processes) != 2 {
		t.Fatalf("selection: %+v", got)
	}
	table = table[1:]
	table[0].PPID = 1
	got = SelectResourceProcesses(table, nil, got.Processes, true)
	if len(got.Processes) != 1 || !got.Processes[0].Orphan || got.Processes[0].Thread.Tag != "old" {
		t.Fatalf("lost provenance: %+v", got)
	}
	unknown := SelectResourceProcesses(table, nil, got.Processes, false)
	if len(unknown.Processes) != 1 || unknown.Processes[0].Orphan {
		t.Fatalf("unknown root established orphan: %+v", unknown)
	}
}
func TestResourceRatesAndInputImmutable(t *testing.T) {
	p, _ := ParseResourcePolicy(nil)
	at := time.Now()
	key := ResourceProcessKey{2, "b"}
	s := ResourceSample{At: at, TableKnown: true, Table: []procutil.Process{{PID: 2, Identity: "b"}}, Slots: []ResourceSlot{{Slot: "r:1", OwnershipKnown: true, Roots: []ResourceRoot{{Key: key, Slot: "r:1"}}}}, Observations: []ResourceUsageObservation{{Key: key, CPUKnown: true, RSSKnown: true, DiskKnown: true, CPUTimeNS: 100, RSSBytes: 100, DiskWriteBytes: 100}}}
	st, v, _ := AdvanceResources(ResourceState{}, s, p)
	if v.Processes[0].CPUPercent != nil {
		t.Fatal("first sample has rate")
	}
	before := st.Previous[key]
	s.At = at.Add(time.Minute)
	s.Observations[0].CPUTimeNS += 60e9
	s.Observations[0].DiskWriteBytes += 120
	next, v, _ := AdvanceResources(st, s, p)
	if v.Processes[0].CPUPercent == nil || *v.Processes[0].CPUPercent != 100 || *v.Processes[0].WriteBytesPerSecond != 2 {
		t.Fatalf("rates: %+v", v.Processes[0])
	}
	if !reflect.DeepEqual(st.Previous[key], before) || next.ObservedCPUTimeNS != 60e9 {
		t.Fatal("mutated input or lifetime counted")
	}
	s.At = s.At.Add(10 * time.Minute)
	_, v, _ = AdvanceResources(next, s, p)
	if v.Processes[0].CPUPercent != nil {
		t.Fatal("rate crossed gap")
	}
}
func TestResourceConflictCycleReuseAndUnknown(t *testing.T) {
	root := ResourceRoot{Key: ResourceProcessKey{10, "a"}, Slot: "r:1"}
	slots := []ResourceSlot{{Slot: "r:1", OwnershipKnown: true, Roots: []ResourceRoot{root}}}
	table := []procutil.Process{{PID: 10, Identity: "a"}, {PID: 20, PPID: 10, Identity: "b"}}
	prior := SelectResourceProcesses(table, slots, nil, true).Processes
	table[1].PPID = 1
	got := SelectResourceProcesses(table, slots, prior, true)
	if !got.Processes[1].Orphan {
		t.Fatal("reparented process not orphan")
	}
	slots[0].OwnershipKnown = false
	got = SelectResourceProcesses(table, slots, prior, true)
	if got.Processes[1].Orphan {
		t.Fatal("unknown ownership marked orphan")
	}
	slots[0].OwnershipKnown = true
	table[1].Identity = "reuse"
	got = SelectResourceProcesses(table, slots, prior, true)
	if len(got.Processes) != 1 {
		t.Fatal("reused PID inherited ownership")
	}
	table[1].Identity = "b"
	table[1].PPID = 10
	slots = append(slots, ResourceSlot{Slot: "r:2", OwnershipKnown: true, Roots: []ResourceRoot{{Key: ResourceProcessKey{20, "b"}, Slot: "r:2"}}})
	got = SelectResourceProcesses(table, slots, nil, true)
	if len(got.Processes) != 1 {
		t.Fatal("conflicting roots double charged")
	}
	table[0].PPID = 20
	got = SelectResourceProcesses(table, slots, nil, true)
	if len(got.Processes) != 0 {
		t.Fatal("cycle accepted")
	}
}
func TestResourceClassifyUnproven(t *testing.T) {
	key := ResourceProcessKey{20, "b"}
	table := []procutil.Process{{PID: 20, PPID: 1, Identity: "b", Name: "nvim"}}
	obs := []ResourceUsageObservation{{Key: key, RSSKnown: true, RSSBytes: 300 << 20}}
	if len(ClassifyUnattributed(table, nil, obs, nil, 256<<20)) != 0 {
		t.Fatal("nvim without embed")
	}
	if len(ClassifyUnattributed(table, nil, obs, map[ResourceProcessKey][]string{key: {"nvim", "--embed"}}, 256<<20)) != 1 {
		t.Fatal("embed missing")
	}
	table[0].Identity = "new"
	if len(ClassifyUnattributed(table, nil, obs, map[ResourceProcessKey][]string{key: {"--embed"}}, 256<<20)) != 0 {
		t.Fatal("reuse attributed")
	}
}
func TestResourceSelectionOrderIndependent(t *testing.T) {
	table := []procutil.Process{{PID: 10, Identity: "a"}, {PID: 20, PPID: 10, Identity: "b"}, {PID: 30, PPID: 20, Identity: "c"}}
	slots := []ResourceSlot{{Slot: "r:1", OwnershipKnown: true, Roots: []ResourceRoot{{Key: ResourceProcessKey{10, "a"}, Slot: "r:1"}}}}
	a := SelectResourceProcesses(table, slots, nil, true)
	table[0], table[2] = table[2], table[0]
	b := SelectResourceProcesses(table, slots, nil, true)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("table order changed selection")
	}
}
func TestResourceUnknownIdentityRetainsProvenance(t *testing.T) {
	old := ResourceProvenance{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{5, "old"}, Slot: "r:1"}, Root: ResourceProcessKey{2, "root"}}
	got := SelectResourceProcesses([]procutil.Process{{PID: 5}}, nil, []ResourceProvenance{old}, true)
	if len(got.Processes) != 1 || got.Processes[0].OrphanKnown {
		t.Fatalf("unknown generation lost or resolved: %+v", got)
	}
}
func TestResourceEpisodeCapPreservesPendingAndSuppressesNewEffects(t *testing.T) {
	p, _ := ParseResourcePolicy(nil)
	state := ResourceState{}
	for i := 0; i < 2*ResourceMaxProcesses+2*ResourceMaxSlots; i++ {
		state.Episodes = append(state.Episodes, ResourceEpisode{Key: fmt.Sprintf("write/old:%d/thread", i), Slot: fmt.Sprintf("old:%d", i), Metric: "slot-write", Phase: "notified", Effect: &ResourceEffect{ID: fmt.Sprint(i)}})
	}
	k := ResourceProcessKey{20, "new"}
	sample := ResourceSample{At: time.Now(), TableKnown: true, Table: []procutil.Process{{PID: 20, Identity: "new"}}, Slots: []ResourceSlot{{Slot: "current:1", OwnershipKnown: true, Roots: []ResourceRoot{{Key: k, Slot: "current:1"}}}}, Observations: []ResourceUsageObservation{{Key: k, RSSKnown: true, RSSBytes: 9 << 30}}}
	next, v, effects := AdvanceResources(state, sample, p)
	if len(next.Episodes) > len(state.Episodes) || len(effects) != 0 || len(v.Errors) == 0 {
		t.Fatalf("unbounded or dropped dedup: episodes=%d effects=%d errors=%v", len(next.Episodes), len(effects), v.Errors)
	}
	for _, e := range next.Episodes {
		if e.Phase != "notified" {
			t.Fatal("pending admission displaced")
		}
	}
}
func TestResourceReplacedThreadRetiresEpisodesAndKeepsActiveTotals(t *testing.T) {
	p, _ := ParseResourcePolicy(nil)
	state := ResourceState{Episodes: []ResourceEpisode{{Key: "write/r:1/old", Slot: "r:1", Metric: "slot-write", Phase: "notified", Effect: &ResourceEffect{ID: "old"}}}}
	for i := 0; i < ResourceMaxSlots; i++ {
		state.SlotTotals = append(state.SlotTotals, ResourceSlotTotal{Slot: "r:1", Thread: ThreadAddress{Tag: ThreadTag(fmt.Sprintf("a%03d", i))}, CPUTimeNS: 42})
	}
	newThread := ThreadAddress{Tag: "zz-current"}
	state.SlotTotals = append(state.SlotTotals, ResourceSlotTotal{Slot: "r:1", Thread: newThread, CPUTimeNS: 99})
	s := ResourceSample{At: time.Now(), TableKnown: true, Slots: []ResourceSlot{{Slot: "r:1", Thread: newThread, OwnershipKnown: false}}}
	unknown, _, _ := AdvanceResources(state, s, p)
	found := false
	for _, e := range unknown.Episodes {
		if e.Key == "write/r:1/old" {
			found = true
		}
	}
	if !found {
		t.Fatal("unknown ownership retired episode")
	}
	s.Slots[0].OwnershipKnown = true
	next, _, _ := AdvanceResources(state, s, p)
	for _, e := range next.Episodes {
		if e.Key == "write/r:1/old" {
			t.Fatal("replaced thread retained episode")
		}
	}
	found = false
	for _, total := range next.SlotTotals {
		if total.Thread == newThread && total.CPUTimeNS == 99 {
			found = true
		}
	}
	if !found {
		t.Fatal("active thread totals dropped behind retired totals")
	}
}
func TestResourceEpisodeUnknownGenerationDoesNotRetireAdmission(t *testing.T) {
	p, _ := ParseResourcePolicy(nil)
	key := ResourceProcessKey{20, "original"}
	state := ResourceState{Episodes: []ResourceEpisode{{Key: "rss/20/original", Process: key, Metric: "process-rss", Phase: "notified", Effect: &ResourceEffect{ID: "admitted", Process: key}}}}
	for _, rows := range [][]procutil.Process{{{PID: 20}}, {{PID: 20, Identity: "other"}, {PID: 20, Identity: "different"}}} {
		next, _, effects := AdvanceResources(state, ResourceSample{At: time.Now(), TableKnown: true, Table: rows}, p)
		if len(next.Episodes) != 1 || next.Episodes[0].Phase != "notified" || len(effects) != 0 {
			t.Fatalf("unknown pruned admitted episode: %+v", next.Episodes)
		}
	}
	for _, rows := range [][]procutil.Process{nil, {{PID: 20, Identity: "replacement"}}, {{PID: 20, Identity: "original", Zombie: true}}} {
		next, _, _ := AdvanceResources(state, ResourceSample{At: time.Now(), TableKnown: true, Table: rows}, p)
		if len(next.Episodes) != 0 {
			t.Fatalf("confirmed terminal retained: %+v", next.Episodes)
		}
	}
}
func TestResourceSlotWriteAlertNamesLargestContributor(t *testing.T) {
	p, _ := ParseResourcePolicy(nil)
	p.WriteDuration = 0
	p.WriteBytesPerSecond = 1
	at := time.Now()
	root := ResourceProcessKey{10, "a"}
	child := ResourceProcessKey{20, "b"}
	s := ResourceSample{At: at, TableKnown: true, Table: []procutil.Process{{PID: 10, Identity: "a", Name: "wrapper"}, {PID: 20, PPID: 10, Identity: "b", Name: "writer"}}, Slots: []ResourceSlot{{Slot: "r:1", OwnershipKnown: true, Roots: []ResourceRoot{{Key: root, Slot: "r:1"}}}}, Observations: []ResourceUsageObservation{{Key: root, DiskKnown: true}, {Key: child, DiskKnown: true}}}
	state, _, _ := AdvanceResources(ResourceState{}, s, p)
	s.At = at.Add(time.Minute)
	s.Observations[0].DiskWriteBytes = 120
	s.Observations[1].DiskWriteBytes = 600
	_, _, effects := AdvanceResources(state, s, p)
	for _, effect := range effects {
		if effect.Metric == "slot-write" {
			if effect.Process != child || effect.Name != "writer" {
				t.Fatalf("missing offender: %+v", effect)
			}
			return
		}
	}
	t.Fatal("missing write alert")
}
