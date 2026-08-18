package wrapcmd

import (
	"os"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
)

type agyCell struct {
	x, y int
	text string
}

func TestAgyComposerActive(t *testing.T) {
	border := func(y, start, length int) []agyCell {
		cells := make([]agyCell, length)
		for offset := range length {
			cells[offset] = agyCell{x: start + offset, y: y, text: "─"}
		}
		return cells
	}
	box := func(top, bottom, start, length, promptX int) []agyCell {
		cells := append(border(top, start, length), border(bottom, start, length)...)
		return append(cells, agyCell{x: promptX, y: top + 1, text: ">"})
	}
	join := func(groups ...[]agyCell) []agyCell {
		var cells []agyCell
		for _, group := range groups {
			cells = append(cells, group...)
		}
		return cells
	}

	tests := []struct {
		name    string
		width   int
		height  int
		cursor  uv.Position
		visible bool
		cells   []agyCell
		want    bool
	}{
		{name: "observed startup geometry", width: 80, height: 38, cursor: uv.Position{X: 2, Y: 30}, visible: true, cells: box(29, 31, 0, 60, 0), want: true},
		{name: "multiline composer", width: 80, height: 38, cursor: uv.Position{X: 8, Y: 12}, visible: true, cells: box(9, 13, 0, 40, 0), want: true},
		{name: "minimum border length", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: box(2, 4, 0, 5, 0), want: true},
		{name: "overlapping offset spans", width: 20, height: 10, cursor: uv.Position{X: 5, Y: 3}, visible: true, cells: join(border(2, 0, 8), border(4, 4, 8), []agyCell{{x: 4, y: 3, text: ">"}}), want: true},
		{name: "maximum box height", width: 40, height: 32, cursor: uv.Position{X: 3, Y: 20}, visible: true, cells: box(2, 27, 0, 20, 0), want: true},
		{name: "last anchored prompt column", width: 20, height: 10, cursor: uv.Position{X: 5, Y: 3}, visible: true, cells: box(2, 4, 0, 10, 5), want: true},
		{name: "intervening border glyphs do not hide coherent pair", width: 30, height: 12, cursor: uv.Position{X: 3, Y: 6}, visible: true, cells: join(box(2, 8, 0, 10, 0), border(5, 15, 5)), want: true},
		{name: "hidden cursor", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, cells: box(2, 4, 0, 10, 0)},
		{name: "lone prompt", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: []agyCell{{x: 0, y: 3, text: ">"}}},
		{name: "lone divider", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: border(2, 0, 10)},
		{name: "borders without prompt", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: join(border(2, 0, 10), border(4, 0, 10))},
		{name: "prompt and bottom without top", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: join(border(4, 0, 10), []agyCell{{x: 0, y: 3, text: ">"}})},
		{name: "short borders", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: box(2, 4, 0, 4, 0)},
		{name: "separated glyphs are not cumulative", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: join(border(2, 0, 3), border(2, 4, 3), border(4, 0, 3), border(4, 4, 3), []agyCell{{x: 0, y: 3, text: ">"}})},
		{name: "non-overlapping borders", width: 30, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: join(border(2, 0, 8), border(4, 12, 8), []agyCell{{x: 0, y: 3, text: ">"}})},
		{name: "prompt beyond anchor", width: 20, height: 10, cursor: uv.Position{X: 7, Y: 3}, visible: true, cells: box(2, 4, 0, 10, 6)},
		{name: "prompt outside overlap", width: 30, height: 10, cursor: uv.Position{X: 6, Y: 3}, visible: true, cells: join(border(2, 0, 10), border(4, 5, 10), []agyCell{{x: 0, y: 3, text: ">"}})},
		{name: "prompt outside vertical box", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: join(border(2, 0, 10), border(4, 0, 10), []agyCell{{x: 0, y: 5, text: ">"}})},
		{name: "cursor on border", width: 20, height: 10, cursor: uv.Position{X: 2, Y: 2}, visible: true, cells: box(2, 4, 0, 10, 0)},
		{name: "cursor outside overlap", width: 30, height: 10, cursor: uv.Position{X: 2, Y: 3}, visible: true, cells: join(border(2, 0, 10), border(4, 5, 10), []agyCell{{x: 5, y: 3, text: ">"}})},
		{name: "box too tall", width: 40, height: 34, cursor: uv.Position{X: 3, Y: 20}, visible: true, cells: box(2, 28, 0, 20, 0)},
		{name: "distant box cannot qualify local cursor", width: 40, height: 30, cursor: uv.Position{X: 2, Y: 20}, visible: true, cells: join(box(2, 4, 0, 10, 0), border(19, 0, 4), []agyCell{{x: 0, y: 20, text: ">"}})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := terminalSnapshot{
				Width:         test.width,
				Height:        test.height,
				Cursor:        test.cursor,
				CursorVisible: test.visible,
				Cells:         make([]uv.Cell, test.width*test.height),
			}
			for _, cell := range test.cells {
				snapshot.Cells[cell.y*test.width+cell.x].Content = cell.text
			}
			if got := agyComposerActive(snapshot); got != test.want {
				t.Fatalf("active = %t, want %t", got, test.want)
			}
		})
	}

	t.Run("current snapshot rejects stale lifecycle evidence", func(t *testing.T) {
		composer := "\x1b[10;1H──────────\x1b[11;1H> work\x1b[13;1H──────────\x1b[?25h\x1b[12;3H"
		mutations := []struct {
			name   string
			feed   string
			resize [2]int
		}{
			{name: "overwrite", feed: "\x1b[11;1Hnot a prompt"},
			{name: "erase", feed: "\x1b[2J\x1b[?25h\x1b[12;3H"},
			{name: "ECH", feed: "\x1b[10;1H\x1b[10X\x1b[?25h\x1b[12;3H"},
			{name: "scroll", feed: "\x1b[20S\x1b[?25h\x1b[12;3H"},
			{name: "resize", resize: [2]int{30, 8}},
			{name: "reset", feed: "\x1bc\x1b[?25h\x1b[3;3H"},
			{name: "alternate screen", feed: "\x1b[?1049h\x1b[?25h\x1b[3;3H"},
		}
		for _, mutation := range mutations {
			t.Run(mutation.name, func(t *testing.T) {
				model := newTerminalModelForTest(t, 30, 30)
				if err := model.Feed([]byte(composer)); err != nil {
					t.Fatal(err)
				}
				if !agyComposerActive(model.Snapshot()) {
					t.Fatal("generated composer was not active before mutation")
				}
				if mutation.resize != [2]int{} {
					if err := model.Resize(mutation.resize[0], mutation.resize[1]); err != nil {
						t.Fatal(err)
					}
				} else if err := model.Feed([]byte(mutation.feed)); err != nil {
					t.Fatal(err)
				}
				if agyComposerActive(model.Snapshot()) {
					t.Fatal("stale composer evidence survived current-screen mutation")
				}
			})
		}
	})
}

