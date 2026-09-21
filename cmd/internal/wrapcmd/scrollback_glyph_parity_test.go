package wrapcmd

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
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
		t.Fatalf("read scrollback.lua: %v", err)
	}
	rows := scrollbackQoderPatternRe.FindAllStringSubmatch(string(data), -1)
	if len(rows) != 1 {
		t.Fatalf("scrollback.lua must carry exactly one qoder PROMPT_PATTERN_BY_AGENT row, found %d", len(rows))
	}
	got := rows[0][1]
	if got == "" {
		got = rows[0][2]
	}

	// A Lua character-class member escape. Only %, ], and - are special inside
	// a class (^ only leading); escaping them here keeps the derivation total
	// for any future glyph.
	escape := strings.NewReplacer("%", "%%", "]", "%]", "-", "%-", "^", "%^")
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
