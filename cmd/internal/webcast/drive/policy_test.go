package drive

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestConfine(t *testing.T) {
	key := func(k string, mods ...string) Act { return Act{A: ActKey, Type: KeyDown, Key: k, Mods: mods} }
	click := func(button string, mods ...string) Act {
		return Act{A: ActClick, ID: 1, X: 1, Y: 1, Button: button, Mods: mods}
	}
	for _, c := range []struct {
		name string
		act  Act
		ok   bool
	}{
		{"left click", click(ButtonLeft), true},
		{"right click (context menu)", click("right"), false},
		{"middle click (new tab)", click("middle"), false},
		{"ctrl+click (new tab)", click(ButtonLeft, ModCtrl), false},
		{"shift+click (new window)", click(ButtonLeft, ModShift), false},
		{"alt+click (download)", click(ButtonLeft, ModAlt), false},
		{"ctrl+wheel (zoom)", Act{A: ActWheel, ID: 1, DY: 100, Mods: []string{ModCtrl}}, false},
		{"wheel", Act{A: ActWheel, ID: 1, DY: 100}, true},
		{"drag move", Act{A: ActMove, Button: ButtonLeft}, true},
		{"Enter", key("Enter"), true},
		{"Tab", key("Tab"), true},
		{"shift+Tab", key("Tab", ModShift), true},
		{"Escape", key("Escape"), true},
		{"ArrowLeft", key("ArrowLeft"), true},
		{"printable", key("q"), true},
		{"shift printable", key("Q", ModShift), true},
		{"space", key(" "), true},
		{"select all", key("a", ModMeta), true},
		{"undo", key("z", ModCtrl), true},
		{"redo", key("z", ModMeta, ModShift), true},
		{"shift+delete (cut on Windows and Linux)", key("Delete", ModShift), false},
		{"shift+backspace", key("Backspace", ModShift), false},
		{"ctrl+tab (next tab)", key("Tab", ModCtrl), false},
		{"ctrl+y redo", key("y", ModCtrl), true},
		{"meta+y (History on macOS)", key("y", ModMeta), false},
		{"shift+escape (task manager)", key("Escape", ModShift), false},
		{"shift+meta+a (tab search)", key("a", ModMeta, ModShift), false},
		{"ctrl+pagedown (next tab)", key("PageDown", ModCtrl), false},
		{"key a dressed as KeyW with meta", Act{A: ActKey, Type: KeyDown, Key: "a", Code: "KeyW", Mods: []string{ModMeta}}, false},
		{"key a with code F12", Act{A: ActKey, Type: KeyDown, Key: "a", Code: "F12"}, false},
		{"Enter with code Escape", Act{A: ActKey, Type: KeyDown, Key: "Enter", Code: "Escape"}, false},
		{"meta+a with its code", Act{A: ActKey, Type: KeyDown, Key: "a", Code: "KeyA", Mods: []string{ModMeta}}, true},
		{"AZERTY a on KeyQ", Act{A: ActKey, Type: KeyDown, Key: "a", Code: "KeyQ"}, true},
		{"numpad enter", Act{A: ActKey, Type: KeyDown, Key: "Enter", Code: "NumpadEnter"}, true},
		{"slash", Act{A: ActKey, Type: KeyDown, Key: "/", Code: "Slash"}, true},
		{"space", Act{A: ActKey, Type: KeyDown, Key: " ", Code: "Space"}, true},
		{"meta+enter (a link in a new tab)", key("Enter", ModMeta), false},
		{"shift+enter (a link in a new window)", key("Enter", ModShift), false},
		{"shift+space (a submit into a new window)", key(" ", ModShift), false},
		{"copy", key("c", ModMeta), false},
		{"cut", key("x", ModCtrl), false},
		{"paste (the operator's clipboard)", key("v", ModMeta), false},
		{"close tab", key("w", ModMeta), false},
		{"address bar", key("l", ModMeta), false},
		{"new tab", key("t", ModCtrl), false},
		{"devtools", key("i", ModMeta, ModAlt), false},
		{"back", key("ArrowLeft", ModAlt), false},
		{"F12", key("F12"), false},
		{"F5", key("F5"), false},
		{"unknown named key", key("BrowserBack"), false},
		{"text", Act{A: ActText, V: "hello"}, true},
		{"text with a control character", Act{A: ActText, V: "a\x1bb"}, false},
		{"ime", Act{A: ActIME, V: "にほ"}, true},
	} {
		err := Confine(c.act)
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok && !errors.Is(err, ErrRefused) {
			t.Errorf("%s: got %v, want ErrRefused", c.name, err)
		}
	}
}

