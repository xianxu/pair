package wrapcmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	xansi "github.com/charmbracelet/x/ansi"

	uv "github.com/charmbracelet/ultraviolet"
)

type agyCell struct {
	x, y  int
	text  string
	plain bool // paint without Agy's captured styling
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
	// Agy's permission picker paints an unstyled ">" selection marker in the
	// same column, between the same rules, as its composer prompt.
	pickerBox := func(top, bottom, start, length, promptX int) []agyCell {
		cells := append(border(top, start, length), border(bottom, start, length)...)
		return append(cells, agyCell{x: promptX, y: top + 1, text: ">", plain: true})
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
		{name: "unstyled picker marker between rules", width: 40, height: 20, cursor: uv.Position{X: 3, Y: 3}, visible: true, cells: pickerBox(2, 5, 0, 30, 0)},
		// HEIGHT CEILING. maxBoxHeight caps the box, so a draft taller than it
		// stops being recognized and the next Return submits it. The ceiling is
		// inherited from the box-structure design rather than measured against
		// Agy; the enclosing rules already prevent pairing distant chrome, so
		// this row exists to make any change to the bound deliberate. Its
		// neighbours "maximum box height" and "box too tall" pin the Codex-side
		// equivalent for the box itself.
		{name: "draft taller than the height ceiling", width: 40, height: 36, cursor: uv.Position{X: 3, Y: 20}, visible: true, cells: box(2, 30, 0, 20, 0)},
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
				target := &snapshot.Cells[cell.y*test.width+cell.x]
				target.Content = cell.text
				if cell.plain {
					continue
				}
				// Match the captured Agy palette: bright-blue prompt, dim rules.
				if cell.text == ">" {
					target.Style.Fg = xansi.BrightBlue
				} else {
					target.Style.Fg = xansi.BrightBlack
				}
			}
			if got := agyComposerActive(snapshot); got != test.want {
				t.Fatalf("active = %t, want %t", got, test.want)
			}
		})
	}

	t.Run("current snapshot rejects stale lifecycle evidence", func(t *testing.T) {
		// Agy paints dim rules (SGR 90) and a bright-blue prompt (SGR 94).
		composer := "\x1b[10;1H\x1b[90m──────────\x1b[11;1H\x1b[94m>\x1b[39m work" +
			"\x1b[13;1H\x1b[90m──────────\x1b[39m\x1b[?25h\x1b[12;3H"
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
	name   string
	stream []byte
	want   bool
}

func TestCodexComposerActiveSnapshotDifferential(t *testing.T) {
	literalComposer, err := os.ReadFile("testdata/tty/codex/0.147.0/composer.raw")
	if err != nil {
		t.Fatalf("read literal Codex composer fixture: %v", err)
	}
	literalOverlay, err := os.ReadFile("testdata/tty/codex/0.147.0/overlay.raw")
	if err != nil {
		t.Fatalf("read literal Codex overlay fixture: %v", err)
	}
	// Codex 0.147.0 paints a bold U+203A at column 0 of the composer's first
	// row and indents continuation rows to column 2. It reuses the same glyph
	// unemphasized as a menu selection marker.
	prompt := "\x1b[12;1H\x1b[1m›\x1b[22m alpha"
	composer := prompt + "\x1b[?25h\x1b[12;9H"
	multiline := prompt + "\x1b[13;3Hbeta\x1b[14;3Hgamma\x1b[?25h\x1b[14;8H"
	cases := []composerDifferentialCase{
		{name: "literal captured composer", stream: literalComposer, want: true},
		{name: "generated captured signature", stream: []byte(composer), want: true},
		{name: "cursor on painted continuation row", stream: []byte(multiline), want: true},
		{
			// Pressing Enter once leaves the cursor on an empty second line;
			// the next Enter must still remap or multi-line entry stops after
			// one newline.
			name:   "cursor on empty continuation row",
			stream: []byte(prompt + "\x1b[?25h\x1b[13;3H"),
			want:   true,
		},
		{
			// C1: a blank line inside the composer must not end the block, or
			// Return submits a half-written message instead of adding a line.
			name:   "blank line between prompt and cursor",
			stream: []byte(prompt + "\x1b[14;3Hworld\x1b[15;3Htail\x1b[?25h\x1b[14;8H"),
			want:   true,
		},
		{
			name:   "two consecutive newlines leave the cursor two rows down",
			stream: []byte(prompt + "\x1b[15;3Htail\x1b[?25h\x1b[14;3H"),
			want:   true,
		},
		{
			// DELIBERATE AMBIGUITY RESOLUTION. Codex begins each frame with
			// \x1b[1;1H\x1b[J and paints the status line after the composer,
			// so a PTY read can split a frame between them. At that instant a
			// real composer continuation row is cell-identical to the status
			// row: painted, blank row above, nothing below. The two cannot be
			// told apart from the snapshot, and this resolves to "not a
			// composer" because #139's contract is that an unrecognized state
			// emits bare CR. Changing this answer is a policy change, not a
			// bug fix.
			name: "mid-frame composer indistinguishable from status row",
			stream: []byte("\x1b[?25h\x1b[1;1H\x1b[J\x1b[12;1H\x1b[1m›\x1b[22m alpha" +
				"\x1b[14;3Hbeta\x1b[14;7H"),
			want: false,
		},
		{
			// The same frame completed: the status line below settles it.
			name: "completed frame paints the status row below the composer",
			stream: []byte("\x1b[?25h\x1b[1;1H\x1b[J\x1b[12;1H\x1b[1m›\x1b[22m alpha" +
				"\x1b[14;3Hbeta\x1b[16;3Hgpt-5.6-sol · ~/workspace/pair\x1b[14;7H"),
			want: true,
		},
		{name: "literal captured overlay", stream: literalOverlay},
		{
			// The update interstitial marks its selected row with the same
			// glyph unemphasized. Enter there must confirm, not insert.
			name:   "unemphasized menu marker with visible cursor",
			stream: []byte("\x1b[12;1H\x1b[22m› 1. Update now\x1b[?25h\x1b[12;9H"),
		},
		{
			// Codex parks the cursor on its status line mid-paint. A blank row
			// separates it from the composer, so distant evidence must not
			// qualify a local cursor.
			name:   "cursor below composer past a blank row",
			stream: []byte(prompt + "\x1b[14;3Hgpt-5.6-sol · ~/workspace/pair\x1b[?25h\x1b[14;20H"),
		},
		{name: "hidden cursor", stream: []byte(composer + "\x1b[?25l")},
		{name: "erased composer", stream: []byte(composer + "\x1b[2J\x1b[?25h\x1b[12;9H")},
		{
			name:   "cursor before the composer text column",
			stream: []byte(prompt + "\x1b[?25h\x1b[12;2H"),
		},
		{
			name:   "prompt beyond the composer height bound",
			stream: []byte(prompt + "\x1b[33;3Htail\x1b[?25h\x1b[33;7H"),
		},
	}

	runComposerSnapshotDifferential(t, cases, codexComposerActive)
}

// claudeBox builds a Claude composer box: a rule row, the prompt row, then a
// closing rule, all painted in one truecolor foreground. Claude repaints both
// the glyph and the rule colour per input mode, so both are parameters.
func claudeBox(top int, glyph, rgb string, body ...string) string {
	rule := "\x1b[38;2;" + rgb + "m" + strings.Repeat("─", 40) + "\x1b[39m"
	out := fmt.Sprintf("\x1b[%d;1H%s", top+1, rule)
	promptFG := ""
	if glyph != "❯" {
		promptFG = "\x1b[38;2;" + rgb + "m"
	}
	out += fmt.Sprintf("\x1b[%d;1H%s%s\x1b[39m %s", top+2, promptFG, glyph, body[0])
	for i, line := range body[1:] {
		out += fmt.Sprintf("\x1b[%d;3H%s", top+3+i, line)
	}
	return out + fmt.Sprintf("\x1b[%d;1H%s", top+2+len(body), rule)
}

func TestClaudeComposerActiveSnapshotDifferential(t *testing.T) {
	const (
		grey = "136;136;136"
		pink = "253;93;177"
	)
	// Default mode: chevron prompt, grey rules, cursor just past the prompt.
	dflt := claudeBox(5, "❯", grey, "alpha") + "\x1b[?25h\x1b[7;9H"
	cases := []composerDifferentialCase{
		{name: "default mode composer", stream: []byte(dflt), want: true},
		{
			// Claude's bash mode repaints BOTH the glyph and the rule colour.
			// A recognizer pinned to the chevron or to grey declines here, and
			// for Claude a decline means the next Return submits the draft.
			name:   "bash mode composer repaints glyph and rule colour",
			stream: []byte(claudeBox(5, "!", pink, "ls -la") + "\x1b[?25h\x1b[7;10H"),
			want:   true,
		},
		{
			name:   "cursor on a painted continuation row",
			stream: []byte(claudeBox(5, "❯", grey, "alpha", "beta") + "\x1b[?25h\x1b[8;7H"),
			want:   true,
		},
		{
			// One Return pressed: the cursor sits on a new, still-empty line.
			name:   "cursor on an empty continuation row",
			stream: []byte(claudeBox(5, "❯", grey, "alpha", "") + "\x1b[?25h\x1b[8;3H"),
			want:   true,
		},
		{
			name:   "blank line inside the draft",
			stream: []byte(claudeBox(5, "❯", grey, "alpha", "", "gamma") + "\x1b[?25h\x1b[9;8H"),
			want:   true,
		},
		{
			name:   "cursor resting on the closing rule",
			stream: []byte(dflt + "\x1b[?25h\x1b[8;3H"),
			want:   true,
		},
		{
			// The rules must agree with each other, or unrelated chrome pairs
			// up into a box that was never a composer.
			name: "rules whose colours disagree",
			stream: []byte("\x1b[6;1H\x1b[38;2;136;136;136m" + strings.Repeat("─", 40) +
				"\x1b[7;1H\x1b[39m❯ alpha\x1b[8;1H\x1b[38;2;253;93;177m" + strings.Repeat("─", 40) +
				"\x1b[39m\x1b[?25h\x1b[7;9H"),
		},
		{
			name:   "prompt with no rule above",
			stream: []byte("\x1b[7;1H❯ alpha\x1b[8;1H\x1b[38;2;136;136;136m" + strings.Repeat("─", 40) + "\x1b[39m\x1b[?25h\x1b[7;9H"),
		},
		{
			name:   "prompt with no rule below",
			stream: []byte("\x1b[6;1H\x1b[38;2;136;136;136m" + strings.Repeat("─", 40) + "\x1b[39m\x1b[7;1H❯ alpha\x1b[10;1Hstatus\x1b[?25h\x1b[7;9H"),
		},
		{name: "hidden cursor", stream: []byte(dflt + "\x1b[?25l")},
		{name: "erased composer", stream: []byte(dflt + "\x1b[2J\x1b[?25h\x1b[7;9H")},
		{name: "cursor before the composer text column", stream: []byte(claudeBox(5, "❯", grey, "alpha") + "\x1b[?25h\x1b[7;2H")},
		{
			// NO HEIGHT CEILING. A ceiling would make Return submit any draft
			// taller than it, which is the lost-draft failure this issue
			// exists to avoid, on Pair's default agent. Nothing distant can
			// pair into a box: the top rule must be immediately above the
			// prompt and the closing rule is the first painted column-0 row
			// below it.
			name:   "draft far taller than any height ceiling",
			stream: []byte(claudeBox(1, "❯", grey, append([]string{"one"}, tallClaudeBody()...)...) + "\x1b[?25h\x1b[25;8H"),
			want:   true,
		},
	}

	runComposerSnapshotDifferential(t, cases, claudeComposerActive)
}

// tallClaudeBody returns continuation lines pushing the cursor past
// claudeComposerMaxRows below the prompt.
func tallClaudeBody() []string {
	body := make([]string, 22)
	for i := range body {
		body[i] = "line"
	}
	return body
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
		{
			// C2: the box grows as the message does; anchoring on rules
			// directly above and below the prompt only ever matched one line.
			name: "two-line composer with cursor on the second line",
			stream: []byte("\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ hello" +
				"\x1b[9;3Hworld\x1b[10;1H\x1b[2m────\x1b[?25h\x1b[9;8H"),
			want: true,
		},
		{
			name: "three-line composer with cursor on the third line",
			stream: []byte("\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ one" +
				"\x1b[9;3Htwo\x1b[10;3Hthree\x1b[11;1H\x1b[2m────\x1b[?25h\x1b[10;8H"),
			want: true,
		},
		{
			// HEIGHT CEILING. museComposerMaxRows caps prompt-to-cursor
			// distance, so a draft taller than it stops being recognized and
			// the next Return submits it. The ceiling is inherited from the
			// box-structure design rather than measured against Muse; the
			// enclosing rules already prevent pairing distant chrome, so this
			// row exists to make any change to the bound deliberate.
			name: "draft taller than the height ceiling",
			stream: []byte("\x1b[1;1H\x1b[2m────\x1b[2;1H\x1b[22m⟩ one" +
				museTallComposerRows() + "\x1b[?25h\x1b[24;8H"),
			want: false,
		},
		{name: "hidden cursor", stream: []byte(qualified + "\x1b[?25l")},
		{name: "bare old U+203A glyph", stream: []byte("\x1b[8;1H› \x1b[?25h\x1b[8;3H")},
		{
			name:   "unqualified U+27E9 glyph",
			stream: []byte("\x1b[8;1H⟩ \x1b[?25h\x1b[8;3H"),
		},
		{
			name:   "stale prompt mutation",
			stream: []byte(qualified + "\x1b[8;1H\x1b[22mnot a prompt\x1b[?25h\x1b[8;3H"),
		},
		{
			name: "one local weak glyph plus distant complete evidence",
			stream: []byte("\x1b[2;1H\x1b[2m────\x1b[3;1H\x1b[22m⟩ \x1b[4;1H\x1b[2m────" +
				"\x1b[20;1H\x1b[22m⟩ \x1b[?25h\x1b[20;3H"),
		},
	}

	runComposerSnapshotDifferential(t, cases, museComposerActive)
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
		{"claude", claudeComposerActive},
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
	recognize composerRecognizer,
) {
	t.Helper()
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			model := newTerminalModelForTest(t, 120, 38)
			if err := model.Feed(test.stream); err != nil {
				t.Fatalf("feed terminal model: %v", err)
			}
			got := recognize(model.Snapshot())
			if got != test.want {
				t.Fatalf("snapshot active = %t, want %t", got, test.want)
			}
		})
	}
}

// museTallComposerRows paints continuation rows 3..23 so the cursor sits more
// than museComposerMaxRows below the prompt.
func museTallComposerRows() string {
	var b strings.Builder
	for row := 3; row <= 23; row++ {
		fmt.Fprintf(&b, "\x1b[%d;3Hline", row)
	}
	b.WriteString("\x1b[25;1H\x1b[2m────")
	return b.String()
}
