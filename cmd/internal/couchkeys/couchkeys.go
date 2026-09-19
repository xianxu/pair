// Package couchkeys declares Couch's keyboard contract: which chords Couch
// takes, where it acts on them, and what they mean (#282).
//
// It is data. couchtty's Interceptor and Console routing derive from it,
// `couch --help` renders it, and Pair's Alt+h page shows it when Couch presents
// the thread. It is a package of its own so Pair can describe Couch's keys
// without importing Couch's console: Couch layers above Pair, and Pair reads
// only this declaration.
package couchkeys

import (
	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// SwitchLegacy is ctrl-space in the LEGACY encoding: ctrl-@ is NUL.
const SwitchLegacy = 0x00

// PreviousLegacy is ctrl+backspace in the LEGACY encoding. Like SwitchLegacy it
// is a bare-byte navigation encoding rather than a multi-byte known sequence.
//
// Accepted cost, deliberate and not a discovery: in legacy encoding 0x08 IS
// ^H, so intercepting ctrl+backspace also takes ctrl-h from the child (readline
// and nvim insert-mode treat it as backspace). Under the Kitty protocol the two
// separate cleanly -- \x1b[104;5u vs \x1b[127;5u -- and zellij pushes the
// protocol, so this only bites with the protocol off.
const PreviousLegacy = 0x08

// NewestPageSequence is ctrl+return under the Kitty protocol: codepoint 13 with
// modifier bitmask 4 encoded as 4+1, the same construction as ctrl-space's row.
//
// It has no legacy form: CR is also ordinary Return. Couch maintains the
// disambiguation flag while it owns a supporting terminal; an unsupported host
// retains Ctrl+Space then Return as the notification-jump fallback.
//
// Named because two sites need the same bytes: its row in the table below, and
// the panel arm of couchtty's onNewestPageHotkey, which hands them to the
// panel's decoder.
const NewestPageSequence = "\x1b[13;5u"

// Scope is where Couch acts on a chord.
type Scope uint8

const (
	// ScopeEveryPane: Couch takes the chord from whichever Pair pane is
	// displayed; Pair never sees it.
	ScopeEveryPane Scope = iota + 1
	// ScopeSwitcher: Couch acts on the chord only while its switcher has focus.
	// While a Pair pane is displayed the same bytes reach Pair (#245), which has
	// its own meaning for them.
	ScopeSwitcher
)

// Action is what Couch does; couchtty maps each to its console dispatch.
type Action uint8

const (
	ActionSwitch Action = iota + 1
	ActionPrevious
	ActionNewestPage
	ActionDetach
	ActionPark
	ActionRelaunch
)

// Binding is one chord Couch declares. Chord is the Pair chord it shares (a
// switcher chord) or takes (an every-pane chord whose bytes are Pair's); zero
// for a Couch-only key.
type Binding struct {
	Action    Action
	Scope     Scope
	Chord     workbenchshortcut.Chord
	Key, Help string
	Encodings [][]byte
}

// bindings is the table, in help order. Help wording avoids "start", "park"
// and "resume": `couch --help` renders it and keeps internal operation names
// off its public surface.
var bindings = []Binding{
	{Action: ActionSwitch, Scope: ScopeEveryPane, Key: "Ctrl+Space", Help: "open the Couch switcher",
		Encodings: [][]byte{{SwitchLegacy}, []byte("\x1b[32;5u")}},
	{Action: ActionPrevious, Scope: ScopeEveryPane, Key: "Ctrl+Backspace", Help: "return to the previous Couch thread",
		Encodings: [][]byte{{PreviousLegacy}, []byte("\x1b[127;5u")}},
	{Action: ActionNewestPage, Scope: ScopeEveryPane, Key: "Ctrl+Return", Help: "jump to the newest notification thread",
		Encodings: [][]byte{[]byte(NewestPageSequence), []byte("\x1b[13;5:1u"), []byte("\x1b[13;5:2u")}},
	// Relaunch is taken from every pane (#284): Pair's own restart refuses under
	// Couch, so passing it through left a key that confirmed and did nothing.
	// From a Pair pane it relaunches the thread on screen, in the switcher the
	// highlighted row.
	pairChord(ScopeEveryPane, ActionRelaunch, workbenchshortcut.ChordAltN, "Alt+n", "relaunch this thread on the current binary, keeping its conversation"),
	pairChord(ScopeEveryPane, ActionRelaunch, workbenchshortcut.ChordCtrlAltN, "Ctrl+Alt+n", "same as Alt+n"),
	pairChord(ScopeSwitcher, ActionDetach, workbenchshortcut.ChordAltD, "Alt+d", "detach every live thread and leave Couch; their sessions keep running"),
	pairChord(ScopeSwitcher, ActionPark, workbenchshortcut.ChordAltX, "Alt+x", "shut down every live thread and leave Couch (asks first)"),
}

// pairChord declares a chord whose bytes are Pair's: a switcher chord shares
// them (outside the switcher they ARE Pair's key), an every-pane chord takes
// them. Either way they come from Pair's table rather than being restated.
func pairChord(scope Scope, action Action, chord workbenchshortcut.Chord, key, help string) Binding {
	return Binding{Action: action, Scope: scope, Chord: chord, Key: key, Help: help, Encodings: workbenchshortcut.ChordEncodings(chord)}
}

// Bindings returns every declared chord in help order, deep-copied so no
// caller can edit the contract every other reader sees.
func Bindings() []Binding {
	out := make([]Binding, len(bindings))
	for i, b := range bindings {
		b.Encodings = make([][]byte, len(bindings[i].Encodings))
		for n, e := range bindings[i].Encodings {
			b.Encodings[n] = append([]byte(nil), e...)
		}
		out[i] = b
	}
	return out
}

const (
	helpTitleEveryPane = "Couch — from every Pair pane (Couch takes these first)"
	helpTitleSwitcher  = "Couch switcher"
)

// HelpSections is Couch's layer of the key help: one section per scope, in
// table order, worded by the table and nothing else. `couch --help` and Pair's
// Alt+h page both render it (#282).
func HelpSections(bs []Binding) []keyhelp.Section {
	var pane, menu []keyhelp.Binding
	opener := ""
	for i, b := range bs {
		row := keyhelp.Binding{Key: b.Key, Desc: b.Help, Order: i, Chord: b.Chord}
		switch b.Scope {
		case ScopeEveryPane:
			row.Context, row.Group = keyhelp.ContextHost, helpTitleEveryPane
			pane = append(pane, row)
			if b.Action == ActionSwitch && opener == "" {
				opener = b.Key
			}
		case ScopeSwitcher:
			row.Context = keyhelp.ContextHostMenu
			menu = append(menu, row)
		}
	}
	title := helpTitleSwitcher
	if opener != "" {
		title += " (open it with " + opener + ")"
	}
	var out []keyhelp.Section
	if len(pane) > 0 {
		out = append(out, keyhelp.Section{Title: helpTitleEveryPane, Bindings: pane})
	}
	if len(menu) > 0 {
		for i := range menu {
			menu[i].Group = title
		}
		out = append(out, keyhelp.Section{Title: title, Bindings: menu})
	}
	return out
}

// Claimed is every Pair chord Couch takes from every pane: the rows Pair's
// layer gives up (keyhelp.Layer).
func Claimed(bs []Binding) []workbenchshortcut.Chord {
	var out []workbenchshortcut.Chord
	for _, b := range bs {
		if b.Scope == ScopeEveryPane && b.Chord != 0 {
			out = append(out, b.Chord)
		}
	}
	return out
}
