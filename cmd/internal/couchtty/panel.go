package couchtty

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// PanelRow is one line of couch's own screen.
type PanelRow struct {
	Address couchcore.ThreadAddress
	// Target is the console-local child id to switch to. It is deliberately
	// separate from Address: durable thread identity and terminal routing
	// address different layers.
	Target string
	// Tree is the working path shown to the operator and used by start. It is
	// not an identity: Brain-style repositories may host several threads in it.
	Tree  couchcore.Worktree
	Label string
	Desc  string
	Live  bool
	// Bell is the point of the panel being a place to LOOK: an actor that
	// wants attention says so here, not only on the status row where it
	// competes for one line.
	Bell bool
}

// PanelTarget is console-local routing state joined onto Couch's durable thread
// summaries. Keeping it separate prevents a hosted-child inventory from
// becoming a second source for labels, descriptions, or parked rows.
type PanelTarget struct {
	Address couchcore.ThreadAddress
	Tree    couchcore.Worktree
	Target  string
	Bell    bool
}

// PanelControl is one operator-entered panel surface. The renderer and README
// checks consume this inventory so a new key cannot ship undocumented.
type PanelControl struct {
	Keys   string
	Action string
}

var panelControls = []PanelControl{
	{Keys: "typeahead", Action: "filter"},
	{Keys: "↑↓", Action: "select"},
	{Keys: "Enter", Action: "switch/start"},
	{Keys: "Ctrl-Space", Action: "start"},
	{Keys: "Escape", Action: "clear/back"},
}

// PanelControls returns the shared, immutable-by-copy key inventory.
func PanelControls() []PanelControl {
	return append([]PanelControl(nil), panelControls...)
}

// PanelModel is the panel as DATA: what to show, filtered, in a stable order.
//
// Pure. The console renders it and #148's advisor can read the same rows, which
// is the "no state the operator can see that an LLM cannot" property stated in
// the project.
type PanelModel struct {
	all []PanelRow

	// shown is the last filtered result, and selection always indexes it.
	shown []PanelRow

	// cursor is the highlighted row, 0-based into shown. A list with no
	// highlight is a list you cannot navigate -- the operator has no way to
	// tell what Enter will do.
	cursor int
}

// Cursor is the highlighted row index.
func (m *PanelModel) Cursor() int { return m.cursor }

// Move steps the highlight, clamping rather than wrapping. Wrapping in a short
// list makes "press down twice" unpredictable.
func (m *PanelModel) Move(delta int) {
	if len(m.shown) == 0 {
		m.cursor = -1
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.shown) {
		m.cursor = len(m.shown) - 1
	}
}

// Selected is the highlighted row.
func (m *PanelModel) Selected() (PanelRow, bool) {
	if m.cursor < 0 || m.cursor >= len(m.shown) {
		return PanelRow{}, false
	}
	return m.shown[m.cursor], true
}

// NewPanelModel builds the rows from couch's own summaries, so a thread that is
// PARKED -- named, no live actor -- is listed exactly as `couch list` lists it.
// That thread is the one this project exists to stop losing, so it is not
// filtered out for being idle.
func NewPanelModel(threads []couchcore.ThreadSummary) *PanelModel {
	m := &PanelModel{all: make([]PanelRow, 0, len(threads))}
	for _, thread := range threads {
		m.all = append(m.all, PanelRow{
			Address: thread.Address,
			Tree:    couchcore.Worktree(thread.WorkingPath),
			Label:   thread.Label(),
			Desc:    thread.DisplaySummary(),
			Live:    thread.Live(),
		})
	}
	m.shown = m.all
	if len(m.shown) == 0 {
		m.cursor = -1
	}
	return m
}

// Rows is everything the panel knows about, unfiltered.
func (m *PanelModel) Rows() []PanelRow { return m.all }

// BindTargets joins ephemeral console routing onto summary-derived rows.
// Multiple hosted children for one thread choose the first target deterministically
// and OR their bell state; the panel remains one row per composite thread address.
func (m *PanelModel) BindTargets(targets []PanelTarget) {
	selected := m.selectedAddress()
	byAddress := map[couchcore.ThreadAddress]PanelTarget{}
	for _, target := range targets {
		key := target.Address
		joined := byAddress[key]
		if joined.Target == "" {
			joined.Tree = target.Tree
			joined.Target = target.Target
		}
		joined.Bell = joined.Bell || target.Bell
		byAddress[key] = joined
	}
	for i := range m.all {
		if target, ok := byAddress[m.all[i].Address]; ok {
			m.all[i].Target = target.Target
			m.all[i].Bell = target.Bell
		}
	}
	m.setShown(m.all, selected)
}

// Shown is the current filtered view -- what the operator is looking at.
func (m *PanelModel) Shown() []PanelRow { return m.shown }

