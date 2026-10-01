package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func aliasCall(t *testing.T, c *Couch, args map[string]string) (any, error) {
	t.Helper()
	for _, op := range Operations() {
		if op.Name == "alias" {
			return DirectStoreExecutor(c)(OperationCall{Name: "alias", Args: args, Context: context.Background(), Operation: op})
		}
	}
	t.Fatal("alias is not declared")
	return nil, nil
}

func TestAliasOperationSetsReadsAndClears(t *testing.T) {
	c, _, fleet := shortNameFixture(t)
	got, err := aliasCall(t, c, map[string]string{"ref": "xianxu", "alias": "blog"})
	if err != nil || got != (RepositoryAliasResult{Repository: "xianxu.dev", Alias: "blog"}) {
		t.Fatalf("set by prefix: %+v %v", got, err)
	}
	got, err = aliasCall(t, c, map[string]string{"ref": "blog:1"})
	if err != nil || got != (RepositoryAliasResult{Repository: "xianxu.dev", Alias: "blog"}) {
		t.Fatalf("read by alias slot: %+v %v", got, err)
	}
	if path, _, err := c.WorkspaceReferencePath(context.Background(), "blog:1"); err != nil || path != conventionalSlot(filepath.Join(fleet, "xianxu.dev"), 1).WorktreeRoot {
		t.Fatalf("alias does not address: %q %v", path, err)
	}
	if _, err := aliasCall(t, c, map[string]string{"ref": "blog", "alias": "x", "clear": "true"}); err == nil {
		t.Fatal("alias and --clear together accepted")
	}
	got, err = aliasCall(t, c, map[string]string{"ref": "xianxu.dev", "clear": "true"})
	if err != nil || got != (RepositoryAliasResult{Repository: "xianxu.dev"}) {
		t.Fatalf("clear: %+v %v", got, err)
	}
}

func TestAliasOperationRefusesShadowedOrTakenNames(t *testing.T) {
	c, _, fleet := shortNameFixture(t)
	if err := os.MkdirAll(filepath.Join(fleet, "web"), 0700); err != nil {
		t.Fatal(err)
	}
	for alias, want := range map[string]string{"web": "shadowed", "brain": "brain", "parley.nvim": "parley.nvim"} {
		if _, err := aliasCall(t, c, map[string]string{"ref": "xianxu.dev", "alias": alias}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", alias, err)
		}
	}
	if _, err := aliasCall(t, c, map[string]string{"ref": "parley", "alias": "pl"}); err != nil {
		t.Fatal(err)
	}
	if _, err := aliasCall(t, c, map[string]string{"ref": "xianxu.dev", "alias": "pl"}); err == nil || !strings.Contains(err.Error(), "already used by parley.nvim (pl)") {
		t.Fatalf("taken alias: %v", err)
	}
}

func TestApplyRepositoryAliasesMatchesScopeNotPath(t *testing.T) {
	scope := func(root string) string {
		s, err := launcher.ResolveRepoScope(root)
		if err != nil {
			t.Fatal(err)
		}
		return s.Key
	}
	slot := ThreadTarget{Kind: ThreadTargetSlot, Slot: conventionalSlot("/w/xianxu.dev", 1)}
	rows := ApplyRepositoryAliases([]ActionableThreadSummary{
		{Address: ThreadAddress{RepoScope: scope("/w/xianxu.dev"), Tag: "primary"}, StartingPath: "/w/xianxu.dev"},
		{Address: ThreadAddress{RepoScope: scope("/w/xianxu.dev/nested"), Tag: "nested"}, StartingPath: "/w/xianxu.dev/nested"},
		{Target: slot, Address: ThreadAddress{RepoScope: "slot", Tag: "slot"}},
		{Address: ThreadAddress{RepoScope: scope("/w/pair"), Tag: "pair"}},
		{Address: ThreadAddress{RepoScope: scope("/w/xianxu.dev"), Tag: "subdir"}, StartingPath: "/w/xianxu.dev/sub"},
	}, []RepositoryName{{Key: "/w/xianxu.dev", Dir: "xianxu.dev", Alias: "blog"}, {Key: "/w/pair", Dir: "pair"}})
	for i, want := range []string{"blog", "", "blog", "", ""} {
		if rows[i].RepositoryAlias != want {
			t.Errorf("row %s: %q want %q", rows[i].Address.Tag, rows[i].RepositoryAlias, want)
		}
	}
	if got := rows[2].Label(); got != "blog:1" {
		t.Fatalf("slot label %q", got)
	}
}

// The shadow check looks where resolution looks: the repository's fleet root,
// even when that is not its primary root's parent directory.
func TestAliasShadowCheckUsesFleetRoot(t *testing.T) {
	c, _, fleet := shortNameFixture(t)
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(elsewhere, "web"), 0700); err != nil {
		t.Fatal(err)
	}
	catalog := c.Slots.(*SlotCatalogFake)
	root := filepath.Join(fleet, "xianxu.dev")
	repo := catalog.Repositories[root]
	repo.Identity.FleetRoot = elsewhere
	catalog.Repositories[root] = repo
	if _, err := aliasCall(t, c, map[string]string{"ref": "xianxu.dev", "alias": "web"}); err == nil || !strings.Contains(err.Error(), "shadowed") {
		t.Fatalf("fleet-root sibling not checked: %v", err)
	}
}
