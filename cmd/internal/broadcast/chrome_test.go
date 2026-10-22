package broadcast

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestChildCannotAuthorizeBroadcast(t *testing.T) {
	f := textFrame(t, 30, 4, "body", liveChrome(PointerSGR+PointerLabel+"\x1b[0m"))
	// Reconstruct the same pixels as an endpoint/panel, without trusted chrome.
	child, err := terminal.PanelFrame(f.Geometry, f.Cells, f.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	for _, rows := range []int{0, -1, f.Geometry.Rows, f.Geometry.Rows + 1} {
		child.ChromeRows = rows
		if IndicatorShown(child) || PointerShown(child) {
			t.Fatalf("child pixels authorized controls with invalid chrome %d", rows)
		}
	}
}

func TestParseNoteDoesNotAssumeControlRow(t *testing.T) {
	if _, err := ParseNote([]byte(`{"cols":30,"rows":4,"box":[0,3,2,1],"color":0,"text":"second chrome row"}`)); err != nil {
		t.Fatalf("structural parser guessed protected row: %v", err)
	}
}

func TestWrappedChromeBroadcastSeams(t *testing.T) {
	body, err := terminal.StyledRows("FLEET-SECRET", 40, 4)
	if err != nil {
		t.Fatal(err)
	}
	child, err := terminal.PanelFrame(terminal.Geometry{Cols: 40, Rows: 4}, body, terminal.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	chrome, err := terminal.StyledRows(liveChrome(PointerSGR+PointerLabel+"\x1b[0m")+"\nsecond row tabs", 40, 2)
	if err != nil {
		t.Fatal(err)
	}
	f, err := terminal.Compose(child, terminal.Geometry{Cols: 40, Rows: 6}, chrome)
	if err != nil {
		t.Fatal(err)
	}
	f.Rows[4].Wrapped = true
	if !IndicatorShown(f) || !PointerShown(f) {
		t.Fatal("first chrome row controls not recognized")
	}
	private, err := ViewerFrame(f, terminal.FramePrivate, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(frameText(t, private), "FLEET-SECRET") {
		t.Fatal("private body leaked")
	}
	if len(private.Rows) != len(f.Rows) || !reflect.DeepEqual(private.Rows[4:], f.Rows[4:]) {
		t.Fatal("chrome row metadata lost")
	}
	if private.ChromeRows != 2 {
		t.Fatal("private placeholder lost chrome metadata")
	}
	for i := 4 * 40; i < len(f.Cells); i++ {
		if !private.Cells[i].Equal(&f.Cells[i]) {
			t.Fatalf("chrome cell %d replaced", i)
		}
	}
	if !IndicatorShown(private) || !PointerShown(private) {
		t.Fatal("private frame controls lost")
	}
	var stream Stream
	m, ok, err := stream.Next(f)
	if err != nil || !ok || m.ChromeRows != 2 {
		t.Fatalf("initial stream: %+v %v %v", m, ok, err)
	}
	joined, ok, err := stream.Join()
	if err != nil || !ok || joined.ChromeRows != 2 {
		t.Fatalf("late join: %+v %v %v", joined, ok, err)
	}
	changed := f.Clone()
	changed.ChromeRows = 1
	m, ok, err = stream.Next(changed)
	if err != nil || !ok || m.ChromeRows != 1 || len(m.Data) != 0 {
		t.Fatalf("metadata-only stream: %+v %v %v", m, ok, err)
	}
}

func TestChromeGuardsMoveAndReleaseSecondRow(t *testing.T) {
	f := textFrame(t, 40, 6, "body", liveChrome("tabs"))
	f.ChromeRows = 2
	now := time.Now()
	marks := NewMarks()
	marks.Add([][2]int{{0, 4}}, 40, 6, 2, 0, now)
	if len(marks.cells) != 0 {
		t.Fatal("first chrome row admitted a mark")
	}
	marks.Add([][2]int{{0, 5}}, 40, 6, 2, 0, now)
	if len(marks.cells) != 1 {
		t.Fatal("second chrome row refused a mark")
	}
	got := marks.Overlay(f, now)
	if got.Cells[200].Style.Bg == f.Cells[200].Style.Bg {
		t.Fatal("second chrome row was guarded")
	}
	f.ChromeRows = 1
	got = marks.Overlay(f, now)
	if !got.Cells[200].Equal(&f.Cells[200]) {
		t.Fatal("existing mark covered moved controls")
	}
	var board NoteBoard
	if _, err := board.Add(Note{Box: CellBox{0, 4, 2, 1}, Text: "control"}, 40, 6, 2); err == nil {
		t.Fatal("note on controls admitted")
	}
	n, err := board.Add(Note{Box: CellBox{0, 5, 2, 1}, Text: "tabs"}, 40, 6, 2)
	if err != nil || !NoteVisible(n, 40, 6, 2) || NoteVisible(n, 40, 6, 1) {
		t.Fatalf("note guard movement: %+v %v", n, err)
	}
}

func TestTintWideCellCannotReachControls(t *testing.T) {
	f := textFrame(t, 40, 6, "body", "tabs")
	f.ChromeRows = 2
	// Hostile wide character straddles the last protected cell.
	i := 4*40 + StatusGuardCols - 1
	f.Cells[i] = terminal.Cell{Content: "界", Width: 2}
	f.Cells[i+1] = terminal.Cell{}
	marks := NewMarks()
	now := time.Now()
	marks.Add([][2]int{{StatusGuardCols, 4}}, 40, 6, 2, 0, now)
	got := marks.Overlay(f, now)
	if !got.Cells[i].Equal(&f.Cells[i]) {
		t.Fatal("wide continuation tinted a protected control cell")
	}
}

func TestViewerChromeNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	src := `import assert from 'node:assert/strict';
import { guardedCell, noteVisible, boxHasPin } from "../broadcast/web/notes.js";
import { frameChromeRows } from "../broadcast/web/viewer.js";
assert.equal(frameChromeRows({rows:6}),1);
assert.equal(frameChromeRows({rows:6,chrome_rows:2}),2);
for (const bad of [0,-1,6,7,1.5,'2',null]) assert.equal(frameChromeRows({rows:6,chrome_rows:bad}),0);
assert.equal(guardedCell(0,4,6,9,2),true);
assert.equal(guardedCell(0,5,6,9,2),false);
assert.equal(noteVisible({col:0,row:5},40,6,9,2),true);
assert.equal(boxHasPin([0,4,2,1],40,6,9,2),false);
for (const bad of [0,-1,6,7,1.5,'2',null]) {
 assert.equal(noteVisible({col:0,row:0},40,6,9,bad),false,JSON.stringify(bad));
}
assert.equal(guardedCell(0,5,6,9),true);
`
	out, err := exec.Command(node, "--input-type=module", "-e", src).CombinedOutput()
	if err != nil {
		t.Fatalf("node chrome guards: %v\n%s", err, out)
	}
}

func TestChromeServerAdmissionAndMetadataEvents(t *testing.T) {
	p, link := notesOn(t)
	f := textFrame(t, 40, 4, "body\n\n"+liveChrome(PointerSGR+PointerLabel+"\x1b[0m"), liveChrome(PointerSGR+PointerLabel+"\x1b[0m"))
	f.ChromeRows = 2
	offer(p.s.hub, f, terminal.FramePublic)
	if cur := p.s.hub.Current(); cur.ChromeRows != 2 || !cur.PointerShown {
		t.Fatalf("current lost chrome: %+v", cur)
	}
	r := stream(t, link)
	readFrame := func() wireFrame {
		t.Helper()
		var m wireFrame
		if err := json.Unmarshal([]byte(readEvent(t, r, "frame")), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	if m := readFrame(); m.ChromeRows != 2 {
		t.Fatalf("join frame: %+v", m)
	}
	if code, body := post(t, link+"note", "application/json", noteBody(0, 2, 0, "controls")); code != 400 {
		t.Fatalf("current controls admitted: %d %q", code, body)
	}
	if code, body := post(t, link+"note", "application/json", noteBody(0, 3, 0, "second chrome row")); code != 204 {
		t.Fatalf("second row refused: %d %q", code, body)
	}
	for _, route := range []string{"point", "note"} {
		body := `{"cols":40,"rows":4,"down":false,"points":[[0,0]],"chrome_rows":1}`
		if route == "note" {
			body = `{"cols":40,"rows":4,"box":[0,2,2,1],"color":0,"text":"spoof","chrome_rows":1}`
		}
		if code, _ := post(t, link+route, "application/json", body); code != 400 {
			t.Fatalf("%s accepted client chrome authority: %d", route, code)
		}
	}
	if code, _ := post(t, link+"point", "application/json", tapBody); code != 204 {
		t.Fatalf("point: %d", code)
	}
	batches := p.batches()
	if len(batches) != 1 || batches[0].ChromeRows != 2 {
		t.Fatalf("point lost trusted authority: %+v", batches)
	}
	f.ChromeRows = 1
	offer(p.s.hub, f, terminal.FramePublic)
	if m := readFrame(); m.ChromeRows != 1 || m.B != "" {
		t.Fatalf("metadata-only frame: %+v", m)
	}
	var late wireFrame
	if err := json.Unmarshal([]byte(readEvent(t, stream(t, link), "frame")), &late); err != nil || late.ChromeRows != 1 {
		t.Fatalf("late metadata: %+v %v", late, err)
	}
	if notes, _ := p.s.Notes(); len(notes) != 1 || notes[0].On.Rows != 4 || NoteVisible(notes[0], 40, 4, 1) {
		t.Fatalf("stored grid/current guards: %+v", notes)
	}
}

func TestNoteChipCannotBlankWideControlNeighbor(t *testing.T) {
	f := textFrame(t, 40, 6, "body", "tabs")
	f.ChromeRows = 2
	i := 4*40 + StatusGuardCols - 1
	f.Cells[i] = terminal.Cell{Content: "界", Width: 2}
	f.Cells[i+1] = terminal.Cell{}
	n := Note{Col: StatusGuardCols, Row: 4, Box: CellBox{StatusGuardCols, 4, 2, 1}}
	got := DrawNotes(f, []Note{n}, NoteStyle{})
	if !got.Cells[i].Equal(&f.Cells[i]) {
		t.Fatal("note blanked protected wide control neighbor")
	}
}
