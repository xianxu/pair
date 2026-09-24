package couchtty

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestGroupedStatusUsesSwitcherOrderAndVisibleAnchor(t *testing.T) {
	primary := groupedRow("/workspace/pair", 0, "primary")
	one := groupedRow("/workspace/pair", 1, "one")
	two := groupedRow("/workspace/pair", 2, "two")
	brain := groupedRow("/workspace/brain", 0, "brain")
	rows := []couchcore.ActionableThreadSummary{two, primary, brain, one}
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 120}), strings.NewReader(""))
	con.menu = NewMenuState(rows, primary.Address)
	con.order = []string{"two", "primary", "one", "brain"}
	con.panes = map[string]*pane{}
	for _, r := range rows {
		id := string(r.Address.Tag)
		con.panes[id] = &pane{tree: couchcore.Worktree(r.StartingPath), thread: r.Address, label: r.Label()}
	}
	con.active = "two"
	con.attention.Mark(one.Address, "ready")
	model := con.statusModelLocked()
	var addresses []couchcore.ThreadAddress
	for _, a := range model.Actors {
		addresses = append(addresses, a.Thread)
	}
	want := []couchcore.ThreadAddress{brain.Address, primary.Address, one.Address, two.Address}
	if !reflect.DeepEqual(addresses, want) {
		t.Fatalf("status order=%v want %v", addresses, want)
	}
	plain := string(ansi.Strip([]byte(RenderStatusRow(120, model).Body)))
	if !strings.Contains(plain, "pair  :1  [:2]") {
		t.Fatalf("group tabs %q", plain)
	}
	if !model.Actors[2].Bell || !model.Actors[3].Active {
		t.Fatal("lost notification/focus")
	}
	delete(con.panes, "primary")
	con.order = []string{"two", "brain", "one"}
	// A parked primary belongs in the switcher, not the status bar.
	primary.State = couchcore.ThreadParked
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{two, primary, one, brain}, two.Address)
	model = con.statusModelLocked()
	plain = string(ansi.Strip([]byte(RenderStatusRow(120, model).Body)))
	if !strings.Contains(plain, "pair:1  [:2]") || len(model.Actors) != 3 {
		t.Fatalf("missing primary context %q", plain)
	}
	for width := 1; width < 80; width++ {
		view := RenderStatusRow(width, model)
		for _, span := range view.Chips {
			for col := span.Start; col < span.End; col++ {
				a, ok := view.ColumnToActor(col)
				if !ok || a != span.Thread || span.End > width {
					t.Fatalf("width%d span%+v: %+v", width, span, a)
				}
			}
		}
	}
}

func TestGroupedStatusSubdirectoryFallbackSurvivesInventory(t *testing.T) {
	row := groupedRow("/workspace/pair", 0, "primary")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 120}), strings.NewReader(""))
	con.order = []string{"primary"}
	con.panes = map[string]*pane{"primary": {tree: "/workspace/pair", thread: row.Address, label: "pair"}}
	before := con.statusModelLocked()
	row.StartingPath = "/workspace/pair/subdir"
	row.WorkingPath = "/workspace/pair/elsewhere"
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	after := con.statusModelLocked()
	if !reflect.DeepEqual(before.Actors, after.Actors) {
		t.Fatalf("fallback changed on refresh: before=%+v after=%+v", before.Actors, after.Actors)
	}
}

func TestGroupedStatusClickActivatesExactNativePane(t *testing.T) {
	con, writer, _, _ := newMouseFixture(t)
	one, two := groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")
	con.mu.Lock()
	con.panes["c1"].thread, con.panes["c1"].tree = one.Address, couchcore.Worktree(one.StartingPath)
	con.panes["c2"].thread, con.panes["c2"].tree = two.Address, couchcore.Worktree(two.StartingPath)
	con.order = []string{"c2", "c1"}
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{two, one}, one.Address)
	con.mu.Unlock()
	con.SetOperationDispatcher(con.ExecuteConsoleOperation)
	con.repaint()
	var column int
	waitFor(t, "grouped tab spans", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		for _, span := range con.statusChips {
			if span.Thread == two.Address {
				column = span.Start + 1
				return true
			}
		}
		return false
	})
	clickAt(t, writer, column, 24)
	waitFor(t, "clicked slot becomes active", func() bool {
		con.mu.Lock()
		defer con.mu.Unlock()
		return con.active == "c2" && con.focus == FocusActor("c2")
	})
}

func TestGroupedThreeWorkspaceActivationTrial(t *testing.T) {
	f := newFixture(t, 24, 120)
	root := filepath.Join(t.TempDir(), "pair")
	var rows []couchcore.ActionableThreadSummary
	for number := 0; number < 3; number++ {
		row := groupedRow(root, number, fmt.Sprintf("trial%d", number))
		if err := os.MkdirAll(row.StartingPath, 0755); err != nil {
			t.Fatal(err)
		}
		if number > 0 {
			if err := os.MkdirAll(filepath.Join(filepath.Dir(row.StartingPath), "ariadne"), 0755); err != nil {
				t.Fatal(err)
			}
		}
		rows = append(rows, row)
		child := ptychild.NewFakeChild(nil)
		t.Cleanup(func() { _ = child.Close() })
		id := fmt.Sprintf("trial%d", number)
		if err := f.con.installObservedThreadActor(context.Background(), id, couchcore.ActorID(id), row.Address, couchcore.Worktree(row.StartingPath), "pair", child, couchcore.ProcessIdentity{}, true); err != nil {
			t.Fatal(err)
		}
	}
	f.con.SetOperationDispatcher(f.con.ExecuteConsoleOperation)
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{rows[2], rows[0], rows[1]}, nil
	})
	waitFor(t, "three workspace inventory", func() bool { return len(f.con.menuSnapshot().Inventory) == 3 })
	defer func() {
		if t.Failed() {
			f.con.mu.Lock()
			defer f.con.mu.Unlock()
			t.Logf("status=%+v chips=%+v focus=%+v active=%s", f.con.statusModelLocked(), f.con.statusChips, f.con.focus, f.con.active)
		}
	}()
	for number, row := range rows {
		var column int
		waitFor(t, "workspace tab", func() bool {
			f.con.mu.Lock()
			defer f.con.mu.Unlock()
			for _, span := range f.con.statusChips {
				if span.Thread == row.Address {
					column = span.Start + 1
					return true
				}
			}
			return false
		})
		clickAt(t, f.stdin, column, 24)
		want := fmt.Sprintf("trial%d", number)
		waitFor(t, "exact workspace activation", func() bool {
			f.con.mu.Lock()
			defer f.con.mu.Unlock()
			return f.con.active == want && f.con.panes[want].tree == couchcore.Worktree(row.StartingPath) && f.con.menu.InFlight.Operation == "" && reflect.DeepEqual(f.con.statusChips, RenderStatusRow(int(f.con.size.Cols), f.con.statusModelLocked()).Chips)
		})
	}
	for _, row := range VisibleMenuThreads(f.con.menuSnapshot()) {
		if strings.Contains(row.WorkingPath, "ariadne") {
			t.Fatal("dependency gained its own row")
		}
	}
}
