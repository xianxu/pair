package mockserve

import (
	"errors"
	"testing"
)

func TestRevisionStateTransitions(t *testing.T) {
	var state RevisionState
	var previous string
	steps := []struct {
		version, digest string
		err             error
		changed         bool
	}{
		{"v1", "a", nil, true}, {"v1", "a", nil, false}, {"v1", "b", nil, true},
		{"v2", "b", nil, true}, {"v2", "", errors.New("broken"), true},
		{"v2", "", errors.New("broken"), false}, {"v2", "", errors.New("different"), true},
		{"v2", "b", nil, true}, {"v2", "b", nil, false},
	}
	for _, s := range steps {
		changed := state.Observe(s.version, s.digest, s.err)
		if changed != s.changed {
			t.Errorf("Observe(%+v) changed=%v", s, changed)
		}
		got := state.Snapshot("life-a")
		if got.Version != s.version {
			t.Errorf("version=%q", got.Version)
		}
		if s.err != nil && got.Error != s.err.Error() || s.err == nil && got.Error != "" {
			t.Errorf("error=%q", got.Error)
		}
		if previous != "" && (got.Revision != previous) != s.changed {
			t.Errorf("revision change=%v want %v", got.Revision != previous, s.changed)
		}
		previous = got.Revision
	}
	before := state.Snapshot("life-a")
	after := state.Snapshot("life-b")
	if before.Revision == after.Revision {
		t.Error("restart must change revision")
	}
	after.Version = "mutated"
	if state.Snapshot("life-a").Version != before.Version {
		t.Error("snapshot aliases state")
	}
}

func TestRevisionDistinguishesEmptyErrorFromHealthy(t *testing.T) {
	var state RevisionState
	state.Observe("v1", "", nil)
	before := state.Snapshot("life")
	if !state.Observe("v1", "", errors.New("")) {
		t.Error("empty error did not change healthy state")
	}
	if after := state.Snapshot("life"); after.Error == "" || after.Revision == before.Revision {
		t.Errorf("error snapshot = %+v", after)
	}
	if !state.Observe("v1", "", nil) {
		t.Error("recovery did not change error state")
	}
}
