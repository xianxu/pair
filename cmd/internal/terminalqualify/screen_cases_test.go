package terminalqualify

import "testing"

func TestScreenCases(t *testing.T) {
	cases := ScreenCases()
	seen := map[string]bool{}
	for _, c := range cases {
		if seen[c.ID] || c.ID == "" {
			t.Fatalf("duplicate/empty ID %q", c.ID)
		}
		seen[c.ID] = true
		if c.Source == "" || len(c.Expected) == 0 || c.Uncovered != "" {
			t.Errorf("invalid executable fixture %s", c.ID)
		}
		if c.Width != 8 || c.Height != 4 {
			t.Errorf("fixture %s must specify geometry", c.ID)
		}
	}
	for _, id := range []string{"sgr-bold", "sgr-faint", "sgr-italic", "sgr-slow-blink", "sgr-rapid-blink", "sgr-reverse", "sgr-conceal", "sgr-strike", "sgr-selective-reset", "sgr-underline-styles", "sgr-underline-color", "sgr-scroll-preservation", "ascii", "utf8-two", "utf8-four", "save-1048", "malformed-utf8", "utf8-wide", "combining", "zwj", "save-dec", "save-csi", "alt-47", "alt-1049", "origin", "scroll-region", "erase", "resize", "history", "hyperlink", "truecolor", "indexed-color", "dcs-framing", "malformed-csi", "oversized-osc"} {
		if !seen[id] {
			t.Errorf("missing obligation %s", id)
		}
	}
}
