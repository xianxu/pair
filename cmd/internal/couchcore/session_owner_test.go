package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// sessionOwnerWorld models the actual terminal separately from stale indexes.
type sessionOwnerWorld struct {
	root       string
	owners     map[string]ThreadAddress
	generation string
}

func (w *sessionOwnerWorld) SessionServers(_ context.Context, name string) ([]launcher.SessionServerIdentity, error) {
	if _, ok := w.owners[name]; !ok {
		return nil, nil
	}
	return []launcher.SessionServerIdentity{{PID: 42, Identity: w.generation, Session: name}}, nil
}
func (w *sessionOwnerWorld) SessionPresent(_ context.Context, name string) (bool, error) {
	_, ok := w.owners[name]
	return ok, nil
}
func (w *sessionOwnerWorld) Socket(string) launcher.SocketState { return launcher.SocketPresent }
func (w *sessionOwnerWorld) SessionPanes(_ context.Context, name string) ([]byte, error) {
	a := w.owners[name]
	p, e := artifactpath.Resolve(artifactpath.Address{DataDir: w.root, RepoScope: a.RepoScope, Tag: string(a.Tag)})
	if e != nil {
		return nil, e
	}
	return json.Marshal([]map[string]any{{"id": 1, "pane_command": fmt.Sprintf("nvim %q", p.Draft())}})
}
func testOwnerProbe(root, name string, address ThreadAddress) launcher.SessionOwnerProbe {
	return launcher.SessionOwnerProbe{IO: &sessionOwnerWorld{root: root, owners: map[string]ThreadAddress{name: address}, generation: "server-1"}}
}

func TestTwoScopeNameCollisionNeverOwnsForeignSession(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := "📁pair-couch-3"
	old := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "old"}
	live := ThreadAddress{RepoScope: "fedcba9876543210", Tag: "live"}
	indexSession(t, root, old, name)
	indexSession(t, root, live, name)
	deleter := &fakeSessionDeleter{}
	checker, _ := sandboxedChecker(t, root, map[string]string{name: "detached"})
	checker.Sessions = deleter
	checker.OwnerProbe = testOwnerProbe(root, name, live)
	stale, err := checker.PairSession(old)
	if err != nil || stale.Present {
		t.Fatalf("stale index grants owner: %+v %v", stale, err)
	}
	owned, err := checker.PairSession(live)
	if err != nil || !owned.Present {
		t.Fatalf("actual owner: %+v %v", owned, err)
	}
	if err := checker.Quiesce(old); err != nil {
		t.Fatal(err)
	}
	if len(deleter.deleted) != 0 {
		t.Fatalf("foreign terminal deleted: %v", deleter.deleted)
	}
	intent := launcher.QuitIntent{Version: launcher.QuitIntentVersion, Kind: launcher.QuitIntentCouch, Request: &launcher.QuitRequestReference{DataDir: root, RepoScope: old.RepoScope, Tag: string(old.Tag), Nonce: "park-1", Attempt: 1}}
	if err := checker.TriggerQuit(name, intent); err == nil {
		t.Fatal("foreign quit accepted")
	}
	if len(deleter.deleted) != 0 {
		t.Fatalf("foreign terminal quit: %v", deleter.deleted)
	}
	detached, err := checker.DetachedSessions(context.Background(), []DetachedCandidate{{Address: old, Agent: "claude"}})
	if err != nil || len(detached) != 0 {
		t.Fatalf("foreign stale name selected warm resume: %+v %v", detached, err)
	}
}

func TestBoundQuitRejectsReplacedServerBeforeWritingIntent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	name := "📁1-2"
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "1-repo-1"}
	world := &sessionOwnerWorld{root: root, owners: map[string]ThreadAddress{name: address}, generation: "first"}
	deleter := &fakeSessionDeleter{}
	checker := NewScopedThreadArtifactCollisionChecker(root)
	checker.Sessions = deleter
	checker.OwnerProbe = launcher.SessionOwnerProbe{IO: world}
	binding, err := checker.NamedPairSessionContext(context.Background(), address, name)
	if err != nil || !binding.Present {
		t.Fatalf("%+v %v", binding, err)
	}
	world.generation = "replacement"
	intent := launcher.QuitIntent{Version: launcher.QuitIntentVersion, Kind: launcher.QuitIntentCouch, Request: &launcher.QuitRequestReference{DataDir: root, RepoScope: address.RepoScope, Tag: string(address.Tag), Nonce: "park-1", Attempt: 1}}
	if err := checker.TriggerBoundQuit(context.Background(), binding, intent); err == nil {
		t.Fatal("replaced server accepted")
	}
	if len(deleter.deleted) != 0 {
		t.Fatalf("deleted replacement: %v", deleter.deleted)
	}
	_ = filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			t.Errorf("refusal wrote %s", path)
		}
		return err
	})
}

func TestSlotAdmissionDisregardsProvenForeignStaleName(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, err := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	old := validThreadRecord(t)
	old.Address.RepoScope = scope.Key
	old.StartingPath = local.slot.WorktreeRoot
	old.WorkingPath = old.StartingPath
	old.Incarnations = nil
	if _, err := local.CreateThread(old); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := "📁pair-couch-3"
	foreign := ThreadAddress{RepoScope: "fedcba9876543210", Tag: "other-slot"}
	indexSession(t, root, old.Address, name)
	indexSession(t, root, foreign, name)
	checker, _ := sandboxedChecker(t, root, map[string]string{name: "detached"})
	checker.OwnerProbe = testOwnerProbe(root, name, foreign)
	env.Couch.Artifacts = checker
	observed, err := env.Couch.ObserveSlotSessions(context.Background(), *local.slot)
	if err != nil || !observed.Absent {
		t.Fatalf("foreign stale name blocked stopped slot: %+v %v", observed, err)
	}
}
