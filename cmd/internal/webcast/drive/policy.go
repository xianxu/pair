package drive

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrRefused: an act that would reach past the page (browser UI, the
// operator's clipboard, new windows, devtools). The injector dispatches
// nothing from a batch with one.
var ErrRefused = errors.New("drive: not allowed while driving")

// namedKeys a driver may press: editing and moving around the page.
var namedKeys = []string{"Enter", "Tab", "Escape", "Backspace", "Delete", "ArrowUp", "ArrowDown", "ArrowLeft",
	"ArrowRight", "Home", "End", "PageUp", "PageDown"}

// Confine is the allow-list every act passes before it reaches the cast tab
// (#466). The rule: what the page alone handles is allowed; anything Chrome
// or the operator's machine would handle is not.
//   - Pointer: the left button only, no modifiers (right opens Chrome's
//     context menu, middle and modified clicks open tabs or windows or
//     download, Ctrl+wheel zooms).
//   - Keys: the named editing keys and printable characters, with Shift;
//     Ctrl or Meta only for select all, undo, redo and submit (Enter). No clipboard keys
//     (copy and cut write the operator's clipboard, paste reads it), no Alt
//     (back and forward), no function keys, no other accelerator (closing
//     the tab, the address bar, devtools).
//   - Text and composition: no control characters.
func Confine(a Act) error {
	refuse := func(what string) error { return fmt.Errorf("%w: %s", ErrRefused, what) }
	switch a.A {
	case ActDown, ActUp, ActMove, ActClick, ActWheel:
		if a.Button != "" && a.Button != ButtonLeft {
			return refuse("only the left button")
		}
		if len(a.Mods) > 0 {
			return refuse("no modifiers with the pointer")
		}
	case ActKey:
		has := func(m string) bool { return slices.Contains(a.Mods, m) }
		if has(ModAlt) {
			return refuse("no Alt")
		}
		printable := utf8.RuneCountInString(a.Key) == 1 && unicode.IsPrint([]rune(a.Key)[0])
		if !printable && !slices.Contains(namedKeys, a.Key) {
			return refuse("that key")
		}
		if has(ModCtrl) || has(ModMeta) {
			switch k := strings.ToLower(a.Key); {
			case k == "z", k == "enter":
			case (k == "a" || k == "y") && !has(ModShift):
			default:
				return refuse("that shortcut")
			}
		}
	case ActText, ActIME:
		for _, r := range a.V {
			if unicode.IsControl(r) {
				return refuse("control characters")
			}
		}
	default:
		return refuse("unknown act")
	}
	return nil
}
