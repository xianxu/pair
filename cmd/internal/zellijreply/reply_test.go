package zellijreply

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSharedContracts(t *testing.T) {
	b, err := os.ReadFile("../../../tests/fixtures/zellij-replies.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Name, Kind, Input string
			Requirements      PaneRequirements
			Valid             bool
		}
	}
	if err := json.Unmarshal(b, &corpus); err != nil {
		t.Fatal(err)
	}
	for _, c := range corpus.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var err error
			switch c.Kind {
			case "clients":
				_, err = ParseClients([]byte(c.Input))
			case "panes":
				_, err = ParsePanes([]byte(c.Input), c.Requirements)
			case "tab":
				_, err = ParseTab([]byte(c.Input))
			default:
				t.Fatal(c.Kind)
			}
			if (err == nil) != c.Valid {
				t.Fatalf("valid=%v error=%v", c.Valid, err)
			}
		})
	}
}

func TestParsedFacts(t *testing.T) {
	clients, err := ParseClients([]byte(" CLIENT_ID  ZELLIJ_PANE_ID RUNNING_COMMAND\n3 terminal_7 nvim file with spaces\n"))
	if err != nil || len(clients) != 1 || clients[0].ID != 3 || clients[0].PaneID != "terminal_7" || clients[0].RunningCommand != "nvim file with spaces" {
		t.Fatalf("%+v %v", clients, err)
	}
	panes, err := ParsePanes([]byte(`[{"id":7,"is_plugin":false,"terminal_command":null,"tab_id":2,"tab_position":1}]`), PaneRequirements{Command: true, Tab: true})
	if err != nil || len(panes) != 1 || panes[0].ID != "7" || panes[0].TabID == nil || *panes[0].TabID != 2 {
		t.Fatalf("%+v %v", panes, err)
	}
	tab, err := ParseTab([]byte(`{"active_swap_layout_name":null,"is_swap_layout_dirty":false,"tab_id":2,"position":1}`))
	if err != nil || tab.ActiveSwapLayoutName != nil || tab.ID == nil || *tab.ID != 2 || tab.Position == nil || *tab.Position != 1 {
		t.Fatalf("%+v %v", tab, err)
	}
}

func TestConfirmOrdering(t *testing.T) {
	for _, mode := range []string{"valid", "unknown-first", "mismatch", "cancel-before", "cancel-first", "cancel-second", "read-error", "invalid-second", "project-cancel", "project-cancel-second", "read-error-second"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel-before" {
				cancel()
			}
			reads, projections := 0, 0
			got, err := Confirm(ctx, func(context.Context) ([]byte, error) {
				reads++
				if mode == "cancel-first" || (mode == "cancel-second" && reads == 2) {
					cancel()
				}
				if mode == "read-error" || (mode == "read-error-second" && reads == 2) {
					return nil, errors.New("read")
				}
				if reads == 2 && (mode == "mismatch" || mode == "invalid-second") {
					return []byte("other"), nil
				}
				return []byte("ok"), nil
			}, func(b []byte) (string, error) {
				projections++
				if mode == "project-cancel" || (mode == "project-cancel-second" && projections == 2) {
					cancel()
				}
				if mode == "unknown-first" || (mode == "invalid-second" && string(b) == "other") {
					return "", errors.New("unknown")
				}
				return string(b), nil
			})
			wantReads := 2
			switch mode {
			case "cancel-before":
				wantReads = 0
			case "unknown-first", "cancel-first", "read-error", "project-cancel":
				wantReads = 1
			}
			if reads != wantReads {
				t.Fatalf("reads=%d want %d", reads, wantReads)
			}
			if mode == "valid" {
				if err != nil || string(got) != "ok" || projections != 2 {
					t.Fatalf("%s %v projections=%d", got, err, projections)
				}
			} else if err == nil || got != nil {
				t.Fatalf("refusal returned %s %v", got, err)
			}
			if mode == "mismatch" && !errors.Is(err, ErrMismatch) {
				t.Fatal(err)
			}
			if mode == "cancel-first" && projections != 0 {
				t.Fatal("projected after cancelled read")
			}
		})
	}
}
func TestAdmissionContext(t *testing.T) {
	ctx, cancel := AdmissionContext(context.Background())
	defer cancel()
	d, ok := ctx.Deadline()
	if !ok || time.Until(d) > 5*time.Second || time.Until(d) < 4*time.Second {
		t.Fatal(d)
	}
	parent, stop := context.WithTimeout(context.Background(), time.Millisecond)
	defer stop()
	child, done := AdmissionContext(parent)
	defer done()
	pd, _ := parent.Deadline()
	cd, _ := child.Deadline()
	if pd != cd {
		t.Fatal("extended deadline")
	}
}

func FuzzParseClients(f *testing.F) {
	f.Add([]byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n0 terminal_1 sh\n"))
	f.Add([]byte("{}"))
	f.Fuzz(func(t *testing.T, b []byte) {
		cs, err := ParseClients(b)
		if err == nil {
			seen := map[int]bool{}
			for _, c := range cs {
				if c.ID < 0 || seen[c.ID] || !(strings.HasPrefix(c.PaneID, "terminal_") || strings.HasPrefix(c.PaneID, "plugin_")) {
					t.Fatalf("invalid admitted client %+v", c)
				}
				seen[c.ID] = true
			}
		}
	})
}
func FuzzParsePanes(f *testing.F) {
	f.Add([]byte(`[{"id":0,"is_plugin":false}]`))
	f.Add([]byte(`{"0":[{"pane_id":"terminal_7","is_plugin":false}]}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		ps, err := ParsePanes(b, PaneRequirements{})
		if err == nil {
			seen := map[string]bool{}
			for _, p := range ps {
				key := p.ID
				if p.IsPlugin {
					key = "p" + key
				}
				if p.ID == "" || seen[key] {
					t.Fatal("invalid admitted identity")
				}
				seen[key] = true
			}
		}
	})
}
func FuzzParseTab(f *testing.F) {
	f.Add([]byte(`{"active_swap_layout_name":null,"is_swap_layout_dirty":false,"tab_id":0,"position":0}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = ParseTab(b) })
}

func TestTolerantPaneTabFieldsRemainAdditive(t *testing.T) {
	panes := zellijpane.Parse([]byte(`[{"id":7,"is_plugin":false,"tab_id":2,"tab_position":1},{"id":8,"is_plugin":false}]`))
	if len(panes) != 2 || panes[0].TabID == nil || *panes[0].TabID != 2 || panes[0].TabPosition == nil || *panes[0].TabPosition != 1 || panes[1].TabID != nil || panes[1].TabPosition != nil {
		t.Fatalf("tab identity lost or fabricated: %+v", panes)
	}
}

func TestConfirmReturnsSecondProjectedSnapshot(t *testing.T) {
	calls := 0
	got, err := Confirm(context.Background(), func(context.Context) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte("target:old-title"), nil
		}
		return []byte("target:new-title"), nil
	}, func(b []byte) (string, error) { return strings.SplitN(string(b), ":", 2)[0], nil })
	if err != nil || string(got) != "target:new-title" || calls != 2 {
		t.Fatalf("snapshot=%s calls=%d err=%v", got, calls, err)
	}
}
