package couchcore

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
)

const MaxHubSlots = 128

// HubSlot is canonical workspace identity, independent of agent conversations.
type HubSlot struct {
	Repository string
	Number     int
}

func (s HubSlot) Validate() error {
	if !workspaceAbsolute(s.Repository) || s.Number < 0 {
		return fmt.Errorf("invalid hub slot identity")
	}
	return nil
}

type HubDefinition struct {
	Root, Workspace string
	Slot            HubSlot
	Claims          HubClaimDocument
}
type HubResolution struct {
	Hub       string
	Conflicts []string
}

// HubOwnership is an immutable value. Apply is its sole runtime transition.
type HubOwnership struct {
	hubs      []HubDefinition
	overrides map[HubSlot]string
	revision  uint64
}

func NewHubOwnership(hubs []HubDefinition) (HubOwnership, error) {
	if len(hubs) == 0 || len(hubs) > MaxHubSlots {
		return HubOwnership{}, fmt.Errorf("expected 1..%d hubs", MaxHubSlots)
	}
	roots := map[string]bool{}
	slots := map[HubSlot]bool{}
	for _, h := range hubs {
		if !workspaceAbsolute(h.Root) || !workspaceAbsolute(h.Workspace) || h.Slot.Validate() != nil || roots[h.Root] || slots[h.Slot] {
			return HubOwnership{}, fmt.Errorf("invalid or duplicate hub %q", h.Root)
		}
		if rel, e := filepath.Rel(h.Workspace, h.Slot.Repository); e != nil || rel == ".." || len(rel) > 3 && rel[:3] == "../" {
			return HubOwnership{}, fmt.Errorf("hub repository is outside workspace")
		}
		if _, e := ParseHubClaims(h.Claims.Bytes()); e != nil {
			return HubOwnership{}, e
		}
		roots[h.Root] = true
		slots[h.Slot] = true
	}
	return HubOwnership{hubs: slices.Clone(hubs), overrides: map[HubSlot]string{}, revision: 1}, nil
}
func (s HubOwnership) Hubs() []HubDefinition { return slices.Clone(s.hubs) }
func (s HubOwnership) Revision() uint64      { return s.revision }
func (s HubOwnership) Main() string {
	if len(s.hubs) == 0 {
		return ""
	}
	return s.hubs[0].Root
}
func (s HubOwnership) IsHub(root string) bool {
	return slices.ContainsFunc(s.hubs, func(h HubDefinition) bool { return h.Root == root })
}

// Wakes is couch --wake's authority (pair#830): only the main hub wakes, and
// only another hub's own slot. A worker is woken by its hub, never from here.
func (s HubOwnership) Wakes(caller, target string) error {
	switch {
	case caller == "" || caller != s.Main():
		return fmt.Errorf("only the main hub (%s) may wake the asleep hubs", s.Main())
	case target == caller:
		return fmt.Errorf("the main hub is already awake")
	case !s.IsHub(target):
		return fmt.Errorf("%s is not a hub; its hub wakes it", target)
	}
	return nil
}
func (s HubOwnership) hubForSlot(slot HubSlot) string {
	for _, h := range s.hubs {
		if h.Slot == slot {
			return h.Root
		}
	}
	return ""
}

// Provisions is the bench verbs' authority (pair#847): only a hub other than
// the main hub, and only on a slot it holds by its own explicit .hub claim.
// Default ownership is the main hub's, so production slots never qualify.
func (s HubOwnership) Provisions(caller string, target HubSlot) error {
	resolution, source := s.resolve(target)
	switch {
	case !s.IsHub(caller):
		return fmt.Errorf("%s is not a hub", caller)
	case caller == s.Main():
		return fmt.Errorf("the main hub (%s) owns production slots; bench slots belong to another hub", caller)
	case source == ownedAsHubSlot && resolution.Hub == caller:
		return fmt.Errorf("a hub cannot drive its own slot")
	case len(resolution.Conflicts) > 0:
		return fmt.Errorf("slot claim conflicts between hubs %s and %v", resolution.Hub, resolution.Conflicts)
	case source == ownedByOverride && resolution.Hub == caller:
		return fmt.Errorf("slot was moved to %s at runtime, not claimed in its .hub", caller)
	case source != ownedByClaim || resolution.Hub != caller:
		return fmt.Errorf("slot is not claimed by hub %s (owner %s)", caller, resolution.Hub)
	}
	return nil
}

