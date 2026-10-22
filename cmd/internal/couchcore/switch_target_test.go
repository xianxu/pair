package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type switchTargetCountedCatalog struct {
	*SlotCatalogFake
	workspaces  []string
	discoveries int
}

func (c *switchTargetCountedCatalog) Discover(context.Context, string) (SlotRepository, error) {
	c.discoveries++
	return SlotRepository{}, errors.New("unrelated slot discovery must not participate in explicit target selection")
}

func (c *switchTargetCountedCatalog) ResolveWorkspace(ctx context.Context, path string) (WorkspaceIdentity, error) {
	c.workspaces = append(c.workspaces, path)
	return c.SlotCatalogFake.ResolveWorkspace(ctx, path)
}

func TestResolveAgentSwitchTargetDirectProbeCostIndependentOfOtherSlots(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			env, local, catalog := switchTargetFixture(t)
			record := slotPreferenceRecord(t, env, local)
			catalog.Workspaces["."] = catalog.Workspaces[local.slot.PrimaryRoot]
			repo := catalog.Repositories[local.slot.PrimaryRoot]
			for n := 2; n < count+2; n++ {
				repo.Slots = append(repo.Slots, SlotCandidate{Identity: conventionalSlot(local.slot.PrimaryRoot, n), Err: errors.New("unrelated slot is unavailable")})
			}
			catalog.Repositories[local.slot.PrimaryRoot] = repo
			counted := &switchTargetCountedCatalog{SlotCatalogFake: catalog}
			env.Couch.Slots = counted
			for range 2 { // Initial resolution and its fresh authorization recheck.
				address, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), local.slot.Repo+":1")
				if err != nil || address != record.Address {
					t.Fatalf("direct target resolution: %+v %v", address, err)
				}
			}
			want := []string{".", local.slot.WorktreeRoot, ".", local.slot.WorktreeRoot}
			if counted.discoveries != 0 || !reflect.DeepEqual(counted.workspaces, want) {
				t.Fatalf("target cost grew with %d unrelated slots: discoveries=%d workspaces=%q", count, counted.discoveries, counted.workspaces)
			}
		})
	}
}

func TestResolveAgentSwitchTargetDirectRecheckPreservesSiblingPrecedence(t *testing.T) {
	env, local, catalog := switchTargetFixture(t)
	record := slotPreferenceRecord(t, env, local)
	catalog.Workspaces["."] = catalog.Workspaces[local.slot.PrimaryRoot]
	if err := env.Couch.Threads.SetRepositoryAlias(local.slot.PrimaryRoot, "short"); err != nil {
		t.Fatal(err)
	}
	counted := &switchTargetCountedCatalog{SlotCatalogFake: catalog}
	env.Couch.Slots = counted
	address, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), "short:1")
	if err != nil || address != record.Address {
		t.Fatalf("initial alias selection: %+v %v", address, err)
	}
	// An exact sibling directory shadows the enrolled alias. Even though the
	// prior selection is still readable, the next check must verify the sibling
	// candidate and refuse its missing identity instead of reusing old authority.
	sibling := filepath.Join(filepath.Dir(local.slot.PrimaryRoot), "short")
	if err := os.Mkdir(sibling, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), "short:1"); err == nil {
		t.Fatal("recheck ignored the new exact sibling")
	}
	want := []string{".", local.slot.WorktreeRoot, ".", NewSlotLayout(sibling, "", 1).Host()}
	if counted.discoveries != 0 || !reflect.DeepEqual(counted.workspaces, want) {
		t.Fatalf("sibling recheck probes: discovery=%d workspaces=%q", counted.discoveries, counted.workspaces)
	}
}

