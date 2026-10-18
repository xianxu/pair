// Package zellijtest provides a private, stateful external Zellij CLI fixture.
package zellijtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

type Pane = map[string]any
type Tab = map[string]any
type Client struct {
	ID             int    `json:"id"`
	PaneID         string `json:"pane_id"`
	RunningCommand string `json:"running_command"`
}
type Session struct {
	Panes           []Pane   `json:"panes"`
	Clients         []Client `json:"clients"`
	Tab             Tab      `json:"tab"`
	FloatingVisible bool     `json:"floating_visible"`
	Generation      string   `json:"generation"`
	Layouts         []string `json:"layouts,omitempty"`
}
type Model struct {
	Sessions map[string]Session `json:"sessions"`
}

// Fault is applied at the global invocation ordinal, before normal dispatch.
// Reply (including an empty string) replaces stdout; Error exits unsuccessfully.
// Model replaces state before this invocation; it can be combined with Reply.
type Fault struct {
	Ordinal int     `json:"ordinal"`
	Reply   *string `json:"reply,omitempty"`
	Error   string  `json:"error,omitempty"`
	Model   *Model  `json:"model,omitempty"`
}
type Query struct {
	Ordinal int      `json:"ordinal"`
	Args    []string `json:"args"`
	Session string   `json:"session"`
}
type Effect struct {
	Query
	Action   string `json:"action"`
	Target   string `json:"target"`
	IsPlugin bool   `json:"is_plugin"`
}
type state struct {
	Model   Model    `json:"model"`
	Faults  []Fault  `json:"faults"`
	Queries []Query  `json:"queries"`
	Effects []Effect `json:"effects"`
	Ordinal int      `json:"ordinal"`
}
type Fixture struct {
	t                testing.TB
	root, executable string
}

func New(t testing.TB) *Fixture {
	t.Helper()
	f := &Fixture{t: t, root: t.TempDir()}
	for _, name := range []string{"bin", "home", "cache", "runtime", "config"} {
		if err := os.Mkdir(filepath.Join(f.root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Work with trimpath builds too: Go package tests start inside the repository.
	for {
		if _, err = os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("cannot find repository go.mod")
		}
		cwd = parent
	}
	script, err := os.ReadFile(filepath.Join(cwd, "tests/lib/fake-zellij.py"))
	if err != nil {
		t.Fatal(err)
	}
	f.executable = filepath.Join(f.root, "bin/zellij")
	if err = os.WriteFile(f.executable, script, 0700); err != nil {
		t.Fatal(err)
	}
	renderer, err := os.ReadFile(filepath.Join(cwd, "tests/lib/zellij_fixture_replies.py"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(f.root, "bin/zellij_fixture_replies.py"), renderer, 0600); err != nil {
		t.Fatal(err)
	}
	f.update(func(s *state) { s.Model = Model{Sessions: map[string]Session{}} })
	return f
}
func (f *Fixture) Path() string { return f.executable }
func (f *Fixture) Root() string { return f.root }

// Env returns overrides suitable for t.Setenv or merging into os.Environ.
func (f *Fixture) Env() []string {
	return []string{
		"PATH=" + filepath.Dir(f.executable) + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + filepath.Join(f.root, "home"), "XDG_CACHE_HOME=" + filepath.Join(f.root, "cache"),
		"XDG_RUNTIME_DIR=" + filepath.Join(f.root, "runtime"), "XDG_CONFIG_HOME=" + filepath.Join(f.root, "config"),
		"ZELLIJ_SOCKET_DIR=" + filepath.Join(f.root, "runtime"), "ZELLIJ_SESSION_NAME=test", "ZELLIJ_PANE_ID=0",
		"PAIR_FAKE_ZELLIJ_STATE=" + filepath.Join(f.root, "state.json"),
	}
}
func (f *Fixture) SetModel(m Model)          { f.update(func(s *state) { s.Model = m }) }
func (f *Fixture) SetFaults(faults ...Fault) { f.update(func(s *state) { s.Faults = faults }) }
func (f *Fixture) Model() Model              { return f.read().Model }
func (f *Fixture) Queries() []Query          { return f.read().Queries }
func (f *Fixture) Effects() []Effect         { return f.read().Effects }
func (f *Fixture) read() state {
	var result state
	f.update(func(s *state) { result = *s })
	return result
}
func (f *Fixture) update(fn func(*state)) {
	f.t.Helper()
	lock, err := os.OpenFile(filepath.Join(f.root, "state.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		f.t.Fatal(err)
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		f.t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	path := filepath.Join(f.root, "state.json")
	var s state
	data, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(data, &s); err != nil {
			f.t.Fatal(err)
		}
	} else if !os.IsNotExist(err) {
		f.t.Fatal(err)
	}
	fn(&s)
	data, err = json.Marshal(s)
	if err != nil {
		f.t.Fatal(err)
	}
	tmp, err := os.CreateTemp(f.root, "state-*")
	if err != nil {
		f.t.Fatal(err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		f.t.Fatal(err)
	}
	if err = tmp.Close(); err != nil {
		f.t.Fatal(err)
	}
	if err = os.Rename(name, path); err != nil {
		f.t.Fatal(err)
	}
}

// DefaultPane supplies valid native geometry/flags, preserving a nullable layout command.
func DefaultPane(id int, command string) Pane {
	return Pane{"id": id, "is_plugin": false, "is_focused": true, "is_floating": false, "is_fullscreen": false, "tab_id": 0, "tab_position": 0, "title": "terminal", "terminal_command": nil, "pane_command": command, "pane_cwd": "/tmp", "pane_x": 0, "pane_y": 0, "pane_columns": 80, "pane_rows": 24}
}
func DefaultTab() Tab {
	return Tab{"tab_id": 0, "position": 0, "name": "test", "active": true, "is_fullscreen_active": false, "is_sync_panes_active": false, "are_floating_panes_visible": false, "other_focused_clients": []int{}, "active_swap_layout_name": "base", "is_swap_layout_dirty": false}
}

// Environment merges fixture overrides with the caller environment for exec.Cmd.
func (f *Fixture) Environment() []string {
	overrides := f.Env()
	keys := map[string]bool{}
	for _, v := range overrides {
		keys[strings.SplitN(v, "=", 2)[0]] = true
	}
	var env []string
	for _, v := range os.Environ() {
		if !keys[strings.SplitN(v, "=", 2)[0]] {
			env = append(env, v)
		}
	}
	return append(env, overrides...)
}
