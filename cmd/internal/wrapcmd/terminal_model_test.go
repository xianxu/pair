package wrapcmd

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

func TestTerminalModelConstructorReturnsEmptySnapshot(t *testing.T) {
	for _, size := range []struct {
		width  int
		height int
	}{{1, 1}, {8, 3}, {80, 24}} {
		model, err := newTerminalModel(size.width, size.height)
		if err != nil {
			t.Fatalf("newTerminalModel(%d, %d): %v", size.width, size.height, err)
		}

		snapshot := model.Snapshot()
		if snapshot.Width != size.width || snapshot.Height != size.height {
			t.Errorf("snapshot dimensions = %dx%d, want %dx%d", snapshot.Width, snapshot.Height, size.width, size.height)
		}
		if snapshot.Cursor.X != 0 || snapshot.Cursor.Y != 0 {
			t.Errorf("snapshot cursor = %v, want (0,0)", snapshot.Cursor)
		}
		if snapshot.CursorVisible {
			t.Error("cursor starts visible, want fail-closed false")
		}
		if snapshot.AltScreen {
			t.Error("alternate screen starts active")
		}
		if len(snapshot.Cells) != size.width*size.height {
			t.Errorf("cell count = %d, want %d", len(snapshot.Cells), size.width*size.height)
		}
		if cell := snapshot.CellAt(0, 0); cell == nil || cell.Content != " " || cell.Width != 1 {
			t.Errorf("CellAt(0,0) = %#v, want x/vt's blank cell", cell)
		}
		for _, point := range [][2]int{{-1, 0}, {0, -1}, {size.width, 0}, {0, size.height}} {
			if cell := snapshot.CellAt(point[0], point[1]); cell != nil {
				t.Errorf("CellAt(%d,%d) = %#v, want nil", point[0], point[1], cell)
			}
		}

		if err := model.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}
}

func TestTerminalModelSnapshotCellAtToleratesMissingBackingCells(t *testing.T) {
	snapshot := terminalSnapshot{Width: 2, Height: 2}
	if cell := snapshot.CellAt(1, 1); cell != nil {
		t.Fatalf("CellAt with missing backing cell = %#v, want nil", cell)
	}
}

func TestTerminalModelConstructorRejectsInvalidDimensions(t *testing.T) {
	for _, size := range [][2]int{{0, 1}, {1, 0}, {-1, 1}, {1, -1}} {
		model, err := newTerminalModel(size[0], size[1])
		if err == nil {
			if model != nil {
				_ = model.Close()
			}
			t.Errorf("newTerminalModel(%d, %d) succeeded, want error", size[0], size[1])
		}
	}
}

func TestTerminalModelConstructorRejectsOversizedDimensions(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name          string
		width, height int
	}{
		{"width above axis limit", 4097, 1},
		{"height above axis limit", 1, 4097},
		{"area above cell limit", 4096, 65},
		{"maximum integer width", maxInt, 1},
		{"maximum integer height", 1, maxInt},
		{"maximum integer area", maxInt, maxInt},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model, err, panicValue := newTerminalModelResult(test.width, test.height)
			if panicValue != nil {
				t.Fatalf("newTerminalModel(%d, %d) panicked: %v", test.width, test.height, panicValue)
			}
			if err == nil {
				if model != nil {
					_ = model.Close()
				}
				t.Fatalf("newTerminalModel(%d, %d) succeeded, want allocation-bound error", test.width, test.height)
			}
		})
	}
}

func TestTerminalModelConstructorRejectsReplyCloserCapabilityFailure(t *testing.T) {
	var actualReplyWriter io.Writer
	model, err := newTerminalModelWithReplyCloserAssertion(80, 24, func(writer io.Writer) bool {
		actualReplyWriter = writer
		return false
	})
	if err == nil {
		if model != nil {
			_ = model.Close()
		}
		t.Fatal("constructor succeeded after reply closer capability rejection")
	}
	if actualReplyWriter == nil {
		t.Fatal("capability assertion did not receive emulator InputPipe")
	}
	if _, ok := actualReplyWriter.(io.Closer); !ok {
		t.Fatalf("pinned emulator InputPipe type %T unexpectedly lacks io.Closer", actualReplyWriter)
	}
}

