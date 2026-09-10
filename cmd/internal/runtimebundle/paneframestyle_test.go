package runtimebundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FULL PANE FRAMES ARE STATED, NOT DEFAULTED (#223).
//
// zellij 0.45 split `pane_frames true` into a style and made the default
// "titles": a title row and no border. pair's layout is built on the full
// frame — the agent pane's top border is its first line and carries the scroll
// indicator — and on 0.45 without the explicit style the operator watched that
// line vanish. Deleting the key is silent on 0.44 (which ignores it) and
// silently wrong on 0.45, so it is pinned here, in both the source config and
// the runtime bundle's mirror — the same source-plus-mirror pairing
// TestEveryTerminalPaneRungIsBorderless uses for main-3.kdl.
func TestConfigStatesFullPaneFrames(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "..", "zellij", "config.kdl"),
		filepath.Join("assets", "runtime", "files", "zellij", "config.kdl"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var frames, style bool
		for _, line := range strings.Split(string(raw), "\n") {
			// A commented-out key is not a setting; strip `//` comments before
			// matching, so documentation that NAMES the key cannot satisfy this.
			code, _, _ := strings.Cut(line, "//")
			switch strings.Join(strings.Fields(code), " ") {
			case "pane_frames true":
				frames = true
			case `pane_frame_style "full"`:
				style = true
			}
		}
		if !frames {
			t.Errorf("%s: pane_frames true is not set — the agent pane's scroll indicator lives in its frame", path)
		}
		if !style {
			t.Errorf(`%s: pane_frame_style "full" is not set — on zellij 0.45 the default "titles" `+
				"style drops the frame's top border, which is the layout's first line (#223)", path)
		}
	}
}
