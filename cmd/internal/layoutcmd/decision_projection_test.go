package layoutcmd

import (
	"bytes"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"path/filepath"
	"strings"
	"testing"
)

type unregisteredAdmissionRuntime struct{ admissionRuntime }

func (r unregisteredAdmissionRuntime) TerminalPaneIDs() ([]string, error) { return nil, nil }

func TestToggleConfirmationComparesDecisionFacts(t *testing.T) {
	for _, mode := range []string{"cosmetic-title", "registered-title", "unregistered-title", "unregistered-command", "id", "tab", "state", "order", "template", "foreground"} {
		t.Run(mode, func(t *testing.T) {
			f := zellijtest.New(t)
			for _, e := range f.Env() {
				k, v, _ := strings.Cut(e, "=")
				t.Setenv(k, v)
			}
			t.Setenv("PAIR_PAIR_WRAP_PID_PATH", filepath.Join(f.Root(), "wrapper.pid"))
			draft := zellijtest.DefaultPane(2, "nvim -u /pair/nvim/init.lua")
			draft["terminal_command"] = "nvim -u /pair/nvim/init.lua"
			terminal := zellijtest.DefaultPane(3, "pair term")
			terminal["terminal_command"] = "pair term"
			terminal["title"] = "terminal"
			terminal["is_focused"] = false
			if mode == "registered-title" || mode == "unregistered-title" {
				terminal["terminal_command"] = ""
			}
			f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{draft, terminal}, Tab: zellijtest.DefaultTab()}}})
			switch mode {
			case "cosmetic-title", "registered-title", "unregistered-title":
				terminal["title"] = "shell"
			case "id":
				terminal["id"] = 4
			case "tab":
				terminal["tab_id"] = 9
			case "state":
				terminal["is_focused"] = true
			case "template":
				terminal["terminal_command"] = "pair term --changed"
			case "foreground":
				terminal["pane_command"] = "vim"
				terminal["pane_cwd"] = "/different"
			}
			rows := []zellijtest.Pane{draft, terminal}
			if mode == "order" {
				rows = []zellijtest.Pane{terminal, draft}
			}
			b, err := json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
			reply := string(b)
			f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &reply})
			w := newFullscreenWorld()
			w.record = "unchanged"
			var rt FullscreenRuntime = admissionRuntime{world: w}
			if mode == "unregistered-title" || mode == "unregistered-command" {
				rt = unregisteredAdmissionRuntime{admissionRuntime{world: w}}
			}
			var diagnostic bytes.Buffer
			code := RunToggleFocused(nil, rt, &diagnostic)
			admitted := mode == "cosmetic-title" || mode == "registered-title" || mode == "foreground"
			if mode == "unregistered-title" || mode == "unregistered-command" {
				if code != 0 || len(f.Effects()) != 0 || w.record != "unchanged" {
					t.Fatalf("unregistered pane admitted: code=%d effects=%v record=%q diagnostic=%s", code, f.Effects(), w.record, diagnostic.String())
				}
			} else if admitted {
				if code != 0 || len(f.Effects()) == 0 {
					t.Fatalf("expected admission: code=%d effects=%v diagnostic=%s", code, f.Effects(), diagnostic.String())
				}
			} else if code == 0 || len(f.Effects()) != 0 || w.record != "unchanged" {
				t.Fatalf("expected refusal: code=%d effects=%v record=%q diagnostic=%s", code, f.Effects(), w.record, diagnostic.String())
			}
		})
	}
}