func TestTerminalModelFeedUsesXVTScreenSemantics(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		stream        string
		assert        func(*testing.T, terminalSnapshot)
	}{
		{
			name:  "printable text and style",
			width: 6, height: 2,
			stream: "\x1b[31mA\x1b[0mB",
			assert: func(t *testing.T, snapshot terminalSnapshot) {
				if a, b := snapshot.CellAt(0, 0), snapshot.CellAt(1, 0); a.Content != "A" || a.Style.Fg == nil || b.Content != "B" || b.Style.Fg != nil {
					t.Fatalf("styled cells = (%#v, %#v), want red A then unstyled B", a, b)
				}
			},
		},
		{
			name:  "cursor addressing",
			width: 6, height: 3,
			stream: "abc\x1b[2;4HZ",
			assert: func(t *testing.T, snapshot terminalSnapshot) {
				if got := snapshot.CellAt(3, 1).Content; got != "Z" {
					t.Fatalf("addressed cell = %q, want Z", got)
				}
				if snapshot.Cursor.X != 4 || snapshot.Cursor.Y != 1 {
					t.Fatalf("cursor = %v, want (4,1)", snapshot.Cursor)
				}
			},
		},
		{
			name:  "erase in line",
			width: 6, height: 2,
			stream: "abc\x1b[2D\x1b[K",
			assert: func(t *testing.T, snapshot terminalSnapshot) {
				if got := snapshotRow(snapshot, 0); got != "a     " {
					t.Fatalf("row = %q, want %q", got, "a     ")
				}
			},
		},
		{
			name:  "scroll at bottom margin",
			width: 4, height: 2,
			stream: "\x1b[1;1H1111\x1b[2;1H2222\x1b[2;1H\n3333",
			assert: func(t *testing.T, snapshot terminalSnapshot) {
				if top, bottom := snapshotRow(snapshot, 0), snapshotRow(snapshot, 1); top != "2222" || bottom != "3333" {
					t.Fatalf("rows = (%q, %q), want (2222, 3333)", top, bottom)
				}
			},
		},
		{
			name:  "autowrap",
			width: 3, height: 2,
			stream: "abcd",
			assert: func(t *testing.T, snapshot terminalSnapshot) {
				if top, bottom := snapshotRow(snapshot, 0), snapshotRow(snapshot, 1); top != "abc" || bottom != "d  " {
					t.Fatalf("rows = (%q, %q), want (abc, d__)", top, bottom)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newTerminalModelForTest(t, test.width, test.height)
			if err := model.Feed([]byte(test.stream)); err != nil {
				t.Fatalf("Feed: %v", err)
			}
			test.assert(t, model.Snapshot())
		})
	}
}

func TestTerminalModelFeedChunkBoundariesMatchOneShot(t *testing.T) {
	streams := []string{
		"plain text",
		"\x1b[31mstyled\x1b[0m\x1b[2;4Hcursor",
		"one\r\ntwo\x1b[1A\x1b[2Kerased",
		"wide: 界; combining: e\u0301",
	}
	for _, stream := range streams {
		wantModel := newTerminalModelForTest(t, 20, 4)
		if err := wantModel.Feed([]byte(stream)); err != nil {
			t.Fatalf("one-shot Feed(%q): %v", stream, err)
		}
		want := wantModel.Snapshot()

		for split := 0; split <= len(stream); split++ {
			gotModel := newTerminalModelForTest(t, 20, 4)
			if err := gotModel.Feed([]byte(stream[:split])); err != nil {
				t.Fatalf("Feed prefix at %d: %v", split, err)
			}
			if err := gotModel.Feed([]byte(stream[split:])); err != nil {
				t.Fatalf("Feed suffix at %d: %v", split, err)
			}
			if got := gotModel.Snapshot(); !reflect.DeepEqual(got, want) {
				t.Fatalf("stream %q split at %d produced\n%#v\nwant one-shot\n%#v", stream, split, got, want)
			}
		}

		bytewise := newTerminalModelForTest(t, 20, 4)
		for i := range []byte(stream) {
			if err := bytewise.Feed([]byte(stream)[i : i+1]); err != nil {
				t.Fatalf("bytewise Feed at %d: %v", i, err)
			}
		}
		if got := bytewise.Snapshot(); !reflect.DeepEqual(got, want) {
			t.Fatalf("stream %q bytewise snapshot differs from one-shot", stream)
		}
	}
}

func TestTerminalModelValidZWJSplitSnapshotsStayCoherent(t *testing.T) {
	const stream = "👩‍💻"
	oneShot := newTerminalModelForTest(t, 8, 2)
	if err := oneShot.Feed([]byte(stream)); err != nil {
		t.Fatal(err)
	}
	oneShotSnapshot := oneShot.Snapshot()

	split := newTerminalModelForTest(t, 8, 2)
	splitAt := len("👩")
	if err := split.Feed([]byte(stream[:splitAt])); err != nil {
		t.Fatal(err)
	}
	if err := split.Feed([]byte(stream[splitAt:])); err != nil {
		t.Fatal(err)
	}
	splitSnapshot := split.Snapshot()

	assertTerminalSnapshotCoherent(t, oneShotSnapshot)
	assertTerminalSnapshotCoherent(t, splitSnapshot)
	if reflect.DeepEqual(splitSnapshot, oneShotSnapshot) {
		t.Fatal("split and one-shot ZWJ snapshots unexpectedly match; regression no longer exercises x/vt's Write-boundary behavior")
	}
	if got := oneShotSnapshot.CellAt(0, 0).Content; got != stream {
		t.Fatalf("one-shot first cell = %q, want %q", got, stream)
	}
	if woman, laptop := splitSnapshot.CellAt(0, 0).Content, splitSnapshot.CellAt(2, 0).Content; woman != "👩" || laptop != "💻" {
		t.Fatalf("split cells = %q/%q, want separate woman/laptop glyphs", woman, laptop)
	}
}

