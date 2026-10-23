package couchcore

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// ThreadAsleep reports whether a full shutdown put this thread to sleep
// (pair#830), and since when. It is the only reader of the mark: asleep means
// the verified park in force is the very park the mark was written for. A mark
// whose park rolled back, was replaced, or was ended by a resume -- including
// one an older build carried through a rewrite -- names another nonce and is
// inert.
func ThreadAsleep(record ThreadRecord) (time.Time, bool) {
	if record.Asleep == nil || record.VerifiedPark == nil || record.Park != nil {
		return time.Time{}, false
	}
	if record.Asleep.Nonce != record.VerifiedPark.Identity.Nonce {
		return time.Time{}, false
	}
	return record.Asleep.At, true
}

func asleepAt(record ThreadRecord) *time.Time {
	at, ok := ThreadAsleep(record)
	if !ok {
		return nil
	}
	return &at
}

// ParkCause is why a park was asked for: the zero value is the operator's
// plain park of one thread; a non-empty cause marks the park asleep.
type ParkCause = threadrecord.SleepCause

// ParkCausePlain is an individual park; it reads parked, never asleep.
const ParkCausePlain ParkCause = ""

// AsleepSlots is the store's asleep slot rows, read from the records alone
// (pair#830): asleep is a fact of a record, so no liveness evidence is
// gathered and the read stays cheap enough for a broker admission.
func (c *Couch) AsleepSlots(ctx context.Context) ([]ActionableThreadSummary, error) {
	if c.Threads == nil {
		return nil, errors.New("no thread store")
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		return nil, err
	}
	return asleepSlotRows(FromSnapshot(snapshot, nil)), nil
}

// asleepSlotRows keeps the projected rows that are slots and asleep: wake
// addresses a slot (repo:N), so an asleep thread outside one is not a target.
func asleepSlotRows(input ThreadProjectionInput) []ActionableThreadSummary {
	var asleep []ActionableThreadSummary
	for _, row := range ProjectActionableThreads(input) {
		if row.AsleepAt != nil && row.Target.Kind == ThreadTargetSlot {
			asleep = append(asleep, row)
		}
	}
	return asleep
}
