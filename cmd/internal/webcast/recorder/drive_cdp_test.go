package recorder

import (
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
)

func TestCDPAct(t *testing.T) {
	scroll := [2]float64{0, 100}
	moved := map[string]any{"type": "mouseMoved", "x": 10.0, "y": 20.0, "modifiers": 0, "button": "none", "buttons": 0}
	press := func(n int) map[string]any {
		return map[string]any{"type": "mousePressed", "x": 10.0, "y": 20.0, "modifiers": 0, "button": "left", "buttons": 1, "clickCount": n}
	}
	release := func(n int) map[string]any {
		return map[string]any{"type": "mouseReleased", "x": 10.0, "y": 20.0, "modifiers": 0, "button": "left", "buttons": 0, "clickCount": n}
	}
	mouse := func(ps ...map[string]any) []cdpCall {
		var out []cdpCall
		for _, p := range ps {
			out = append(out, cdpCall{"Input.dispatchMouseEvent", p})
		}
		return out
	}
	key := func(p map[string]any) []cdpCall { return []cdpCall{{"Input.dispatchKeyEvent", p}} }
	for _, c := range []struct {
		name string
		act  drive.Act
		want []cdpCall
	}{
		{"click, page to viewport", drive.Act{A: drive.ActClick, ID: 1, X: 10, Y: 120, Button: "left", Count: 1}, mouse(moved, press(1), release(1))},
		{"double click", drive.Act{A: drive.ActClick, ID: 1, X: 10, Y: 120, Button: "left", Count: 2}, mouse(moved, press(1), release(1), press(2), release(2))},
		{"down", drive.Act{A: drive.ActDown, ID: 1, X: 10, Y: 120, Button: "left"}, mouse(moved, press(1))},
		{"drag move", drive.Act{A: drive.ActMove, X: 10, Y: 120, Button: "left"},
			mouse(map[string]any{"type": "mouseMoved", "x": 10.0, "y": 20.0, "modifiers": 0, "button": "left", "buttons": 1})},
		{"hover move", drive.Act{A: drive.ActMove, X: 10, Y: 120}, mouse(moved)},
		{"up", drive.Act{A: drive.ActUp, X: 10, Y: 120, Button: "left"}, mouse(release(1))},
		{"wheel", drive.Act{A: drive.ActWheel, ID: 1, X: 10, Y: 120, DY: 300},
			mouse(map[string]any{"type": "mouseWheel", "x": 10.0, "y": 20.0, "modifiers": 0, "deltaX": 0.0, "deltaY": 300.0})},
		{"letter", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "a", Code: "KeyA"},
			key(map[string]any{"type": "keyDown", "key": "a", "code": "KeyA", "text": "a", "modifiers": 0, "windowsVirtualKeyCode": 65})},
		{"shifted letter", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "A", Mods: []string{"shift"}},
			key(map[string]any{"type": "keyDown", "key": "A", "text": "A", "modifiers": 8, "windowsVirtualKeyCode": 65})},
		{"punctuation has no virtual key", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "."},
			key(map[string]any{"type": "keyDown", "key": ".", "text": ".", "modifiers": 0, "windowsVirtualKeyCode": 0})},
		{"enter", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "Enter"},
			key(map[string]any{"type": "keyDown", "key": "Enter", "text": "\r", "modifiers": 0, "windowsVirtualKeyCode": 13})},
		{"arrow", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "ArrowLeft", Mods: []string{"shift"}},
			key(map[string]any{"type": "rawKeyDown", "key": "ArrowLeft", "modifiers": 8, "windowsVirtualKeyCode": 37})},
		{"key up", drive.Act{A: drive.ActKey, Type: drive.KeyUp, Key: "Enter"},
			key(map[string]any{"type": "keyUp", "key": "Enter", "modifiers": 0, "windowsVirtualKeyCode": 13})},
		{"select all", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "a", Mods: []string{"meta"}},
			key(map[string]any{"type": "rawKeyDown", "key": "a", "modifiers": 4, "windowsVirtualKeyCode": 65, "commands": []string{"selectAll"}})},
		{"undo", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "z", Mods: []string{"ctrl"}},
			key(map[string]any{"type": "rawKeyDown", "key": "z", "modifiers": 2, "windowsVirtualKeyCode": 90, "commands": []string{"undo"}})},
		{"redo", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "z", Mods: []string{"meta", "shift"}},
			key(map[string]any{"type": "rawKeyDown", "key": "z", "modifiers": 12, "windowsVirtualKeyCode": 90, "commands": []string{"redo"}})},
		{"ctrl+y", drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "y", Mods: []string{"ctrl"}},
			key(map[string]any{"type": "rawKeyDown", "key": "y", "modifiers": 2, "windowsVirtualKeyCode": 89, "commands": []string{"redo"}})},
		{"text", drive.Act{A: drive.ActText, V: "invoice"}, []cdpCall{{"Input.insertText", map[string]any{"text": "invoice"}}}},
		{"ime", drive.Act{A: drive.ActIME, V: "にほ", Sel: &[2]int{1, 2}},
			[]cdpCall{{"Input.imeSetComposition", map[string]any{"text": "にほ", "selectionStart": 1, "selectionEnd": 2}}}},
		{"ime caret at end", drive.Act{A: drive.ActIME, V: "にほ"},
			[]cdpCall{{"Input.imeSetComposition", map[string]any{"text": "にほ", "selectionStart": 2, "selectionEnd": 2}}}},
	} {
		if got := cdpAct(c.act, scroll); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %#v\nwant %#v", c.name, got, c.want)
		}
	}
}