func TestTerminalModelSnapshotCellsAreIndependent(t *testing.T) {
	model := newTerminalModelForTest(t, 4, 2)
	if err := model.Feed([]byte("A")); err != nil {
		t.Fatal(err)
	}

	first := model.Snapshot()
	first.CellAt(0, 0).Content = "mutated"
	if err := model.Feed([]byte("B")); err != nil {
		t.Fatal(err)
	}
	second := model.Snapshot()

	if got := second.CellAt(0, 0).Content; got != "A" {
		t.Fatalf("mutating first snapshot changed model cell to %q", got)
	}
	if got := first.CellAt(1, 0).Content; got != " " {
		t.Fatalf("later Feed changed first snapshot cell to %q", got)
	}
}

func TestTerminalModelControlObserverVisibilityTransitions(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   bool
	}{
		{"show authorizes visibility", "\x1b[?25h", true},
		{"hide revokes visibility", "\x1b[?25h\x1b[?25l", false},
		{"RIS revokes visibility", "\x1b[?25h\x1bc", false},
		{"1047 enter revokes visibility", "\x1b[?25h\x1b[?1047h", false},
		{"1047 leave revokes visibility", "\x1b[?25h\x1b[?1047l", false},
		{"1049 enter revokes visibility", "\x1b[?25h\x1b[?1049h", false},
		{"1049 leave revokes visibility", "\x1b[?25h\x1b[?1049l", false},
		{"later show reauthorizes replacement screen", "\x1b[?25h\x1b[?1049h\x1b[?25h", true},
		{"ordinary controls preserve authorization", "\x1b[?25h\x1b[31m\x1b[2;3H", true},
		{"embedded CSI control does not invent a transition", "\x1b[?25h\x1b[\x00A", true},
		{"nonexact cursor-mode control fails closed", "\x1b[?25h\x1b[?25;1000h", false},
		{"grouped hide revokes visibility", "\x1b[?25h\x1b[?1000;25l", false},
		{"grouped alt mode revokes visibility", "\x1b[?25h\x1b[?1000;1049h", false},
		{"grouped show does not authorize visibility", "\x1b[?1000;25h", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var observer terminalControlObserver
			observer.Feed([]byte(test.stream))
			if observer.visible != test.want {
				t.Fatalf("visible = %v, want %v", observer.visible, test.want)
			}
			if state := terminalObserverState(observer); state != "GroundState" {
				t.Fatalf("parser state = %s after complete stream", state)
			}
		})
	}
}

func TestTerminalModelControlObserverC1CSITransitions(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   bool
	}{
		{"show authorizes", "\x9b?25h", true},
		{"hide revokes", "\x1b[?25h\x9b?25l", false},
		{"1047 enter revokes", "\x1b[?25h\x9b?1047h", false},
		{"1047 leave revokes", "\x1b[?25h\x9b?1047l", false},
		{"1049 enter revokes", "\x1b[?25h\x9b?1049h", false},
		{"1049 leave revokes", "\x1b[?25h\x9b?1049l", false},
		{"grouped show stays fail closed", "\x9b?1000;25h", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var oneShot terminalControlObserver
			oneShot.Feed([]byte(test.stream))
			if oneShot.visible != test.want || terminalObserverState(oneShot) != "GroundState" {
				t.Fatalf("one-shot visible/state = %v/%s, want %v/GroundState", oneShot.visible, terminalObserverState(oneShot), test.want)
			}

			for split := 0; split <= len(test.stream); split++ {
				var splitObserver terminalControlObserver
				splitObserver.Feed([]byte(test.stream[:split]))
				splitObserver.Feed([]byte(test.stream[split:]))
				if !terminalObserversEqual(splitObserver, oneShot) {
					t.Fatalf("split %d visible/state = %v/%s, want one-shot %v/%s", split, splitObserver.visible, terminalObserverState(splitObserver), oneShot.visible, terminalObserverState(oneShot))
				}
			}
		})
	}
}