// Every key a driver could name, against every modifier set, compared with
// an expectation written here by hand (not derived from Confine's table), and
// with the Chrome and Blink bindings that must stay out.
func TestConfineAllowsExactlyTheseChords(t *testing.T) {
	// The expectation: key -> the exact modifier sets allowed.
	want := map[string][]string{
		"Enter": {""}, " ": {""}, "Tab": {"", "shift"}, "Escape": {""}, "Backspace": {""}, "Delete": {""},
		"ArrowUp": {"", "shift"}, "ArrowDown": {"", "shift"}, "ArrowLeft": {"", "shift"}, "ArrowRight": {"", "shift"},
		"Home": {"", "shift"}, "End": {"", "shift"}, "PageUp": {"", "shift"}, "PageDown": {"", "shift"},
		"a": {"", "shift", "ctrl", "meta"}, "z": {"", "shift", "ctrl", "meta", "ctrl+shift", "meta+shift"},
		"y": {"", "shift", "ctrl"}, "c": {"", "shift"}, "v": {"", "shift"}, "x": {"", "shift"}, "w": {"", "shift"},
		"l": {"", "shift"}, "t": {"", "shift"}, "q": {"", "shift"}, "1": {"", "shift"}, "/": {"", "shift"},
		"F12": nil, "F5": nil, "Insert": nil, "BrowserBack": nil, "Meta": nil,
	}
	mods := []string{ModCtrl, ModMeta, ModShift, ModAlt}
	for k, allowed := range want {
		for set := 0; set < 1<<len(mods); set++ {
			var ms []string
			for i, m := range mods {
				if set&(1<<i) != 0 {
					ms = append(ms, m)
				}
			}
			chord := strings.Join(ms, "+")
			ok := Confine(Act{A: ActKey, Type: KeyDown, Key: k, Mods: ms}) == nil
			if ok != slices.Contains(allowed, chord) {
				t.Errorf("%q with %q: Confine allows %v, want %v", k, chord, ok, !ok)
			}
		}
	}
	// Chrome's accelerator table (chrome/browser/ui/accelerator_table.cc,
	// global_keyboard_shortcuts_mac.mm), Blink's editing key bindings
	// (editing_behavior.cc: Shift+Delete cut, Shift+Insert paste), and
	// Blink's activation rules (a link or submit activated by Enter or Space
	// takes the event's modifiers as its navigation policy).
	for _, b := range [][]string{
		{"Delete", ModShift}, {"Insert", ModShift}, {"Insert", ModCtrl}, {"c", ModMeta}, {"c", ModCtrl}, {"x", ModCtrl},
		{"v", ModMeta}, {"v", ModCtrl, ModShift}, {"y", ModMeta}, {"Escape", ModShift}, {"Tab", ModCtrl},
		{"PageUp", ModCtrl}, {"PageDown", ModMeta}, {"w", ModMeta}, {"t", ModCtrl}, {"l", ModMeta}, {"q", ModMeta},
		{"ArrowLeft", ModAlt}, {"ArrowLeft", ModMeta}, {"Home", ModAlt}, {"F12"}, {"F5"}, {"i", ModMeta, ModAlt},
		{"a", ModMeta, ModShift}, {"1", ModCtrl}, {"BrowserBack"},
		{"Enter", ModCtrl}, {"Enter", ModMeta}, {"Enter", ModShift}, {"Enter", ModAlt}, {" ", ModShift},
		{" ", ModCtrl}, {" ", ModMeta},
	} {
		if Confine(Act{A: ActKey, Type: KeyDown, Key: b[0], Mods: b[1:]}) == nil {
			t.Errorf("a Chrome or Blink binding passes: %v", b)
		}
	}
}
