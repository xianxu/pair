package couchcore

import (
	"fmt"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// RecordPreviousSession commits evidence while the exact source park is still
// authoritative. Registration may clear VerifiedPark but keeps this descriptor.
func (s *ThreadStore) RecordPreviousSession(address ThreadAddress, revision uint64, previous PreviousSession) (ThreadRecord, error) {
	return s.updateExistingThread(address, revision, func(r *ThreadRecord) error {
		if r.VerifiedPark == nil || toPersistedParkIdentity(r.VerifiedPark.Identity) != previous.Source {
			return fmt.Errorf("previous session does not name the verified source park")
		}
		if !reflect.DeepEqual(previous.Capture, r.VerifiedPark.Scrollback) {
			return fmt.Errorf("previous capture does not match verified source park")
		}
		if err := previous.Validate(toPersistedThreadAddress(address)); err != nil {
			return err
		}
		r.PreviousSession = threadrecord.ClonePreviousSession(&previous)
		return nil
	})
}

func previousSessionEvidence(source, parked ThreadRecord) *PreviousSession {
	if parked.VerifiedPark == nil {
		return nil
	}
	return threadrecord.ClonePreviousSession(&PreviousSession{
		Source: toPersistedParkIdentity(parked.VerifiedPark.Identity),
		Agent:  RecordAgent(source), Capture: parked.VerifiedPark.Scrollback,
		CapturedAt: parked.VerifiedPark.ParkedAt,
	})
}