// ownershipSource says which rule Resolve applied.
type ownershipSource uint8

const (
	ownedAsHubSlot ownershipSource = iota + 1
	ownedByOverride
	ownedByClaim
	ownedByDefault
)

func (s HubOwnership) Resolve(slot HubSlot) HubResolution {
	resolution, _ := s.resolve(slot)
	return resolution
}

func (s HubOwnership) resolve(slot HubSlot) (HubResolution, ownershipSource) {
	if slot.Validate() != nil {
		return HubResolution{}, 0
	}
	if hub := s.hubForSlot(slot); hub != "" {
		return HubResolution{Hub: hub}, ownedAsHubSlot
	}
	if hub := s.overrides[slot]; hub != "" {
		return HubResolution{Hub: hub}, ownedByOverride
	}
	out := HubResolution{Hub: s.Main()}
	best := 0
	for _, h := range s.hubs {
		rank := 0
		for _, c := range h.Claims.Claims() {
			candidate := claimSlot(h, c)
			if candidate.Repository != slot.Repository {
				continue
			}
			if c.Slot == -1 && rank < 1 {
				rank = 1
			}
			if c.Slot == slot.Number {
				rank = 2
			}
		}
		if rank > best {
			out = HubResolution{Hub: h.Root}
			best = rank
		} else if rank > 0 && rank == best {
			out.Conflicts = append(out.Conflicts, h.Root)
		}
	}
	if best == 0 {
		return out, ownedByDefault
	}
	return out, ownedByClaim
}

type HubMove struct {
	Slots          []HubSlot
	Caller, Target string
	Revision       uint64
}

// Apply admits an ephemeral ownership move. Persistence is separately planned
// from Hubs even when the requested runtime owner already matches.
func (s HubOwnership) Apply(move HubMove) (HubOwnership, error) {
	if move.Revision != s.revision {
		return s, fmt.Errorf("hub ownership changed; retry the move")
	}
	if !s.IsHub(move.Caller) || !s.IsHub(move.Target) {
		return s, fmt.Errorf("move-slot requires a live hub caller and active target hub")
	}
	if len(move.Slots) == 0 || len(move.Slots) > MaxHubSlots {
		return s, fmt.Errorf("move requires 1..%d slots", MaxHubSlots)
	}
	for _, slot := range move.Slots {
		if err := slot.Validate(); err != nil {
			return s, err
		}
		if s.hubForSlot(slot) != "" {
			return s, fmt.Errorf("hub slots cannot be moved")
		}
		owner := s.Resolve(slot).Hub
		if owner != move.Caller && owner != s.Main() {
			return s, fmt.Errorf("slot belongs to hub %s; a hub may move only its own or main's slots", owner)
		}
	}
	next := s
	next.overrides = make(map[HubSlot]string, len(s.overrides)+len(move.Slots))
	for slot, hub := range s.overrides {
		next.overrides[slot] = hub
	}
	changed := false
	for _, slot := range move.Slots {
		if s.Resolve(slot).Hub != move.Target {
			next.overrides[slot] = move.Target
			changed = true
		}
	}
	if len(next.overrides) > MaxHubSlots {
		return s, fmt.Errorf("runtime hub move capacity reached; restart Couch to clear ephemeral moves")
	}
	if changed {
		next.revision++
	}
	return next, nil
}

// WithClaims publishes a confirmed durable configuration without discarding
// unrelated ephemeral moves. The adapter must supply the observed revision.
func (s HubOwnership) WithClaims(hubs []HubDefinition, revision uint64) (HubOwnership, error) {
	if revision != s.revision {
		return s, fmt.Errorf("hub ownership changed; retry")
	}
	next, err := NewHubOwnership(hubs)
	if err != nil {
		return s, err
	}
	if len(next.hubs) != len(s.hubs) {
		return s, fmt.Errorf("active hubs changed")
	}
	changed := false
	for i, h := range next.hubs {
		old := s.hubs[i]
		changed = changed || !bytes.Equal(h.Claims.Bytes(), old.Claims.Bytes())
		if h.Root != old.Root || h.Workspace != old.Workspace || h.Slot != old.Slot {
			return s, fmt.Errorf("active hubs changed")
		}
	}
	if !changed {
		return s, nil
	}
	next.overrides = s.overrides
	next.revision = s.revision + 1
	return next, nil
}
