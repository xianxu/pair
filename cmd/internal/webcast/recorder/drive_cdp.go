package recorder

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
)

// cdpCall is one CDP input command.
type cdpCall struct {
	Method string
	Params map[string]any
}

// CDP's modifier bits.
const (
	modAlt   = 1
	modCtrl  = 2
	modMeta  = 4
	modShift = 8
)

// namedKeyCodes: the Windows virtual key codes of the keys a driver may
// press by name (drive's chord table).
var namedKeyCodes = map[string]int{
	"Enter": 13, " ": 32, "Tab": 9, "Escape": 27, "Backspace": 8, "Delete": 46,
	"ArrowLeft": 37, "ArrowUp": 38, "ArrowRight": 39, "ArrowDown": 40,
	"Home": 36, "End": 35, "PageUp": 33, "PageDown": 34,
}

// cdpAct maps one confined act to the CDP calls that perform it in the
// tab: pointer coordinates from top-page CSS px to the viewport by the top
// document's scroll. Keys derive everything from Key (Code only fills the
// DOM event's code); the editing chords Confine allows carry Chrome's
// editing command, which a synthetic Ctrl or Meta doesn't run on macOS.
func cdpAct(a drive.Act, scroll [2]float64) []cdpCall {
	x, y := a.X-scroll[0], a.Y-scroll[1]
	mouse := func(typ string, extra map[string]any) cdpCall {
		p := map[string]any{"type": typ, "x": x, "y": y, "modifiers": mods(a.Mods)}
		for k, v := range extra {
			p[k] = v
		}
		return cdpCall{"Input.dispatchMouseEvent", p}
	}
	button := func(pressed bool) map[string]any {
		if a.Button == "" {
			return map[string]any{"button": "none", "buttons": 0}
		}
		b := map[string]any{"button": a.Button, "buttons": 1}
		if !pressed {
			b["buttons"] = 0
		}
		return b
	}
	with := func(m map[string]any, k string, v any) map[string]any { m[k] = v; return m }
	switch a.A {
	case drive.ActMove:
		return []cdpCall{mouse("mouseMoved", button(true))}
	case drive.ActDown:
		return []cdpCall{mouse("mouseMoved", map[string]any{"button": "none", "buttons": 0}), mouse("mousePressed", with(button(true), "clickCount", 1))}
	case drive.ActUp:
		return []cdpCall{mouse("mouseReleased", with(button(false), "clickCount", 1))}
	case drive.ActClick:
		out := []cdpCall{mouse("mouseMoved", map[string]any{"button": "none", "buttons": 0})}
		for n := 1; n <= a.Count; n++ {
			out = append(out, mouse("mousePressed", with(button(true), "clickCount", n)), mouse("mouseReleased", with(button(false), "clickCount", n)))
		}
		return out
	case drive.ActWheel:
		return []cdpCall{mouse("mouseWheel", map[string]any{"deltaX": a.DX, "deltaY": a.DY})}
	case drive.ActKey:
		return []cdpCall{{"Input.dispatchKeyEvent", keyParams(a)}}
	case drive.ActText:
		return []cdpCall{{"Input.insertText", map[string]any{"text": a.V}}}
	case drive.ActIME:
		p := map[string]any{"text": a.V, "selectionStart": utf8.RuneCountInString(a.V), "selectionEnd": utf8.RuneCountInString(a.V)}
		if a.Sel != nil {
			p["selectionStart"], p["selectionEnd"] = a.Sel[0], a.Sel[1]
		}
		return []cdpCall{{"Input.imeSetComposition", p}}
	}
	return nil
}

func mods(ms []string) int {
	bits := 0
	for _, m := range ms {
		switch m {
		case drive.ModAlt:
			bits |= modAlt
		case drive.ModCtrl:
			bits |= modCtrl
		case drive.ModMeta:
			bits |= modMeta
		case drive.ModShift:
			bits |= modShift
		}
	}
	return bits
}

func keyParams(a drive.Act) map[string]any {
	m := mods(a.Mods)
	p := map[string]any{"key": a.Key, "modifiers": m}
	if a.Code != "" {
		p["code"] = a.Code
	}
	code, named := namedKeyCodes[a.Key]
	printable := !named && utf8.RuneCountInString(a.Key) == 1
	if printable {
		// Letters and digits have their own virtual key; for anything else
		// a guess could name another key (".", 46, is Delete's), so none.
		if c := strings.ToUpper(a.Key)[0]; len(a.Key) == 1 && (c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			code = int(c)
		}
	}
	p["windowsVirtualKeyCode"] = code
	if a.Type == drive.KeyUp {
		p["type"] = "keyUp"
		return p
	}
	editing := m&(modCtrl|modMeta) != 0
	switch {
	case editing:
		p["type"] = "rawKeyDown"
		if c := editCommand(strings.ToLower(a.Key), a.Mods); c != "" {
			p["commands"] = []string{c}
		}
	case printable:
		p["type"], p["text"] = "keyDown", a.Key
	case a.Key == "Enter":
		p["type"], p["text"] = "keyDown", "\r"
	case a.Key == " ":
		p["type"], p["text"] = "keyDown", " "
	default:
		p["type"] = "rawKeyDown"
	}
	return p
}

// editCommand is the editing command of a Ctrl/Meta chord Confine allows.
func editCommand(key string, ms []string) string {
	shift := slices.Contains(ms, drive.ModShift)
	switch {
	case key == "a":
		return "selectAll"
	case key == "z" && shift, key == "y":
		return "redo"
	case key == "z":
		return "undo"
	}
	return ""
}
