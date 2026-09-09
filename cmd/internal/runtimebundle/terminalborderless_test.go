package runtimebundle

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// EVERY `name="terminal"` PANE IS BORDERLESS, and the count is asserted so a
// rung added later cannot quietly reframe the pane.
//
// The frame used to be the pane's label. Since #199 M3 the pane draws its own
// tab strip in a row it reserves, which is what makes taking the frame off an
// improvement rather than a loss — and it is also why this cannot be left to a
// spot check: `main-3.kdl` carries NINE terminal rungs (the layout ladder plus
// both split halves), and a frame returning on one of them is invisible until
// an operator happens to step onto that rung.
//
// It reads BOTH copies. `zellij/layouts/main-3.kdl` is the SOURCE; the copy
// under `cmd/internal/runtimebundle/assets/runtime/files/` is a generated mirror
// that `make test` regenerates before any test runs, so asserting only the
// mirror would pass on a mirror regenerated from an unedited source, and
// asserting only the source would miss a mirror that failed to regenerate.
func TestEveryTerminalPaneRungIsBorderless(t *testing.T) {
	const wantRungs = 9
	pane := regexp.MustCompile(`pane\s+name="terminal"`)

	for _, path := range []string{
		filepath.Join("..", "..", "..", "zellij", "layouts", "main-3.kdl"),
		filepath.Join("assets", "runtime", "files", "zellij", "layouts", "main-3.kdl"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		rungs := 0
		for i, line := range lines {
			if !pane.MatchString(line) {
				continue
			}
			rungs++
			// THE PANE'S OWN BLOCK, delimited by brace depth -- not a
			// fixed two-line window.
			//
			// The window version let a NEIGHBOUR satisfy the check: a rung
			// missing borderless passed whenever the pane declared right after
			// it had the attribute, which made 3 of these 9 rungs incapable of
			// failing. A guard that cannot fail on a third of its enumeration
			// is worse than none, because the count still reads as coverage.
			window := paneBlock(lines, i)
			if !strings.Contains(window, "borderless=true") {
				t.Errorf("%s:%d — a terminal pane rung without borderless=true:\n  %s",
					filepath.Base(path), i+1, strings.TrimSpace(line))
			}
		}
		if rungs != wantRungs {
			t.Errorf("%s declares %d terminal pane rungs, want %d — a rung was added or "+
				"removed without updating this enumeration", filepath.Base(path), rungs, wantRungs)
		}
	}
}

// paneBlock returns the lines belonging to the pane declared at `start`, and
// nothing after it.
//
// A pane is either self-contained on one line (`pane ... }` or no brace at all)
// or opens a block that closes when brace depth returns to where it began.
// Either way the span stops before the next sibling, which is the whole point:
// an attribute on a neighbour must never satisfy this pane.
func paneBlock(lines []string, start int) string {
	depth := 0
	var b strings.Builder
	for i := start; i < len(lines); i++ {
		line := lines[i]
		b.WriteString(line)
		b.WriteString("\n")
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if i > start && depth <= 0 {
			break
		}
		if depth <= 0 && strings.Contains(line, "}") {
			break // self-closed on the declaring line
		}
		if depth == 0 && i == start {
			break // no block at all: a one-line pane with no braces
		}
	}
	return b.String()
}
