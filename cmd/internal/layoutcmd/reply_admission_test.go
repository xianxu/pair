package layoutcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestLayoutReturnConfirmsDecisionFactsWithoutGeometry(t *testing.T) {
	for _, mode := range []string{"geometry-only", "target-change", "tab-change"} {
		t.Run(mode, func(t *testing.T) {
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
			terminal["pane_columns"] = 100
			if mode == "target-change" {
				terminal["id"] = 4
			}
			if mode == "tab-change" {
				terminal["tab_id"] = 9
			}
			b, _ := json.Marshal([]zellijtest.Pane{draft, terminal})
			reply := string(b)
			f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &reply})
			w := newFullscreenWorld()
			w.record = "2 swap=base"
			var diagnostic bytes.Buffer
			code := RunToggleFocused(nil, admissionRuntime{world: w}, &diagnostic)
			if mode == "geometry-only" {
				if code != 0 || w.record != "" || len(f.Effects()) != 2 {
					t.Fatalf("healthy geometry change code=%d record=%q effects=%v diagnostic=%s", code, w.record, f.Effects(), diagnostic.String())
				}
				return
			}
			if code != 1 || len(f.Effects()) != 0 || w.record != "2 swap=base" {
				t.Fatalf("mode=%s code=%d record=%q effects=%v", mode, code, w.record, f.Effects())
			}
		})
	}
}

func TestLayoutReplyRefusalUsesIndependentBroker(t *testing.T) {
	assertLayoutRefusalUsesIndependentBroker(t, `{"wrong":"kind"}`)
}

func TestLayoutMalformedNativeReplyRefusesVisiblyWithoutEffects(t *testing.T) {
	for _, body := range []struct{ name, raw string }{
		{"empty", ""}, {"truncated", `[{"id":3`},
		{"warning-prefix", "warning: native diagnostic\n[]"},
		{"plain-wrong-reply", "Connected"},
	} {
		t.Run(body.name, func(t *testing.T) { assertLayoutRefusalUsesIndependentBroker(t, body.raw) })
	}
}

func assertLayoutRefusalUsesIndependentBroker(t *testing.T, wrong string) {
	t.Helper()
	f := zellijtest.New(t)
	for _, e := range f.Env() {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	binding := filepath.Join(f.Root(), "wrapper.pid")
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {}}})
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &wrong}, zellijtest.Fault{Ordinal: 2, Reply: &wrong}, zellijtest.Fault{Ordinal: 3, Reply: &wrong})
	wantQueries := 1
	if wrong == "" {
		wantQueries = 3
	}
	w := newFullscreenWorld()
	w.record = "preserved"
	var diagnostic bytes.Buffer
	if code := RunToggleFocused(nil, admissionRuntime{world: w}, &diagnostic); code != 1 {
		t.Fatalf("code=%d", code)
	}
	select {
	case msg := <-broker.Messages():
		if msg != "Pair action refused: terminal context could not be confirmed." {
			t.Fatalf("message=%q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("refusal did not reach independent wrapper transport")
	}
	if len(f.Effects()) != 0 || w.record != "preserved" || len(f.Queries()) != wantQueries {
		t.Fatalf("effects=%v record=%q queries=%v", f.Effects(), w.record, f.Queries())
	}
}

type splitAdmissionRuntime struct{ admissionRuntime }

func (r splitAdmissionRuntime) TerminalPaneIDs() ([]string, error) {
	return []string{"3", "4"}, nil
}

func TestLayoutAmbiguousFirstReplyStopsBeforeConfirmation(t *testing.T) {
	f := zellijtest.New(t)
	for _, e := range f.Env() {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", filepath.Join(f.Root(), "wrapper.pid"))
	draft := zellijtest.DefaultPane(2, "nvim")
	draft["title"] = "draft"
	draft["terminal_command"] = "nvim -u /pair/nvim/init.lua"
	a := zellijtest.DefaultPane(3, "pair term")
	a["terminal_command"] = "pair term"
	a["is_floating"] = true
	b := zellijtest.DefaultPane(4, "pair term")
	b["terminal_command"] = "pair term"
	b["is_floating"] = true
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{draft, a, b}, Tab: zellijtest.DefaultTab()}}})
	w := newFullscreenWorld()
	w.record = "preserved"
	var diagnostic bytes.Buffer
	if code := RunToggleFocused(nil, splitAdmissionRuntime{admissionRuntime{world: w}}, &diagnostic); code != 1 || len(f.Queries()) != 1 || len(f.Effects()) != 0 || w.record != "preserved" {
		t.Fatalf("code=%d queries=%v effects=%v record=%q", code, f.Queries(), f.Effects(), w.record)
	}
}

func TestLayoutRefusalRetainsPresenterFailureWithDiscardedStderr(t *testing.T) {
	w := newFullscreenWorld()
	w.fail = "save"
	w.notifyError = errors.New("private broker unavailable")
	if code := RunToggleFocused(nil, w, io.Discard); code != 1 {
		t.Fatalf("code=%d", code)
	}
	if len(w.logs) != 2 || !strings.Contains(w.logs[1], "refusal notification failed") || !strings.Contains(w.logs[1], "private broker unavailable") {
		t.Fatalf("retained logs=%v", w.logs)
	}
	if len(w.notices) != 1 {
		t.Fatalf("notices=%v", w.notices)
	}
}
