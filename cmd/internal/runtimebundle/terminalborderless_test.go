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
			// borderless may sit on the same line as the pane, or on the
			// following line for the multi-line rungs.
			window := line
			if i+1 < len(lines) {
				window += "\n" + lines[i+1]
			}
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
