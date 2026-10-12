package drive

import (
	"errors"
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
