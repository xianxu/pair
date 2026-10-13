package couchcore

import (
	"fmt"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/orientation"
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
		if !reflect.DeepEqual(previous.Binding, r.SessionBinding) {
			return fmt.Errorf("previous binding does not match verified source session")
		}
		if err := previous.Validate(toPersistedThreadAddress(address)); err != nil {
			return err
		}
		r.PreviousSession = threadrecord.ClonePreviousSession(&previous)
		return nil
	})
}

func previousSessionEvidence(source, parked ThreadRecord, evidence orientation.OrientationContext) *PreviousSession {
	if parked.VerifiedPark == nil {
		return nil
	}
	agent := RecordAgent(source)
	if evidence.SourceAgent != "" {
		agent = evidence.SourceAgent
	}
	previous := &PreviousSession{
		Source:  toPersistedParkIdentity(parked.VerifiedPark.Identity),
		Binding: cloneSessionBinding(source.SessionBinding), Agent: agent,
		NativeSession: evidence.SourceSession, Transcripts: evidence.NativeTranscripts,
		Capture: parked.VerifiedPark.Scrollback, CapturedAt: parked.VerifiedPark.ParkedAt,
		Unavailable: append([]string(nil), evidence.Unavailable...),
	}
	if previous.Capture == nil {
		previous.Unavailable = append(previous.Unavailable, "source park has no preserved tty capture")
	}
	if len(previous.Transcripts) == 0 {
		previous.Unavailable = append(previous.Unavailable, "previous native transcript is unavailable")
	}
	return threadrecord.ClonePreviousSession(previous)
}