type composerDifferentialCase struct {
	name              string
	stream            []byte
	want              bool
	allowedCorrection string
}

func TestCodexComposerActiveSnapshotDifferential(t *testing.T) {
	composer := "\x1b[19;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[21;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[?25h\x1b[20;3H"
	cases := []composerDifferentialCase{
		{name: "generated composer", stream: []byte(composer), want: true},
		{name: "hidden cursor", stream: []byte(composer + "\x1b[?25l")},
		{name: "erased composer", stream: []byte(composer + "\x1b[2J\x1b[?25h\x1b[20;3H")},
		{name: "composer away from cursor", stream: []byte(composer + "\x1b[?25h\x1b[30;3H")},
		{
			name: "one local painted row plus distant complete evidence",
			stream: []byte("\x1b[9;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[10;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[?25h\x1b[20;3H"),
		},
	}

	runComposerSnapshotDifferential(t, cases, func(stream []byte) bool {
		tracker := newCodexComposerTracker()
		tracker.resize(38, 120)
		tracker.feed(stream)
		return tracker.state().active()
	}, codexComposerActive)
}

func TestMuseComposerActiveSnapshotDifferential(t *testing.T) {
	literal, err := os.ReadFile("testdata/tty/muse/0.1.0-R708.1/composer.raw")
	if err != nil {
		t.Fatalf("read literal Muse fixture: %v", err)
	}
	qualified := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ \x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;3H"
	qualifiedNonEmpty := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ work on #140" +
		"\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;15H"
	qualifiedCursorAbove := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ work on #140" +
		"\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[7;15H"
	qualifiedCursorBelow := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ work on #140" +
		"\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[9;15H"
	cases := []composerDifferentialCase{
		{name: "literal captured composer", stream: literal, want: true},
		{name: "generated captured signature", stream: []byte(qualified), want: true},
		{name: "qualified non-empty prompt", stream: []byte(qualifiedNonEmpty), want: true},
		{name: "qualified prompt one row above cursor", stream: []byte(qualifiedCursorBelow), want: true},
		{name: "qualified prompt one row below cursor", stream: []byte(qualifiedCursorAbove), want: true},
		{name: "hidden cursor", stream: []byte(qualified + "\x1b[?25l")},
		{name: "bare old U+203A glyph", stream: []byte("\x1b[8;1H› \x1b[?25h\x1b[8;3H")},
		{
			name:              "unqualified U+27E9 glyph",
			stream:            []byte("\x1b[8;1H⟩ \x1b[?25h\x1b[8;3H"),
			allowedCorrection: "unqualified glyph",
		},
		{
			name:              "stale prompt mutation",
			stream:            []byte(qualified + "\x1b[8;1H\x1b[22mnot a prompt\x1b[?25h\x1b[8;3H"),
			allowedCorrection: "stale mutation",
		},
		{
			name: "one local weak glyph plus distant complete evidence",
			stream: []byte("\x1b[2;1H\x1b[2m────\x1b[3;1H\x1b[22m⟩ \x1b[4;1H\x1b[2m────" +
				"\x1b[20;1H\x1b[22m⟩ \x1b[?25h\x1b[20;3H"),
			allowedCorrection: "unqualified glyph",
		},
	}

	runComposerSnapshotDifferential(t, cases, func(stream []byte) bool {
		tracker := newMuseComposerTracker()
		tracker.resize(38, 120)
		tracker.feed(stream)
		return tracker.state().active()
	}, museComposerActive)
}

