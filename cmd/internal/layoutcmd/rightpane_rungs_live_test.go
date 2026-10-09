package layoutcmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// The #417 focus restore depends on Pair's swap layouts: re-embedding a floated
// pane does not restore its tiled slot, so leaving focus re-applies the recorded
// swap layout. This runs the shipped zellij/layouts/main-3.kdl (commands swapped
// for receivers) through the full cycle on every rung, unsplit and split.
// Run on zellij upgrades and layout edits:
// PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run '^TestRightPaneRungsZellijConformance$' -count=1
func TestRightPaneRungsZellijConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for disposable main-3 swap-layout conformance")
	}
	base := t.TempDir()
	t.Setenv("TMPDIR", "/tmp") // Darwin's Unix socket pathname limit.
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR"} {
		t.Setenv(key, t.TempDir())
	}
	t.Setenv("SHELL", "/bin/sh")
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "ZELLIJ_PANE_ID", "PAIR_SESSION", "PAIR_TAG", "PAIR_DATA_DIR"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(base, "config.kdl")
	if err := os.WriteFile(config, []byte("show_startup_tips false\nshow_release_notes false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("../../../zellij/layouts/main-3.kdl")
	if err != nil {
		t.Fatal(err)
	}
	receivers := 0
	body := regexp.MustCompile(`(?s)command "sh"\s*\n\s*(//[^\n]*\n\s*)*args "-c" "[^\n]*"`).ReplaceAllStringFunc(string(src), func(string) string {
		receivers++
		command := `while IFS= read -r line; do printf '%s\n' "$line" >> "$1"; done`
		return fmt.Sprintf("command \"/bin/sh\"\n args \"-c\" %s \"receiver\" %s\n", strconv.Quote(command), strconv.Quote(filepath.Join(base, fmt.Sprintf("p%d", receivers))))
	})
	if receivers != 3 {
		t.Fatalf("main-3.kdl: replaced %d pane commands, want 3 (agent, draft, terminal)", receivers)
	}
	layout := filepath.Join(base, "layout.kdl")
	if err := os.WriteFile(layout, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, "pair-rp-"+hex.EncodeToString(entropy[:]), pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Logf("fixture close: %v", err)
		}
	})
	rt := &fullscreenLiveRuntime{ctx: ctx, session: fixture.Session, store: workbenchshortcut.FullscreenReturnStore{DataDir: t.TempDir(), Tag: "rungs"}}
	type snapshot struct {
		geometry string
		swap     string
		dirty    bool
	}
	observe := func() ([]zellijpane.Pane, snapshot) {
		t.Helper()
		raw, err := rt.ListPanesJSON()
		if err != nil {
			t.Fatal(err)
		}
		var panes []zellijpane.Pane
		var parts []string
		for _, p := range zellijpane.Parse(raw) {
			if p.IsPlugin {
				continue
			}
			panes = append(panes, p)
			parts = append(parts, fmt.Sprintf("%s:%d,%d,%dx%d,float=%v", p.ID, p.X, p.Y, p.Columns, p.Rows, p.IsFloating))
		}
		sort.Strings(parts)
		tab, err := rt.CurrentTabJSON()
		if err != nil {
			t.Fatal(err)
		}
		swap, dirty, err := ParseTabLayout(tab)
		if err != nil {
			t.Fatal(err)
		}
		return panes, snapshot{strings.Join(parts, " "), swap, dirty}
	}
	byTitle := func(panes []zellijpane.Pane, title string) []zellijpane.Pane {
		var out []zellijpane.Pane
		for _, p := range panes {
			if p.Title == title {
				out = append(out, p)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Y < out[j].Y })
		return out
	}
	action := func(args ...string) {
		t.Helper()
		if err := rt.RunZellijAction(args...); err != nil {
			t.Fatal(err)
		}
		time.Sleep(150 * time.Millisecond)
	}
	press := func(caller string) {
		t.Helper()
		rt.caller = caller
		var stderr bytes.Buffer
		if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
			t.Fatalf("press from %s: exit %d: %s", caller, code, &stderr)
		}
		time.Sleep(150 * time.Millisecond)
	}
	scoped, err := artifactpath.ResolveScoped(rt.store.DataDir, rt.store.Tag)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := scoped.FullscreenDiagnostics()
	var panes []zellijpane.Pane
	fullscreenLiveWait(t, ctx, "main-3 panes", func() bool {
		raw, err := rt.ListPanesJSON()
		if err != nil {
			t.Fatal(err)
		}
		panes = zellijpane.Parse(raw)
		return len(byTitle(panes, "terminal")) == 1 && len(byTitle(panes, "agent")) == 1 && len(byTitle(panes, "draft")) == 1
	})
	if _, err := fixture.WriteInput([]byte("x\n")); err != nil { // Register the attached client.
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	draft := byTitle(panes, "draft")[0].ID
	// current-tab-info resolves the tab through the calling pane, as it does
	// for a press from inside Pair.
	rt.caller = draft
	cycle := func(name string, target string) {
		t.Helper()
		t.Log(name)
		before, want := observe()
		rt.terminals = nil
		for _, p := range byTitle(before, "terminal") {
			rt.terminals = append(rt.terminals, p.ID)
		}
		rt.last = target
		press(draft)
		if _, got := observe(); !strings.Contains(got.geometry, target+":") || !strings.Contains(got.geometry, "float=true") {
			t.Fatalf("%s: focus not entered: %s", name, got.geometry)
		}
		press(target)
		press(target)
		fullscreenLiveWait(t, ctx, name+" restored", func() bool {
			_, got := observe()
			return got.geometry == want.geometry
		})
		if record, err := rt.store.Read(); err != nil || record != "" {
			t.Fatalf("%s: record %q %v", name, record, err)
		}
		if data, err := os.ReadFile(diagnostics); err == nil && len(data) > 0 {
			t.Fatalf("%s: diagnostics %s", name, data)
		}
	}
	terminal := byTitle(panes, "terminal")[0].ID
	cycle("base", terminal)
	action("next-swap-layout")
	cycle("minimized", terminal)
	action("next-swap-layout")
	cycle("third", terminal)
	action("next-swap-layout") // back to base
	// Alt+Shift+d's shape: a new terminal below, leaving the rung name dirty.
	action("focus-pane-id", terminal)
	action("new-pane", "--direction", "down", "--name", "terminal", "--", "/bin/sh", "-c", "sleep 600")
	panes, _ = observe()
	halves := byTitle(panes, "terminal")
	if len(halves) != 2 {
		t.Fatalf("split: %d terminals", len(halves))
	}
	cycle("dirty-split-bottom", halves[1].ID)
	cycle("split-top", halves[0].ID)
	action("next-swap-layout")
	cycle("split-next-rung-top", halves[0].ID)
	// The top-half fix-up must survive later rung changes.
	action("next-swap-layout")
	panes, _ = observe()
	if got := byTitle(panes, "terminal"); len(got) != 2 || got[0].ID != halves[0].ID {
		t.Fatalf("half order lost after rung change: %+v", got)
	}
}