func TestRecheckAgentSwitchTargetUsesFreshSelectionWithoutWorkspaceProbe(t *testing.T) {
	env, local, catalog := switchTargetFixture(t)
	record := slotPreferenceRecord(t, env, local)
	catalog.Workspaces["."] = catalog.Workspaces[local.slot.PrimaryRoot]
	if err := env.Couch.Threads.SetRepositoryAlias(local.slot.PrimaryRoot, "short"); err != nil {
		t.Fatal(err)
	}
	counted := &switchTargetCountedCatalog{SlotCatalogFake: catalog}
	env.Couch.Slots = counted
	address, identity, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), "short:1")
	if err != nil || address != record.Address {
		t.Fatal(err)
	}
	for range 2 {
		if err := env.Couch.RecheckAgentSwitchTarget(context.Background(), "short:1", address, identity); err != nil {
			t.Fatal(err)
		}
	}
	if counted.discoveries != 0 || !reflect.DeepEqual(counted.workspaces, []string{".", local.slot.WorktreeRoot}) {
		t.Fatalf("selection recheck repeated workspace discovery: %+v", counted)
	}
	if err := os.Mkdir(filepath.Join(filepath.Dir(local.slot.PrimaryRoot), "short"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.RecheckAgentSwitchTarget(context.Background(), "short:1", address, identity); err == nil {
		t.Fatal("new sibling did not revoke admission selection")
	}
}

func switchTargetFixture(t *testing.T) (*testEnv, *ThreadStore, *SlotCatalogFake) {
	t.Helper()
	env, local := slotRecoverySemanticFixture(t)
	catalog := env.Couch.Slots.(*SlotCatalogFake)
	catalog.Workspaces[local.slot.PrimaryRoot] = catalog.Repositories[local.slot.PrimaryRoot].Identity
	// Selection must require neither lifecycle services nor workspace repair.
	env.Couch.Proc, env.Couch.Artifacts, env.Couch.Runner, env.Couch.Workspaces = nil, nil, nil, nil
	return env, local, catalog
}

func switchTargetPrimaryRecord(t *testing.T, env *testEnv, primary, path, tag string) ThreadRecord {
	t.Helper()
	record := recordAtCheckout(t, primary, path, tag)
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.LatestLaunchProfile = &profile
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: IncarnationLive, RepoIdentity: filepath.Join(primary, ".git"), LaunchProfile: &profile}}
	record, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "target-park-" + tag, Address: record.Address, PID: 42, ProcessIdentity: "pair-helper"}
	record, err = env.Couch.Threads.BeginPark(record.Address, record.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	record, err = env.Couch.Threads.FinalizePark(context.Background(), record.Address, record.Revision, identity, 1, env.Now)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestResolveAgentSwitchTargetRetainedPrimaryAndNumbered(t *testing.T) {
	env, local, catalog := switchTargetFixture(t)
	numbered := slotPreferenceRecord(t, env, local)
	primary := switchTargetPrimaryRecord(t, env, local.slot.PrimaryRoot, local.slot.PrimaryRoot, "primary")
	switchTargetPrimaryRecord(t, env, local.slot.PrimaryRoot, filepath.Join(local.slot.PrimaryRoot, "sub"), "subdirectory")
	if err := env.Couch.Threads.SetRepositoryAlias(local.slot.PrimaryRoot, "short"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		ref    string
		record ThreadRecord
		path   string
		number int
	}{
		{local.slot.Repo + ":0", primary, local.slot.PrimaryRoot, 0},
		{local.slot.Repo + ":1", numbered, local.slot.WorktreeRoot, 1},
		{"short:1", numbered, local.slot.WorktreeRoot, 1},
	} {
		t.Run(test.ref, func(t *testing.T) {
			before, err := snapshotDirectoryForTest(env.Couch.Threads.root)
			if err != nil {
				t.Fatal(err)
			}
			localBefore, err := snapshotDirectoryForTest(local.root)
			if err != nil {
				t.Fatal(err)
			}
			address, identity, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), test.ref)
			if err != nil || address != test.record.Address || !reflect.DeepEqual(identity, catalog.Workspaces[test.path]) || identity.Slot == nil || *identity.Slot != test.number {
				t.Fatalf("resolved=%+v %+v err=%v", address, identity, err)
			}
			after, err := snapshotDirectoryForTest(env.Couch.Threads.root)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("selection changed global store: %v", err)
			}
			localAfter, err := snapshotDirectoryForTest(local.root)
			if err != nil || !reflect.DeepEqual(localBefore, localAfter) {
				t.Fatalf("selection changed local store: %v", err)
			}
		})
	}
}

func TestResolveAgentSwitchTargetPreservesPartialSnapshotError(t *testing.T) {
	path := "/fleet/worktree/repo-slot1/repo"
	record := recordAtCheckout(t, path, path, "readable")
	backendErr := errors.New("partial backend read failed")
	snapshot := ThreadSnapshot{
		Records: []ThreadRecord{record},
		Slots: []SlotInventoryObservation{{
			Identity: conventionalSlot("/fleet/repo", 1),
			Address:  record.Address,
			Err:      backendErr,
		}},
	}
	if address, err := selectAgentSwitchTarget(snapshot, 1, path); !errors.Is(err, backendErr) || address != (ThreadAddress{}) {
		t.Fatalf("partial backend error was lost: address=%+v err=%v", address, err)
	}
}