func TestTerminalModelC1CSIMatchesXVTSemantics(t *testing.T) {
	emulator := vt.NewEmulator(8, 3)
	var visibilityEvents []bool
	var altEvents []bool
	emulator.SetCallbacks(vt.Callbacks{
		CursorVisibility: func(visible bool) { visibilityEvents = append(visibilityEvents, visible) },
		AltScreen:        func(active bool) { altEvents = append(altEvents, active) },
	})
	if _, err := emulator.Write([]byte("\x9b?25l\x9b?25h\x9b?1049h\x9b?1049l")); err != nil {
		t.Fatal(err)
	}
	if len(visibilityEvents) < 2 || !reflect.DeepEqual(visibilityEvents[:2], []bool{false, true}) {
		t.Fatalf("x/vt C1 cursor callbacks = %v, want false then true", visibilityEvents)
	}
	if !reflect.DeepEqual(altEvents, []bool{true, false}) {
		t.Fatalf("x/vt C1 alt callbacks = %v, want enter then leave", altEvents)
	}
	visibilityEvents = nil
	altEvents = nil
	if _, err := emulator.Write([]byte("\xc2\x9b?25l\x1b]0;\x9b?25l\x07\x1bPq\x9b?1049h\x1b\\")); err != nil {
		t.Fatal(err)
	}
	if len(visibilityEvents) != 0 || len(altEvents) != 0 {
		t.Fatalf("x/vt treated UTF-8/string payload C1 bytes as controls: visibility=%v alt=%v", visibilityEvents, altEvents)
	}
	if err := emulator.Close(); err != nil {
		t.Fatal(err)
	}

	model := newTerminalModelForTest(t, 8, 3)
	for _, step := range []struct {
		stream    string
		visible   bool
		alternate bool
	}{
		{"\x9b?25h", true, false},
		{"\x9b?25l", false, false},
		{"\x9b?25h\x9b?1049h", false, true},
		{"\x9b?25h", true, true},
		{"\x9b?1049l", false, false},
	} {
		if err := model.Feed([]byte(step.stream)); err != nil {
			t.Fatal(err)
		}
		snapshot := model.Snapshot()
		if snapshot.CursorVisible != step.visible || snapshot.AltScreen != step.alternate {
			t.Fatalf("after %q state = visible:%v alt:%v, want visible:%v alt:%v", step.stream, snapshot.CursorVisible, snapshot.AltScreen, step.visible, step.alternate)
		}
	}
}

func TestTerminalModelControlObserverIgnoresC1BytesOutsideGround(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   bool
	}{
		{"UTF-8 C1-looking show is text", "\xc2\x9b?25h", false},
		{"UTF-8 C1-looking hide preserves show", "\x1b[?25h\xc2\x9b?25l", true},
		{"OSC payload show is data", "\x1b]0;title\x9b?25h\x07", false},
		{"OSC payload hide preserves show", "\x1b[?25h\x1b]0;title\x9b?25l\x07", true},
		{"C1 OSC payload show is data", "\x9d0;title\x9b?25h\x9c", false},
		{"DCS payload show is data", "\x1bPqpayload\x9b?25h\x1b\\", false},
		{"DCS payload hide preserves show", "\x1b[?25h\x1bPqpayload\x9b?25l\x1b\\", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var oneShot terminalControlObserver
			oneShot.Feed([]byte(test.stream))
			if oneShot.visible != test.want {
				t.Fatalf("one-shot visible = %v, want %v", oneShot.visible, test.want)
			}
			for split := 0; split <= len(test.stream); split++ {
				var splitObserver terminalControlObserver
				splitObserver.Feed([]byte(test.stream[:split]))
				splitObserver.Feed([]byte(test.stream[split:]))
				if splitObserver.visible != oneShot.visible {
					t.Fatalf("split %d visible = %v, want one-shot %v", split, splitObserver.visible, oneShot.visible)
				}
			}
		})
	}
}

func TestTerminalModelControlObserverIsChunkEquivalent(t *testing.T) {
	streams := []string{
		"plain\x1b[?25h\x1b[31mstyled",
		"\x1b[?25h\x1b[?25l\x1b[?25h",
		"\x1b[?25h\x1b[?1049hdraw\x1b[?25h\x1b[?1049l",
		"\x1b[?25h\x1bc\x1b[?25h",
		"\x1b[?25h\x1b[\x00A",
	}

	for _, stream := range streams {
		var want terminalControlObserver
		want.Feed([]byte(stream))

		for split := 0; split <= len(stream); split++ {
			var got terminalControlObserver
			got.Feed([]byte(stream[:split]))
			got.Feed([]byte(stream[split:]))
			if !terminalObserversEqual(got, want) {
				t.Fatalf("stream %q split at %d: visible/state = %v/%s, want %v/%s", stream, split, got.visible, terminalObserverState(got), want.visible, terminalObserverState(want))
			}
		}

		var bytewise terminalControlObserver
		for _, b := range []byte(stream) {
			bytewise.Feed([]byte{b})
		}
		if !terminalObserversEqual(bytewise, want) {
			t.Fatalf("stream %q bytewise visible/state = %v/%s, want %v/%s", stream, bytewise.visible, terminalObserverState(bytewise), want.visible, terminalObserverState(want))
		}
	}
}

func TestTerminalModelControlObserverBoundsStringStorage(t *testing.T) {
	small := []byte("\x1b]0;payload\x9b?25h\x07")
	large := []byte("\x1b]0;" + strings.Repeat("x", 1<<20) + "\x9b?25h\x07")
	allocations := func(stream []byte) float64 {
		return testing.AllocsPerRun(10, func() {
			var observer terminalControlObserver
			observer.Feed(stream)
			if observer.visible {
				panic("OSC payload authorized visibility")
			}
		})
	}
	if smallAllocs, largeAllocs := allocations(small), allocations(large); largeAllocs > smallAllocs+1 {
		t.Fatalf("large OSC allocations = %.0f, small = %.0f; parser storage is not bounded", largeAllocs, smallAllocs)
	}
}

