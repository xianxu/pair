package couchtty

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestHubRepositorySlotsLeadBothViews(t *testing.T) {
	claims, err := couchcore.ParseHubClaims([]byte("ops-echo\necho\n"))
	if err != nil {
		t.Fatal(err)
	}
	ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{
		{Root: "/src/ops", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/ops"}},
		{Root: "/src/ops-echo", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/ops-echo"}, Claims: claims},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rows []couchcore.ActionableThreadSummary
	for _, repo := range []string{"ops", "ariadne", "brain", "pair", "ops-echo", "echo"} {
		for _, n := range []int{0, 1, 2, 10} {
			rows = append(rows, groupedRow("/src/"+repo, n, repo+string(rune('a'+n))))
		}
	}
	want := strings.Fields("ops ops:1 ops:2 ops:10 ariadne ariadne:1 ariadne:2 ariadne:10 brain brain:1 brain:2 brain:10 pair pair:1 pair:2 pair:10 ops-echo ops-echo:1 ops-echo:2 ops-echo:10 echo echo:1 echo:2 echo:10")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 60, Cols: 240}), nil)
	t.Cleanup(con.Stop)
	con.panes = map[string]*pane{}
	for _, row := range rows {
		id := string(row.Address.Tag)
		con.order = append(con.order, id)
		con.panes[id] = &pane{tree: couchcore.Worktree(row.StartingPath), thread: row.Address}
	}
	for seed := int64(0); seed < 8; seed++ {
		rand.New(rand.NewSource(seed)).Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		state := NewMenuState(rows, couchcore.ThreadAddress{})
		state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: ownership})
		labels, _, _ := presentationLabels(PresentThreads(menuRows(state), nil, state.Hub, state.HubOwnership))
		if !reflect.DeepEqual(labels, want) {
			t.Fatalf("seed %d shared order %v, want %v", seed, labels, want)
		}
		// Inventory order drives switcher navigation, independently of rendering.
		for i, row := range state.Inventory {
			if label := PresentThreads([]couchcore.ActionableThreadSummary{row}, nil, "")[0].Label; label != want[i] {
				t.Fatalf("switcher row %d: %q, want %q", i, label, want[i])
			}
		}
		lines, _ := renderRootMenuFrame(state, state.CurrentFrame(), 240, 60, time.Now(), false)
		body := string(ansi.Strip([]byte(strings.Join(lines, "\n"))))
		previous := -1
		for _, label := range want {
			at := strings.Index(body, " "+label+"  ")
			if at <= previous {
				t.Fatalf("switcher label %q out of order:\n%s", label, body)
			}
			previous = at
		}
		con.menu = state
		model := con.statusModelLocked()
		var tabLabels []string
		for _, actor := range model.Actors {
			tabLabels = append(tabLabels, actor.Label)
		}
		if !reflect.DeepEqual(tabLabels, want) {
			t.Fatalf("tab order %v, want %v", tabLabels, want)
		}
		if got, want := rowPlain(RenderStatusRow(240, model)), "ops  :1  :2  :10  ariadne  :1  :2  :10  brain  :1  :2  :10  pair  :1  :2  :10 │ ops-echo  :1  :2  :10  echo  :1  :2  :10"; got != want {
			t.Fatalf("tab bar %q, want %q", got, want)
		}
	}
}
