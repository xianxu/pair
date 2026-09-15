package terminalqualify

import (
	vt "github.com/charmbracelet/x/vt"
	"strings"
)

const xtermSource = "https://invisible-island.net/xterm/ctlseqs/ctlseqs.html"
const unicodeSource = "https://www.unicode.org/reports/tr29/"
const hyperlinkSource = "https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda"

// ScreenCases uses literal cells and zero-based positions. Split is an additional
// fragmentation check, never a substitute for the independent expected cells.
func ScreenCases() []Case {
	cases := []Case{
		{ID: "ascii", Capability: "ASCII text and cursor", Input: "abc", Expected: Observation{"cell:0,0": "a", "cell:1,0": "b", "cell:2,0": "c", "cursor": "3,0"}, Split: true},
		{ID: "utf8-two", Capability: "two-byte UTF-8", Source: unicodeSource, Input: "éX", Expected: Observation{"cell:0,0": "é", "cell:1,0": "X", "cursor": "2,0"}, Split: true},
		{ID: "utf8-four", Capability: "four-byte UTF-8", Source: unicodeSource, Input: "😀X", Expected: Observation{"cell:0,0": "😀", "cell-width:0,0": "2", "cell:2,0": "X", "cursor": "3,0"}, Split: true},
		{ID: "save-1048", Capability: "DEC1048 cursor save/restore", Input: "\x1b[2;3H\x1b[?1048h\x1b[4;6H\x1b[?1048lX", Expected: Observation{"cell:2,1": "X", "cursor": "3,1"}, Split: true},
		{ID: "malformed-utf8", Capability: "malformed UTF-8 recovery without poisoning later text", Source: unicodeSource, Input: "\xff\x1b[2J\x1b[HZ", Expected: Observation{"cell:0,0": "Z", "cursor": "1,0"}, Split: true},
		{ID: "utf8-wide", Capability: "UTF-8 wide cells", Source: unicodeSource, Input: "界X", Expected: Observation{"cell:0,0": "界", "cell-width:0,0": "2", "cell:2,0": "X", "cursor": "3,0"}, Split: true},
		{ID: "combining", Capability: "combining graphemes", Source: unicodeSource, Input: "e\u0301X", Expected: Observation{"cell:0,0": "e\u0301", "cell-width:0,0": "1", "cell:1,0": "X", "cursor": "2,0"}, Split: true},
		{ID: "zwj", Capability: "ZWJ graphemes", Source: unicodeSource, Input: "👩\u200d💻X", Expected: Observation{"cell:0,0": "👩\u200d💻", "cell-width:0,0": "2", "cell:2,0": "X", "cursor": "3,0"}, Split: true},
		{ID: "save-dec", Capability: "DEC cursor save/restore", Input: "\x1b[2;3H\x1b7\x1b[4;6H\x1b8X", Expected: Observation{"cell:2,1": "X", "cursor": "3,1"}, Split: true},
		{ID: "save-csi", Capability: "CSI cursor save/restore", Input: "\x1b[2;3H\x1b[s\x1b[4;6H\x1b[uX", Expected: Observation{"cell:2,1": "X", "cursor": "3,1"}, Split: true},
		{ID: "alt-47", Capability: "legacy alternate buffer", Input: "A\x1b[?47h\x1b[HX\x1b[?47l", Expected: Observation{"cell:0,0": "A", "alt": "false"}, Split: true},
		{ID: "alt-1049", Capability: "alternate buffer and cursor restoration", Input: "AB\x1b[?1049h\x1b[HX\x1b[?1049lC", Expected: Observation{"cell:0,0": "A", "cell:1,0": "B", "cell:2,0": "C", "cursor": "3,0", "alt": "false"}, Split: true},
		{ID: "alt-1047", Capability: "alternate buffer selection", Input: "A\x1b[?1047h\x1b[HX\x1b[?1047l", Expected: Observation{"cell:0,0": "A", "alt": "false"}, Split: true},
		{ID: "origin", Capability: "margins and origin mode", Input: "\x1b[2;4r\x1b[?6h\x1b[1;2HX", Expected: Observation{"cell:1,1": "X", "cursor": "2,1"}, Split: true},
		{ID: "scroll-region", Capability: "scroll region isolation", Input: "A\x1b[2;1HB\x1b[3;1HC\x1b[4;1HD\x1b[2;3r\x1b[3;1H\nX", Expected: Observation{"cell:0,0": "A", "cell:0,1": "C", "cell:0,2": "X", "cell:0,3": "D"}, Split: true},
		{ID: "erase", Capability: "erase display", Input: "ABC\x1b[2J\x1b[HZ", Expected: Observation{"cell:0,0": "Z", "cell:1,0": " ", "cell:2,0": " ", "cursor": "1,0"}, Split: true},
		{ID: "erase-line", Capability: "erase line from cursor", Input: "ABCDE\x1b[1;3H\x1b[KZ", Expected: Observation{"cell:0,0": "A", "cell:1,0": "B", "cell:2,0": "Z", "cell:3,0": " "}, Split: true},
		{ID: "resize", Capability: "geometry and retained content", Input: "AB", Action: func(e *vt.Emulator) { e.Resize(10, 5) }, Expected: Observation{"width": "10", "height": "5", "cell:0,0": "A", "cell:1,0": "B"}},
		{ID: "history", Capability: "normal-screen history", Input: "A\r\nB\r\nC\r\nD\r\nE", Expected: Observation{"cell:0,0": "B", "cell:0,3": "E", "history-lines": "1"}},
		{ID: "hyperlink", Capability: "OSC8 URL and parameters", Source: hyperlinkSource, Input: "\x1b]8;id=probe;https://example.com/\x1b\\X\x1b]8;;\x1b\\Y", Expected: Observation{"cell:0,0": "X", "link:0,0": "https://example.com/", "link-params:0,0": "id=probe", "link:1,0": ""}, Split: true},
		{ID: "truecolor", Capability: "SGR truecolor", Input: "\x1b[38;2;18;52;86;48;2;101;67;33mX", Expected: Observation{"fg:0,0": "#123456", "bg:0,0": "#654321"}, Split: true},
		{ID: "indexed-color", Capability: "SGR indexed color", Input: "\x1b[38;5;196;48;5;22mX", Expected: Observation{"fg:0,0": "ansi:196", "bg:0,0": "ansi:22"}, Split: true},
		{ID: "dcs-framing", Capability: "unknown DCS string framing", Input: "A\x1bP999zignored\x1b\\B", Expected: Observation{"cell:0,0": "A", "cell:1,0": "B", "cursor": "2,0"}, Split: true},
		{ID: "malformed-csi", Capability: "cancelled malformed CSI recovery", Input: "A\x1b[31;\x18B", Expected: Observation{"cell:0,0": "A", "cell:1,0": "B", "cursor": "2,0"}, Split: true},
		{ID: "unterminated-osc", Capability: "unterminated string framing", Input: "A\x1b]999;payload", Expected: Observation{"cell:0,0": "A", "cell:1,0": " ", "cursor": "1,0"}, Split: true},
		// Fixed-size adversarial input exercises recovery; it does not establish an
		// asymptotic memory ceiling (that obligation is explicit in Coverage).
		{ID: "oversized-osc", Capability: "large unknown string recovery", Input: "\x1b]999;" + strings.Repeat("x", 16384) + "\x1b\\Z", Expected: Observation{"cell:0,0": "Z", "cursor": "1,0"}},
	}
	for i := range cases {
		cases[i].Width = 8
		cases[i].Height = 4
		if cases[i].Source == "" {
			cases[i].Source = xtermSource
		}
	}
	return cases
}
