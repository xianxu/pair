package drive

import (
	"errors"
	"slices"
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
		{"submit", key("Enter", ModMeta), true},
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

// Every key a driver could name, against every modifier set: Confine
// allows exactly the table's chords, and none of Chrome's or Blink's known
// bindings.
func TestConfineIsExactlyTheChordTable(t *testing.T) {
	mods := []string{ModCtrl, ModMeta, ModShift, ModAlt}
	keys := append(slices.Clone(namedKeys), "a", "z", "y", "c", "v", "x", "w", "l", "t", "q", "1", " ", "/", "F12", "Insert", "BrowserBack")
	for _, k := range keys {
		for set := 0; set < 1<<len(mods); set++ {
			var ms []string
			for i, m := range mods {
				if set&(1<<i) != 0 {
					ms = append(ms, m)
				}
			}
			want := chordAllowed(k, ms)
			got := Confine(Act{A: ActKey, Type: KeyDown, Key: k, Mods: ms}) == nil
			if got != want {
				t.Errorf("%q %v: Confine allows %v, the table %v", k, ms, got, want)
			}
		}
	}
	// Chrome and Blink bindings (any platform) that must stay out.
	for _, b := range [][]string{
		{"Delete", ModShift}, {"Insert", ModShift}, {"Insert", ModCtrl}, {"c", ModMeta}, {"c", ModCtrl}, {"x", ModCtrl},
		{"v", ModMeta}, {"v", ModCtrl, ModShift}, {"y", ModMeta}, {"Escape", ModShift}, {"Tab", ModCtrl},
		{"PageUp", ModCtrl}, {"PageDown", ModMeta}, {"w", ModMeta}, {"t", ModCtrl}, {"l", ModMeta}, {"q", ModMeta},
		{"ArrowLeft", ModAlt}, {"ArrowLeft", ModMeta}, {"Home", ModAlt}, {"F12"}, {"F5"}, {"i", ModMeta, ModAlt},
		{"a", ModMeta, ModShift}, {"Enter", ModAlt}, {"1", ModCtrl}, {"BrowserBack"},
	} {
		if Confine(Act{A: ActKey, Type: KeyDown, Key: b[0], Mods: b[1:]}) == nil {
			t.Errorf("a Chrome or Blink binding passes: %v", b)
		}
	}
}
