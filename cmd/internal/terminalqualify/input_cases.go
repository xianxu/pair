package terminalqualify

import (
	uv "github.com/charmbracelet/ultraviolet"
	vt "github.com/charmbracelet/x/vt"
)

const kittySource = "https://sw.kovidgoyal.net/kitty/keyboard-protocol/"

// InputCases compares wire bytes to documented encodings. All events are sent
// through the public candidate API; lack of negotiation is a failure, not a skip.
func InputCases() []Case {
	cases := []Case{
		{ID: "keyboard-query", Capability: "Kitty keyboard negotiation", Source: kittySource, Input: "\x1b[>1u\x1b[?u", Expected: Observation{"replies": "\x1b[?1u"}, Split: true},
		{ID: "ctrl-return", Capability: "existing Ctrl-Return shortcut", Source: kittySource, Input: "\x1b[>1u", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyPressEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl}) }, Expected: Observation{"replies": "\x1b[13;5u"}},
		{ID: "alt-up", Capability: "existing Alt-Up shortcut", Input: "", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyPressEvent{Code: uv.KeyUp, Mod: uv.ModAlt}) }, Expected: Observation{"replies": "\x1b[1;3A"}},
		{ID: "key-repeat", Capability: "Kitty repeat encoding", Source: kittySource, Input: "\x1b[>3u", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyPressEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl, IsRepeat: true}) }, Expected: Observation{"replies": "\x1b[13;5:2u"}},
		{ID: "key-release", Capability: "Kitty release encoding", Source: kittySource, Input: "\x1b[>3u", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyReleaseEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl}) }, Expected: Observation{"replies": "\x1b[13;5:3u"}},
		{ID: "application-cursor", Capability: "application cursor mode", Input: "\x1b[?1h", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyPressEvent{Code: uv.KeyUp}) }, Expected: Observation{"replies": "\x1bOA"}},
		{ID: "normal-cursor", Capability: "normal cursor mode", Input: "\x1b[?1h\x1b[?1l", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyPressEvent{Code: uv.KeyUp}) }, Expected: Observation{"replies": "\x1b[A"}},
		{ID: "application-keypad", Capability: "application keypad", Input: "\x1b=", Action: func(e *vt.Emulator) { e.SendKey(uv.KeyPressEvent{Code: uv.KeyKpEnter}) }, Expected: Observation{"replies": "\x1bOM"}},
		{ID: "mouse-off", Capability: "mouse disabled", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft}) }, Expected: Observation{"replies": ""}},
		{ID: "mouse-click", Capability: "SGR mouse click", Input: "\x1b[?1000h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft}) }, Expected: Observation{"replies": "\x1b[<0;3;2M"}},
		{ID: "mouse-release", Capability: "SGR mouse release", Input: "\x1b[?1000h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseReleaseEvent{X: 2, Y: 1, Button: uv.MouseLeft}) }, Expected: Observation{"replies": "\x1b[<0;3;2m"}},
		{ID: "mouse-click-suppresses-motion", Capability: "click mode rejects drag motion", Input: "\x1b[?1000h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseMotionEvent{X: 2, Y: 1, Button: uv.MouseLeft}) }, Expected: Observation{"replies": ""}},
		{ID: "mouse-drag", Capability: "SGR button motion", Input: "\x1b[?1002h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseMotionEvent{X: 2, Y: 1, Button: uv.MouseLeft}) }, Expected: Observation{"replies": "\x1b[<32;3;2M"}},
		{ID: "mouse-drag-suppresses-hover", Capability: "button mode rejects no-button motion", Input: "\x1b[?1002h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseMotionEvent{X: 2, Y: 1, Button: uv.MouseNone}) }, Expected: Observation{"replies": ""}},
		{ID: "mouse-all-motion", Capability: "SGR all motion", Input: "\x1b[?1003h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseMotionEvent{X: 2, Y: 1, Button: uv.MouseNone}) }, Expected: Observation{"replies": "\x1b[<35;3;2M"}},
		{ID: "mouse-replace-mode", Capability: "mouse tracking modes replace each other", Input: "\x1b[?1003h\x1b[?1000h\x1b[?1006h", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseMotionEvent{X: 2, Y: 1, Button: uv.MouseNone}) }, Expected: Observation{"replies": ""}},
		{ID: "mouse-disable", Capability: "mouse reporting reset", Input: "\x1b[?1000h\x1b[?1006h\x1b[?1000l", Action: func(e *vt.Emulator) { e.SendMouse(uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft}) }, Expected: Observation{"replies": ""}},
		{ID: "focus", Capability: "focus reports", Input: "\x1b[?1004h", Action: func(e *vt.Emulator) { e.Focus(); e.Blur() }, Expected: Observation{"replies": "\x1b[I\x1b[O"}},
		{ID: "focus-off", Capability: "disabled focus suppression", Action: func(e *vt.Emulator) { e.Focus(); e.Blur() }, Expected: Observation{"replies": ""}},
		{ID: "paste", Capability: "bracketed paste", Input: "\x1b[?2004h", Action: func(e *vt.Emulator) { e.Paste("a\nb") }, Expected: Observation{"replies": "\x1b[200~a\nb\x1b[201~"}},
		{ID: "paste-off", Capability: "ordinary paste", Action: func(e *vt.Emulator) { e.Paste("ab") }, Expected: Observation{"replies": "ab"}},
		{ID: "cursor-query", Capability: "cursor position report", Input: "\x1b[2;3H\x1b[6n", Expected: Observation{"replies": "\x1b[2;3R"}, Split: true},
		{ID: "status-query", Capability: "device status report", Input: "\x1b[5n", Expected: Observation{"replies": "\x1b[0n"}, Split: true},
		{ID: "mode-query", Capability: "private mode report", Input: "\x1b[?1h\x1b[?1$p", Expected: Observation{"replies": "\x1b[?1;1$y"}, Split: true},
		{ID: "title", Capability: "OSC title callback", Input: "\x1b]2;probe title\x1b\\", Expected: Observation{"title": "probe title"}, Split: true},
		{ID: "cwd", Capability: "OSC working directory callback", Input: "\x1b]7;file://localhost/tmp/probe\x1b\\", Expected: Observation{"cwd": "file://localhost/tmp/probe"}, Split: true},
		{ID: "bell", Capability: "bell callback", Input: "\a\a", Expected: Observation{"bells": "2"}, Split: true},
		{ID: "cursor-visible", Capability: "cursor visibility callback", Input: "\x1b[?25l", Expected: Observation{"cursor-visible": "false"}, Split: true},
		{ID: "cursor-style", Capability: "cursor shape callback", Input: "\x1b[5 q", Expected: Observation{"cursor-style": "2,true"}, Split: true},
		{ID: "sync-query", Capability: "synchronized output negotiation", Source: "https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036", Input: "\x1b[?2026h\x1b[?2026$p", Expected: Observation{"replies": "\x1b[?2026;1$y"}, Split: true},
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
