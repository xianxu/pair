package layoutcmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// Run explicitly on supported Zellij upgrades and fullscreen/focus changes:
// PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run '^TestFullscreenZellijConformance$' -count=1
func TestFullscreenZellijConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for disposable native fullscreen conformance")
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
	if err := os.WriteFile(config, []byte("pane_frames false\nshow_startup_tips false\nshow_release_notes false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// Tiny receiver processes give an independent observation of actual client
	// focus. They do not source user shell configuration or invoke Pair agents.
	pane := func(name string) string {
		command := `while IFS= read -r line; do printf '%s\n' "$line" >> "$1"; done`
		return fmt.Sprintf("pane name=%s command=\"/bin/sh\" { args \"-c\" %s \"receiver\" %s; }\n", strconv.Quote(name), strconv.Quote(command), strconv.Quote(filepath.Join(base, name)))
	}
	layout := filepath.Join(base, "layout.kdl")
	body := "layout {\n pane split_direction=\"vertical\" {\n pane {\n" + pane("agent") + pane("draft") + "}\n pane {\n" + pane("terminal-top") + pane("terminal-bottom") + "}\n }\n}\n"
	if err := os.WriteFile(layout, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, "pair-fs-"+hex.EncodeToString(entropy[:]), pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			// A short-lived session may have no resurrection directory yet:
			// delete-session --force kills it but returns 2 for that missing
			// directory. Verify server disappearance independently of this code.
			t.Logf("fixture close: %v; checking server disappearance", err)
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		fullscreenLiveWait(t, cleanupCtx, "disposable server shutdown", func() bool {
			out, err := exec.CommandContext(cleanupCtx, "zellij", "list-sessions", "--no-formatting").CombinedOutput()
			if err != nil && !strings.Contains(string(out), "No active zellij sessions found") {
				t.Fatalf("verify fixture shutdown: %v: %s", err, out)
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
	rt := &fullscreenLiveRuntime{ctx: ctx, session: fixture.Session, store: workbenchshortcut.FullscreenReturnStore{DataDir: t.TempDir(), Tag: "conformance"}}
	names := []string{"agent", "draft", "terminal-top", "terminal-bottom"}
	var panes map[string]zellijpane.Pane
	observe := func() map[string]zellijpane.Pane {
		t.Helper()
		raw, err := rt.ListPanesJSON()
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]zellijpane.Pane{}
		for _, p := range zellijpane.Parse(raw) {
			for _, name := range names {
				if p.Title == name && !p.IsPlugin {
					found[name] = p
				}
			}
		}
		return found
	}
	fullscreenLiveWait(t, ctx, "four layout panes", func() bool { panes = observe(); return len(panes) == 4 })
	rt.last = panes["terminal-bottom"].ID
	rt.caller = panes["agent"].ID
	rt.terminals = []string{panes["terminal-top"].ID, rt.last}
	for _, name := range names {
		p := panes[name]
		if p.IsFullscreen == nil || *p.IsFullscreen || p.Columns <= 0 || p.Rows <= 0 {
			t.Fatalf("invalid initial pane %s: %+v", name, p)
		}
	}
	if panes["terminal-top"].X <= panes["draft"].X || panes["terminal-bottom"].Y <= panes["terminal-top"].Y {
		t.Fatalf("not a split right terminal layout: %+v", panes)
	}
	action := func(args ...string) {
		t.Helper()
		if err := rt.RunZellijAction(args...); err != nil {
			t.Fatal(err)
		}
	}
	sequence := 0
	assertFocus := func(name string) {
		t.Helper()
		fullscreenLiveWait(t, ctx, "focus observation for "+name, func() bool { return observe()[name].IsFocused })
		sequence++
		marker := fmt.Sprintf("focus-%d", sequence)
		if _, err := fixture.WriteInput([]byte(marker + "\n")); err != nil {
			t.Fatal(err)
		}
		fullscreenLiveWait(t, ctx, "client input reaches "+name, func() bool {
			for _, other := range names {
				data, err := os.ReadFile(filepath.Join(base, other))
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				if strings.Contains(string(data), marker+"\n") {
					if other != name {
						clients, _ := rt.command("list-clients")
						t.Logf("clients: %s", clients)
						t.Fatalf("input reached %s, wanted %s", other, name)
					}
					return true
				}
			}
			return false
		})
	}
	toggle := func(caller string) {
		t.Helper()
		rt.caller = panes[caller].ID
		var stderr bytes.Buffer
		if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
			t.Errorf("toggle from %s: exit %d: %s", caller, code, &stderr)
		}
	}
	assertTiled := func(original map[string]zellijpane.Pane) {
		t.Helper()
		fullscreenLiveWait(t, ctx, "exact restored tiling", func() bool {
			got := observe()
			if len(got) != len(original) {
				return false
			}
			for name, want := range original {
				p := got[name]
				if p.ID != want.ID || p.IsFullscreen == nil || *p.IsFullscreen || p.X != want.X || p.Y != want.Y || p.Columns != want.Columns || p.Rows != want.Rows {
					return false
				}
			}
			return true
		})
	}
	assertExpanded := func(target string, original map[string]zellijpane.Pane) {
		t.Helper()
		fullscreenLiveWait(t, ctx, "observed fullscreen "+target, func() bool {
			got := observe()
			p, ok := got[target]
			if !ok || p.IsFullscreen == nil || !*p.IsFullscreen || !p.IsFocused || p.Columns <= original[target].Columns || p.Rows <= original[target].Rows {
				return false
			}
			for name, q := range got {
				if name != target && q.IsFullscreen != nil && *q.IsFullscreen {
					t.Fatalf("unexpected fullscreen pane %s", name)
				}
			}
			return true
		})
		assertFocus(target)
	}
	roundTrip := func(caller, target string) {
		action("focus-pane-id", panes[caller].ID)
		assertFocus(caller)
		original := observe()
		toggle(caller)
		assertExpanded(target, original)
		if got, err := rt.store.Read(); err != nil || got != panes[caller].ID {
			t.Fatalf("return record %q, %v", got, err)
		}
		// The second press originates in the now-focused fullscreen terminal.
		toggle(target)
		assertTiled(original)
		assertFocus(caller)
		if got, err := rt.store.Read(); err != nil || got != "" {
			t.Errorf("return record not cleared: %q, %v", got, err)
		}
	}
	// Zellij routes CLI focus actions to its last keyboard-active client. The
	// fixture starts without keystrokes; establish its real client before any
	// CLI focus action, otherwise focus can belong to a transient CLI client.
	assertFocus("agent")
	for _, tc := range []struct{ name, caller, target string }{
		{"draft-to-recorded-bottom", "draft", "terminal-bottom"},
		{"agent-to-recorded-bottom", "agent", "terminal-bottom"},
		{"same-right-bottom", "terminal-bottom", "terminal-bottom"},
		{"invoking-top-overrides-recorded-bottom", "terminal-top", "terminal-top"},
	} {
		t.Log(tc.name)
		roundTrip(tc.caller, tc.target)
	}
	t.Log("manually-resized-split")
	{
		before := observe()
		action("resize", "increase", "down", "--pane-id", panes["terminal-top"].ID)
		fullscreenLiveWait(t, ctx, "manual split resize", func() bool { return observe()["terminal-top"].Rows != before["terminal-top"].Rows })
		roundTrip("draft", "terminal-bottom")
	}
	t.Log("native-focus-away-exits-fullscreen")
	{
		assertFocus("draft") // The preceding round trip restored this pane.
		original := observe()
		toggle("draft")
		assertExpanded("terminal-bottom", original)
		action("focus-pane-id", panes["agent"].ID)
		assertTiled(original)
		assertFocus("agent")
		// The record intentionally survives native focus-away; the next actual
		// executor invocation must derive direction from the tiled observation.
		toggle("agent")
		assertExpanded("terminal-bottom", original)
		toggle("terminal-bottom")
		assertTiled(original)
		assertFocus("agent")
		if got, err := rt.store.Read(); err != nil || got != "" {
			t.Fatalf("stale return record: %q, %v", got, err)
		}
	}
}

type fullscreenLiveRuntime struct {
	ctx                   context.Context
	session, caller, last string
	terminals             []string
	store                 workbenchshortcut.FullscreenReturnStore
}

func (r *fullscreenLiveRuntime) command(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "zellij", append([]string{"--session", r.session, "action"}, args...)...)
	command.Env = append(os.Environ(), "ZELLIJ_PANE_ID="+r.caller)
	out, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fixture zellij %v: %w: %s", args, err, out)
	}
	return out, nil
}
func (r *fullscreenLiveRuntime) ListPanesJSON() ([]byte, error) {
	return r.command("list-panes", "--json", "--all")
}
func (r *fullscreenLiveRuntime) RunZellijAction(args ...string) error {
	_, err := r.command(args...)
	return err
}
func (r *fullscreenLiveRuntime) CurrentPaneID() string                              { return r.caller }
func (r *fullscreenLiveRuntime) LastTerminalPaneID() (string, error)                { return r.last, nil }
func (r *fullscreenLiveRuntime) TerminalPaneIDs() ([]string, error)                 { return r.terminals, nil }
func (r *fullscreenLiveRuntime) FullscreenStore() workbenchshortcut.FullscreenStore { return r.store }

func fullscreenLiveWait(t *testing.T, ctx context.Context, description string, ready func() bool) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for !ready() {
		select {
		case <-ctx.Done():
			t.Fatalf("%s: %v", description, ctx.Err())
		case <-deadline.C:
			t.Fatalf("timed out waiting for %s", description)
		case <-time.After(20 * time.Millisecond):
		}
	}
}
