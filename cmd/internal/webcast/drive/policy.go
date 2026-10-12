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
//   - Keys: the named editing keys and printable characters, with Shift
//     (not Shift+Escape: Chrome's task manager). Ctrl or Meta only for
//     select all (A), undo and redo (Z, Shift+Z), Ctrl+Y redo (Meta+Y opens
//     History on macOS) and Enter. No clipboard keys (copy and cut write
//     the operator's clipboard, paste reads it), no Alt (back and forward),
//     no function keys, no other accelerator (closing the tab, the address
//     bar, devtools). Code, when given, must name Key's own key, so an
//     injector that reads it can't be led to another (key "a", code "KeyW").
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
		ctrl, meta, shift := has(ModCtrl), has(ModMeta), has(ModShift)
		if shift && a.Key == "Escape" {
			return refuse("that shortcut")
		}
		if ctrl || meta {
			switch k := strings.ToLower(a.Key); {
			case k == "z":
			case (k == "a" || k == "enter") && !shift:
			case k == "y" && ctrl && !meta && !shift:
			default:
				return refuse("that shortcut")
			}
		}
		if !codeFits(a.Key, a.Code, ctrl || meta) {
			return refuse("a code that isn't the key's")
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

// punctuationCodes are the physical keys that type symbols.
var punctuationCodes = []string{"Minus", "Equal", "BracketLeft", "BracketRight", "Backslash", "Semicolon", "Quote",
	"Comma", "Period", "Slash", "Backquote", "IntlBackslash", "IntlRo", "IntlYen", "Space", "NumpadAdd",
	"NumpadSubtract", "NumpadMultiply", "NumpadDivide", "NumpadDecimal", "NumpadComma", "NumpadEqual"}

// codeFits: an empty code, or one naming key's own physical key. A named key
// has its own code (Enter also NumpadEnter). A printable character comes
// from a character key: with Ctrl or Meta, exactly its US key (KeyA for
// "a"); without, any character key, since layouts move letters around.
func codeFits(key, code string, modified bool) bool {
	switch {
	case code == "":
		return true
	case slices.Contains(namedKeys, key):
		return code == key || (key == "Enter" && code == "NumpadEnter")
	case key == " ":
		return code == "Space"
	case modified:
		return code == "Key"+strings.ToUpper(key)
	}
	for _, p := range []string{"Key", "Digit", "Numpad"} {
		if rest, ok := strings.CutPrefix(code, p); ok && len(rest) == 1 && (unicode.IsUpper(rune(rest[0])) || unicode.IsDigit(rune(rest[0]))) {
			return true
		}
	}
	return slices.Contains(punctuationCodes, code)
}
