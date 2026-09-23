package couchcore

import "fmt"

type ThreadTargetKind string

const (
	ThreadTargetOrdinary ThreadTargetKind = "ordinary"
	ThreadTargetSlot     ThreadTargetKind = "slot"
)

// ThreadTarget identifies either an ordinary conversation or a durable slot.
// A slot remains addressable when it has no readable native conversation.
type ThreadTarget struct {
	Kind    ThreadTargetKind
	Address ThreadAddress
	Slot    SlotIdentity
}

// ThreadRowKey keeps UI selection stable when a slot changes conversations.
type ThreadRowKey struct {
	Kind     ThreadTargetKind
	Address  ThreadAddress
	SlotPath string
}

func (t ThreadTarget) Validate() error {
	switch t.Kind {
	case ThreadTargetOrdinary:
		if t.Slot != (SlotIdentity{}) {
			return fmt.Errorf("ordinary target also contains a slot")
		}
		return validateThreadAddress(t.Address)
	case ThreadTargetSlot:
		if t.Address != (ThreadAddress{}) {
			return fmt.Errorf("slot target also contains a native address")
		}
		return t.Slot.Validate()
	default:
		return fmt.Errorf("unknown thread target kind %q", t.Kind)
	}
}

func (t ThreadTarget) RowKey() (ThreadRowKey, error) {
	if err := t.Validate(); err != nil {
		return ThreadRowKey{}, err
	}
	if t.Kind == ThreadTargetSlot {
		return ThreadRowKey{Kind: t.Kind, SlotPath: t.Slot.WorktreeRoot}, nil
	}
	return ThreadRowKey{Kind: t.Kind, Address: t.Address}, nil
}
