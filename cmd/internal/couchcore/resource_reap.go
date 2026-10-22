package couchcore

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// ProvenOrphanTarget freezes the monitor's exact process and ownership witnesses.
// Received bytes are claims, never a signal permit without a fresh proof.
type ProvenOrphanTarget struct {
	Key       ResourceProcessKey `json:"key"`
	Root      ResourceProcessKey `json:"root"`
	Slot      string             `json:"slot"`
	Thread    ThreadAddress      `json:"thread"`
	Workspace WorkspaceIdentity  `json:"workspace"`
}

func (t ProvenOrphanTarget) Validate() error {
	if t.Key.PID <= 0 || t.Key.Identity == "" || t.Root.PID <= 0 || t.Root.Identity == "" {
		return fmt.Errorf("missing exact process generation")
	}
	if err := validateThreadAddress(t.Thread); err != nil {
		return err
	}
	if err := t.Workspace.validate(); err != nil {
		return err
	}
	if t.Workspace.Address == nil || t.Workspace.Slot == nil || *t.Workspace.Address != t.Slot {
		return fmt.Errorf("target lacks canonical slot ownership")
	}
	return nil
}

// ProvenOrphanProof is built exclusively by the sampler worker's complete
// ownership stage, including detached-root enrichment and its unknown evidence.
type ProvenOrphanProof struct {
	Table                         []procutil.Process
	Selection                     ResourceSelection
	TableKnown, InventoryComplete bool
	UnknownDetached               []ResourceProcessKey
}

func orphanTarget(p ResourceProvenance) ProvenOrphanTarget {
	return ProvenOrphanTarget{Key: p.Key, Root: p.Root, Slot: p.Slot, Thread: p.Thread, Workspace: p.Workspace}
}

// ProvenOrphanTargets returns only unambiguous positive monitor witnesses. It
// does not discover children or reconstruct ownership from process names.
func ProvenOrphanTargets(view ResourceView) []ProvenOrphanTarget {
	counts := map[int]int{}
	for _, p := range view.Orphans {
		counts[p.Key.PID]++
	}
	var out []ProvenOrphanTarget
	for _, p := range view.Orphans {
		t := orphanTarget(p.ResourceProvenance)
		if counts[p.Key.PID] == 1 && p.Orphan && p.OrphanKnown && t.Validate() == nil {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key.PID < out[j].Key.PID })
	return out
}

// DecideProvenOrphanSignal returns sent only as a permission for one subsequent
// guarded effect, never as evidence that the signal happened. Unknown evidence
// cannot establish absence or revoke a live detached process's protection.
func DecideProvenOrphanSignal(target ProvenOrphanTarget, proof ProvenOrphanProof) (string, string) {
	if err := target.Validate(); err != nil {
		return "refused", err.Error()
	}
	if !proof.TableKnown || !proof.InventoryComplete || len(proof.Table) > ResourceMaxTableRows || len(proof.Selection.Errors) > 0 {
		return "refused", "complete current ownership proof unavailable"
	}
	rows := make(map[int]procutil.Process, len(proof.Table))
	duplicates := map[int]bool{}
	for _, p := range proof.Table {
		if _, ok := rows[p.PID]; ok {
			duplicates[p.PID] = true
		}
		rows[p.PID] = p
	}
	p, present := rows[target.Key.PID]
	if duplicates[target.Key.PID] || present && p.Identity == "" {
		return "refused", "current process identity is ambiguous or unavailable"
	}
	if !present || p.Zombie || p.Identity != target.Key.Identity {
		return "skipped", "original process is gone or changed"
	}
	unknown := map[ResourceProcessKey]bool{}
	for _, k := range proof.UnknownDetached {
		unknown[k] = true
	}
	seen := map[int]bool{}
	for cur := p; ; {
		if seen[cur.PID] || duplicates[cur.PID] || cur.Identity == "" || unknown[resourceKey(cur)] {
			return "refused", "detached or ancestor ownership is unknown"
		}
		seen[cur.PID] = true
		parent, ok := rows[cur.PPID]
		if !ok || cur.PPID <= 0 {
			break
		}
		cur = parent
	}
	var found *ResourceProvenance
	for i := range proof.Selection.Processes {
		q := &proof.Selection.Processes[i]
		if q.Key == target.Key {
			if found != nil {
				return "refused", "ambiguous current provenance"
			}
			found = q
		}
	}
	if found == nil || !found.Orphan || !found.OrphanKnown || !reflect.DeepEqual(orphanTarget(*found), target) {
		return "refused", "current ownership no longer proves the exact orphan"
	}
	return "sent", ""
}

// ResourceOwnershipComplete decides whether this collection could have lost an
// ownership witness before legacy tree admission. The sampler keeps any prior
// loss sticky; this value only describes the supplied collection. Zombie death
// is positive evidence even when its former generation can no longer be read.
func ResourceOwnershipComplete(sample ResourceSample, selection ResourceSelection, unknown []ResourceProcessKey) bool {
	if !sample.TableKnown || !sample.InventoryComplete || sample.InventoryOmitted != 0 ||
		len(sample.Table) > ResourceMaxTableRows || len(sample.Slots) > ResourceMaxSlots ||
		len(selection.Processes) > ResourceMaxProcesses || len(selection.Errors) != 0 || len(unknown) != 0 {
		return false
	}
	seen := make(map[int]bool, len(sample.Table))
	for _, row := range sample.Table {
		if row.PID <= 0 || seen[row.PID] || !row.Zombie && row.Identity == "" {
			return false
		}
		seen[row.PID] = true
	}
	slots := make(map[string]bool, len(sample.Slots))
	for _, slot := range sample.Slots {
		if !slot.OwnershipKnown || slot.Slot == "" || slots[slot.Slot] {
			return false
		}
		slots[slot.Slot] = true
		for _, root := range slot.Roots {
			if root.Key.PID <= 0 || root.Key.Identity == "" {
				return false
			}
		}
	}
	return true
}
