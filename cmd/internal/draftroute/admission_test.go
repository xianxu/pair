package draftroute

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

type nativeRouteRuntime struct {
	cached string
	caller string
}

func (r nativeRouteRuntime) CachedDraftPaneID() (string, bool) { return r.cached, r.cached != "" }
func (r nativeRouteRuntime) CurrentPaneID() string             { return r.caller }
func (nativeRouteRuntime) ListPanesJSON() ([]byte, error) {
	return layoutcmd.OSRuntime{}.ListPanesJSON()
}
func (nativeRouteRuntime) ListPanesJSONContext(ctx context.Context) ([]byte, error) {
	return layoutcmd.OSRuntime{}.ListPanesJSONContext(ctx)
}
func (nativeRouteRuntime) RunZellijAction(args ...string) error {
	return layoutcmd.OSRuntime{}.RunZellijAction(args...)
}

func draftRoutingModel(draftID int) zellijtest.Model {
	pane := func(id, tab int, command string) zellijtest.Pane {
		return zellijtest.Pane{
			"id": id, "tab_id": tab, "tab_position": tab, "is_plugin": false, "is_focused": id == 4,
			"is_floating": false, "is_fullscreen": false, "pane_x": 0, "pane_y": 0, "pane_columns": 80, "pane_rows": 20, "terminal_command": command,
		}
	}
	return zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{
		pane(4, 1, "pair term"), pane(draftID, 1, "nvim -u /pair/nvim/init.lua /data/draft.md"),
		pane(8, 2, "nvim -u /pair/nvim/init.lua /data/foreign-draft.md"),
	}}}}
}

func TestRouteLuaRequiresConfirmedScopedUniqueDraft(t *testing.T) {
	for _, scenario := range []string{"healthy", "one-crossed-first", "one-crossed-confirmation", "wrong-kind-first", "wrong-kind-confirmation", "displaced-first", "changed-target", "missing-caller", "missing-draft", "ambiguous-draft", "changed-tab"} {
		t.Run(scenario, func(t *testing.T) {
			f := zellijtest.New(t)
			for _, entry := range f.Env() {
				key, value, _ := strings.Cut(entry, "=")
				t.Setenv(key, value)
			}
			model := draftRoutingModel(2)
			wrong := "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
			switch scenario {
			case "wrong-kind-first":
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &wrong}, zellijtest.Fault{Ordinal: 2, Reply: &wrong}, zellijtest.Fault{Ordinal: 3, Reply: &wrong})
			case "wrong-kind-confirmation":
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &wrong}, zellijtest.Fault{Ordinal: 3, Reply: &wrong}, zellijtest.Fault{Ordinal: 4, Reply: &wrong})
			case "one-crossed-first":
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &wrong})
			case "one-crossed-confirmation":
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &wrong})
			case "displaced-first":
				body, err := json.Marshal(draftRoutingModel(3).Sessions["test"].Panes)
				if err != nil {
					t.Fatal(err)
				}
				reply := string(body)
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &reply})
			case "changed-target":
				changed := draftRoutingModel(3)
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &changed})
			case "changed-tab":
				changed := draftRoutingModel(2)
				changed.Sessions["test"].Panes[0]["tab_id"] = 2
				changed.Sessions["test"].Panes[0]["tab_position"] = 2
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &changed})
			case "missing-caller":
				session := model.Sessions["test"]
				session.Panes = session.Panes[1:]
				model.Sessions["test"] = session
			case "missing-draft":
				session := model.Sessions["test"]
				session.Panes = append(session.Panes[:1], session.Panes[2:]...)
				model.Sessions["test"] = session
			case "ambiguous-draft":
				session := model.Sessions["test"]
				another := draftRoutingModel(3).Sessions["test"].Panes[1]
				session.Panes = append(session.Panes, another)
				model.Sessions["test"] = session
			}
			f.SetModel(model)
			err := RouteLua(nativeRouteRuntime{caller: "4"}, "PairConfirmQuit", true)
			if scenario == "healthy" || strings.HasPrefix(scenario, "one-crossed-") {
				if err != nil {
					t.Fatal(err)
				}
				if effects := f.Effects(); len(effects) != 5 {
					t.Fatalf("effects=%+v", effects)
				} else {
					for _, effect := range effects {
						if effect.Target != "terminal_2" {
							t.Fatalf("unscoped target=%+v", effect)
						}
					}
				}
				reads := 0
				for _, query := range f.Queries() {
					if strings.Contains(strings.Join(query.Args, " "), "list-panes") {
						reads++
					}
				}
				wantReads := 2
				if strings.HasPrefix(scenario, "one-crossed-") {
					wantReads = 3
				}
				if reads != wantReads {
					t.Fatalf("queries=%+v; want exactly %d reads", f.Queries(), wantReads)
				}
			} else {
				wantReads := 2
				switch scenario {
				case "wrong-kind-first":
					wantReads = 3
				case "wrong-kind-confirmation":
					wantReads = 4
				case "missing-caller", "missing-draft", "ambiguous-draft":
					wantReads = 1
				}
				if len(f.Queries()) != wantReads {
					t.Fatalf("refusal queries=%+v; want exactly %d reads", f.Queries(), wantReads)
				}
				if err == nil || len(f.Effects()) != 0 {
					t.Fatalf("uncertain route error=%v effects=%+v", err, f.Effects())
				}
				for _, query := range f.Queries() {
					if !strings.Contains(strings.Join(query.Args, " "), "list-panes") {
						t.Fatalf("uncertain route attempted mutation: %+v", query)
					}
				}
			}
		})
	}
}
