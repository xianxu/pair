package termcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func (f *fakeRuntime) FullscreenStore() workbenchshortcut.FullscreenStore { return f }
func (f *fakeRuntime) Read() (string, error)                              { return f.fullscreenRecord, nil }
func (f *fakeRuntime) Write(id string) error                              { f.fullscreenRecord = id; return nil }
func (f *fakeRuntime) Clear() error                                       { f.fullscreenRecord = ""; return nil }
func (f *fakeRuntime) TryLock() (func(), bool, error)                     { return func() {}, true, nil }
func (f *fakeRuntime) CurrentTabJSON() ([]byte, error) {
	return []byte(`{"tab_id":1,"position":1,"active_swap_layout_name":"BASE","is_swap_layout_dirty":false}`), nil
}
func (f *fakeRuntime) NudgeWrap() error           { return nil }
func (f *fakeRuntime) NotifyRefusal(string) error { return nil }
func (f *fakeRuntime) LogFailure(err error) {
	f.fullscreenErrors = append(f.fullscreenErrors, err.Error())
}

func TestTerminalFullscreenFailureIsLoggedAndReported(t *testing.T) {
	for _, direct := range []bool{true, false} {
		rt := &fakeRuntime{currentPaneID: "4", failList: true}
		mux := &fakeMux{}
		if direct {
			if !handleTerminalChord(workbenchshortcut.ChordAltShiftEnter, mux, rt) {
				t.Fatal("not handled")
			}
		} else {
			err := runDecision(workbenchshortcut.ShortcutDecision{Disposition: workbenchshortcut.DispositionHandle, Action: workbenchshortcut.ActionToggleFocusedLayout}, workbenchPanes{}, rt, strings.NewReader(""), io.Discard)
			if err == nil {
				t.Fatal("layout refusal silently succeeded")
			}
		}
		wantReported := 0
		if direct {
			wantReported = 1
		}
		if len(rt.fullscreenErrors) != 1 || len(mux.reported) != wantReported || len(rt.ops) != 0 {
			t.Fatalf("logs=%v UI=%v actions=%v", rt.fullscreenErrors, mux.reported, rt.ops)
		}
	}
}

func TestTerminalToggleRefusalReturnsFailureAndNotifiesBroker(t *testing.T) {
	f := zellijtest.New(t)
	for _, entry := range f.Env() {
		key, value, _ := strings.Cut(entry, "=")
		t.Setenv(key, value)
	}
	//lint:allow tmp-literal — the real notification broker appends a 32-hex digest and PID; macOS TMPDIR can exceed its 100-byte socket limit
	socketDir, err := os.MkdirTemp("/tmp", "p647-term-notify-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(socketDir) })
	t.Setenv("PAIR_NOTIFY_SOCKET_DIR", socketDir)
	t.Setenv("PAIR_DATA_DIR", f.Root())
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("ZELLIJ_PANE_ID", "4")
	binding := filepath.Join(f.Root(), "wrapper.pid")
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	f.SetModel(shortcutAdmissionModel(2))
	wrong := "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &wrong})
	var stderr bytes.Buffer
	if code := Run([]string{"--test-shortcut", "Alt+Shift+Enter"}, strings.NewReader(""), io.Discard, &stderr); code != 1 {
		t.Fatalf("uncertain toggle code=%d stderr=%q", code, &stderr)
	}
	select {
	case message := <-broker.Messages():
		if message != "Pair action refused: terminal context could not be confirmed." {
			t.Fatalf("notice=%q", message)
		}
	case <-testwait.After(t):
		t.Fatal("terminal refusal never reached broker")
	}
	if !strings.Contains(stderr.String(), "refused") || len(f.Effects()) != 0 || len(f.Queries()) != 1 {
		t.Fatalf("diagnostic=%q effects=%+v queries=%+v", &stderr, f.Effects(), f.Queries())
	}
}

type nativeReturnFocusRuntime struct {
	*fakeRuntime
	quiet bool
}

func (r nativeReturnFocusRuntime) RunZellijAction(args ...string) error {
	if args[0] == "focus-pane-id" {
		if r.quiet {
			return (OSRuntime{}).RunZellijActionQuiet(args...)
		}
		return (OSRuntime{}).RunZellijAction(args...)
	}
	return r.fakeRuntime.RunZellijAction(args...)
}

func TestTerminalNativeAlreadyFocusedReturnClearsRecord(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte("#!/bin/sh\nprintf 'Pane Terminal(2) is already focused\\n' >&2\nexit 2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	for _, quiet := range []bool{false, true} {
		f := &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}, fullscreenRecord: "2 swap=BASE", panesJSON: `[{"id":2,"is_floating":false,"is_fullscreen":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md","is_plugin":false,"is_focused":false,"pane_x":0,"pane_y":0,"pane_columns":75,"pane_rows":39,"tab_id":1,"tab_position":1},{"id":4,"is_floating":true,"is_fullscreen":false,"title":"terminal","terminal_command":"pair term","is_plugin":false,"is_focused":false,"pane_x":0,"pane_y":0,"pane_columns":75,"pane_rows":39,"tab_id":1,"tab_position":1}]`}
		if code := layoutcmd.RunToggleFocused(nil, nativeReturnFocusRuntime{f, quiet}, io.Discard); code != 0 || f.fullscreenRecord != "" || len(f.fullscreenErrors) != 0 {
			t.Fatalf("quiet=%v code=%d record=%q errors=%v", quiet, code, f.fullscreenRecord, f.fullscreenErrors)
		}
	}
}
