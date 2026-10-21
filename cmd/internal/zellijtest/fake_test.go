package zellijtest

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func sample() Model {
	return Model{Sessions: map[string]Session{"test": {Panes: []Pane{DefaultPane(0, "sh"), DefaultPane(2, "nvim")}, Clients: []Client{{ID: 7, PaneID: "terminal_0", RunningCommand: "sh"}}, Tab: DefaultTab(), Layouts: []string{"base", "wide"}}}}
}

func TestFakeVersionIsObservedWithoutEffects(t *testing.T) {
	f := New(t)
	if got := invoke(t, f, "--version"); got != "zellij 0.45.1\n" {
		t.Fatalf("version %q", got)
	}
	queries := f.Queries()
	if len(queries) != 1 || len(queries[0].Args) != 1 || queries[0].Args[0] != "--version" || len(f.Effects()) != 0 {
		t.Fatalf("version bypassed observation or caused effects: %+v %+v", queries, f.Effects())
	}
}
func invoke(t *testing.T, f *Fixture, args ...string) string {
	t.Helper()
	c := exec.Command(f.Path(), args...)
	c.Env = f.Environment()
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("%v: %v: %s", args, e, b)
	}
	return string(b)
}
func TestFakeModelTransitionsAndExactTargets(t *testing.T) {
	f := New(t)
	f.SetModel(sample())
	if got := invoke(t, f, "list-sessions", "--short", "--no-formatting"); got != "test\n" {
		t.Fatalf("sessions %q", got)
	}
	if got := invoke(t, f, "action", "list-clients"); !strings.Contains(got, "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n7 terminal_0 sh") {
		t.Fatalf("clients %q", got)
	}
	invoke(t, f, "action", "focus-pane-id", "2")
	invoke(t, f, "action", "write-chars", "--pane-id", "2", "hello")
	invoke(t, f, "action", "toggle-fullscreen")
	invoke(t, f, "action", "next-swap-layout")
	var panes []Pane
	if err := json.Unmarshal([]byte(invoke(t, f, "action", "list-panes", "--json", "--command", "--geometry")), &panes); err != nil {
		t.Fatal(err)
	}
	if panes[0]["is_focused"] != false || panes[1]["is_focused"] != true || panes[1]["is_fullscreen"] != true || panes[1]["input"] != "hello" {
		t.Fatalf("panes %+v", panes)
	}
	if got := f.Model().Sessions["test"].Tab["active_swap_layout_name"]; got != "wide" {
		t.Fatalf("layout %v", got)
	}
	effects := f.Effects()
	if len(effects) != 4 || effects[0].Target != "terminal_2" || effects[1].Target != "terminal_2" {
		t.Fatalf("effects %+v", effects)
	}
	if len(f.Queries()) != 7 {
		t.Fatalf("queries %+v", f.Queries())
	}
}
func TestFakeFaultsDoNotInventEffects(t *testing.T) {
	f := New(t)
	f.SetModel(sample())
	wrong := "{}"
	empty := ""
	changed := sample()
	s := changed.Sessions["test"]
	s.Panes[0]["pane_command"] = "other"
	changed.Sessions["test"] = s
	f.SetFaults(Fault{Ordinal: 1, Reply: &wrong}, Fault{Ordinal: 2, Reply: &empty}, Fault{Ordinal: 3, Error: "disconnected"}, Fault{Ordinal: 4, Model: &changed})
	if got := invoke(t, f, "action", "list-clients"); got != "{}" {
		t.Fatal(got)
	}
	if got := invoke(t, f, "action", "focus-pane-id", "2"); got != "" {
		t.Fatal(got)
	}
	c := exec.Command(f.Path(), "action", "list-panes", "--json")
	c.Env = f.Environment()
	if err := c.Run(); err == nil {
		t.Fatal("disconnect succeeded")
	}
	if got := invoke(t, f, "action", "list-panes", "--json"); !strings.Contains(got, "other") {
		t.Fatal(got)
	}
	if len(f.Effects()) != 0 || len(f.Queries()) != 4 {
		t.Fatalf("effects %v queries %v", f.Effects(), f.Queries())
	}
}
func TestFakeConcurrentCLIsKeepAllEffects(t *testing.T) {
	f := New(t)
	f.SetModel(sample())
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			invoke(t, f, "--session", "test", "action", "write-chars", "--pane-id", "0", "x")
		}()
	}
	wg.Wait()
	if len(f.Effects()) != 12 || len(f.Queries()) != 12 || f.Model().Sessions["test"].Panes[0]["input"] != "xxxxxxxxxxxx" {
		t.Fatalf("state %+v effects %v", f.Model(), f.Effects())
	}
}

