package couchcore

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func legacyProofWorld(family string) (LegacyOrphanTarget, ProvenOrphanProof, ResourceView) {
	target := LegacyOrphanTarget{Family: family, Root: ResourceProcessKey{PID: 10, Identity: "root"}, Key: ResourceProcessKey{PID: 11, Identity: "child"}}
	proof := ProvenOrphanProof{TableKnown: true, InventoryComplete: true, Table: []procutil.Process{{PID: 10, PPID: 1, Identity: "root"}, {PID: 11, PPID: 10, Identity: "child"}, {PID: 12, PPID: 11, Identity: "descendant"}}}
	view := ResourceView{}
	if family == "orphan-nvim" {
		view.OrphanNvim = []OrphanNvim{{Key: target.Root}}
	} else {
		view.LeftRunning = []LeftRunningOrphan{{Process: ProcessIdentity{PID: 10, Identity: "root"}}}
	}
	return target, proof, view
}

func TestLegacyOrphanTargetValidate(t *testing.T) {
	for _, family := range []string{"orphan-nvim", "left-running"} {
		target, _, _ := legacyProofWorld(family)
		if err := target.Validate(); err != nil {
			t.Fatal(err)
		}
		for _, change := range []func(*LegacyOrphanTarget){func(t *LegacyOrphanTarget) { t.Family = "" }, func(t *LegacyOrphanTarget) { t.Family = "proven" }, func(t *LegacyOrphanTarget) { t.Key.PID = 0 }, func(t *LegacyOrphanTarget) { t.Key.Identity = "" }, func(t *LegacyOrphanTarget) { t.Root.PID = -1 }, func(t *LegacyOrphanTarget) { t.Root.Identity = "" }} {
			bad := target
			change(&bad)
			if bad.Validate() == nil {
				t.Fatalf("accepted %+v", bad)
			}
		}
	}
}

func TestDecideLegacyOrphanSignal(t *testing.T) {
	for _, family := range []string{"orphan-nvim", "left-running"} {
		t.Run(family, func(t *testing.T) {
			for _, tc := range []struct {
				name, want string
				change     func(*LegacyOrphanTarget, *ProvenOrphanProof, *ResourceView)
			}{
				{"exact descendant", "sent", func(*LegacyOrphanTarget, *ProvenOrphanProof, *ResourceView) {}},
				{"exact root", "sent", func(t *LegacyOrphanTarget, _ *ProvenOrphanProof, _ *ResourceView) { t.Key = t.Root }},
				{"target absent", "skipped", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Table = append(p.Table[:1], p.Table[2:]...)
				}},
				{"target reused", "skipped", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Table[1].Identity = "replacement"
				}},
				{"target dead", "skipped", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table[1].Zombie = true }},
				{"root absent surviving child", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table = p.Table[1:] }},
				{"root reused surviving child", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table[0].Identity = "new-root" }},
				{"root dead surviving child", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table[0].Zombie = true }},
				{"reparented survivor", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table[1].PPID = 1 }},
				{"root no longer eligible", "refused", func(_ *LegacyOrphanTarget, _ *ProvenOrphanProof, v *ResourceView) {
					v.OrphanNvim = nil
					v.LeftRunning = nil
				}},
				{"incomplete table", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.TableKnown = false }},
				{"incomplete slots", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.InventoryComplete = false }},
				{"incomplete proof reason", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.IncompleteReason = "slot unknown"
				}},
				{"classification conflict", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Selection.Errors = []string{"conflict"}
				}},
				{"unknown detached evidence", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.UnknownDetached = []ResourceProcessKey{{PID: 99, Identity: "unknown"}}
				}},
				{"duplicate pid", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Table = append(p.Table, p.Table[1])
				}},
				{"unknown identity", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table[2].Identity = "" }},
				{"cyclic ancestry", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) { p.Table[0].PPID = 12 }},
				{"unrelated cycle", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Table = append(p.Table, procutil.Process{PID: 90, PPID: 91, Identity: "a"}, procutil.Process{PID: 91, PPID: 90, Identity: "b"})
				}},
				{"duplicate eligible root", "refused", func(_ *LegacyOrphanTarget, _ *ProvenOrphanProof, v *ResourceView) {
					v.OrphanNvim = append(v.OrphanNvim, v.OrphanNvim...)
					v.LeftRunning = append(v.LeftRunning, v.LeftRunning...)
				}},
				{"owned target", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Selection.Processes = []ResourceProvenance{{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{PID: 11, Identity: "child"}}}}
				}},
				{"protected descendant", "refused", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Selection.Processes = []ResourceProvenance{{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{PID: 12, Identity: "descendant"}}, Orphan: true, OrphanKnown: true}}
				}},
				{"incomplete orphan descendant", "refused", func(_ *LegacyOrphanTarget, _ *ProvenOrphanProof, v *ResourceView) {
					v.Orphans = []ResourceProcessView{{ResourceProvenance: ResourceProvenance{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{PID: 12, Identity: "descendant"}}}}}
				}},
				{"view owned descendant", "refused", func(_ *LegacyOrphanTarget, _ *ProvenOrphanProof, v *ResourceView) {
					v.Processes = []ResourceProcessView{{ResourceProvenance: ResourceProvenance{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{PID: 12, Identity: "descendant"}}}}}
				}},
				{"unrelated owned process", "sent", func(_ *LegacyOrphanTarget, p *ProvenOrphanProof, _ *ResourceView) {
					p.Table = append(p.Table, procutil.Process{PID: 99, PPID: 1, Identity: "other"})
					p.Selection.Processes = []ResourceProvenance{{ResourceRoot: ResourceRoot{Key: ResourceProcessKey{PID: 99, Identity: "other"}}}}
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					target, proof, view := legacyProofWorld(family)
					tc.change(&target, &proof, &view)
					before, _ := json.Marshal([]any{target, proof, view})
					status, detail := DecideLegacyOrphanSignal(target, proof, view)
					after, _ := json.Marshal([]any{target, proof, view})
					if status != tc.want {
						t.Fatalf("got %s (%s), want %s", status, detail, tc.want)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatal("decision mutated evidence")
					}
				})
			}
		})
	}
}