func TestTerminalModelControlObserverBoundsCompleteControlEqually(t *testing.T) {
	var overflowed terminalControlObserver
	overflowed.Feed([]byte("\x1b[?18446744073709551641h")) // 2^64 + 25 wraps to 25 in an int.
	if overflowed.visible {
		t.Fatal("overflowed numeric parameter authorized visibility")
	}

	stream := "\x1b[?" + strings.Repeat("1;", terminalControlParamsMax+4) + "25h"
	var oneShot terminalControlObserver
	oneShot.Feed([]byte(stream))
	if oneShot.visible {
		t.Fatal("oversized parameter list authorized visibility")
	}

	var split terminalControlObserver
	for _, b := range []byte(stream) {
		split.Feed([]byte{b})
	}
	if !terminalObserversEqual(split, oneShot) {
		t.Fatalf("bytewise visible/state = %v/%s, want one-shot %v/%s", split.visible, terminalObserverState(split), oneShot.visible, terminalObserverState(oneShot))
	}
}

func TestTerminalModelCursorVisibilityComesOnlyFromExplicitControls(t *testing.T) {
	model := newTerminalModelForTest(t, 8, 3)
	for _, step := range []struct {
		stream string
		want   bool
	}{
		{"text and cursor movement\x1b[2;2H", false},
		{"\x1b[?25h", true},
		{"\x1b[31mstyle\x1b[0m", true},
		{"\x1b[?1049h", false},
		{"\x1b[?25h", true},
		{"\x1b[?1049l", false},
	} {
		if err := model.Feed([]byte(step.stream)); err != nil {
			t.Fatalf("Feed(%q): %v", step.stream, err)
		}
		if got := model.Snapshot().CursorVisible; got != step.want {
			t.Fatalf("after %q CursorVisible = %v, want %v", step.stream, got, step.want)
		}
	}
}

func TestTerminalModelResizePublishesBoundedXVTState(t *testing.T) {
	model := newTerminalModelForTest(t, 5, 2)
	if err := model.Feed([]byte("abcde\r\nfghij\x1b[?25h")); err != nil {
		t.Fatal(err)
	}
	if err := model.Resize(3, 4); err != nil {
		t.Fatalf("Resize: %v", err)
	}

	snapshot := model.Snapshot()
	if snapshot.Width != 3 || snapshot.Height != 4 {
		t.Fatalf("dimensions = %dx%d, want 3x4", snapshot.Width, snapshot.Height)
	}
	if snapshot.Cursor.X < 0 || snapshot.Cursor.X >= snapshot.Width || snapshot.Cursor.Y < 0 || snapshot.Cursor.Y >= snapshot.Height {
		t.Fatalf("cursor %v outside resized bounds %dx%d", snapshot.Cursor, snapshot.Width, snapshot.Height)
	}
	if len(snapshot.Cells) != 12 || snapshot.CellAt(2, 3) == nil || snapshot.CellAt(3, 3) != nil {
		t.Fatalf("resized cell grid is not 3x4: len=%d", len(snapshot.Cells))
	}
	if !snapshot.CursorVisible {
		t.Fatal("resize discarded explicit cursor visibility without replacing the active screen")
	}
}

func TestTerminalModelResizeRejectsInvalidDimensions(t *testing.T) {
	model := newTerminalModelForTest(t, 5, 2)
	before := model.Snapshot()
	for _, size := range [][2]int{{0, 1}, {1, 0}, {-1, 1}, {1, -1}} {
		if err := model.Resize(size[0], size[1]); err == nil {
			t.Errorf("Resize(%d, %d) succeeded, want error", size[0], size[1])
		}
		if got := model.Snapshot(); got.Width != before.Width || got.Height != before.Height {
			t.Fatalf("failed resize changed dimensions to %dx%d", got.Width, got.Height)
		}
	}
}

func TestTerminalModelResizeRejectsOversizedDimensionsWithoutMutation(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name       string
		cols, rows int
	}{
		{"width above axis limit", 4097, 1},
		{"height above axis limit", 1, 4097},
		{"area above cell limit", 4096, 65},
		{"maximum integer width", maxInt, 1},
		{"maximum integer height", 1, maxInt},
		{"maximum integer area", maxInt, maxInt},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newTerminalModelForTest(t, 6, 3)
			if err := model.Feed([]byte("primary\x1b[?1049h\x1b[2;3H\x1b[31mZ\x1b[?25h")); err != nil {
				t.Fatal(err)
			}
			before := model.Snapshot()

			err, panicValue := terminalModelResizeResult(model, test.cols, test.rows)
			if panicValue != nil {
				t.Fatalf("Resize(%d, %d) panicked: %v", test.cols, test.rows, panicValue)
			}
			if err == nil {
				t.Fatalf("Resize(%d, %d) succeeded, want allocation-bound error", test.cols, test.rows)
			}
			if got := model.Snapshot(); !reflect.DeepEqual(got, before) {
				t.Fatalf("rejected Resize(%d, %d) changed snapshot:\n got %#v\nwant %#v", test.cols, test.rows, got, before)
			}
		})
	}
}