func TestComposerRecognizersRejectAdversarialSnapshotsWithoutBlocking(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	snapshots := []struct {
		name     string
		snapshot terminalSnapshot
	}{
		{"max width", terminalSnapshot{Width: maxInt, Height: 2, CursorVisible: true}},
		{"max height", terminalSnapshot{Width: 2, Height: maxInt, CursorVisible: true}},
		{"max area", terminalSnapshot{Width: 4096, Height: 4096, CursorVisible: true}},
		{"tiny cell slice", terminalSnapshot{Width: 120, Height: 38, CursorVisible: true, Cells: make([]uv.Cell, 1)}},
	}
	recognizers := []struct {
		name      string
		recognize composerRecognizer
	}{
		{"codex", codexComposerActive},
		{"agy", agyComposerActive},
		{"muse", museComposerActive},
	}

	for _, snapshot := range snapshots {
		t.Run(snapshot.name, func(t *testing.T) {
			if snapshotCoordinatesValid(snapshot.snapshot) {
				t.Error("adversarial snapshot coordinates unexpectedly valid")
			}
			for _, recognizer := range recognizers {
				done := make(chan bool, 1)
				go func() { done <- recognizer.recognize(snapshot.snapshot) }()
				select {
				case got := <-done:
					if got {
						t.Errorf("%s recognizer accepted adversarial snapshot", recognizer.name)
					}
				case <-time.After(100 * time.Millisecond):
					t.Errorf("%s recognizer blocked on adversarial snapshot", recognizer.name)
				}
			}
		})
	}
}

func runComposerSnapshotDifferential(
	t *testing.T,
	cases []composerDifferentialCase,
	legacy func([]byte) bool,
	recognize composerRecognizer,
) {
	t.Helper()
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			old := legacy(test.stream)
			model := newTerminalModelForTest(t, 120, 38)
			if err := model.Feed(test.stream); err != nil {
				t.Fatalf("feed terminal model: %v", err)
			}
			new := recognize(model.Snapshot())
			if !old && new {
				t.Fatalf("forbidden differential: old=false new=true")
			}
			if test.allowedCorrection == "" && new != old {
				t.Fatalf("unapproved differential: old=%t new=%t", old, new)
			}
			if test.allowedCorrection != "" && (!old || new) {
				t.Fatalf("%s correction = old:%t new:%t, want old:true new:false", test.allowedCorrection, old, new)
			}
			if new != test.want {
				t.Fatalf("snapshot active = %t, want %t", new, test.want)
			}
		})
	}
}
