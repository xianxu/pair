package keyhelp

import "github.com/xianxu/pair/cmd/internal/workbenchshortcut"

// Group titles, in render order.
const (
	groupDraft    = "Draft — compose and send"
	groupHistory  = "Draft — history and queue"
	groupPanes    = "Panes and layout"
	groupTerminal = "Terminal tabs (in the right terminal)"
	groupSession  = "Session"
)

var groupOrder = []string{groupDraft, groupHistory, groupPanes, groupTerminal, groupSession}

// entry is one curated row. It holds NO wording: Desc comes from the named Source.
//
// The exception is SourceZellij, whose two binds (Alt+h, Alt+l) have no upstream
// prose anywhere — zellij's KDL has no description field — so Help is authored here
// and the comment says so rather than leaving a silent inconsistency.
type entry struct {
	Key     string // the SOURCE's own spelling, e.g. "<M-CR>" or "Alt h"
	Display string // display override; empty means derive from Key
	Group   string
	Order   int
	Context Context
	Source  Source
	Help    string // ONLY for SourceZellij
}

// catalog decides INCLUSION, GROUP and ORDER — never wording.
//
// Why inclusion is explicit rather than "document every keymap": the sources carry
// editor mechanics (autopair, jump-over, completion cycling, spell digits, a
// quit-blocked warning) whose descs are perfectly good for `:map` output and
// actively wrong as workbench keybinding help. Auto-including them would replace
// #132's empty help with noisy help. The internal list is the other half of the
// decision, and TestEveryNvimKeymapIsClassified makes both halves mandatory.
type catalog struct {
	include  []entry
	internal []string
}

