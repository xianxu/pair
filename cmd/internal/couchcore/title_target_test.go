package couchcore

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
)

func TestProjectTitleTargetExactObservation(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
	binding := &couchidentity.SessionBinding{Name: "pair-title-fixture", ScopeKey: address.RepoScope, Tag: string(address.Tag), StartNonce: "title-start", Legacy: true}
	record := ThreadRecord{Address: address, SessionBinding: binding, LatestLaunchProfile: &LaunchProfile{Agent: "codex"}}
	row := ActionableThreadSummary{Address: address, Agent: "codex", State: ThreadLive}
	observation := SessionObservation{State: SessionPresent, Name: binding.Name}
	target, ok := ProjectTitleTarget(record, row, observation, 7)
	if !ok || target.Session != binding.Name || target.StartNonce != binding.StartNonce || target.LaunchOrdinal != 7 {
		t.Fatalf("target=%+v ok=%v", target, ok)
	}
	for _, change := range []func(*ThreadRecord, *ActionableThreadSummary, *SessionObservation, *uint64){
		func(_ *ThreadRecord, _ *ActionableThreadSummary, s *SessionObservation, _ *uint64) {
			s.Name = "pair-foreign"
		},
		func(_ *ThreadRecord, _ *ActionableThreadSummary, s *SessionObservation, _ *uint64) {
			s.State = SessionUnresolved
		},
		func(_ *ThreadRecord, r *ActionableThreadSummary, _ *SessionObservation, _ *uint64) {
			r.Agent = "claude"
		},
		func(_ *ThreadRecord, r *ActionableThreadSummary, _ *SessionObservation, _ *uint64) {
			r.State = ThreadParked
		},
		func(_ *ThreadRecord, _ *ActionableThreadSummary, _ *SessionObservation, n *uint64) { *n = 0 },
		func(r *ThreadRecord, _ *ActionableThreadSummary, _ *SessionObservation, _ *uint64) {
			r.SessionBinding = nil
		},
	} {
		r, s, o, n := record, row, observation, uint64(7)
		change(&r, &s, &o, &n)
		if got, ok := ProjectTitleTarget(r, s, o, n); ok {
			t.Fatalf("unproved target=%+v", got)
		}
	}
	row.TitleTarget = &target
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), binding.StartNonce) || strings.Contains(string(raw), binding.Name) {
		t.Fatalf("private target in row JSON: %s", raw)
	}
}

// titleOrdinalArtifacts retains a current ordinal beside the existing stateful
// session world, so accepted inventory and pre-effect checks share authority.
type titleOrdinalArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	ordinal uint64
}

func (f *titleOrdinalArtifacts) CurrentTitleOrdinal(_ context.Context, _ ThreadAddress, _ string) (uint64, error) {
	return f.ordinal, nil
}

func TestTitleTargetCurrentRejectsReplacedLaunch(t *testing.T) {
	env, record := registeredActorFixture(t)
	binding := &couchidentity.SessionBinding{Name: "pair-title-current", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), StartNonce: "title-current", Legacy: true}
	record.SessionBinding = binding
	record, err := env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error { r.SessionBinding = binding; return nil })
	if err != nil {
		t.Fatal(err)
	}
	artifacts := &titleOrdinalArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, ordinal: 3}
	env.Couch.Artifacts = artifacts
	target := TitleTarget{Address: record.Address, Agent: "claude", Session: binding.Name, StartNonce: binding.StartNonce, LaunchOrdinal: 3}
	if !env.Couch.TitleTargetCurrent(context.Background(), target) {
		t.Fatal("current target rejected")
	}
	artifacts.ordinal = 4
	if env.Couch.TitleTargetCurrent(context.Background(), target) {
		t.Fatal("replaced launch admitted")
	}
}