// Filter narrows the rows by INJECTING the match rule rather than restating it.
//
// resolve is `couch.ResolveThreadReference` in production: one rule serving
// the CLI, panel, and #148's advisor. Restating it here is the drift Decision 12 exists
// to prevent -- and the earlier plan text got the rule's own field list wrong,
// which is what a second copy does.
//
// An empty query is not a resolution: it means "show everything", and asking
// the resolver would make the panel's DEFAULT view depend on a match rule.
func (m *PanelModel) Filter(query string, resolve func(string) []couchcore.ThreadAddress) []PanelRow {
	selected := m.selectedAddress()
	if query == "" || resolve == nil {
		m.setShown(m.all, selected)
		return m.shown
	}
	want := map[couchcore.ThreadAddress]bool{}
	for _, address := range resolve(query) {
		want[address] = true
	}
	// Filtered in the ORIGINAL order rather than the resolver's: numbered
	// selection is only safe if rows do not move under the operator's fingers,
	// and a resolver is free to return whatever order it likes.
	out := make([]PanelRow, 0, len(want))
	for _, r := range m.all {
		if want[r.Address] {
			out = append(out, r)
		}
	}
	m.setShown(out, selected)
	return out
}

func (m *PanelModel) selectedAddress() couchcore.ThreadAddress {
	if row, ok := m.Selected(); ok {
		return row.Address
	}
	return couchcore.ThreadAddress{}
}

// SelectAddress selects a visible row by durable composite thread identity.
func (m *PanelModel) SelectAddress(address couchcore.ThreadAddress) bool {
	for i, row := range m.shown {
		if row.Address == address {
			m.cursor = i
			return true
		}
	}
	return false
}

// SelectTree is the compatibility boundary retained for Pair #146 callers.
// A working path is no longer a durable identity, so selection succeeds only
// when the visible list contains exactly one thread at that path.
func (m *PanelModel) SelectTree(tree couchcore.Worktree) bool {
	match := -1
	for i, row := range m.shown {
		if row.Tree != tree {
			continue
		}
		if match >= 0 {
			return false
		}
		match = i
	}
	if match < 0 {
		return false
	}
	m.cursor = match
	return true
}

func (m *PanelModel) setShown(rows []PanelRow, selected couchcore.ThreadAddress) {
	m.shown = rows
	m.cursor = -1
	if selected != (couchcore.ThreadAddress{}) && m.SelectAddress(selected) {
		return
	}
	if len(rows) > 0 {
		m.cursor = 0
	}
}

// clampCursor keeps the highlight on a row that exists: filtering can shrink
// the list under it, and a cursor past the end selects nothing.
func (m *PanelModel) clampCursor() {
	if len(m.shown) == 0 {
		m.cursor = -1
		return
	}
	if m.cursor >= len(m.shown) {
		m.cursor = len(m.shown) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// RenderPanel draws the panel for the operator.
//
// Deliberately plain -- a list to read, not chrome. But it MUST show three
// things or it is not usable: which row is selected, which actors want
// attention, and what the keys are. The first cut showed a bare list and the
// operator had no way to tell that arrows, Enter or Escape did anything.
func RenderPanel(rows []PanelRow, cursor int) string {
	var b strings.Builder
	b.WriteString("couch — actors\r\n\r\n")
	if len(rows) == 0 {
		b.WriteString("  (no match)\r\n")
		return b.String()
	}
	for i, r := range rows {
		marker := "  "
		if i == cursor {
			marker = "▸ "
		}
		state := " "
		if !r.Live {
			// A parked thread stays listed: it is exactly the one an operator
			// loses track of.
			state = "·"
		}
		bell := " "
		if r.Bell {
			bell = "*"
		}
		fmt.Fprintf(&b, "%s%s%s %s", marker, state, bell, sanitize(r.Label))
		if r.Desc != "" {
			fmt.Fprintf(&b, "  — %s", sanitize(r.Desc))
		}
		b.WriteString("\r\n")
	}
	return b.String()
}

// RenderPanelWithQuery draws the panel plus the typeahead buffer and the keys,
// so the operator can see why the list narrowed and what to press.
func RenderPanelWithQuery(query string, rows []PanelRow, cursor int) string {
	var b strings.Builder
	b.WriteString(RenderPanel(rows, cursor))
	b.WriteString("\r\n")
	if query != "" {
		fmt.Fprintf(&b, "  filter: %s\r\n", sanitize(query))
	}
	controls := make([]string, 0, len(panelControls))
	for _, control := range panelControls {
		controls = append(controls, control.Keys+" "+control.Action)
	}
	b.WriteString("  " + strings.Join(controls, " · ") + "\r\n")
	return b.String()
}

// PanelActions is what the operator can do from the panel.
//
// Names only, and every one must be a name in couchcore.Operations(): the panel
// DISPATCHES through that table rather than implementing anything, so there is
// no operator action the advisor cannot also perform (#148's design test).
//
// It lists what is WIRED, not what is planned. The first version returned four
// names with nothing behind them and the audit passed anyway -- a subset check
// is satisfied by a list that does nothing, which is why the audit now also
// requires each name to be reachable from a keystroke.
func PanelActions() []string {
	return []string{"start"}
}

// PanelActionKeys maps each action to the key that invokes it, so the audit can
// check the action is reachable rather than merely declared.
func PanelActionKeys() map[string][]string {
	return map[string][]string{
		"start": {"Ctrl-Space", "Enter parked"},
	}
}
