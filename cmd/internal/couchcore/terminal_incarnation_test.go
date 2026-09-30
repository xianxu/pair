package couchcore

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The index may stay on the old terminal while both old and proposed server
// identities exist independently. This models the launcher's exact-name CREATE.
type incarnationWorld struct {
	*FakeThreadArtifactCollisionChecker
	index   string
	live    map[string]bool
	deleted []string
}

func (w *incarnationWorld) PairSession(a ThreadAddress) (PairSessionBinding, error) {
	return w.PairSessionContext(context.Background(), a)
}
func (w *incarnationWorld) PairSessionContext(_ context.Context, _ ThreadAddress) (PairSessionBinding, error) {
	return PairSessionBinding{Name: w.index, Present: w.live[w.index]}, nil
}
func (w *incarnationWorld) NamedPairSessionContext(_ context.Context, _ ThreadAddress, name string) (PairSessionBinding, error) {
	return PairSessionBinding{Name: name, Present: w.live[name]}, nil
}
func (w *incarnationWorld) Quiesce(_ ThreadAddress) error {
	w.deleted = append(w.deleted, w.index)
	w.live[w.index] = false
	return nil
}
func (w *incarnationWorld) QuiesceNamed(_ context.Context, _ ThreadAddress, name string) error {
	w.deleted = append(w.deleted, name)
	w.live[name] = false
	return nil
}

func incarnationFixture(t *testing.T) (*testEnv, ThreadRecord, *incarnationWorld) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	env.Couch.Identities = couchidentity.IdentityStore{HostDir: t.TempDir(), StoreDir: env.Dir}
	env.Couch.resumeRegistrationTimeout = 40 * time.Millisecond
	record := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	world := &incarnationWorld{FakeThreadArtifactCollisionChecker: env.Artifacts, index: "📁legacy-old", live: map[string]bool{}}
	env.Couch.Artifacts = world
	return env, record, world
}

func TestColdAttachedConversationCannotCreateSecondTerminal(t *testing.T) {
	env, record, world := incarnationFixture(t)
	world.live[world.index] = true
	launches := 0
	env.Runner.AfterAcknowledge = func(id string) error {
		launches++
		for _, entry := range env.Runner.Child(id).Env {
			if raw, ok := strings.CutPrefix(entry, launcher.CouchSessionIntentEnv+"="); ok {
				var intent launcher.CouchSessionIntent
				if err := json.Unmarshal([]byte(raw), &intent); err != nil {
					return err
				}
				world.live[intent.Name] = true
			}
		}
		return nil
	}
	_, _, err := env.Couch.Resume(record.Address)
	if err == nil {
		t.Fatal("attached conversation was cold-created")
	}
	if launches != 0 || len(world.live) != 1 || !world.live[world.index] || len(world.deleted) != 0 {
		t.Fatalf("second terminal or original changed: launches=%d live=%v deleted=%v", launches, world.live, world.deleted)
	}
}

func TestColdRegistrationAndCleanupIgnoreOldIndexedTerminal(t *testing.T) {
	env, record, world := incarnationFixture(t)
	var proposed string
	env.Runner.AfterAcknowledge = func(id string) error {
		for _, entry := range env.Runner.Child(id).Env {
			if raw, ok := strings.CutPrefix(entry, launcher.CouchSessionIntentEnv+"="); ok {
				var intent launcher.CouchSessionIntent
				if err := json.Unmarshal([]byte(raw), &intent); err != nil {
					return err
				}
				proposed = intent.Name
			}
		}
		// The previous terminal reappears; the proposed one never materializes.
		world.live[world.index] = true
		env.Artifacts.SetPaneSidecar(record.Address, "claude")
		return nil
	}
	_, _, err := env.Couch.Resume(record.Address)
	if err == nil {
		t.Fatal("old indexed terminal falsely registered proposed terminal")
	}
	if proposed == "" || proposed == world.index {
		t.Fatalf("missing distinct proposed terminal: %q", proposed)
	}
	if !world.live[world.index] {
		t.Fatalf("old terminal deleted: %v", world.deleted)
	}
	for _, name := range world.deleted {
		if name != proposed {
			t.Fatalf("cleanup targeted %q instead of proposed %q", name, proposed)
		}
	}
	if strings.Contains(err.Error(), "Pair STARTED") {
		t.Fatalf("old terminal produced false startup diagnostic: %v", err)
	}
}

func TestColdLegacyMissingIndexIsNotTerminalAbsence(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPairSession(record.Address, "", false)
	if _, _, err := env.Couch.Resume(record.Address); err == nil {
		t.Fatal("missing legacy association authorized cold create")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("unproven absence launched a helper: %v", env.Runner.Ops)
	}
}

func TestInterruptedBoundStartIgnoresOldRegistrationBeforeHelper(t *testing.T) {
	env, record, world := incarnationFixture(t)
	world.live[world.index] = true
	const nonce = "start-0123456789abcdef"
	claimed, err := env.Couch.Threads.CommitStartClaim(record.Address, record.Revision, "/repo/.git", env.Now, StartEvent{Kind: StartClaimed, Nonce: nonce, Owner: SupervisorOwner{PID: 777, Identity: "dead-owner"}, Profile: record.LatestLaunchProfile, Shape: StartColdResume})
	if err != nil {
		t.Fatal(err)
	}
	proposed := couchidentity.SessionBinding{C: 1, M: 1, Name: "📁1-1", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), StartNonce: nonce}
	if _, err := env.Couch.Threads.AdvanceStart(record.Address, claimed.Revision, StartEvent{Kind: StartSessionBound, Nonce: nonce, Binding: &proposed}); err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.reconcileInterruptedStarts(); err != nil {
		t.Fatal(err)
	}
	after, err := env.Couch.Threads.GetThread(record.Address)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Incarnations) != 0 || after.VerifiedPark == nil {
		t.Fatalf("old registration occupied absent proposed terminal: %+v", after)
	}
	if !world.live[world.index] || len(world.deleted) != 0 {
		t.Fatalf("recovery changed old terminal: %+v", world)
	}
}
