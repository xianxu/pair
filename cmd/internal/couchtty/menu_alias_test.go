package couchtty

import (
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// livePrimaryMenuRow is a live :0 row whose scope proves its repository root.
func livePrimaryMenuRow(t *testing.T) couchcore.ActionableThreadSummary {
	t.Helper()
	scope, err := launcher.ResolveRepoScope("/w/xianxu.dev")
	if err != nil {
		t.Fatal(err)
	}
	return couchcore.ActionableThreadSummary{
		Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-primary"}, StartingPath: "/w/xianxu.dev",
		WorkingPath: "/w/xianxu.dev", State: couchcore.ThreadLive,
	}
}

func TestAliasOfferedOnlyOnLivePrimary(t *testing.T) {
	primary := livePrimaryMenuRow(t)
	if !slices.Contains(menuActionItems(primary), "alias") {
		t.Fatalf("live :0 lacks alias: %v", menuActionItems(primary))
	}
	parked := primary
	parked.State = couchcore.ThreadParked
	slot := menuSlotRow(1, "couch-slot")
	slot.State = couchcore.ThreadLive
	unrooted := primary
	unrooted.StartingPath = ""
	for name, row := range map[string]couchcore.ActionableThreadSummary{"parked": parked, "slot": slot, "no repository root": unrooted} {
		if slices.Contains(menuActionItems(row), "alias") {
			t.Errorf("%s offers alias: %v", name, menuActionItems(row))
		}
	}
}

func TestAliasTextFrameDispatchesRepositoryRoot(t *testing.T) {
	row := livePrimaryMenuRow(t)
	for _, tc := range []struct {
		input string
		want  map[string]string
	}{
		{"blog", map[string]string{"ref": "/w/xianxu.dev", "alias": "blog"}},
		{"", map[string]string{"ref": "/w/xianxu.dev", "clear": "true"}},
	} {
		state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
		appendMenuFrame(&state, MenuFrame{Kind: MenuFrameText, RowKey: menuRowKey(row), Thread: row.Address, Action: "alias", Input: tc.input})
		_, effects := reduceTextKey(state, PanelKey{Kind: KeyEnter})
		if len(effects) != 1 || effects[0].Operation != "alias" || len(effects[0].Args) != len(tc.want) {
			t.Fatalf("%q: effects %+v", tc.input, effects)
		}
		for k, v := range tc.want {
			if effects[0].Args[k] != v {
				t.Fatalf("%q: arg %s = %q, want %q (%+v)", tc.input, k, effects[0].Args[k], v, effects[0].Args)
			}
		}
	}
}

func TestPresentThreadsLabelsAliasedRepository(t *testing.T) {
	primary := livePrimaryMenuRow(t)
	primary.RepositoryAlias = "blog"
	slot := couchcore.SlotIdentity{Repo: "xianxu.dev", RepoIdentity: "/w/xianxu.dev/.git", PrimaryRoot: "/w/xianxu.dev", EnvironmentRoot: "/w/worktree/xianxu.dev-slot1", WorktreeRoot: "/w/worktree/xianxu.dev-slot1/xianxu.dev", Number: 1}
	target := couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: slot}
	key, _ := target.RowKey()
	slotRow := couchcore.ActionableThreadSummary{Target: target, RowKey: key, Address: couchcore.ThreadAddress{RepoScope: "slot-scope", Tag: "couch-slot"}, State: couchcore.ThreadLive, RepositoryAlias: "blog"}
	labels := map[string]bool{}
	for _, p := range PresentThreads([]couchcore.ActionableThreadSummary{primary, slotRow}, nil) {
		labels[p.Label] = true
	}
	if !labels["blog"] || !labels["blog:1"] {
		t.Fatalf("labels %v", labels)
	}
	if got := slotRow.Label(); got != "blog:1" {
		t.Fatalf("slot row label %q", got)
	}
}
