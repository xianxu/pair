package termcmd

import (
	"io"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func (f *fakeRuntime) FullscreenStore() workbenchshortcut.FullscreenStore { return f }
func (f *fakeRuntime) Read() (string, error)                              { return f.fullscreenRecord, nil }
func (f *fakeRuntime) Write(id string) error                              { f.fullscreenRecord = id; return nil }
func (f *fakeRuntime) Clear() error                                       { f.fullscreenRecord = ""; return nil }
func (f *fakeRuntime) TryLock() (func(), bool, error)                     { return func() {}, true, nil }
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
