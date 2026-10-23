package couchcore

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
)

func repositorySlotProfileFixture(t *testing.T) (*Couch, *ThreadStore, []SlotIdentity) {
	t.Helper()
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	f.git(f.Primary, "worktree", "add", "-b", "main-slot2", f.host(2), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := repoPreferenceStoreFixture(t)
	if err := store.withLock(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	c := &Couch{Threads: store, Git: ExecGit{}, Path: OSPathOps{}, RootAgent: "codex"}
	var slots []SlotIdentity
	for _, candidate := range repository.Slots {
		slots = append(slots, candidate.Identity)
	}
	if len(slots) != 2 {
		t.Fatalf("slot fixture got %d slots", len(slots))
	}
	return c, store, slots
}

func TestRepoPreferenceSlotProfileSharesArgumentsAcrossWorktrees(t *testing.T) {
	t.Parallel()
	c, store, slots := repositorySlotProfileFixture(t)
	want := LaunchProfile{Agent: "codex", Argv: []string{"--model", "remembered"}}
	if err := store.rememberRepoConfiguration(slots[0].RepoIdentity, want, nil); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.repoPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, slot := range slots {
		local := newSlotThreadStore(store.namespace, slot)
		local.readOnly = true
		got, err := c.slotLaunchProfile(context.Background(), local, slot, slot.WorktreeRoot, "codex")
		if err != nil || !reflect.DeepEqual(got.Profile, want) {
			t.Fatalf("shared profile = %+v, %v", got, err)
		}
		if _, err := os.Lstat(local.root); !os.IsNotExist(err) {
			t.Fatalf("preview initialized local metadata: %v", err)
		}
	}
	after, err := os.ReadFile(store.repoPreferencesPath())
	if err != nil || string(before) != string(after) {
		t.Fatalf("preview changed repository preferences: %v", err)
	}
}

func TestRepoPreferenceSlotProfileFoldsLegacyWithoutInheritingAgent(t *testing.T) {
	t.Parallel()
	c, store, slots := repositorySlotProfileFixture(t)
	pref := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: slots[0].RepoIdentity, PhysicalPath: slots[0].WorktreeRoot, LastAgent: "claude", ArgvByAgent: map[string][]string{"codex": {"--model", "legacy"}, "claude": {"--claude-only"}}, Revision: 1}
	if err := writePathLaunchPreferenceForTest(store, pref); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(store.namespace, slots[1])
	local.readOnly = true
	got, err := c.slotLaunchProfile(context.Background(), local, slots[1], slots[1].WorktreeRoot, "")
	want := LaunchProfile{Agent: "codex", Argv: []string{"--model", "legacy"}}
	if err != nil || !reflect.DeepEqual(got.Profile, want) || got.AgentSource != AgentSourceRoot {
		t.Fatalf("legacy repository profile = %+v, %v", got, err)
	}
	if _, err := os.Lstat(store.repoPreferencesPath()); !os.IsNotExist(err) {
		t.Fatalf("legacy preview persisted a repository document: %v", err)
	}
}

func TestRepoPreferenceSlotProfileDoesNotBorrowOrCrossAgent(t *testing.T) {
	t.Parallel()
	c, store, slots := repositorySlotProfileFixture(t)
	if err := store.rememberRepoConfiguration(slots[0].RepoIdentity, LaunchProfile{Agent: "claude", Argv: []string{"--other-agent"}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.rememberRepoConfiguration("/other/.git", LaunchProfile{Agent: "codex", Argv: []string{"--other-repo"}}, nil); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(store.namespace, slots[0])
	got, err := c.slotLaunchProfile(context.Background(), local, slots[0], slots[0].WorktreeRoot, "")
	if err != nil || got.Profile.Agent != "codex" || len(got.Profile.Argv) != 0 {
		t.Fatalf("arguments leaked: %+v %v", got, err)
	}
}

func TestRepoPreferenceSlotProfileRefusesLegacyConflict(t *testing.T) {
	t.Parallel()
	c, store, slots := repositorySlotProfileFixture(t)
	for i, slot := range slots {
		pref := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: slot.RepoIdentity, PhysicalPath: slot.WorktreeRoot, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {[]string{"--first", "--second"}[i]}}, Revision: 1}
		if err := writePathLaunchPreferenceForTest(store, pref); err != nil {
			t.Fatal(err)
		}
	}
	local := newSlotThreadStore(store.namespace, slots[0])
	_, err := c.slotLaunchProfile(context.Background(), local, slots[0], slots[0].WorktreeRoot, "codex")
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflicting legacy arguments accepted: %v", err)
	}
}
