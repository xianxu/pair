package couchcore

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func provenReapWorld(t *testing.T) (ProvenOrphanTarget, ProvenOrphanProof) {
	t.Helper()
	b, _ := json.Marshal(identityJSON(t, true))
	ws, err := ParseWorkspaceIdentity(b)
	if err != nil {
		t.Fatal(err)
	}
	thread := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "1-my-repo-2"}
	root := ResourceRoot{Key: ResourceProcessKey{10, "old"}, Slot: "my-repo:2", Thread: thread, Workspace: ws}
	before := []procutil.Process{{PID: 10, PPID: 1, Identity: "old", Name: "zellij"}, {PID: 11, PPID: 10, Identity: "child", Name: "codex"}}
	slots := []ResourceSlot{{Slot: root.Slot, Thread: thread, Workspace: ws, OwnershipKnown: true, Roots: []ResourceRoot{root}}}
	first := SelectResourceProcesses(before, slots, nil, true, true)
	slots[0].Roots = []ResourceRoot{{Key: ResourceProcessKey{20, "new"}, Slot: root.Slot, Thread: thread, Workspace: ws}}
	table := append(before, procutil.Process{PID: 20, PPID: 1, Identity: "new", Name: "zellij"})
	selection := SelectResourceProcesses(table, slots, first.Processes, true, true)
	proof := ProvenOrphanProof{Table: table, Selection: selection, TableKnown: true, InventoryComplete: true}
	view := ResourceView{}
	for _, p := range selection.Processes {
		if p.Orphan {
			view.Orphans = append(view.Orphans, ResourceProcessView{ResourceProvenance: p})
		}
	}
	targets := ProvenOrphanTargets(view)
	if len(targets) != 2 {
		t.Fatalf("targets %+v", targets)
	}
	return targets[1], proof
}

func TestProvenOrphanTargetsUseExactPriorOwnership(t *testing.T) {
	target, proof := provenReapWorld(t)
	var view ResourceView
	for _, p := range proof.Selection.Processes {
		view.Orphans = append(view.Orphans, ResourceProcessView{ResourceProvenance: p})
	}
	before := append([]ResourceProcessView(nil), view.Orphans...)
	got := ProvenOrphanTargets(view)
	if len(got) != 2 || got[1].Key != target.Key {
		t.Fatalf("targets %+v", got)
	}
	if !reflect.DeepEqual(view.Orphans, before) {
		t.Fatal("mutated input")
	}
	// Every missing witness removes eligibility; current rows never enter.
	for _, change := range []func(*ResourceProvenance){
		func(p *ResourceProvenance) { p.Key.Identity = "" }, func(p *ResourceProvenance) { p.Root.Identity = "" },
		func(p *ResourceProvenance) { p.OrphanKnown = false }, func(p *ResourceProvenance) { p.Orphan = false },
		func(p *ResourceProvenance) { p.Workspace.RepoIdentity = "" }, func(p *ResourceProvenance) { p.Slot = "other:2" },
		func(p *ResourceProvenance) { p.Thread = ThreadAddress{} },
	} {
		row := view.Orphans[0]
		change(&row.ResourceProvenance)
		if got := ProvenOrphanTargets(ResourceView{Orphans: []ResourceProcessView{row}}); len(got) != 0 {
			t.Fatalf("accepted %+v", row)
		}
	}
	view.Orphans = append(view.Orphans, view.Orphans[0])
	if got := ProvenOrphanTargets(view); len(got) != 1 {
		t.Fatalf("ambiguous duplicate accepted %+v", got)
	}
}

