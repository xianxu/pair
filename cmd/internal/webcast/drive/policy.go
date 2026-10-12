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

// Chords: the closed table of what a driver may press, as key x exact
// modifier set ("" none; modifiers in the order ctrl, meta, shift). A chord is
// here only if no Chrome or Blink binding handles it on any platform Couch
// runs viewers' casts from (macOS, Linux, Windows); everything else is
// refused. Known bindings kept out: Shift+Delete and Shift+Insert (cut,
// paste), Ctrl/Meta+C/X/V (clipboard), Meta+Y (History), Shift+Escape (task
// manager), Ctrl+Tab and Ctrl+PageUp/PageDown (tabs), Alt+anything (back,
// forward, menus), F-keys (reload, devtools, fullscreen).
//
// The activation keys, Enter and Space, take no modifier at all: they are a
// click's keyboard form, and Blink takes the navigation policy from the
// activating event's modifiers (Ctrl/Meta opens a new tab, Shift a new
// window), so every modifier the pointer is refused is refused here too.
var chords = map[string][]string{
	"Enter":      {""},
	" ":          {""},
	"Tab":        {"", "shift"},
	"Escape":     {""},
	"Backspace":  {""},
	"Delete":     {""},
	"ArrowUp":    {"", "shift"},
	"ArrowDown":  {"", "shift"},
	"ArrowLeft":  {"", "shift"},
	"ArrowRight": {"", "shift"},
	"Home":       {"", "shift"},
	"End":        {"", "shift"},
	"PageUp":     {"", "shift"},
	"PageDown":   {"", "shift"},
}

// printableChords: any printable character takes these; a few letters take
// more (select all, undo, redo).
var (
	printableChords = []string{"", "shift"}
	letterChords    = map[string][]string{
		"a": {"ctrl", "meta"},
		"z": {"ctrl", "meta", "ctrl+shift", "meta+shift"},
		"y": {"ctrl"},
	}
)

// namedKeys a driver may press: editing and moving around the page.
var namedKeys = func() []string {
	out := make([]string, 0, len(chords))
	for k := range chords {
		if k != " " { // Space is a printable key with its own row
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}()

// chordOf is a modifier set in the table's form.
func chordOf(mods []string) string {
	var out []string
	for _, m := range []string{ModCtrl, ModMeta, ModShift, ModAlt} {
		if slices.Contains(mods, m) {
			out = append(out, m)
		}
	}
	return strings.Join(out, "+")
}

// chordAllowed: key with mods is in the table.
func chordAllowed(key string, mods []string) bool {
	chord := chordOf(mods)
	if allowed, ok := chords[key]; ok {
		return slices.Contains(allowed, chord)
	}
	if utf8.RuneCountInString(key) != 1 || !unicode.IsPrint([]rune(key)[0]) {
		return false
	}
	return slices.Contains(printableChords, chord) || slices.Contains(letterChords[strings.ToLower(key)], chord)
}

// Confine is the allow-list every act passes before it reaches the cast tab
// (#466). The rule: what the page alone handles is allowed; anything Chrome
// or the operator's machine would handle is not.
//   - Pointer: the left button only, no modifiers (right opens Chrome's
//     context menu, middle and modified clicks open tabs or windows or
//     download, Ctrl+wheel zooms).
//   - Keys: only the chords in the table above (key x exact modifiers),
//     each one no Chrome or Blink binding handles. Code, when given, must
//     name Key's own key, so an injector that reads it can't be led to
//     another (key "a", code "KeyW").
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
		if !chordAllowed(a.Key, a.Mods) {
			return refuse("that key or shortcut")
		}
		modified := slices.Contains(a.Mods, ModCtrl) || slices.Contains(a.Mods, ModMeta)
		if !codeFits(a.Key, a.Code, modified) {
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