// Catalog is the curated help table.
var Catalog = catalog{
	include: []entry{
		// --- Draft: compose and send -------------------------------------
		{Key: "<M-CR>", Display: "Alt+⏎", Group: groupDraft, Order: 10, Context: ContextDraft, Source: SourceNvim},
		{Key: "<S-M-CR>", Display: "Shift+Alt+⏎", Group: groupDraft, Order: 20, Context: ContextDraft, Source: SourceNvim},
		{Key: "<C-c>", Display: "Ctrl+c", Group: groupDraft, Order: 30, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-i>", Display: "Alt+i", Group: groupDraft, Order: 40, Context: ContextDraft, Source: SourceNvim},
		{Key: "<C-_>", Display: "Ctrl+/", Group: groupDraft, Order: 50, Context: ContextDraft, Source: SourceNvim},
		{Key: "'<M-' .. i .. '>'", Display: "Alt+1…9", Group: groupDraft, Order: 60, Context: ContextDraft, Source: SourceNvim},

		// --- Draft: history and queue ------------------------------------
		{Key: "<M-Left>", Display: "Alt+←", Group: groupHistory, Order: 10, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-Right>", Display: "Alt+→", Group: groupHistory, Order: 20, Context: ContextDraft, Source: SourceNvim},
		{Key: "<S-M-Left>", Display: "Shift+Alt+←", Group: groupHistory, Order: 30, Context: ContextDraft, Source: SourceNvim},
		{Key: "<S-M-Right>", Display: "Shift+Alt+→", Group: groupHistory, Order: 40, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-q>", Display: "Alt+q", Group: groupHistory, Order: 50, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-BS>", Display: "Alt+⌫", Group: groupHistory, Order: 60, Context: ContextDraft, Source: SourceNvim},
		{Key: "<S-M-BS>", Display: "Shift+Alt+⌫", Group: groupHistory, Order: 70, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-/>", Display: "Alt+/", Group: groupHistory, Order: 80, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-b>", Display: "Alt+b", Group: groupHistory, Order: 90, Context: ContextDraft, Source: SourceNvim},

		// --- Panes and layout --------------------------------------------
		{Key: "<M-j>", Display: "Alt+j", Group: groupPanes, Order: 10, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-k>", Display: "Alt+k", Group: groupPanes, Order: 20, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-Up>", Display: "Alt+↑", Group: groupPanes, Order: 30, Context: ContextGlobal, Source: SourceGlobal},
		{Key: "<M-Down>", Display: "Alt+↓", Group: groupPanes, Order: 40, Context: ContextGlobal, Source: SourceGlobal},
		{Key: "<M-c>", Display: "Alt+c", Group: groupPanes, Order: 50, Context: ContextGlobal, Source: SourceGlobal},

		// --- Terminal tabs (pane-local; wording from roleBindings) --------
		{Key: "Alt+t", Group: groupTerminal, Order: 10, Context: ContextTerminal, Source: SourceRole},
		{Key: "Alt+w", Group: groupTerminal, Order: 20, Context: ContextTerminal, Source: SourceRole},
		{Key: "Alt+r", Group: groupTerminal, Order: 30, Context: ContextTerminal, Source: SourceRole},
		{Key: "Alt+Shift+d", Group: groupTerminal, Order: 40, Context: ContextTerminal, Source: SourceRole},
		{Key: "Alt+k", Group: groupTerminal, Order: 50, Context: ContextTerminal, Source: SourceRole},
		{Key: "Alt+Shift+⏎", Group: groupTerminal, Order: 60, Context: ContextTerminal, Source: SourceRole},

		// --- Session ------------------------------------------------------
		{Key: "Alt h", Display: "Alt+h", Group: groupSession, Order: 10, Context: ContextGlobal, Source: SourceZellij,
			Help: "show this keybinding help"},
		{Key: "Alt l", Display: "Alt+l", Group: groupSession, Order: 20, Context: ContextGlobal, Source: SourceZellij,
			Help: "open the changelog"},
		{Key: "<M-C>", Display: "Alt+Shift+c", Group: groupSession, Order: 30, Context: ContextDraft, Source: SourceNvim},
		{Key: "<M-d>", Display: "Alt+d", Group: groupSession, Order: 40, Context: ContextGlobal, Source: SourceGlobal},
		{Key: "<M-N>", Display: "Alt+Shift+n", Group: groupSession, Order: 50, Context: ContextGlobal, Source: SourceGlobal},
		{Key: "<M-n>", Display: "Alt+n", Group: groupSession, Order: 60, Context: ContextGlobal, Source: SourceGlobal},
		{Key: "<C-M-n>", Display: "Ctrl+Alt+n", Group: groupSession, Order: 70, Context: ContextGlobal, Source: SourceGlobal},
		{Key: "<M-x>", Display: "Alt+x", Group: groupSession, Order: 80, Context: ContextGlobal, Source: SourceGlobal},
	},

	// Editor mechanics and duplicate spellings. Each is a deliberate decision, not
	// an oversight — that is the whole point of requiring classification.
	internal: []string{
		"ZZ", "ZQ", // vim quit keys, remapped to a "quit is Alt+x" warning
		"<Tab>", "<S-Tab>", "<CR>", "<LeftMouse>", // completion-popup mechanics
		"<BS>",    // autopair smart-delete
		"z=",      // spell popup
		"<C-M-c>", // duplicate spelling of <M-C> (compact), already listed
		// The DRAFT rows for these are deliberate no-ops whose desc reads
		// "right-terminal tab helper disabled in draft". Their real behaviour is
		// documented in the Terminal tabs group, worded from roleBindings.
		"<M-t>", "<M-w>", "<M-r>",
	},
}

// Includes reports whether a source key is surfaced in the help.
func (c catalog) Includes(key string) bool {
	for _, e := range c.include {
		if e.Key == key {
			return true
		}
	}
	return false
}

// IsInternal reports whether a source key is deliberately withheld.
func (c catalog) IsInternal(key string) bool {
	for _, k := range c.internal {
		if k == key {
			return true
		}
	}
	return false
}

// Keys returns every documented source key, for the stale-entry drift test.
func (c catalog) Keys() []string {
	out := make([]string, 0, len(c.include))
	for _, e := range c.include {
		out = append(out, e.Key)
	}
	return out
}

// roleChordKey maps a pane-local chord to the catalog spelling used for it.
//
// Deliberately NOT workbenchshortcut.ChordName: that is a routing name round-tripped
// by ChordFromName (wrapcmd), i.e. a wire format. Coupling help display to it would
// mean a cosmetic rewording silently breaks chord routing (#132 non-goal).
func roleChordKey(c workbenchshortcut.Chord) string {
	switch c {
	case workbenchshortcut.ChordAltT:
		return "Alt+t"
	case workbenchshortcut.ChordAltW:
		return "Alt+w"
	case workbenchshortcut.ChordAltR:
		return "Alt+r"
	case workbenchshortcut.ChordAltShiftD:
		return "Alt+Shift+d"
	case workbenchshortcut.ChordAltK:
		return "Alt+k"
	case workbenchshortcut.ChordAltShiftEnter:
		return "Alt+Shift+⏎"
	}
	return ""
}