func TestTerminalModelSnapshotTracksActiveScreen(t *testing.T) {
	for _, mode := range []string{"1047", "1049"} {
		t.Run(mode, func(t *testing.T) {
			model := newTerminalModelForTest(t, 4, 2)
			if err := model.Feed([]byte("M\x1b[?25h")); err != nil {
				t.Fatal(err)
			}
			if err := model.Feed([]byte("\x1b[?" + mode + "h")); err != nil {
				t.Fatal(err)
			}
			alternate := model.Snapshot()
			if !alternate.AltScreen || alternate.CursorVisible {
				t.Fatalf("alternate identity/visibility = (%v,%v), want (true,false)", alternate.AltScreen, alternate.CursorVisible)
			}
			if got := alternate.CellAt(0, 0).Content; got != " " {
				t.Fatalf("new alternate screen retained primary cell %q", got)
			}

			if err := model.Feed([]byte("A\x1b[?25h")); err != nil {
				t.Fatal(err)
			}
			if snapshot := model.Snapshot(); !snapshot.CursorVisible || snapshot.CellAt(0, 0).Content != "A" {
				t.Fatalf("alternate update = %#v", snapshot)
			}

			if err := model.Feed([]byte("\x1b[?" + mode + "l")); err != nil {
				t.Fatal(err)
			}
			primary := model.Snapshot()
			if primary.AltScreen || primary.CursorVisible {
				t.Fatalf("primary identity/visibility = (%v,%v), want (false,false)", primary.AltScreen, primary.CursorVisible)
			}
			if got := primary.CellAt(0, 0).Content; got != "M" {
				t.Fatalf("restored primary cell = %q, want M", got)
			}
		})
	}
}

func TestTerminalModelDrainsRepliesAndClosesDeterministically(t *testing.T) {
	model, err := newTerminalModel(20, 4)
	if err != nil {
		t.Fatal(err)
	}

	feedDone := make(chan error, 1)
	go func() {
		feedDone <- model.Feed([]byte("\x1b[5n\x1b[6n\x1b[c"))
	}()
	select {
	case err := <-feedDone:
		if err != nil {
			t.Fatalf("reply-producing Feed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reply-producing Feed blocked; emulator replies are not being drained")
	}

	closeDone := make(chan error, 1)
	go func() { closeDone <- model.Close() }()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close blocked waiting for reply drainer")
	}

	if err := model.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if err := model.Feed([]byte("after close")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Feed after close = %v, want io.ErrClosedPipe", err)
	}
	if err := model.Resize(10, 2); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Resize after close = %v, want io.ErrClosedPipe", err)
	}
	if err := model.Resize(0, 0); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("invalid Resize after close = %v, want closed-state precedence", err)
	}
}

func TestTerminalModelSnapshotAfterCloseIsFinalAndIndependent(t *testing.T) {
	model, err := newTerminalModel(4, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Feed([]byte("AB\x1b[?25h")); err != nil {
		t.Fatal(err)
	}
	want := model.Snapshot()
	if err := model.Close(); err != nil {
		t.Fatal(err)
	}

	first := model.Snapshot()
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("snapshot after close = %#v, want final pre-close %#v", first, want)
	}
	first.CellAt(0, 0).Content = "mutated"
	second := model.Snapshot()
	if got := second.CellAt(0, 0).Content; got != "A" {
		t.Fatalf("post-close snapshot mutation changed final state to %q", got)
	}
}

func TestTerminalModelConcurrentCloseIsIdempotent(t *testing.T) {
	model, err := newTerminalModel(20, 4)
	if err != nil {
		t.Fatal(err)
	}

	const callers = 16
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- model.Close()
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent Close calls blocked")
	}
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("Close: %v", err)
		}
	}
}

