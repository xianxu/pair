package couchcore

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// pair#597: the only cross-slot refusal is another positively live slot whose
// recorded conversation is the same one; stale process or readiness evidence
// (the old native-owner scan's input) can never refuse.
func TestRecoveryConversationInUseIsRegistryOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(*Couch, *FakeProcOps, *FakeThreadArtifactCollisionChecker, ThreadRecord, ThreadRecord)
		want    string
	}{
		{name: "same conversation in another live slot", want: "conversation native is in use by pair:0"},
		{name: "requested conversation in another live slot", want: "in use by", arrange: func(_ *Couch, _ *FakeProcOps, a *FakeThreadArtifactCollisionChecker, _, other ThreadRecord) {
			a.SetNativeBinding(other.Address, "codex", sessioninventory.BindingProvisional, "")
			a.nativeBindings[nativeBindingKey{Address: other.Address, Agent: "codex"}] = NativeBindingResolution{Status: sessioninventory.BindingProvisional, RequestedNativeID: "native"}
		}},
		{name: "other slot dead", arrange: func(_ *Couch, p *FakeProcOps, _ *FakeThreadArtifactCollisionChecker, _, _ ThreadRecord) { p.Kill(90) }},
		{name: "other slot liveness unknown", arrange: func(_ *Couch, p *FakeProcOps, _ *FakeThreadArtifactCollisionChecker, _, _ ThreadRecord) {
			p.SetUnknown(90)
		}},
		{name: "other slot pid reused", arrange: func(_ *Couch, p *FakeProcOps, _ *FakeThreadArtifactCollisionChecker, _, _ ThreadRecord) {
			p.Set(90, "unrelated")
		}},
		{name: "other slot different conversation", arrange: func(_ *Couch, _ *FakeProcOps, a *FakeThreadArtifactCollisionChecker, _, other ThreadRecord) {
			a.SetNativeBinding(other.Address, "codex", sessioninventory.BindingEstablished, "different")
		}},
		{name: "other slot conversation unknown", arrange: func(_ *Couch, _ *FakeProcOps, a *FakeThreadArtifactCollisionChecker, _, other ThreadRecord) {
			delete(a.nativeBindings, nativeBindingKey{Address: other.Address, Agent: "codex"})
		}},
		{name: "own slot records it", arrange: func(_ *Couch, _ *FakeProcOps, a *FakeThreadArtifactCollisionChecker, own, other ThreadRecord) {
			a.SetNativeBinding(other.Address, "codex", sessioninventory.BindingEstablished, "different")
			a.SetNativeBinding(own.Address, "codex", sessioninventory.BindingEstablished, "native")
		}},
		{name: "stale native owner evidence ignored", arrange: func(_ *Couch, _ *FakeProcOps, a *FakeThreadArtifactCollisionChecker, _, other ThreadRecord) {
			a.SetNativeBinding(other.Address, "codex", sessioninventory.BindingEstablished, "different")
			a.SetRecoveryNativeWrappers(FakeRecoveryNativeWrapper{Address: other.Address, Agent: "codex", PID: 1111})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			profile := LaunchProfile{Agent: "codex", Argv: []string{}}
			create := func(tag string, pid int) ThreadRecord {
				r := validThreadRecord(t)
				r.Address.Tag = ThreadTag(tag)
				r.StartingPath, r.WorkingPath = ns.Dir(), ns.Dir()
				r.LatestLaunchProfile = &profile
				r.Incarnations = []ThreadIncarnation{{PID: pid, Identity: "wrapper", State: IncarnationLive, LaunchProfile: &profile}}
				created, err := store.CreateThread(r)
				if err != nil {
					t.Fatal(err)
				}
				return created
			}
			own, other := create("couch-1111111111111111", 42), create("couch-2222222222222222", 90)
			proc := NewFakeProcOps()
			proc.Set(42, "wrapper")
			proc.Set(90, "wrapper")
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetNativeBinding(other.Address, "codex", sessioninventory.BindingEstablished, "native")
			couch := &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Git: NewFakeGit(map[GitCall]string{{Dir: ns.Dir(), Args: "rev-parse --show-toplevel"}: "/work/pair"})}
			if tc.arrange != nil {
				tc.arrange(couch, proc, artifacts, own, other)
			}
			err := couch.checkRecoveryConversationInUse(context.Background(), own, RecoveryChoice{Agent: "codex", NativeID: "native"})
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("err=%v, want %q", err, tc.want)
			}
		})
	}
}

// pair#597 close review BR-1: the ordinary Couch cold resume (resume,
// relaunch) takes no native admission either, so it must pass the same
// registry check before any child starts.
func TestColdResumeRefusesConversationLiveInAnotherSlot(t *testing.T) {
	for _, collide := range []bool{true, false} {
		env := newTestEnv(t, "/repo")
		profile := LaunchProfile{Agent: "claude", Argv: []string{}}
		parked := createParkedThreadInCouch(t, env, profile)
		env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
		other := validThreadRecord(t)
		other.Address.Tag = "couch-2222222222222222"
		other.StartingPath, other.WorkingPath = "/repo", "/repo"
		other.LatestLaunchProfile = &profile
		other.Incarnations = []ThreadIncarnation{{PID: 77, Identity: "other-helper", State: IncarnationLive, LaunchProfile: &profile}}
		if _, err := env.Couch.Threads.CreateThread(other); err != nil {
			t.Fatal(err)
		}
		env.Proc.Set(77, "other-helper")
		otherID := "native-root-2"
		if collide {
			otherID = "native-root-1"
		}
		env.Artifacts.SetNativeBinding(other.Address, "claude", sessioninventory.BindingEstablished, otherID)
		env.Runner.AfterAcknowledge = func(id string) error {
			env.Artifacts.SetPairSession(parked.Address, continuationChildSession(t, env.Runner, id), true)
			return nil
		}
		_, _, err := env.Couch.Resume(parked.Address)
		if collide && (err == nil || !strings.Contains(err.Error(), "conversation native-root-1 is in use by") || len(env.Runner.order) != 0) {
			t.Fatalf("collision admitted: err=%v children=%v", err, env.Runner.order)
		}
		if !collide && err != nil {
			t.Fatalf("different conversation refused: %v", err)
		}
	}
}
