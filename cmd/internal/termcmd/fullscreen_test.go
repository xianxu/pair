package termcmd

import (
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func (f *fakeRuntime) FullscreenStore() workbenchshortcut.FullscreenStore { return f }
func (f *fakeRuntime) Read() (string, error)                              { return f.fullscreenRecord, nil }
func (f *fakeRuntime) Write(id string) error                              { f.fullscreenRecord = id; return nil }
func (f *fakeRuntime) Clear() error                                       { f.fullscreenRecord = ""; return nil }
func (f *fakeRuntime) TryLock() (func(), bool, error)                     { return func() {}, true, nil }
func (f *fakeRuntime) CurrentTabJSON() ([]byte, error) {
	return []byte(`{"active_swap_layout_name":"BASE","is_swap_layout_dirty":false}`), nil
}
func (f *fakeRuntime) NudgeWrap() error { return nil }
func (f *fakeRuntime) LogFailure(err error) {
	f.fullscreenErrors = append(f.fullscreenErrors, err.Error())
}

func TestTerminalFullscreenFailureIsLoggedAndNotShown(t *testing.T) {
	for _, direct := range []bool{true, false} {
		rt := &fakeRuntime{failList: true}
		mux := &fakeMux{}
		if direct {
			if !handleTerminalChord(workbenchshortcut.ChordAltShiftEnter, mux, rt) {
				t.Fatal("not handled")
			}
		} else {
			err := runDecision(workbenchshortcut.ShortcutDecision{Disposition: workbenchshortcut.DispositionHandle, Action: workbenchshortcut.ActionToggleFocusedLayout}, workbenchPanes{}, rt, strings.NewReader(""), io.Discard)
			if err != nil {
				t.Fatalf("error escaped to UI: %v", err)
			}
		}
		if len(rt.fullscreenErrors) != 1 || len(mux.reported) != 0 || len(rt.ops) != 0 {
			t.Fatalf("logs=%v UI=%v actions=%v", rt.fullscreenErrors, mux.reported, rt.ops)
		}
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
		f := &fakeRuntime{currentPaneID: "4", fullscreenRecord: "2 swap=BASE", panesJSON: `[{"id":2,"is_floating":false,"is_fullscreen":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"},{"id":4,"is_floating":true,"is_fullscreen":false,"title":"terminal","terminal_command":"pair term"}]`}
		if code := layoutcmd.RunToggleFocused(nil, nativeReturnFocusRuntime{f, quiet}, io.Discard); code != 0 || f.fullscreenRecord != "" || len(f.fullscreenErrors) != 0 {
			t.Fatalf("quiet=%v code=%d record=%q errors=%v", quiet, code, f.fullscreenRecord, f.fullscreenErrors)
		}
	}
}
