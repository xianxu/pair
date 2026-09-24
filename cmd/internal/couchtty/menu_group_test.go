package couchtty

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func groupedRow(root string, number int, tag string) couchcore.ActionableThreadSummary {
	path := root
	target := couchcore.ThreadTarget{}
	if number > 0 {
		repo := filepath.Base(root)
		env := filepath.Join(filepath.Dir(root), "worktree", fmt.Sprintf("%s-slot%d", repo, number))
		path = filepath.Join(env, repo)
		target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: repo, RepoIdentity: root + "/.git", PrimaryRoot: root, EnvironmentRoot: env, WorktreeRoot: path, Number: number}}
	}
	scope, _ := launcher.ResolveRepoScope(path)
	address := couchcore.ThreadAddress{RepoScope: scope.Key, Tag: couchcore.ThreadTag(tag)}
	if tag == "" {
		address = couchcore.ThreadAddress{}
	}
	if number == 0 {
		target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: address}
	}
	key, _ := target.RowKey()
	return couchcore.ActionableThreadSummary{Target: target, RowKey: key, Address: address, StartingPath: path, WorkingPath: path, State: couchcore.ThreadLive}
}

func TestGroupedMenuOrderingRenderingAndRefreshRouting(t *testing.T) {
	primary := groupedRow("/workspace/pair", 0, "primary")
	one := groupedRow("/workspace/pair", 1, "one")
	two := groupedRow("/workspace/pair", 2, "")
	two.State, two.Reason = couchcore.ThreadUnusable, couchcore.ReasonUnreadable
	ten := groupedRow("/workspace/pair", 10, "ten")
	brain := groupedRow("/workspace/brain", 0, "brain")
	inventory := []couchcore.ActionableThreadSummary{ten, two, primary, brain, one}
	state := NewMenuState(inventory, primary.Address)
	var got []couchcore.ThreadRowKey
	for _, row := range VisibleMenuThreads(state) {
		got = append(got, menuRowKey(row))
	}
	want := []couchcore.ThreadRowKey{brain.RowKey, primary.RowKey, one.RowKey, two.RowKey, ten.RowKey}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visible order = %v, want %v", got, want)
	}
	// Navigation uses displayed order, starting at the first displayed member.
	for range 3 {
		state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	}
	if state.CurrentFrame().SelectedKey != two.RowKey {
		t.Fatalf("selection = %+v", state.CurrentFrame())
	}
	replacement := groupedRow("/workspace/pair", 2, "replacement")
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{replacement, ten, one, brain, primary}, Generation: 1})
	if state.CurrentFrame().SelectedKey != two.RowKey {
		t.Fatal("refresh lost durable slot selection")
	}
	view := RenderMenuView(state, 130, 20, time.Unix(1, 0), false)
	plain := string(ansi.Strip([]byte(view.Body)))
	for _, fragment := range []string{"  pair  /workspace/pair", "    pair:1  /workspace/worktree/pair-slot1/pair", "▸   pair:2  /workspace/worktree/pair-slot2/pair", "    pair:10  /workspace/worktree/pair-slot10/pair"} {
		if !strings.Contains(plain, fragment) {
			t.Errorf("missing %q in\n%s", fragment, plain)
		}
	}
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Args["tag"] != "replacement" || effects[0].Args["repo-scope"] != replacement.Address.RepoScope {
		t.Fatalf("wrong live activation: %+v", effects)
	}
	for _, extent := range view.Extents {
		if extent.RowKey != two.RowKey {
			continue
		}
		key, address, ok := view.PointToRow(extent.Start, 0)
		if !ok {
			t.Fatal("missing click target")
		}
		_, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, RowKey: key, Address: address})
		if len(effects) != 1 || effects[0].Args["tag"] != "replacement" {
			t.Fatalf("wrong clicked slot: %+v", effects)
		}
	}
}

func TestGroupedMenuCustomNameStillShowsWorkspaceAndFilters(t *testing.T) {
	row := groupedRow("/workspace/pair", 2, "two")
	row.Name = "bugfix"
	row.State = couchcore.ThreadParked
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.Frames[0].Filter = "bugfix"
	plain := string(ansi.Strip([]byte(RenderMenu(state, 120, 15, time.Unix(1, 0), false))))
	for _, part := range []string{"pair:2", row.WorkingPath, "bugfix", "parked"} {
		if !strings.Contains(plain, part) {
			t.Errorf("missing %q in %s", part, plain)
		}
	}
	state.Frames[0].Filter = "pair:2"
	if rows := VisibleMenuThreads(state); len(rows) != 1 || menuRowKey(rows[0]) != row.RowKey {
		t.Fatalf("workspace search: %+v", rows)
	}
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "open-slot" || effects[0].Args["path"] != row.WorkingPath {
		t.Fatalf("parked activation: %+v", effects)
	}
}
