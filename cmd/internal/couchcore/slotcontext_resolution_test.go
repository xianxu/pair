package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shortNameFixture enrolls parley.nvim, xianxu.dev, brain and brainstorm under
// one fleet root; the caller's context is pair, which is not enrolled.
func shortNameFixture(t *testing.T) (*Couch, *ThreadStore, string) {
	t.Helper()
	fleet := t.TempDir()
	repo := func(name string, slots ...int) SlotRepository {
		primary := filepath.Join(fleet, name)
		if err := os.MkdirAll(primary, 0700); err != nil {
			t.Fatal(err)
		}
		r := SlotRepository{Identity: WorkspaceIdentity{Repo: name, PrimaryRoot: primary, FleetRoot: fleet, WorktreeRoot: primary, Kind: "primary"}}
		for _, n := range slots {
			r.Slots = append(r.Slots, SlotCandidate{Identity: conventionalSlot(primary, n), Verified: true})
		}
		return r
	}
	catalog := &SlotCatalogFake{Repositories: map[string]SlotRepository{}, Workspaces: map[string]WorkspaceIdentity{}}
	var roots []string
	for _, r := range []SlotRepository{repo("parley.nvim", 1, 2), repo("xianxu.dev", 1), repo("brain"), repo("brainstorm")} {
		catalog.Repositories[r.Identity.PrimaryRoot] = r
		roots = append(roots, r.Identity.PrimaryRoot)
	}
	pair := repo("pair")
	catalog.Repositories[pair.Identity.PrimaryRoot] = pair
	catalog.Workspaces["."] = pair.Identity
	store := aliasTestStore(t, roots...)
	return &Couch{Slots: catalog, Threads: store}, store, fleet
}

func TestSlotReferenceResolvesUniquePrefixAndAlias(t *testing.T) {
	t.Parallel()

	c, store, fleet := shortNameFixture(t)
	ctx := context.Background()
	path, recognized, err := c.WorkspaceReferencePath(ctx, "parley:1")
	if err != nil || !recognized || path != conventionalSlot(filepath.Join(fleet, "parley.nvim"), 1).WorktreeRoot {
		t.Fatalf("prefix: %q %v %v", path, recognized, err)
	}
	if path, _, err = c.WorkspaceReferencePath(ctx, "parley.nvim:2"); err != nil || path != conventionalSlot(filepath.Join(fleet, "parley.nvim"), 2).WorktreeRoot {
		t.Fatalf("exact: %q %v", path, err)
	}
	if err := store.SetRepositoryAlias(filepath.Join(fleet, "xianxu.dev"), "blog"); err != nil {
		t.Fatal(err)
	}
	if path, _, err = c.WorkspaceReferencePath(ctx, "blog:1"); err != nil || path != conventionalSlot(filepath.Join(fleet, "xianxu.dev"), 1).WorktreeRoot {
		t.Fatalf("alias: %q %v", path, err)
	}
	if path, _, err = c.WorkspaceReferencePath(ctx, "blog:0"); err != nil || path != filepath.Join(fleet, "xianxu.dev") {
		t.Fatalf("alias primary: %q %v", path, err)
	}
	if path, _, err = c.WorkspaceReferencePath(ctx, "xianxu.dev:1"); err != nil || path != conventionalSlot(filepath.Join(fleet, "xianxu.dev"), 1).WorktreeRoot {
		t.Fatalf("directory name after alias: %q %v", path, err)
	}
}

func TestSlotReferenceOutsideRepositoryUsesPrefix(t *testing.T) {
	t.Parallel()

	c, _, fleet := shortNameFixture(t)
	c.Slots.(*SlotCatalogFake).Errors = map[string]error{".": errors.New("not in a repository")}
	target, id, err := c.resolveSlotInput(context.Background(), "parley:2")
	if err != nil || target == nil || target.Slot.Number != 2 || id.PrimaryRoot != filepath.Join(fleet, "parley.nvim") {
		t.Fatalf("no context: %+v %+v %v", target, id, err)
	}
}

