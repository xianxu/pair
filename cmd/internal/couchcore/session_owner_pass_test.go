package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type ownerPassSource struct {
	calls  int
	states map[string]launcher.ServerState
	err    error
}

func (s *ownerPassSource) ServerStates(context.Context) (map[string]launcher.ServerState, error) {
	s.calls++
	return s.states, s.err
}

type ownerPassIO struct {
	*sessionOwnerWorld
	reads int
}

func (w *ownerPassIO) SessionServers(ctx context.Context, name string) ([]launcher.SessionServerIdentity, error) {
	w.reads++
	return w.sessionOwnerWorld.SessionServers(ctx, name)
}
func ownerPassChecker(t *testing.T) (ScopedThreadArtifactCollisionChecker, *ownerPassSource, *ownerPassIO, ThreadAddress) {
	t.Helper()
	data := t.TempDir()
	a := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "pass-test"}
	world := &ownerPassIO{sessionOwnerWorld: &sessionOwnerWorld{root: data, owners: map[string]ThreadAddress{"session": a}, generation: "server-1"}}
	source := &ownerPassSource{states: map[string]launcher.ServerState{"session": {Verdict: launcher.ServerReachable, Server: launcher.SessionServerIdentity{PID: 42, Identity: "server-1", Session: "session"}}}}
	c := NewScopedThreadArtifactCollisionChecker(data)
	c.OwnerProbe = launcher.SessionOwnerProbe{IO: world}
	c.Servers = source
	return c, source, world, a
}
func TestCheckerOwnerPassSharesOnlyInitialEvidence(t *testing.T) {
	c, source, world, a := ownerPassChecker(t)
	ctx, cancel := c.WithSessionOwnerPass(t.Context())
	defer cancel()
	var binding PairSessionBinding
	for range 2 {
		var err error
		binding, err = c.NamedPairSessionContext(ctx, a, "session")
		if err != nil || !binding.Present {
			t.Fatalf("binding=%+v err=%v", binding, err)
		}
	}
	if source.calls != 1 || world.reads != 2 {
		t.Fatalf("captures=%d live reads=%d", source.calls, world.reads)
	}
	if err := c.RevalidatePairSession(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if world.reads != 3 {
		t.Fatalf("revalidate reused cache: %d", world.reads)
	}
	if _, err := c.NamedPairSessionContext(withoutSessionOwnerPass(ctx), a, "session"); err != nil {
		t.Fatal(err)
	}
	if world.reads != 5 || source.calls != 1 {
		t.Fatalf("registration bypass reads=%d captures=%d", world.reads, source.calls)
	}
	other := NewScopedThreadArtifactCollisionChecker(c.GlobalDataDir)
	other.OwnerProbe = c.OwnerProbe
	other.Servers = source
	if _, err := other.NamedPairSessionContext(ctx, a, "session"); err != nil {
		t.Fatal(err)
	}
	if world.reads != 7 {
		t.Fatalf("different checker consumed pass: reads=%d", world.reads)
	}
	copyWithOtherRoot := c
	copyWithOtherRoot.GlobalDataDir = t.TempDir()
	if b, _ := copyWithOtherRoot.NamedPairSessionContext(ctx, a, "session"); b.Present {
		t.Fatal("different root gained ownership")
	}
	if world.reads != 9 {
		t.Fatalf("different root consumed pass: reads=%d", world.reads)
	}
}
func TestCheckerOwnerPassMemoizesCaptureFailure(t *testing.T) {
	c, source, world, a := ownerPassChecker(t)
	source.err = errors.New("capture unavailable")
	ctx, cancel := c.WithSessionOwnerPass(t.Context())
	defer cancel()
	for range 2 {
		b, err := c.NamedPairSessionContext(ctx, a, "session")
		if !errors.Is(err, source.err) || b.Present {
			t.Fatalf("b=%+v err=%v", b, err)
		}
	}
	if source.calls != 1 || world.reads != 0 {
		t.Fatalf("capture fallback: captures=%d live=%d", source.calls, world.reads)
	}
}

// Model a birth that the pass's prelaunch absence cannot see. Registration
// must inspect live evidence instead of polling an immutable absence forever.
type registrationPassArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	staleReads, liveReads int
}

func (f *registrationPassArtifacts) PairSessionContext(ctx context.Context, a ThreadAddress) (PairSessionBinding, error) {
	if b, _ := ctx.Value(sessionOwnerPassKey{}).(sessionOwnerPassBinding); b.scope != nil {
		f.staleReads++
		return PairSessionBinding{}, nil
	}
	f.liveReads++
	return f.FakeThreadArtifactCollisionChecker.PairSessionContext(ctx, a)
}
func TestResumeRegistrationBypassesPrelaunchOwnerPassAbsence(t *testing.T) {
	env, a := warmDetachedThread(t)
	artifacts := &registrationPassArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts}
	env.Couch.Artifacts = artifacts
	ctx := context.WithValue(t.Context(), sessionOwnerPassKey{}, sessionOwnerPassBinding{scope: &sessionOwnerScope{}})
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if err := env.Couch.awaitResumeRegistration(ctx, a, nil); err != nil {
		t.Fatalf("birth never observed: %v", err)
	}
	if artifacts.staleReads != 0 || artifacts.liveReads != 1 {
		t.Fatalf("stale=%d live=%d", artifacts.staleReads, artifacts.liveReads)
	}
}
