package layoutcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"path/filepath"
	"strings"
	"testing"
)

type admissionRuntime struct {
	OSRuntime
	world *fullscreenWorld
}

func (r admissionRuntime) CurrentPaneID() string                              { return "2" }
func (r admissionRuntime) FullscreenStore() workbenchshortcut.FullscreenStore { return r.world }
func (r admissionRuntime) LastTerminalPaneID() (string, error)                { return "3", nil }
func (r admissionRuntime) TerminalPaneIDs() ([]string, error)                 { return []string{"3"}, nil }
func (r admissionRuntime) NudgeWrap() error                                   { return nil }

func TestLayoutReplyAdmissionRefusesBeforeEffects(t *testing.T) {
	for _, mode := range []string{"healthy", "wrong-first", "wrong-confirmation", "target-change", "wrong-tab"} {
		t.Run(mode, func(t *testing.T) {
			f := zellijtest.New(t)
			for _, e := range f.Env() {
				k, v, _ := strings.Cut(e, "=")
				t.Setenv(k, v)
			}
			draft := zellijtest.DefaultPane(2, "nvim -u /pair/nvim/init.lua")
			draft["terminal_command"] = "nvim -u /pair/nvim/init.lua"
			draft["title"] = "draft"
			terminal := zellijtest.DefaultPane(3, "pair term")
			terminal["terminal_command"] = "pair term"
			terminal["is_focused"] = false
			f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{draft, terminal}, Tab: zellijtest.DefaultTab()}}})
			t.Setenv("PAIR_PAIR_WRAP_PID_PATH", filepath.Join(f.Root(), "wrapper.pid"))
			w := newFullscreenWorld()
			w.record = "unchanged"
			if mode != "healthy" {
				ordinal := 2
				reply := `{"tab_id":0}`
				if mode == "wrong-first" {
					ordinal = 1
				}
				if mode == "target-change" {
					terminal["id"] = 4
					b, _ := json.Marshal([]zellijtest.Pane{draft, terminal})
					reply = string(b)
				}
				if mode == "wrong-tab" {
					ordinal = 3
					tab := zellijtest.DefaultTab()
					tab["tab_id"] = 9
					b, _ := json.Marshal(tab)
					reply = string(b)
				}
				f.SetFaults(zellijtest.Fault{Ordinal: ordinal, Reply: &reply})
			}
			var diagnostic bytes.Buffer
			code := RunToggleFocused(nil, admissionRuntime{world: w}, &diagnostic)
			if mode == "healthy" {
				if code != 0 {
					t.Fatalf("healthy refused: %s", diagnostic.String())
				}
				if len(f.Effects()) == 0 {
					t.Fatal("healthy action had no effects")
				}
				return
			}
			if code == 0 || len(f.Effects()) != 0 || w.record != "unchanged" {
				t.Fatalf("mode=%s code=%d effects=%v record=%q diagnostic=%s", mode, code, f.Effects(), w.record, diagnostic.String())
			}
		})
	}
}

func TestLayoutReplyRefusalRetainsCompletedProgress(t *testing.T) {
	for _, ordinal := range []int{4, 5} {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			f := zellijtest.New(t)
			for _, e := range f.Env() {
				k, v, _ := strings.Cut(e, "=")
				t.Setenv(k, v)
			}
			t.Setenv("PAIR_PAIR_WRAP_PID_PATH", filepath.Join(f.Root(), "wrapper.pid"))
			draft := zellijtest.DefaultPane(2, "nvim")
			draft["title"] = "draft"
			draft["terminal_command"] = "nvim -u /pair/nvim/init.lua"
			terminal := zellijtest.DefaultPane(3, "pair term")
			terminal["terminal_command"] = "pair term"
			terminal["is_floating"] = true
			f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{draft, terminal}, Tab: zellijtest.DefaultTab(), FloatingVisible: true}}})
			wrong := `{"unrelated":true}`
			f.SetFaults(zellijtest.Fault{Ordinal: ordinal, Reply: &wrong})
			w := newFullscreenWorld()
			w.record = "2 swap=base"
			var diagnostic bytes.Buffer
			if code := RunToggleFocused(nil, admissionRuntime{world: w}, &diagnostic); code != 1 {
				t.Fatalf("code=%d diagnostic=%s", code, diagnostic.String())
			}
			effects := f.Effects()
			if len(effects) != 1 || effects[0].Action != "toggle-pane-embed-or-floating" || w.record != "2 swap=base" {
				t.Fatalf("effects=%v record=%q", effects, w.record)
			}
			if f.Model().Sessions["test"].Panes[1]["is_floating"] != false {
				t.Fatal("completed embed was rolled back")
			}
		})
	}
}