func TestSlotReferenceRefusalsListCandidates(t *testing.T) {
	t.Parallel()

	c, _, _ := shortNameFixture(t)
	ctx := context.Background()
	_, _, err := c.WorkspaceReferencePath(ctx, "bra:0")
	if !errors.Is(err, ErrRepositoryAmbiguous) || !strings.Contains(err.Error(), "brain, brainstorm") {
		t.Fatalf("ambiguous: %v", err)
	}
	_, _, err = c.WorkspaceReferencePath(ctx, "nope:1")
	if !errors.Is(err, ErrRepositoryNotFound) || !strings.Contains(err.Error(), "parley.nvim") {
		t.Fatalf("miss: %v", err)
	}
	_, _, err = c.WorkspaceReferencePath(ctx, "parley:7")
	if err == nil || !strings.Contains(err.Error(), "slot parley.nvim:7 does not exist (existing: parley.nvim:0, parley.nvim:1, parley.nvim:2)") {
		t.Fatalf("slot miss: %v", err)
	}
}

func TestExistingSiblingDirectoryBeatsPrefix(t *testing.T) {
	t.Parallel()

	c, _, fleet := shortNameFixture(t)
	sibling := filepath.Join(fleet, "parley")
	if err := os.MkdirAll(sibling, 0700); err != nil {
		t.Fatal(err)
	}
	// The sibling is not a known repository, so its own discovery error
	// surfaces; it is never silently rerouted to parley.nvim.
	_, _, err := c.WorkspaceReferencePath(context.Background(), "parley:1")
	if err == nil || !strings.Contains(err.Error(), sibling) || strings.Contains(err.Error(), "parley.nvim") {
		t.Fatalf("sibling: %v", err)
	}
}

func TestRepositoryHubSlotsAgreesWithReferenceResolution(t *testing.T) {
	env, f, _ := recoverAcceptanceFixture(t, 3)
	slots, err := env.Couch.RepositoryHubSlots(t.Context(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 3 {
		t.Fatalf("slots = %v", slots)
	}
	resolver := env.Couch.Slots.(SlotWorkspaceResolver)
	for n := 1; n <= 3; n++ {
		ref := WorkspaceReference{Repo: filepath.Base(f.Primary), Number: n}.String()
		path, recognized, err := env.Couch.WorkspaceReferencePath(t.Context(), ref)
		if err != nil || !recognized {
			t.Fatalf("reference %s: %v %v", ref, recognized, err)
		}
		id, err := resolver.ResolveWorkspace(t.Context(), path)
		if err != nil {
			t.Fatal(err)
		}
		want := HubSlot{Repository: id.PrimaryRoot, Number: *id.Slot}
		if slots[n] != want {
			t.Fatalf("%s: %v want %v", ref, slots[n], want)
		}
	}
}

func TestRepositoryHubSlotsRejectsUnverifiedRows(t *testing.T) {
	c, _, fleet := shortNameFixture(t)
	primary := filepath.Join(fleet, "parley.nvim")
	catalog := c.Slots.(*SlotCatalogFake)
	repository := catalog.Repositories[primary]
	repository.Slots[0].Verified = false
	repository.Slots[1].Err = errors.New("replaced workspace")
	catalog.Repositories[primary] = repository
	before := len(catalog.Calls)
	slots, err := c.RepositoryHubSlots(t.Context(), primary)
	if err != nil || len(slots) != 0 {
		t.Fatalf("unverified rows: %v %v", slots, err)
	}
	if calls := len(catalog.Calls) - before; calls != 1 {
		t.Fatalf("Discover calls %d, want one", calls)
	}
	catalog.Errors = map[string]error{primary: errors.New("inventory unavailable")}
	if _, err := c.RepositoryHubSlots(t.Context(), primary); err == nil || !strings.Contains(err.Error(), "inventory unavailable") {
		t.Fatalf("discovery failure: %v", err)
	}
	c.Slots = nil
	if _, err := c.RepositoryHubSlots(t.Context(), primary); err == nil {
		t.Fatal("missing catalog accepted")
	}
}