func TestFakeDefaultsExposeNativeTabIdentity(t *testing.T) {
	f := New(t)
	f.SetModel(sample())
	var panes []Pane
	if err := json.Unmarshal([]byte(invoke(t, f, "action", "list-panes", "--json")), &panes); err != nil {
		t.Fatal(err)
	}
	var tab Tab
	if err := json.Unmarshal([]byte(invoke(t, f, "action", "current-tab-info", "--json")), &tab); err != nil {
		t.Fatal(err)
	}
	if panes[0]["tab_id"] != float64(0) || tab["tab_id"] != float64(0) || panes[0]["tab_position"] != tab["position"] {
		t.Fatalf("pane %v tab %v", panes[0], tab)
	}
}

func TestFakeSessionAndPluginIdentity(t *testing.T) {
	f := New(t)
	model := sample()
	s := model.Sessions["test"]
	plugin := DefaultPane(0, "")
	plugin["is_plugin"] = true
	s.Panes = append(s.Panes, plugin)
	model.Sessions["test"] = s
	model.Sessions["other"] = Session{Panes: []Pane{DefaultPane(0, "other")}, Tab: DefaultTab()}
	f.SetModel(model)
	invoke(t, f, "--session", "test", "action", "write-chars", "--pane-id", "plugin_0", "plugin")
	invoke(t, f, "--session", "other", "action", "write-chars", "--pane-id", "0", "other")
	effects := f.Effects()
	if len(effects) != 2 || !effects[0].IsPlugin || effects[0].Target != "plugin_0" || effects[1].Session != "other" {
		t.Fatalf("effects %v", effects)
	}
	s = f.Model().Sessions["test"]
	if _, exists := s.Panes[0]["input"]; exists || s.Panes[2]["input"] != "plugin" {
		t.Fatalf("panes %v", s.Panes)
	}
}

func TestFakeVisibilityStateAndInvalidMutation(t *testing.T) {
	f := New(t)
	f.SetModel(sample())
	invoke(t, f, "action", "show-floating-panes")
	if got := invoke(t, f, "action", "are-floating-panes-visible"); got != "true\n" {
		t.Fatal(got)
	}
	invoke(t, f, "action", "hide-floating-panes")
	visibility := exec.Command(f.Path(), "action", "are-floating-panes-visible")
	visibility.Env = f.Environment()
	got, err := visibility.Output()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || string(got) != "false\n" {
		t.Fatalf("native false visibility: stdout=%q error=%v", got, err)
	}
	c := exec.Command(f.Path(), "action", "focus-pane-id", "99")
	c.Env = f.Environment()
	if err := c.Run(); err == nil {
		t.Fatal("absent target accepted")
	}
	if len(f.Effects()) != 2 {
		t.Fatalf("effects %v", f.Effects())
	}
}

func TestFakeEmptySessionHasNativeEmptyCollections(t *testing.T) {
	f := New(t)
	f.SetModel(Model{Sessions: map[string]Session{"test": {Tab: DefaultTab()}}})
	if got := invoke(t, f, "action", "list-clients"); got != "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" {
		t.Fatalf("clients %q", got)
	}
	if got := invoke(t, f, "action", "list-panes", "--json"); got != "[]\n" {
		t.Fatalf("panes %q", got)
	}
	if len(f.Queries()) != 2 {
		t.Fatalf("queries %v", f.Queries())
	}
}

func TestFakeSemanticKeysPreserveExactPaneAndChord(t *testing.T) {
	f := New(t)
	f.SetModel(sample())
	invoke(t, f, "action", "send-keys", "--pane-id", "2", "Alt Enter")
	effects := f.Effects()
	if len(effects) != 1 || effects[0].Target != "terminal_2" || effects[0].Action != "send-keys" || strings.Join(effects[0].Args, "|") != "action|send-keys|--pane-id|2|Alt Enter" {
		t.Fatalf("semantic key effect %+v", effects)
	}
	panes := f.Model().Sessions["test"].Panes
	keys, ok := panes[1]["keys"].([]any)
	if !ok || len(keys) != 1 || keys[0] != "Alt Enter" || panes[1]["input"] != nil || panes[0]["keys"] != nil {
		t.Fatalf("semantic chord became raw bytes or reached another pane: %+v", panes)
	}
}
