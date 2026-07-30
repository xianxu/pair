// Package keyhelp renders Pair's in-session keybinding help — what Alt+h shows
// (#132).
//
// Wording is NOT authored here. It is derived from the sources that already own
// it — the `desc = 'pair: …'` on each vim.keymap.set, workbenchshortcut's
// GlobalBinding.Help and RoleBinding.Help — so the help cannot drift from the
// bindings the way it did when #99 M5c retired bin/pair-shell and left Alt+h
// paging a CLI usage block whose last line read "In-session keybindings are on
// Alt+h."
//
// Two rules earn their keep here, both learned from real breakage:
//
//   - A binding is identified by (key, CONTEXT), never by key alone. Alt+t/w/r are
//     deliberate no-ops in the draft but new/close/rename tab in the terminal, and
//     Shift+Alt+⏎ means append-no-send in the draft and toggle-layout in the
//     terminal. One row per key would publish one of each pair as a lie.
//   - Every row names the source its wording comes from. There is no "whichever
//     source has prose wins" fallback: that rule is exactly what would ship
//     "right-terminal tab helper disabled in draft" as Alt+t's description.
package keyhelp

// Context is the pane a binding applies in. It is half of a binding's identity.
type Context int

const (
	ContextGlobal   Context = iota // works from any Pair pane
	ContextDraft                   // the nvim draft pane
	ContextTerminal                // the right workbench terminal
)

func (c Context) String() string {
	switch c {
	case ContextDraft:
		return "draft"
	case ContextTerminal:
		return "terminal"
	default:
		return "global"
	}
}

// Source names where a binding's wording comes from. Explicit per row so no join
// can silently borrow the wrong sentence.
type Source int

const (
	SourceNvim   Source = iota // desc = 'pair: …' in nvim/init.lua
	SourceGlobal               // workbenchshortcut.GlobalBinding.Help
	SourceRole                 // workbenchshortcut.RoleBinding.Help
	SourceZellij               // authored in the catalog: zellij Run binds have no upstream prose
)

// Binding is one rendered help row.
type Binding struct {
	Key     string // display form, e.g. "Alt+⏎"
	Desc    string // derived from Source; never authored in this struct
	Context Context
	Group   string
	Order   int
}

// Section is a titled, ordered group of bindings.
type Section struct {
	Title    string
	Bindings []Binding
}