func TestProvenOrphanSignalDecision(t *testing.T) {
	target, original := provenReapWorld(t)
	for _, tc := range []struct {
		name, want string
		change     func(*ProvenOrphanProof)
	}{
		{"old beside same-thread current", "sent", func(*ProvenOrphanProof) {}},
		{"unknown table", "refused", func(p *ProvenOrphanProof) { p.TableKnown = false }},
		{"unknown inventory", "refused", func(p *ProvenOrphanProof) { p.InventoryComplete = false }},
		{"current ownership", "refused", func(p *ProvenOrphanProof) { p.Selection.Processes[1].Orphan = false }},
		{"unknown detached ancestor", "refused", func(p *ProvenOrphanProof) { p.UnknownDetached = []ResourceProcessKey{{10, "old"}} }},
		{"unrelated unknown", "sent", func(p *ProvenOrphanProof) { p.UnknownDetached = []ResourceProcessKey{{99, "other"}} }},
		{"replacement", "skipped", func(p *ProvenOrphanProof) { p.Table[1].Identity = "replacement" }},
		{"missing identity", "refused", func(p *ProvenOrphanProof) { p.Table[1].Identity = "" }},
		{"ambiguous pid", "refused", func(p *ProvenOrphanProof) { p.Table = append(p.Table, p.Table[1]) }},
		{"unavailable selection", "refused", func(p *ProvenOrphanProof) { p.Selection.Processes = nil }},
		{"conflict", "refused", func(p *ProvenOrphanProof) { p.Selection.Errors = []string{"conflicting process ownership"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := original
			p.Table = append([]procutil.Process(nil), original.Table...)
			p.Selection.Processes = append([]ResourceProvenance(nil), original.Selection.Processes...)
			tc.change(&p)
			status, detail := DecideProvenOrphanSignal(target, p)
			if status != tc.want {
				t.Fatalf("%s: %s", status, detail)
			}
		})
	}
}

func TestResourceOwnershipCompleteRejectsUncertaintyAndLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ResourceSample, *ResourceSelection, *[]ResourceProcessKey)
	}{
		{"unknown table", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.TableKnown = false }},
		{"incomplete inventory", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.InventoryComplete = false }},
		{"omitted inventory", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.InventoryOmitted = 1 }},
		{"invalid omission", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.InventoryOmitted = -1 }},
		{"table capacity", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Table = make([]procutil.Process, ResourceMaxTableRows+1)
		}},
		{"slot capacity", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Slots = make([]ResourceSlot, ResourceMaxSlots+1)
		}},
		{"selection capacity", func(_ *ResourceSample, s *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Processes = make([]ResourceProvenance, ResourceMaxProcesses+1)
		}},
		{"selection error", func(_ *ResourceSample, s *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Errors = []string{"conflicting process ownership"}
		}},
		{"unknown detached", func(_ *ResourceSample, _ *ResourceSelection, u *[]ResourceProcessKey) {
			*u = []ResourceProcessKey{{PID: 99, Identity: "unrelated"}}
		}},
		{"duplicate PID", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Table = append(s.Table, s.Table[0])
		}},
		{"zero PID", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.Table[0].PID = 0 }},
		{"negative PID", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.Table[0].PID = -1 }},
		{"unknown live identity", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.Table[0].Identity = "" }},
		{"empty slot label", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) { s.Slots[0].Slot = "" }},
		{"duplicate slot label", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Slots = append(s.Slots, s.Slots[0])
		}},
		{"zero root PID", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Slots[0].Roots = []ResourceRoot{{Key: ResourceProcessKey{Identity: "known"}}}
		}},
		{"negative root PID", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Slots[0].Roots = []ResourceRoot{{Key: ResourceProcessKey{PID: -1, Identity: "known"}}}
		}},
		{"missing root identity", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Slots[0].Roots = []ResourceRoot{{Key: ResourceProcessKey{PID: 1}}}
		}},

		{"unknown slot ownership", func(s *ResourceSample, _ *ResourceSelection, _ *[]ResourceProcessKey) {
			s.Slots[0].OwnershipKnown = false
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sample := ResourceSample{TableKnown: true, InventoryComplete: true, Table: []procutil.Process{{PID: 1, Identity: "live"}}, Slots: []ResourceSlot{{Slot: "repo:0", OwnershipKnown: true}}}
			selection := ResourceSelection{}
			var unknown []ResourceProcessKey
			tc.mutate(&sample, &selection, &unknown)
			if ResourceOwnershipComplete(sample, selection, unknown) {
				t.Fatal("uncertain ownership was declared complete")
			}
		})
	}
}

func TestResourceOwnershipCompleteAcceptsPositiveZombieDeathAndEmptyWorld(t *testing.T) {
	for _, rows := range [][]procutil.Process{nil, {{PID: 1, Zombie: true}}, {{PID: 1, Identity: "old", Zombie: true}, {PID: 2, Identity: "live"}}} {
		sample := ResourceSample{TableKnown: true, InventoryComplete: true, Table: rows}
		if !ResourceOwnershipComplete(sample, ResourceSelection{}, nil) {
			t.Fatalf("authoritative world refused: %+v", rows)
		}
	}
}

func TestResourceOwnershipCompleteIndependentProcessAndSlotGrowth(t *testing.T) {
	for _, processes := range []int{8, 16} {
		for _, slots := range []int{8, 16} {
			sample := ResourceSample{TableKnown: true, InventoryComplete: true}
			for i := 0; i < processes; i++ {
				sample.Table = append(sample.Table, procutil.Process{PID: i + 1, Identity: "known"})
			}
			for i := 0; i < slots; i++ {
				sample.Slots = append(sample.Slots, ResourceSlot{Slot: fmt.Sprintf("repo:%d", i), OwnershipKnown: true})
			}
			before, _ := json.Marshal(sample)
			if !ResourceOwnershipComplete(sample, ResourceSelection{}, nil) {
				t.Fatalf("complete P=%d S=%d refused", processes, slots)
			}
			after, _ := json.Marshal(sample)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("mutated ownership inputs")
			}
		}
	}
}

func TestResourceOwnershipCompleteRetainsGenericMonitorCompatibility(t *testing.T) {
	sample := ResourceSample{TableKnown: true, InventoryComplete: true, Table: []procutil.Process{{PID: 1, Identity: "live"}}, Slots: []ResourceSlot{{Slot: "repo:0", OwnershipKnown: true}}, Errors: []string{"sampler CPU start: unavailable"}}
	if !ResourceOwnershipComplete(sample, ResourceSelection{}, nil) {
		t.Fatal("known absent root or unrelated metric error revoked completeness")
	}
	root := ResourceRoot{Key: ResourceProcessKey{PID: 1, Identity: "live"}}
	sample.Slots[0].Roots = []ResourceRoot{root, root}
	if !ResourceOwnershipComplete(sample, ResourceSelection{}, nil) {
		t.Fatal("redundant identical root witness was treated as ownership loss")
	}
}
