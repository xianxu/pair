package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func TestHubCLIExplicitDirectoriesAndDefault(t *testing.T) {
	defaultCall, err := ParseCLI(nil, couchcore.Operations())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaultCall, cliInvocation{kind: cliLaunch, path: ".", layout: couchcore.DefaultLayout, tabStyle: couchtty.DefaultTabStyle}) {
		t.Fatalf("default changed: %+v", defaultCall)
	}
	for _, args := range [][]string{{".", "../bench"}, {"--layout3", ".", "../bench"}, {"--", ".", "../bench"}} {
		got, err := ParseCLI(args, couchcore.Operations())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.hubPaths, []string{".", "../bench"}) {
			t.Fatalf("%v: %+v", args, got)
		}
	}
	for _, args := range [][]string{{".", "--bogus"}, {".", ""}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}
func TestHubMoveCLI(t *testing.T) {
	got, err := ParseCLI([]string{"--move-slot", "pair:7,pair:8", "--to-hub", "bench:0", "--persist"}, couchcore.Operations())
	if err != nil {
		t.Fatal(err)
	}
	if got.messageOp != "move-slot" || got.ref != "pair:7,pair:8" || got.toHub != "bench:0" || !got.persistHub {
		t.Fatalf("%+v", got)
	}
	for _, args := range [][]string{{"--move-slot", "pair", "--to-hub", "bench:0"}, {"--move-slot", "pair:7", "--persist"}, {"--move-slot", "pair:7", "--to-hub", "bench:0", "--persist", "--persist"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}

func TestHubPreparationReadsOnlyActiveFiles(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	paths := []string{filepath.Join(root, "ops"), filepath.Join(root, "bench")}
	catalog := &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{}}
	for _, p := range paths {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
		n := 0
		catalog.Workspaces[p] = couchcore.WorkspaceIdentity{WorktreeRoot: p, PrimaryRoot: p, FleetRoot: root, Slot: &n}
	}
	c := &couchcore.Couch{Slots: catalog}
	if err := os.WriteFile(filepath.Join(paths[1], ".hub"), []byte("pair:7\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hubs, err := prepareExplicitHubs(context.Background(), c, paths)
	if err != nil {
		t.Fatal(err)
	}
	if got := hubs.snapshot().Resolve(couchcore.HubSlot{Repository: filepath.Join(root, "pair"), Number: 7}).Hub; got != paths[1] {
		t.Fatal(got)
	}
	if err := os.WriteFile(filepath.Join(paths[1], ".hub"), []byte("!bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareExplicitHubs(context.Background(), c, paths); err == nil {
		t.Fatal("corrupt claims treated as empty")
	}
	if _, err := prepareExplicitHubs(context.Background(), c, paths[:1]); err != nil {
		t.Fatal("read omitted hub", err)
	}
	if _, err := prepareExplicitHubs(context.Background(), c, []string{paths[0], paths[0]}); err == nil {
		t.Fatal("duplicate hub")
	}
}

func TestHubStartupOperationUsesDeclaredDispatch(t *testing.T) {
	called := false
	_, err := couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: func(call couchcore.OperationCall) (any, error) {
		called = true
		if call.Args["path"] != "/work/bench" {
			t.Fatal(call)
		}
		return nil, nil
	}}, couchcore.OperationCall{Name: "start-hub", Args: map[string]string{"path": "/work/bench"}, Implicit: true, Context: context.Background()})
	if err != nil || !called {
		t.Fatalf("start-hub dispatch: %v called=%v", err, called)
	}
}

func TestHubConflictWarningsNameEqualSpecificityOnly(t *testing.T) {
	h, defs, _ := moveFixture(t)
	defs[0].Claims, _ = couchcore.ParseHubClaims([]byte("pair\npair:7\n"))
	defs[1].Claims, _ = couchcore.ParseHubClaims([]byte("pair\npair:8\n"))
	state, err := couchcore.NewHubOwnership(defs)
	if err != nil {
		t.Fatal(err)
	}
	h.state = state
	warnings := hubConflictWarnings(state)
	if len(warnings) != 1 || !strings.Contains(warnings[0], defs[0].Root) || !strings.Contains(warnings[0], defs[1].Root) || !strings.Contains(warnings[0], "pair") {
		t.Fatalf("warnings %v", warnings)
	}
}

func TestHubStartupQueueWaitsForEachAdoption(t *testing.T) {
	_, defs, _ := moveFixture(t)
	var finished []func(any, error)
	var roots []string
	queueExplicitHubs(defs[1:], func(_ string, _ string, prepare func(context.Context) (couchcore.OperationCall, error), _ func(), finish func(any, error)) error {
		call, err := prepare(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, call.Args["path"])
		finished = append(finished, finish)
		return nil
	}, func(string, error) {})
	if len(roots) != 1 || roots[0] != defs[1].Root {
		t.Fatalf("eager starts %v", roots)
	}
	finished[0](nil, nil)
	if len(roots) != 2 || roots[1] != defs[2].Root {
		t.Fatalf("order %v", roots)
	}
	finished[1](nil, nil)
	if len(roots) != 2 {
		t.Fatal("extra launch")
	}
}
