package broadcast

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestViewerFramePublicUnchanged(t *testing.T) {
	f := textFrame(t, 30, 4, "agent output", liveChrome("tabs"))
	got, err := ViewerFrame(f, terminal.FramePublic, false)
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatalf("public frame changed: %v", err)
	}
}

func TestViewerFrameShowSwitcherUnchanged(t *testing.T) {
	f := textFrame(t, 30, 4, "FLEET-SECRET", liveChrome("tabs"))
	got, err := ViewerFrame(f, terminal.FramePrivate, true)
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatalf("shown switcher changed: %v", err)
	}
}

func TestViewerFramePrivateIsPlaceholder(t *testing.T) {
	for _, rows := range []int{1, 2, 6} {
		f := textFrame(t, 40, rows, "FLEET-SECRET /path/to/thread\nnotes FLEET-SECRET", liveChrome("tabs"))
		f.Cursor = terminal.Cursor{X: 3, Y: 0, Visible: true}
		got, err := ViewerFrame(f, terminal.FramePrivate, false)
		if err != nil {
			t.Fatal(err)
		}
		if got.Geometry != f.Geometry {
			t.Fatalf("rows=%d: geometry %+v", rows, got.Geometry)
		}
		if got.Cursor.Visible {
			t.Fatalf("rows=%d: placeholder shows a cursor", rows)
		}
		text := frameText(t, got)
		if strings.Contains(text, "FLEET-SECRET") || strings.Contains(text, "/path/to") {
			t.Fatalf("rows=%d: private body leaked: %q", rows, text)
		}
		cols := f.Geometry.Cols
		last := f.Cells[len(f.Cells)-cols:]
		if !reflect.DeepEqual(got.Cells[len(got.Cells)-cols:], last) {
			t.Fatalf("rows=%d: tab bar not carried through", rows)
		}
		if rows > 1 && !strings.Contains(text, PlaceholderText) {
			t.Fatalf("rows=%d: placeholder text missing: %q", rows, text)
		}
	}
}
