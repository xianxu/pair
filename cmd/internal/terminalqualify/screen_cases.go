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
		// Attribute observations expose the pinned API's mask: bold=1, faint=2,
		// italic=4, blink=8, rapid blink=16, reverse=32, conceal=64, strike=128.
		// Literal protocol expectations remain independent of the candidate parser.
		{ID: "sgr-bold", Capability: "SGR bold set and reset", Input: "\x1b[1mX\x1b[22mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "1", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-faint", Capability: "SGR faint set and reset", Input: "\x1b[2mX\x1b[22mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "2", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-italic", Capability: "SGR italic set and reset", Input: "\x1b[3mX\x1b[23mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "4", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-slow-blink", Capability: "SGR slow-blink set and reset", Input: "\x1b[5mX\x1b[25mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "8", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-rapid-blink", Capability: "SGR rapid-blink set and reset", Input: "\x1b[6mX\x1b[25mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "16", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-reverse", Capability: "SGR reverse set and reset", Input: "\x1b[7mX\x1b[27mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "32", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-conceal", Capability: "SGR conceal set and reset", Input: "\x1b[8mX\x1b[28mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "64", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},
		{ID: "sgr-strike", Capability: "SGR strike set and reset", Input: "\x1b[9mX\x1b[29mY", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "128", "cell:1,0": "Y", "attrs:1,0": "0"}, Split: true},

		{ID: "sgr-selective-reset", Capability: "SGR selective reset preserves unrelated attributes", Input: "\x1b[1;2;3;7;8;9mA\x1b[22mB\x1b[23mC\x1b[27mD\x1b[28mE\x1b[29mF\x1b[1;4mG\x1b[0mH", Expected: Observation{"attrs:0,0": "231", "attrs:1,0": "228", "attrs:2,0": "224", "attrs:3,0": "192", "attrs:4,0": "128", "attrs:5,0": "0", "attrs:6,0": "1", "underline:6,0": "1", "attrs:7,0": "0", "underline:7,0": "0"}, Split: true},
		{ID: "sgr-underline-styles", Capability: "SGR underline styles and selective reset", Input: "\x1b[1;4mA\x1b[4:2mB\x1b[4:3mC\x1b[4:4mD\x1b[4:5mE\x1b[24mF", Expected: Observation{"underline:0,0": "1", "underline:1,0": "2", "underline:2,0": "3", "underline:3,0": "4", "underline:4,0": "5", "underline:5,0": "0", "attrs:5,0": "1"}, Split: true},
		{ID: "sgr-underline-color", Capability: "SGR underline color set, reset, and preservation", Input: "\x1b[4;58;2;18;52;86mA\x1b[24mB\x1b[4;58;5;196mC\x1b[59mD\x1b[58;5;22mE\x1b[0mF", Expected: Observation{"underline-color:0,0": "#123456", "underline-color:1,0": "#123456", "underline:1,0": "0", "underline-color:2,0": "ansi:196", "underline-color:3,0": "", "underline:3,0": "1", "underline-color:4,0": "ansi:22", "underline-color:5,0": "", "underline:5,0": "0"}, Split: true},
		{ID: "sgr-scroll-preservation", Capability: "screen scrolling retains complete cell style after pen reset", Input: "\x1b[2;1H\x1b[1;3;7;4:3;58;2;18;52;86mX\x1b[0m\x1b[4;1H\n", Expected: Observation{"cell:0,0": "X", "attrs:0,0": "37", "underline:0,0": "3", "underline-color:0,0": "#123456", "attrs:0,3": "0", "underline:0,3": "0", "underline-color:0,3": ""}, Split: true},

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
