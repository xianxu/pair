package couchcore

import (
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

// ParkCause is why a park was asked for: the zero value is the operator's
// plain park of one thread; a non-empty cause marks the park asleep.
type ParkCause = threadrecord.SleepCause

// ParkCausePlain is an individual park; it reads parked, never asleep.
const ParkCausePlain ParkCause = ""
