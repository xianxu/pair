package layoutcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type terminalRecoveryFocusControl struct{ *fullscreenLiveRuntime }

func (r terminalRecoveryFocusControl) RunZellijAction(args ...string) error {
	if len(args) != 0 && args[0] == "new-pane" {
		filtered := make([]string, 0, len(args))
		for _, arg := range args {
			if arg != "--no-focus" {
				filtered = append(filtered, arg)
			}
		}
		args = filtered
	}
	return r.fullscreenLiveRuntime.RunZellijAction(args...)
}

// This probes native new-pane placement and client focus through the same
// routing seam used by nvim and the agent wrapper. Only the pane's application
// is a receiver: no operator pane, shell configuration, or agent is involved.
func TestTerminalRecoveryZellijConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for disposable native terminal recovery conformance")
	}
	base := t.TempDir()
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(key, t.TempDir())
	}
	// Avoid t.TempDir's long test-name component in Unix socket paths while
	// keeping all scratch on the invocation's configured temporary filesystem.
	runtimeDir, err := os.MkdirTemp("", "tr-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtimeDir) })
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(runtimeDir, "s"))
	t.Setenv("ZELLIJ_CONFIG_DIR", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "ZELLIJ_PANE_ID", "PAIR_SESSION", "PAIR_TAG", "PAIR_DATA_DIR"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, data string, mode os.FileMode) string {
		t.Helper()
		path := filepath.Join(base, name)
		if err := os.WriteFile(path, []byte(data), mode); err != nil {
			t.Fatal(err)
		}
		return path
	}
	// Preserve the production creation argv, including 'pair term', without
	// launching a real user's shell or depending on a separately built Pair.
	write("pair", "#!/bin/sh\n[ \"$1\" = term ] || exit 2\nwhile IFS= read -r line; do printf '%s\\n' \"$line\" >> \"$PAIR_RECOVERY_INPUT/$ZELLIJ_PANE_ID\"; done\n", 0700)
	t.Setenv("PATH", base+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR_RECOVERY_INPUT", base)
	config := write("config.kdl", "pane_frames false\nshow_startup_tips false\nshow_release_notes false\nsession_serialization false\n", 0600)
	callerInput := filepath.Join(base, "caller-input")
	callerScript := `while IFS= read -r line; do printf '%s\n' "$line" >> "$1"; done`
	layout := write("layout.kdl", fmt.Sprintf("layout {\n pane name=\"recovery-caller\" command=\"/bin/sh\" { args \"-c\" %s \"receiver\" %s; }\n}\n", strconv.Quote(callerScript), strconv.Quote(callerInput)), 0600)
	foreignLayout := write("foreign.kdl", "layout {\n pane name=\"foreign-terminal\" command=\"pair\" { args \"term\"; }\n}\n", 0600)
	var entropy [6]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, "tr-"+hex.EncodeToString(entropy[:]), pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("disposable fixture cleanup: %v", err)
		}
	})
	rt := &fullscreenLiveRuntime{ctx: ctx, session: fixture.Session, queryLog: t.Logf}
	observe := func() []zellijpane.Pane {
		t.Helper()
		raw, err := rt.queryContext(ctx, "list-panes", "--json", "--command", "--state", "--geometry", "--all")
		if err != nil {
			t.Fatal(err)
		}
		return zellijpane.Parse(raw)
	}
	find := func(title string) (zellijpane.Pane, bool) {
		for _, p := range observe() {
			if p.Title == title && !p.IsPlugin {
				return p, true
			}
		}
		return zellijpane.Pane{}, false
	}
	var caller, foreign zellijpane.Pane
	fullscreenLiveWait(t, ctx, "invoking pane", func() bool { var ok bool; caller, ok = find("recovery-caller"); return ok })
	rt.caller = caller.ID
	assertCallerInput := func(marker string) {
		t.Helper()
		if _, err := fixture.WriteInput([]byte(marker + "\n")); err != nil {
			t.Fatal(err)
		}
		fullscreenLiveWait(t, ctx, "client input remains in invoking pane", func() bool {
			data, err := os.ReadFile(callerInput)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			return strings.Contains(string(data), marker+"\n")
		})
	}
	// Establish the fixture's keyboard-active client before issuing the route.
	assertCallerInput("initial-caller")
	if err := rt.RunZellijAction("new-tab", "--layout", foreignLayout); err != nil {
		t.Fatal(err)
	}
	fullscreenLiveWait(t, ctx, "foreign terminal", func() bool { var ok bool; foreign, ok = find("foreign-terminal"); return ok })
	if caller.TabID == nil || foreign.TabID == nil || *caller.TabID == *foreign.TabID {
		t.Fatalf("fixture did not create separate tabs: caller=%+v foreign=%+v", caller, foreign)
	}
	if err := rt.RunZellijAction("focus-pane-id", caller.ID); err != nil {
		t.Fatal(err)
	}
	assertCallerInput("before-recovery")
	foreign, _ = find("foreign-terminal")
	rt.last, rt.terminals = foreign.ID, []string{foreign.ID}
	if err := SwitchRightTerminalTabContext(ctx, rt, workbenchshortcut.ChordAltShiftT); err != nil {
		t.Fatal(err)
	}
	var recovered zellijpane.Pane
	fullscreenLiveWait(t, ctx, "visible recovered terminal in invoking tab", func() bool {
		for _, p := range observe() {
			if p.ID != caller.ID && p.TabID != nil && *p.TabID == *caller.TabID && !p.IsPlugin && p.Title == "terminal" && p.Columns > 0 && p.Rows > 0 && !p.IsFloating {
				recovered = p
				return true
			}
		}
		return false
	})
	assertCallerInput("after-recovery")
	if current, ok := find("recovery-caller"); !ok || !current.IsFocused || recovered.IsFocused {
		t.Fatalf("recovery moved client focus: caller=%+v recovered=%+v", current, recovered)
	}
	if current, ok := find("foreign-terminal"); !ok || !reflect.DeepEqual(current, foreign) {
		t.Fatalf("foreign terminal changed: before=%+v after=%+v", foreign, current)
	}
	if data, err := os.ReadFile(filepath.Join(base, foreign.ID)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	} else if len(data) != 0 {
		t.Fatalf("foreign terminal received input: %q", data)
	}
	// Negative control: native default creation steals the actual client's
	// focus. Removing only --no-focus must make input reach the new receiver,
	// proving the positive assertion above cannot pass on per-tab focus flags.
	if err := rt.RunZellijAction("close-pane", "--pane-id", recovered.ID); err != nil {
		t.Fatal(err)
	}
	fullscreenLiveWait(t, ctx, "recovered pane removed for focus control", func() bool {
		for _, p := range observe() {
			if !p.IsPlugin && p.ID == recovered.ID {
				return false
			}
		}
		return true
	})
	if err := SwitchRightTerminalTabContext(ctx, terminalRecoveryFocusControl{rt}, workbenchshortcut.ChordAltShiftT); err != nil {
		t.Fatal(err)
	}
	var control zellijpane.Pane
	fullscreenLiveWait(t, ctx, "default creation focuses new terminal", func() bool {
		p, ok := find("terminal")
		control = p
		return ok && p.IsFocused && p.ID != recovered.ID
	})
	const controlMarker = "negative-control-default-creation"
	if _, err := fixture.WriteInput([]byte(controlMarker + "\n")); err != nil {
		t.Fatal(err)
	}
	fullscreenLiveWait(t, ctx, "default creation redirects native client input", func() bool {
		data, err := os.ReadFile(filepath.Join(base, control.ID))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return strings.Contains(string(data), controlMarker+"\n")
	})
	if data, err := os.ReadFile(callerInput); err != nil || strings.Contains(string(data), controlMarker) {
		t.Fatalf("negative control did not distinguish native client focus: %q, %v", data, err)
	}
}
