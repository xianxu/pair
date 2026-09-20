package layoutcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// This complements native conformance: bytes enter the attached client and
// traverse the actual Zellij config, nvim mapping, wrapper, and terminal mux.
// No coding agent is launched; pair wrap proxies an isolated /bin/sh child.
func TestFullscreenChordZellijLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for disposable actual-chord smoke")
	}
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	bin := filepath.Join(base, "bin", "pair")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/pair-go")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build isolated pair: %v: %s", err, out)
	}
	// Erase inherited selected-thread paths before exporting this fixture's
	// complete canonical bindings. Neither editor nor wrapper can reach them.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") || strings.HasPrefix(key, "ZELLIJ") || strings.HasPrefix(key, "COUCH_") {
			t.Setenv(key, "")
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("TMPDIR", "/tmp")
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR"} {
		t.Setenv(key, t.TempDir())
	}
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("ENV", "")
	t.Setenv("PS1", "smoke$ ")
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	data := t.TempDir()
	paths, err := artifactpath.ResolveScoped(data, "chord-smoke")
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := paths.EnvironmentBindings("generic")
	if err != nil {
		t.Fatal(err)
	}
	for _, binding := range bindings {
		t.Setenv(binding.Name, binding.Path)
	}
	t.Setenv("PAIR_DATA_DIR", data)
	t.Setenv("PAIR_TAG", paths.Tag())
	t.Setenv("PAIR_HOME", base)
	t.Setenv("PAIR_AGENT", "generic")
	t.Setenv("PAIR_RETENTION_PROTOCOL", "")
	t.Setenv("PAIR_NVIM_PID_FILE", paths.NvimPID("draft"))
	if err := os.WriteFile(paths.Draft(), []byte("fullscreen chord smoke\n"), 0600); err != nil {
		t.Fatal(err)
	}
	pane := func(name, command string, args ...string) string {
		quoted := make([]string, len(args))
		for i, arg := range args {
			quoted[i] = strconv.Quote(arg)
		}
		return fmt.Sprintf("pane name=%s borderless=true command=%s { args %s; }\n", strconv.Quote(name), strconv.Quote(command), strings.Join(quoted, " "))
	}
	layout := filepath.Join(base, "layout.kdl")
	body := "layout { pane split_direction=\"vertical\" {\n pane {\n" +
		pane("agent", bin, "wrap", "/bin/sh", "-i") +
		pane("draft", "nvim", "-u", filepath.Join(repo, "nvim", "init.lua"), paths.Draft()) +
		"}\n pane {\n" + pane("terminal-top", bin, "term") + pane("terminal-bottom", bin, "term") + "}\n }\n}\n"
	if err := os.WriteFile(layout, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, "pair-chord-"+hex.EncodeToString(entropy[:]), pairlifecycletest.ZellijOptions{ConfigFile: filepath.Join(repo, "zellij", "config.kdl"), LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Logf("fixture close (fresh session may lack resurrection data): %v", err)
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fullscreenLiveWait(t, cleanup, "chord fixture shutdown", func() bool {
			out, err := exec.CommandContext(cleanup, "zellij", "list-sessions", "--no-formatting").CombinedOutput()
			if err != nil && !strings.Contains(string(out), "No active zellij sessions found") {
				t.Fatalf("verify shutdown: %v: %s", err, out)
			}
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(line)
				if len(fields) > 0 && fields[0] == fixture.Session && !strings.Contains(line, "EXITED") {
					return false
				}
			}
			return true
		})
	})
	rt := &fullscreenLiveRuntime{ctx: ctx, session: fixture.Session}
	observe := func() map[string]zellijpane.Pane {
		t.Helper()
		raw, err := rt.ListPanesJSON()
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]zellijpane.Pane{}
		for _, p := range zellijpane.Parse(raw) {
			if !p.IsPlugin {
				out[p.ID] = p
			}
		}
		return out
	}
	var agent, draft string
	var terminals []zellijpane.Pane
	fullscreenLiveWait(t, ctx, "real wrapper/editor/terminal startup", func() bool {
		terminals = nil
		for _, p := range observe() {
			switch workbenchshortcut.RoleForPane(p) {
			case workbenchshortcut.PaneRoleLeftAgent:
				agent = p.ID
			case workbenchshortcut.PaneRoleLeftDraft:
				draft = p.ID
			case workbenchshortcut.PaneRoleRightTerminal:
				terminals = append(terminals, p)
			}
		}
		_, err := os.Stat(paths.DraftPane())
		return agent != "" && draft != "" && len(terminals) == 2 && err == nil
	})
	sort.Slice(terminals, func(i, j int) bool { return terminals[i].Y < terminals[j].Y })
	top, bottom := terminals[0].ID, terminals[1].ID
	store := workbenchshortcut.FullscreenReturnStore{DataDir: data, Tag: paths.Tag()}
	if err := (workbenchshortcut.LastTerminalPaneStore{DataDir: data, Tag: paths.Tag()}).Write(bottom); err != nil {
		t.Fatal(err)
	}
	fullscreenLiveWait(t, ctx, "both live terminal registrations", func() bool {
		ids, err := (workbenchshortcut.TerminalPaneRegistry{DataDir: data, Tag: paths.Tag()}).LiveIDs(func(pid int) bool { return pid > 0 })
		return err == nil && workbenchshortcut.Registered(ids, top) && workbenchshortcut.Registered(ids, bottom)
	})
	input := func(value string) {
		t.Helper()
		if _, err := fixture.WriteInput([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	focused := func(id string) bool {
		out, err := rt.command("list-clients")
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "terminal_"+id {
				return true
			}
		}
		return false
	}
	// Register the attached keyboard client before CLI focus actions; otherwise
	// Zellij may associate those actions with a transient CLI client instead.
	input("\r")
	fullscreenLiveWait(t, ctx, "attached agent client", func() bool { return focused(agent) })
	focus := func(id string) {
		t.Helper()
		if !focused(id) {
			if err := rt.RunZellijAction("focus-pane-id", id); err != nil {
				t.Fatal(err)
			}
		}
		fullscreenLiveWait(t, ctx, "client focus "+id, func() bool { return focused(id) })
	}
	strip := func(id string) {
		t.Helper()
		var screen string
		defer func() {
			if t.Failed() {
				t.Logf("last strip observation for pane %s: %q", id, screen)
			}
		}()
		fullscreenLiveWait(t, ctx, "terminal strip redraw for "+id, func() bool {
			raw, err := rt.ListPanesJSON()
			if err != nil {
				t.Fatal(err)
			}
			var geometry []struct {
				ID          int  `json:"id"`
				Plugin      bool `json:"is_plugin"`
				ContentRows int  `json:"pane_content_rows"`
			}
			if err := json.Unmarshal(raw, &geometry); err != nil {
				t.Fatal(err)
			}
			rows := 0
			for _, pane := range geometry {
				if !pane.Plugin && strconv.Itoa(pane.ID) == id {
					rows = pane.ContentRows
				}
			}
			out, err := rt.command("dump-screen", "--pane-id", id)
			if err != nil {
				t.Fatal(err)
			}
			screen = string(out)
			lines := strings.Split(strings.TrimSuffix(screen, "\n"), "\n")
			// After an alternate-screen child shrinks, dump-screen can retain
			// rows below the reported content height. Check the visible pane,
			// including its exact bottom row, rather than those offscreen rows.
			return rows > 0 && len(lines) >= rows && strings.Contains(lines[rows-1], "[terminal 1]") && strings.Count(strings.Join(lines[:rows], "\n"), "[terminal 1]") == 1
		})
	}
	strip(top)
	strip(bottom)
	chord := "\x1b[13;4u" // Actual Shift+Alt+Return, delivered to the attached client.
	for _, tc := range []struct {
		name, caller, target string
		nvimChild            bool
	}{
		{"draft nvim mapping", draft, bottom, false},
		{"generic agent wrapper", agent, bottom, false},
		{"terminal shell bottom", bottom, bottom, false},
		{"terminal shell top", top, top, false},
		{"terminal nvim child", bottom, bottom, true},
	} {
		t.Log(tc.name)
		focus(tc.caller)
		childReady := func() {
			t.Helper()
			fullscreenLiveWait(t, ctx, "nvim child screen", func() bool {
				out, err := rt.command("dump-screen", "--pane-id", bottom)
				if err != nil {
					t.Fatal(err)
				}
				return strings.Contains(string(out), "PAIR_NVIM_CHILD_READY")
			})
		}
		if tc.nvimChild {
			// Split the marker in the shell command so command echo cannot
			// satisfy readiness: only nvim's rendered buffer contains it.
			input("nvim -u NONE -i NONE -n -c 'call setline(1, \"PAIR_NVIM_\" . \"CHILD_READY\")' -c 'set nomodified'\r")
			childReady()
			strip(bottom)
		}
		original := observe()
		input(chord)
		fullscreenLiveWait(t, ctx, tc.name+" fullscreen", func() bool {
			p := observe()[tc.target]
			return p.IsFullscreen != nil && *p.IsFullscreen && p.Columns > original[tc.target].Columns && p.Rows > original[tc.target].Rows
		})
		fullscreenLiveWait(t, ctx, "fullscreen client focus", func() bool { return focused(tc.target) })
		if got, err := store.Read(); err != nil || got != tc.caller {
			t.Fatalf("return record = %q, %v; want %s", got, err, tc.caller)
		}
		strip(tc.target)
		if tc.nvimChild {
			childReady()
		}
		input(chord) // Collapse is always received by the real fullscreen pair term.
		fullscreenLiveWait(t, ctx, tc.name+" restored geometry", func() bool {
			got := observe()
			for _, id := range []string{agent, draft, top, bottom} {
				p, want := got[id], original[id]
				if p.ID != want.ID || p.IsFullscreen == nil || *p.IsFullscreen || p.X != want.X || p.Y != want.Y || p.Rows != want.Rows || p.Columns != want.Columns {
					return false
				}
			}
			return true
		})
		fullscreenLiveWait(t, ctx, "return client focus", func() bool { return focused(tc.caller) })
		fullscreenLiveWait(t, ctx, "return record cleared", func() bool { got, err := store.Read(); return err == nil && got == "" })
		strip(top)
		strip(bottom)
		if tc.nvimChild {
			childReady()
		}
	}
	if data, err := os.ReadFile(paths.FullscreenDiagnostics()); err == nil && len(data) > 0 {
		t.Fatalf("fullscreen failure diagnostics: %s", data)
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
