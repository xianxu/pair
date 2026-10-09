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
	for _, path := range zellijConfigPaths {
		settings := configSettings(t, path)
		frames, style := settings["pane_frames true"], settings[`pane_frame_style "full"`]
		if !frames {
			t.Errorf("%s: pane_frames true is not set — the agent pane's scroll indicator lives in its frame", path)
		}
		if !style {
			t.Errorf(`%s: pane_frame_style "full" is not set — on zellij 0.45 the default "titles" `+
				"style drops the frame's top border, which is the layout's first line (#223)", path)
		}
	}
}

// A SCROLLED-BACK PANE STILL TAKES KEYS (#416).
//
// zellij 0.45's default `scroll_mode_sync true` enters Scroll mode when the
// focused pane is scrolled, and Scroll mode passes no unbound key to the pane.
// pair clears zellij's default keybinds, so a wheel scroll left every keystroke
// dropped instead of snapping back to the bottom. Like the frame style, the key
// is unknown to 0.44 and its absence is silent, so it is pinned here.
func TestConfigDisablesScrollModeSync(t *testing.T) {
	for _, path := range zellijConfigPaths {
		if !configSettings(t, path)["scroll_mode_sync false"] {
			t.Errorf("%s: scroll_mode_sync false is not set — a scrolled-back pane would swallow keystrokes (#416)", path)
		}
	}
}

// zellijConfigPaths is the source config and the runtime bundle's mirror.
var zellijConfigPaths = []string{
	filepath.Join("..", "..", "..", "zellij", "config.kdl"),
	filepath.Join("assets", "runtime", "files", "zellij", "config.kdl"),
}

// configSettings returns the config's lines as whitespace-normalized settings.
// A commented-out key is not a setting; `//` comments are stripped before
// matching, so documentation that NAMES a key cannot satisfy a check.
func configSettings(t *testing.T, path string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	settings := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		code, _, _ := strings.Cut(line, "//")
		settings[strings.Join(strings.Fields(code), " ")] = true
	}
	return settings
}
