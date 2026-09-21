package wrapcmd

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/changelogcmd"
)

// scrollbackQoderPatternRe extracts the qoder row of nvim/scrollback.lua's
// PROMPT_PATTERN_BY_AGENT table: `  qoder  = [[<pattern>]],` — or the leveled
// `[=[<pattern>]=]` form, needed whenever the pattern's own character class
// ends the row (its closing `]` would fuse with the `]]` delimiter).
var scrollbackQoderPatternRe = regexp.MustCompile(`(?m)^\s*qoder\s*=\s*\[\[(.*?)\]\],\s*$|^\s*qoder\s*=\s*\[=\[(.*?)\]=\],\s*$`)

// TestScrollbackQoderPatternTracksPromptAuthority pins the one qoder glyph
// consumer that cannot read qoderPromptGlyphs: nvim/scrollback.lua's Alt+b
// prompt-jump pattern. The expected pattern is DERIVED here from
// qoderPromptGlyphs + qoderPromptCol (the single authority), so adding,
// removing, or renaming a glyph — or moving the prompt column — turns this
// red until the Lua row is updated, rather than leaving Alt+b silently
// unable to find qoder turns.
func TestScrollbackQoderPatternTracksPromptAuthority(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "nvim", "scrollback.lua"))
	if err != nil {
		t.Fatalf("read nvim/scrollback.lua (the qoder pattern's parity fixture — if the file moved, this test moved out of sync): %v", err)
	}
	rows := scrollbackQoderPatternRe.FindAllStringSubmatch(string(data), -1)
	if len(rows) != 1 {
		t.Fatalf("scrollback.lua must carry exactly one qoder PROMPT_PATTERN_BY_AGENT row, found %d", len(rows))
	}
	got := rows[0][1]
	if got == "" {
		got = rows[0][2]
	}

	// A Vim collection escape. The Lua row is consumed by vim.fn.search, a Vim
	// regex, so the derived class must escape the Vim dialect: backslash first,
	// then the collection-specials ], -, ^ (^ only matters leading, which a
	// first-sorted glyph can be). A Lua-pattern dialect here (%, %], %-) would
	// derive a wrong class for a future glyph containing those characters.
	escape := strings.NewReplacer(`\`, `\\`, `]`, `\]`, `-`, `\-`, `^`, `\^`)
	glyphs := make([]string, 0, len(qoderPromptGlyphs))
	for glyph := range qoderPromptGlyphs {
		glyphs = append(glyphs, escape.Replace(glyph))
	}
	sort.Strings(glyphs)

	want := "^" + strings.Repeat(" ", qoderPromptCol) + "[" + strings.Join(glyphs, "") + "]"
	if got != want {
		t.Fatalf("scrollback.lua qoder pattern %q does not derive from qoderPromptGlyphs at qoderPromptCol %d (want %q); update PROMPT_PATTERN_BY_AGENT", got, qoderPromptCol, want)
	}
}

// TestDistillQoderGlyphTracksPromptAuthority pins the other out-of-package qoder
// glyph consumer: changelogcmd's promptGlyphChar row (read by scanTurnBoundaries
// and trimLiveTail) restated " >" from this authority by hand. The expected
// value is DERIVED here from qoderPromptCol + the default-mode glyph, so moving
// the column or renaming the glyph reddens distill's row instead of silently
// desyncing the changelog boundary detection (#300 M4 review BR-43).
func TestDistillQoderGlyphTracksPromptAuthority(t *testing.T) {
	want := strings.Repeat(" ", qoderPromptCol) + ">"
	if got := changelogcmd.PromptGlyph("qoder"); got != want {
		t.Fatalf("changelogcmd qoder glyph %q does not derive from qoderPromptCol %d + %q (want %q); update promptGlyphChar", got, qoderPromptCol, ">", want)
	}
	// The omitted yolo `*` is a decision, not drift: distill's captured evidence
	// covers default mode only, and a missed boundary degrades gracefully (extra
	// lookback) where a wrong boundary corrupts the log. Pin the authority as
	// still carrying `*` so its removal is a deliberate registry change, and pin
	// distill as not picking it up so adding `*` there is a deliberate decision
	// made in this test's presence.
	if !qoderPromptGlyphs["*"] {
		t.Fatal("qoderPromptGlyphs no longer carries the yolo `*` glyph — if that is deliberate, drop the omission rows here and in distill")
	}
	if got := changelogcmd.PromptGlyph("qoder"); strings.Contains(got, "*") {
		t.Fatalf("changelogcmd qoder glyph %q must stay default-mode only (yolo `*` is a deliberate omission)", got)
	}
}
