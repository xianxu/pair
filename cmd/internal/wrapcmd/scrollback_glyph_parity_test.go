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

// promptPatternAuthorities lists every harness whose prompt glyph authority
// lives in Go, with the column its submitted prompt is echoed at in the
// transcript. Qoder echoes at its composer column; Grok one column right of
// it. The out-of-package consumers (nvim/scrollback.lua's Alt+b pattern and
// changelogcmd's distill glyph) must derive from these rows.
var promptPatternAuthorities = []struct {
	agent       string
	glyphs      map[string]bool
	echoCol     int
	distillWith string // the default-mode glyph distill keys on
}{
	{"qoder", qoderPromptGlyphs, qoderPromptCol, ">"},
	{"grok", grokPromptGlyphs, grokEchoPromptCol, "❯"},
}

// scrollbackPatternRe extracts an agent's row of nvim/scrollback.lua's
// PROMPT_PATTERN_BY_AGENT table: `  <agent> = [[<pattern>]],` — or the leveled
// `[=[<pattern>]=]` form, needed whenever the pattern's own character class
// ends the row (its closing `]` would fuse with the `]]` delimiter).
func scrollbackPatternRe(agent string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^\s*` + agent + `\s*=\s*\[\[(.*?)\]\],\s*$|^\s*` + agent + `\s*=\s*\[=\[(.*?)\]=\],\s*$`)
}

// TestScrollbackPatternsTrackPromptAuthority pins the glyph consumer that
// cannot read the Go authorities: nvim/scrollback.lua's Alt+b prompt-jump
// pattern. Each expected pattern is DERIVED from its authority's glyphs and
// echo column, so adding, removing, or renaming a glyph — or moving the column
// — turns this red until the Lua row is updated, rather than leaving Alt+b
// silently unable to find that harness's turns.
func TestScrollbackPatternsTrackPromptAuthority(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "nvim", "scrollback.lua"))
	if err != nil {
		t.Fatalf("read nvim/scrollback.lua (the prompt patterns' parity fixture — if the file moved, this test moved out of sync): %v", err)
	}
	// A Vim collection escape. The Lua row is consumed by vim.fn.search, a Vim
	// regex, so the derived class must escape the Vim dialect: backslash first,
	// then the collection-specials ], -, ^ (^ only matters leading, which a
	// first-sorted glyph can be). A Lua-pattern dialect here (%, %], %-) would
	// derive a wrong class for a future glyph containing those characters.
	escape := strings.NewReplacer(`\`, `\\`, `]`, `\]`, `-`, `\-`, `^`, `\^`)
	for _, authority := range promptPatternAuthorities {
		rows := scrollbackPatternRe(authority.agent).FindAllStringSubmatch(string(data), -1)
		if len(rows) != 1 {
			t.Errorf("scrollback.lua must carry exactly one %s PROMPT_PATTERN_BY_AGENT row, found %d", authority.agent, len(rows))
			continue
		}
		got := rows[0][1]
		if got == "" {
			got = rows[0][2]
		}
		glyphs := make([]string, 0, len(authority.glyphs))
		for glyph := range authority.glyphs {
			glyphs = append(glyphs, escape.Replace(glyph))
		}
		sort.Strings(glyphs)
		want := "^" + strings.Repeat(" ", authority.echoCol) + "[" + strings.Join(glyphs, "") + "]"
		if got != want {
			t.Errorf("scrollback.lua %s pattern %q does not derive from its glyph authority at echo column %d (want %q); update PROMPT_PATTERN_BY_AGENT", authority.agent, got, authority.echoCol, want)
		}
	}
}

// TestDistillGlyphsTrackPromptAuthority pins the other out-of-package glyph
// consumer: changelogcmd's promptGlyphChar row (read by scanTurnBoundaries and
// trimLiveTail). Each expected value is DERIVED from the echo column + the
// default-mode glyph, so moving the column or renaming the glyph reddens
// distill's row instead of silently desyncing the changelog boundary detection
// (#300 M4 review BR-43).
func TestDistillGlyphsTrackPromptAuthority(t *testing.T) {
	for _, authority := range promptPatternAuthorities {
		want := strings.Repeat(" ", authority.echoCol) + authority.distillWith
		if got := changelogcmd.PromptGlyph(authority.agent); got != want {
			t.Errorf("changelogcmd %s glyph %q does not derive from echo column %d + %q (want %q); update promptGlyphChar", authority.agent, got, authority.echoCol, authority.distillWith, want)
		}
	}
	// The omitted qoder yolo `*` is a decision, not drift: distill's captured
	// evidence covers default mode only, and a missed boundary degrades
	// gracefully (extra lookback) where a wrong boundary corrupts the log. Pin
	// the authority as still carrying `*` so its removal is a deliberate
	// registry change, and pin distill as not picking it up.
	if !qoderPromptGlyphs["*"] {
		t.Fatal("qoderPromptGlyphs no longer carries the yolo `*` glyph — if that is deliberate, drop the omission rows here and in distill")
	}
	if got := changelogcmd.PromptGlyph("qoder"); strings.Contains(got, "*") {
		t.Fatalf("changelogcmd qoder glyph %q must stay default-mode only (yolo `*` is a deliberate omission)", got)
	}
}

// TestGrokEchoPromptColMatchesCapture pins grokEchoPromptCol against the
// frozen finished-turn capture: the submitted prompt's echo row carries a
// grokPromptGlyphs glyph at exactly that column, above the live composer box.
func TestGrokEchoPromptColMatchesCapture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "prompt-echo", "grok", "1.0.46", "echo.raw"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := newTerminalModel(120, 38)
	if err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	if err := model.Feed(raw); err != nil {
		t.Fatal(err)
	}
	snapshot := model.Snapshot()
	found := false
	for y := 0; y < snapshot.Height; y++ {
		cell := snapshot.CellAt(grokEchoPromptCol, y)
		if cell == nil || !grokPromptGlyphs[cell.Content] {
			continue
		}
		for x := 0; x < grokEchoPromptCol; x++ {
			if c := snapshot.CellAt(x, y); c != nil && strings.TrimSpace(c.Content) != "" {
				t.Fatalf("echo row %d paints column %d before the glyph: %q", y, x, c.Content)
			}
		}
		found = true
	}
	if !found {
		t.Fatalf("no %v glyph at echo column %d in echo.raw", grokPromptGlyphs, grokEchoPromptCol)
	}
}
