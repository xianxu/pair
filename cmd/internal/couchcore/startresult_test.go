package couchcore

import "testing"

func TestStartResultAdoptionRequiresAHandle(t *testing.T) {
	for _, hasRecord := range []bool{false, true} {
		for _, hasHandle := range []bool{false, true} {
			input := StartResult{}
			if hasRecord {
				input.Record = ActorRecord{PID: 42, Identity: "retained-record"}
			}
			if hasHandle {
				input.Handle = &fakeHandle{}
			}
			got, adopt := input.Started()
			if adopt != hasHandle {
				t.Errorf("record=%t handle=%t: adoption=%t, want %t", hasRecord, hasHandle, adopt, hasHandle)
			}
			if got.Handle != input.Handle || got.Record.PID != input.Record.PID || got.Record.Identity != input.Record.Identity {
				t.Errorf("Started changed its child payload: %+v", got)
			}
		}
	}
}

func TestStartResultPreservesConfigurationSelection(t *testing.T) {
	retained, explicit := &AcceptedConfiguration{}, &AcceptedConfiguration{}
	for _, handle := range []Handle{nil, &fakeHandle{}} {
		for _, override := range []*AcceptedConfiguration{nil, explicit} {
			input := StartResult{Record: ActorRecord{configuration: retained}, Configuration: override, Handle: handle}
			got, adopt := input.Started()
			want := retained
			if override != nil {
				want = override
			}
			if got.Configuration != want || adopt != (handle != nil) {
				t.Errorf("configuration=%p want=%p adoption=%t handle=%v", got.Configuration, want, adopt, handle)
			}
		}
	}
}
