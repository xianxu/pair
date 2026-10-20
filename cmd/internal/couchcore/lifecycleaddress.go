package couchcore

import (
	"fmt"
)

// LifecycleAddress is what the lifecycle primitives act on (pair#695 §3): a
// slot's checkout. The lease, failure records and orphan origins are keyed by
// Key(). It has one kind; a team is an ordinary :0 slot (the operator's one
// team per repository), so there is no second arm.
type LifecycleAddress struct {
	// Root is the slot's physical checkout and alone forms the key, so a :0
	// primary (no numbered SlotIdentity) is addressed by its root.
	Slot SlotIdentity
	Root string
}

// NewSlotAddress addresses a slot by its physical checkout root, which the
// caller has already canonicalized (symlinks resolved; this does no IO). A
// numbered slot's root must be its own checkout; a :0 primary passes a zero
// SlotIdentity. #683's failure slot key is this address's Key().
func NewSlotAddress(slot SlotIdentity, physicalRoot string) (LifecycleAddress, error) {
	a := LifecycleAddress{Slot: slot, Root: physicalRoot}
	return a, a.Validate()
}

// Validate is a pure shape check: a clean absolute root that, for a numbered
// slot, is the slot's own checkout.
func (a LifecycleAddress) Validate() error {
	if !workspaceAbsolute(a.Root) {
		return fmt.Errorf("slot root %q is not a clean absolute path", a.Root)
	}
	if a.Slot != (SlotIdentity{}) {
		if err := a.Slot.validateLocation(); err != nil {
			return err
		}
		if a.Slot.WorktreeRoot != a.Root {
			return fmt.Errorf("slot root %q is not the slot's checkout %q", a.Root, a.Slot.WorktreeRoot)
		}
	}
	return nil
}

// Key is the address's stable identity, "slot:<root>". File names derived
// from it are digests, never the raw key.
func (a LifecycleAddress) Key() string {
	if a.Root == "" {
		return ""
	}
	return "slot:" + a.Root
}