func TestResolveAgentSwitchTargetRefusesMissingUnreadableAndAmbiguous(t *testing.T) {
	for _, fault := range []string{"primary-missing", "numbered-missing", "primary-subdirectory", "primary-ambiguous", "primary-unreadable", "numbered-unreadable"} {
		t.Run(fault, func(t *testing.T) {
			env, local, _ := switchTargetFixture(t)
			target := local.slot.Repo + ":0"
			wantCode := SlotOpNoThread
			switch fault {
			case "numbered-missing":
				target = local.slot.Repo + ":1"
			case "primary-subdirectory":
				switchTargetPrimaryRecord(t, env, local.slot.PrimaryRoot, filepath.Join(local.slot.PrimaryRoot, "sub"), "subdirectory")
			case "primary-ambiguous":
				switchTargetPrimaryRecord(t, env, local.slot.PrimaryRoot, local.slot.PrimaryRoot, "first")
				switchTargetPrimaryRecord(t, env, local.slot.PrimaryRoot, local.slot.PrimaryRoot, "second")
				wantCode = SlotOpAmbiguous
			case "primary-unreadable":
				record := switchTargetPrimaryRecord(t, env, local.slot.PrimaryRoot, local.slot.PrimaryRoot, "corrupt")
				if err := os.WriteFile(env.Couch.Threads.recordPath(record.Address), []byte("broken"), 0600); err != nil {
					t.Fatal(err)
				}
				wantCode = ""
			case "numbered-unreadable":
				slotPreferenceRecord(t, env, local)
				if err := os.WriteFile(filepath.Join(local.root, "thread.json"), []byte("broken"), 0600); err != nil {
					t.Fatal(err)
				}
				target, wantCode = local.slot.Repo+":1", ""
			}
			address, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), target)
			if err == nil || address != (ThreadAddress{}) {
				t.Fatalf("unsafe retained selection=%+v err=%v", address, err)
			}
			if wantCode != "" && slotOperationCode(err) != wantCode {
				t.Fatalf("refusal=%v want code %q", err, wantCode)
			}
			if strings.Contains(fault, "unreadable") && !strings.Contains(err.Error(), "unreadable") {
				t.Fatalf("unreadable record was hidden: %v", err)
			}
		})
	}
}

func TestResolveAgentSwitchTargetRejectsInvalidReferencesAndWorkspace(t *testing.T) {
	env, local, catalog := switchTargetFixture(t)
	slotPreferenceRecord(t, env, local)
	for _, target := range []string{"self", ":1", local.slot.Repo, local.slot.WorktreeRoot, local.slot.Repo + ":99"} {
		if _, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), target); err == nil {
			t.Fatalf("accepted non-slot target %q", target)
		}
	}
	for _, fault := range []string{"schema", "wrong-path", "wrong-number"} {
		t.Run(fault, func(t *testing.T) {
			original := catalog.Workspaces[local.slot.WorktreeRoot]
			changed := original
			switch fault {
			case "schema":
				changed.SchemaVersion = 999
			case "wrong-path":
				changed = catalog.Workspaces[local.slot.PrimaryRoot]
			case "wrong-number":
				zero := 0
				changed.Slot = &zero
			}
			catalog.Workspaces[local.slot.WorktreeRoot] = changed
			t.Cleanup(func() { catalog.Workspaces[local.slot.WorktreeRoot] = original })
			if _, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), local.slot.Repo+":1"); err == nil {
				t.Fatal("accepted invalid or mismatched workspace identity")
			}
		})
	}
}

func TestResolveAgentSwitchTargetCancellationAndPendingJournalAreReadOnly(t *testing.T) {
	env, local, catalog := switchTargetFixture(t)
	record := slotPreferenceRecord(t, env, local)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := len(catalog.Calls)
	if _, _, err := env.Couch.ResolveAgentSwitchTarget(ctx, local.slot.Repo+":1"); !errors.Is(err, context.Canceled) || len(catalog.Calls) != calls {
		t.Fatalf("cancelled resolution probed workspace: %v", err)
	}
	interrupted := errors.New("interrupted optional fixture transaction")
	local.hooks.AfterJournal = func() error { return interrupted }
	if _, err := local.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Description = "must remain pending"; return nil }); !errors.Is(err, interrupted) {
		t.Fatal(err)
	}
	before, err := snapshotDirectoryForTest(local.root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := env.Couch.ResolveAgentSwitchTarget(context.Background(), local.slot.Repo+":1"); err == nil {
		t.Fatal("selection accepted pending recovery")
	}
	after, err := snapshotDirectoryForTest(local.root)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("selection replayed pending journal: %v", err)
	}
}