func TestTerminalModelConcurrentCloseWaitsForSharedResult(t *testing.T) {
	shutdownStarted := make(chan struct{})
	releaseShutdown := make(chan struct{})
	shutdownErr := errors.New("injected reply closer error")
	model, err := newTerminalModel(8, 3)
	if err != nil {
		t.Fatal(err)
	}
	actualReplyCloser := model.replyCloser
	model.replyCloser = closeFunc(func() error {
		close(shutdownStarted)
		<-releaseShutdown
		if err := actualReplyCloser.Close(); err != nil {
			return err
		}
		return shutdownErr
	})
	if err := model.Feed([]byte("AB\x1b[?25h")); err != nil {
		t.Fatal(err)
	}
	wantSnapshot := model.Snapshot()

	firstClose := make(chan error, 1)
	go func() { firstClose <- model.Close() }()
	select {
	case <-shutdownStarted:
	case <-time.After(time.Second):
		t.Fatal("Close did not reach coupled reply closer")
	}

	const followers = 8
	followerResults := make(chan error, followers)
	for range followers {
		go func() { followerResults <- model.Close() }()
	}
	feedResult := make(chan error, 1)
	go func() { feedResult <- model.Feed([]byte("after closing")) }()
	resizeResult := make(chan error, 1)
	go func() { resizeResult <- model.Resize(4, 2) }()
	snapshotResult := make(chan terminalSnapshot, 1)
	go func() { snapshotResult <- model.Snapshot() }()

	select {
	case err := <-feedResult:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("Feed while closing = %v, want io.ErrClosedPipe", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Feed blocked behind shutdown")
	}
	select {
	case err := <-resizeResult:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("Resize while closing = %v, want io.ErrClosedPipe", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Resize blocked behind shutdown")
	}
	select {
	case snapshot := <-snapshotResult:
		if !reflect.DeepEqual(snapshot, wantSnapshot) {
			t.Errorf("Snapshot while closing = %#v, want final %#v", snapshot, wantSnapshot)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Snapshot blocked behind shutdown")
	}
	select {
	case err := <-followerResults:
		t.Errorf("concurrent Close returned before shutdown completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseShutdown)
	select {
	case err := <-firstClose:
		if !errors.Is(err, shutdownErr) {
			t.Errorf("first Close = %v, want stored shutdown error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first Close did not complete after release")
	}
	for range followers {
		select {
		case err := <-followerResults:
			if !errors.Is(err, shutdownErr) {
				t.Errorf("follower Close = %v, want stored shutdown error", err)
			}
		case <-time.After(time.Second):
			t.Fatal("follower Close did not complete after release")
		}
	}
	if err := model.Close(); !errors.Is(err, shutdownErr) {
		t.Fatalf("repeated Close = %v, want stored shutdown error", err)
	}
}

func TestTerminalModelConcurrentAPICompletesUnderDeadline(t *testing.T) {
	model, err := newTerminalModel(20, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer model.Close() //nolint:errcheck

	const workers = 12
	start := make(chan struct{})
	unexpected := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for step := 0; step < 200; step++ {
				if worker == 0 && step == 50 {
					if err := model.Close(); err != nil {
						unexpected <- fmt.Errorf("Close: %w", err)
						return
					}
					continue
				}
				switch (worker + step) % 3 {
				case 0:
					err := model.Feed([]byte("x\x1b[31mY\x1b[0m\x1b[6n"))
					if err != nil && !errors.Is(err, io.ErrClosedPipe) {
						unexpected <- fmt.Errorf("Feed: %w", err)
						return
					}
				case 1:
					err := model.Resize(10+(step%11), 3+(step%5))
					if err != nil && !errors.Is(err, io.ErrClosedPipe) {
						unexpected <- fmt.Errorf("Resize: %w", err)
						return
					}
				case 2:
					snapshot := model.Snapshot()
					if snapshot.Width <= 0 || snapshot.Height <= 0 || len(snapshot.Cells) != snapshot.Width*snapshot.Height {
						unexpected <- fmt.Errorf("incoherent snapshot dimensions %dx%d with %d cells", snapshot.Width, snapshot.Height, len(snapshot.Cells))
						return
					}
					if snapshot.Cursor.X < 0 || snapshot.Cursor.X >= snapshot.Width || snapshot.Cursor.Y < 0 || snapshot.Cursor.Y >= snapshot.Height {
						unexpected <- fmt.Errorf("cursor %v outside %dx%d snapshot", snapshot.Cursor, snapshot.Width, snapshot.Height)
						return
					}
				}
			}
		}()
	}
	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent terminal API blocked")
	}
	close(unexpected)
	for err := range unexpected {
		t.Error(err)
	}
}

func FuzzTerminalModelFeedChunkPartitions(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("plain\r\ntext"),
		[]byte("\x1b[31mstyled\x1b[0m\x1b[2;3Hcursor\x1b[K"),
		[]byte("wide 界 and e\u0301\x1b[?1049halt\x1b[?1049l"),
		[]byte("\x9b?25hC1 visible\x9b?1049halt\x9b?1049l\x9b?25l"),
		[]byte("\xc2\x9b?25h\x1b]0;\x9b?25l\x07\x1bPq\x9b?1049h\x1b\\"),
		[]byte("\x1b[5n\x1b[6n\x1b[c"),
	} {
		f.Add(seed, []byte{1, 2, 3, 5, 8})
	}
	// x/vt flushes graphemes at Write boundaries. Malformed UTF-8 and valid
	// extended graphemes must both produce safe snapshots for every partition.
	f.Add([]byte("00000\xe000ׅ"), []byte("A"))
	f.Add([]byte("👩‍💻"), []byte{3})

	f.Fuzz(func(t *testing.T, raw, chunkPlan []byte) {
		if len(raw) > 2048 {
			raw = raw[:2048]
		}
		chunked, err := newTerminalModel(20, 6)
		if err != nil {
			t.Fatal(err)
		}
		for offset, planIndex := 0, 0; offset < len(raw); planIndex++ {
			size := 1
			if len(chunkPlan) > 0 {
				size += int(chunkPlan[planIndex%len(chunkPlan)]) % 31
			}
			end := min(offset+size, len(raw))
			if err := chunked.Feed(raw[offset:end]); err != nil {
				t.Fatal(err)
			}
			offset = end
		}
		got := chunked.Snapshot()
		assertTerminalSnapshotCoherent(t, got)
		original := got.CellAt(0, 0).Content
		got.CellAt(0, 0).Content = original + "\x00snapshot mutation"
		fresh := chunked.Snapshot()
		assertTerminalSnapshotCoherent(t, fresh)
		if fresh.CellAt(0, 0).Content != original {
			t.Fatalf("mutating snapshot changed model cell from %q to %q", original, fresh.CellAt(0, 0).Content)
		}
		if err := chunked.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzTerminalModelControlObserverChunkPartitions(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("\x1b[?25h\x1b[31mtext\x1b[?25l"),
		[]byte("\x1b[?25h\x1b[?1049h\x1b[?25h\x1b[?1049l"),
		[]byte("\x9b?25h\x9b?1049h\x9b?25h\x9b?1049l"),
		[]byte("\x9b?1000;25h\x9b?25l"),
		[]byte("\xc2\x9b?25h\x1b]0;\x9b?25l\x07\x1bPq\x9b?1049h\x1b\\"),
		[]byte("\x1b[?25h\x1bc"),
		[]byte("\x1b[?25h\x1b[\x00A"),
		[]byte("\x1b[?" + strings.Repeat("1;", terminalControlParamsMax+4) + "25h"),
	} {
		f.Add(seed, []byte{1, 2, 7})
	}

	f.Fuzz(func(t *testing.T, raw, chunkPlan []byte) {
		if len(raw) > 2048 {
			raw = raw[:2048]
		}
		var want terminalControlObserver
		want.Feed(raw)

		var got terminalControlObserver
		for offset, planIndex := 0, 0; offset < len(raw); planIndex++ {
			size := 1
			if len(chunkPlan) > 0 {
				size += int(chunkPlan[planIndex%len(chunkPlan)]) % 31
			}
			end := min(offset+size, len(raw))
			got.Feed(raw[offset:end])
			offset = end
		}
		if !terminalObserversEqual(got, want) {
			t.Fatalf("chunked visible/state %v/%s differs from one-shot %v/%s for raw %q and plan %v", got.visible, terminalObserverState(got), want.visible, terminalObserverState(want), raw, chunkPlan)
		}
	})
}

func snapshotRow(snapshot terminalSnapshot, y int) string {
	var row strings.Builder
	for x := 0; x < snapshot.Width; x++ {
		row.WriteString(snapshot.CellAt(x, y).Content)
	}
	return row.String()
}

func assertTerminalSnapshotCoherent(t testing.TB, snapshot terminalSnapshot) {
	t.Helper()
	if snapshot.Width <= 0 || snapshot.Height <= 0 || len(snapshot.Cells) != snapshot.Width*snapshot.Height {
		t.Fatalf("incoherent snapshot dimensions %dx%d with %d cells", snapshot.Width, snapshot.Height, len(snapshot.Cells))
	}
	if snapshot.Cursor.X < 0 || snapshot.Cursor.X >= snapshot.Width || snapshot.Cursor.Y < 0 || snapshot.Cursor.Y >= snapshot.Height {
		t.Fatalf("cursor %v outside %dx%d snapshot", snapshot.Cursor, snapshot.Width, snapshot.Height)
	}
	if snapshot.CellAt(0, 0) == nil || snapshot.CellAt(snapshot.Width-1, snapshot.Height-1) == nil || snapshot.CellAt(snapshot.Width, snapshot.Height) != nil {
		t.Fatal("snapshot cell access is not bounds-safe")
	}
}

func newTerminalModelForTest(t testing.TB, width, height int) *terminalModel {
	t.Helper()
	model, err := newTerminalModel(width, height)
	if err != nil {
		t.Fatalf("newTerminalModel(%d, %d): %v", width, height, err)
	}
	t.Cleanup(func() {
		if err := model.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return model
}

func terminalObserverState(observer terminalControlObserver) string {
	if observer.parser == nil {
		return "GroundState"
	}
	return observer.parser.StateName()
}

func terminalObserversEqual(left, right terminalControlObserver) bool {
	return left.visible == right.visible && terminalObserverState(left) == terminalObserverState(right)
}

func newTerminalModelResult(width, height int) (model *terminalModel, err error, panicValue any) {
	defer func() { panicValue = recover() }()
	model, err = newTerminalModel(width, height)
	return model, err, nil
}

func terminalModelResizeResult(model *terminalModel, cols, rows int) (err error, panicValue any) {
	defer func() { panicValue = recover() }()
	err = model.Resize(cols, rows)
	return err, nil
}

type closeFunc func() error

func (close closeFunc) Close() error { return close() }
