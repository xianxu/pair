package couchcore

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// All fixtures here are values: projection needs neither a store nor artifacts.
func TestPreviousSessionEvidencePure(t *testing.T) {
	t.Parallel()
	address := ThreadAddress{RepoScope: "scope", Tag: "thread"}
	at := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	source := ThreadRecord{Address: address, LatestLaunchProfile: &LaunchProfile{Agent: "codex"}, Incarnations: []ThreadIncarnation{{LaunchProfile: &LaunchProfile{Agent: "claude"}}}}
	park := VerifiedPark{Identity: ParkIdentity{Address: address, Nonce: "exact-park", PID: 42, ProcessIdentity: "source-process"}, ParkedAt: at, Scrollback: &pairlifecycle.PreservedScrollback{Agent: "claude", Token: "exact-capture", Events: true}}
	parked := ThreadRecord{Address: address, LatestLaunchProfile: &LaunchProfile{Agent: "agy"}, VerifiedPark: &park}
	if got := previousSessionEvidence(source, ThreadRecord{Address: address}); got != nil {
		t.Fatalf("missing park synthesized evidence: %+v", got)
	}
	want := PreviousSession{Source: threadrecord.ParkIdentity{Address: threadrecord.Address{RepoScope: "scope", Tag: "thread"}, Nonce: "exact-park", PID: 42, ProcessIdentity: "source-process"}, Agent: "claude", CapturedAt: at, Capture: &pairlifecycle.PreservedScrollback{Agent: "claude", Token: "exact-capture", Events: true}}
	got := previousSessionEvidence(source, parked)
	if !reflect.DeepEqual(got, &want) {
		t.Fatalf("wrong source projection: got %+v want %+v", got, want)
	}
	got.Capture.Token = "output-edit"
	if park.Scrollback.Token != "exact-capture" {
		t.Fatal("projected capture aliases input")
	}
	got = previousSessionEvidence(source, parked)
	park.Scrollback.Token = "input-edit"
	if got.Capture.Token != "exact-capture" {
		t.Fatal("input capture aliases projection")
	}
	park.Scrollback = nil
	got = previousSessionEvidence(source, parked)
	want.Capture = nil
	if !reflect.DeepEqual(got, &want) {
		t.Fatalf("missing capture lost park coordinates: %+v", got)
	}
	source.Incarnations = nil
	got = previousSessionEvidence(source, parked)
	if got.Agent != "codex" {
		t.Fatalf("missing incarnation ignored latest source: %+v", got)
	}
}
