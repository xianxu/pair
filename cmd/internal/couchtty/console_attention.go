package couchtty

import (
	"image/color"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// The attention list (pair#673): what waits on the operator, owned here as a
// couchcore.AttentionList. Couch's sources report facts (ObserveAttention);
// the status row shows 🔔N, Ctrl+Alt+a or a click on it opens the list over
// the top right, and picking an item jumps to its slot through switchTo, the
// funnel Ctrl+Return and chip clicks use. The panel is drawn on the
// operator-only layer, so a broadcast never shows it.

// AttentionStore persists the list; couchcore.AttentionStore in production.
type AttentionStore interface {
	Load() ([]couchcore.AttentionItem, error)
	Save([]couchcore.AttentionItem) error
}

// attentionState is a leaf: its lock is taken under c.mu (the status model)
// and on the presenter goroutine (the layer), and never takes another.
type attentionState struct {
	mu   sync.Mutex
	list couchcore.AttentionList
	// save serializes saves so a newer snapshot never lands before an older.
	save  sync.Mutex
	store AttentionStore
	open  bool
	sel   int            // index into the open panel's items
	box   NotesMenu      // the open panel as last laid out
	ids   []string       // the panel's item IDs; menu item n is ids[n-1]
	shown []attentionRow // what the panel lists, newest first
}

type attentionRow struct {
	item couchcore.AttentionItem
	text string
	bg   color.Color
}

var (
	attentionNotifyBg   = ansi.IndexedColor(214) // the bell's amber
	attentionPromptBg   = ansi.IndexedColor(160) // a blocked chip's red
	attentionQuestionBg = ansi.IndexedColor(75)  // the actionable-link blue
)

const attentionTitle = "Waiting on you"

// SetAttentionStore loads the saved list, before Run. A list that can't be
// read starts empty, said in a notice; Couch still starts.
func (c *Console) SetAttentionStore(store AttentionStore) {
	items, err := store.Load()
	c.attn.mu.Lock()
	c.attn.store = store
	c.attn.list = couchcore.RestoreAttention(items, c.now())
	c.attn.mu.Unlock()
	if err != nil {
		c.setNotice("Attention list not restored: " + err.Error())
	}
	c.repaint()
}

// ObserveAttention folds one source fact into the list. It never blocks on
// the console: it saves and repaints from the caller's goroutine.
func (c *Console) ObserveAttention(f couchcore.AttentionFact) {
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	c.attn.list = c.attn.list.Apply(f, c.now())
	c.layoutAttentionLocked(cols, rows)
	c.attn.mu.Unlock()
	c.saveAttention()
	c.repaint()
}

func (c *Console) saveAttention() {
	c.attn.save.Lock()
	defer c.attn.save.Unlock()
	c.attn.mu.Lock()
	store, saved := c.attn.store, c.attn.list.Saved(c.now())
	c.attn.mu.Unlock()
	if store == nil {
		return
	}
	if err := store.Save(saved); err != nil {
		c.setNotice("Attention list not saved: " + err.Error())
	}
}

// attentionRows is the panel's rows, newest first: the slot, what it waits
// on, and when, with a chip of its kind's color. A visited item says so.
func attentionRows(items []couchcore.AttentionItem) []attentionRow {
	rows := make([]attentionRow, 0, len(items))
	for _, item := range items {
		text := item.Title
		bg := attentionNotifyBg
		switch item.Kind {
		case couchcore.AttentionNotify:
			if item.Body != "" {
				text = strings.TrimSpace(text + ": " + item.Body)
				text = strings.TrimPrefix(text, ": ")
			}
		case couchcore.AttentionPrompt:
			bg = attentionPromptBg
		case couchcore.AttentionQuestion:
			bg = attentionQuestionBg
		}
		state := ""
		if item.Visited {
			state = " (visited)"
		}
		rows = append(rows, attentionRow{item: item, bg: bg,
			text: item.Since.Local().Format("15:04") + "  " + item.Label + state + "  " + text})
	}
	return rows
}

// attentionScreen is the whole screen, status row included, read before
// c.attn.mu is taken (c.mu is never taken under it).
func (c *Console) attentionScreen() (cols, rows int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int(c.size.Cols), int(c.size.Rows) + 1
}

// layoutAttentionLocked lays the open panel out again over a cols×rows
// screen; with nothing left in the list the panel closes. Caller holds
// c.attn.mu.
func (c *Console) layoutAttentionLocked(cols, screenRows int) {
	if !c.attn.open {
		return
	}
	rows := attentionRows(c.attn.list.Items())
	c.attn.shown, c.attn.ids = rows, make([]string, len(rows))
	items := make([]menuItem, len(rows))
	for i, r := range rows {
		c.attn.ids[i] = r.item.ID
		items[i] = menuItem{ID: i + 1, Text: r.text, Chip: true, Bg: r.bg}
	}
	c.attn.box = renderMenuAt(attentionTitle, items, cols, screenRows, true)
	if !c.attn.box.Open() {
		c.attn.open = false
		return
	}
	c.attn.sel = min(max(c.attn.sel, 0), len(c.attn.box.items)-1)
}

// toggleAttention is Ctrl+Alt+a and a click on 🔔N.
func (c *Console) toggleAttention() {
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	c.attn.open = !c.attn.open && c.attn.list.Len() > 0
	c.attn.sel = 0
	c.layoutAttentionLocked(cols, rows)
	open := c.attn.open
	c.attn.mu.Unlock()
	if !open && c.attentionCount() == 0 {
		c.setNotice("Nothing is waiting on you.")
	}
	c.repaint()
}

func (c *Console) attentionCount() int {
	c.attn.mu.Lock()
	defer c.attn.mu.Unlock()
	return c.attn.list.Len()
}

func (c *Console) closeAttention() {
	c.attn.mu.Lock()
	was := c.attn.open
	c.attn.open = false
	c.attn.mu.Unlock()
	if was {
		c.repaint()
	}
}

// selectAttention picks the panel's item n (zero-based): the list applies its
// select rule, the panel closes, and the console jumps to the item's slot.
func (c *Console) selectAttention(n int) {
	c.attn.mu.Lock()
	if !c.attn.open || n < 0 || n >= len(c.attn.ids) {
		c.attn.mu.Unlock()
		return
	}
	var item couchcore.AttentionItem
	var ok bool
	c.attn.list, item, ok = c.attn.list.Select(c.attn.ids[n])
	c.attn.open = false
	c.attn.mu.Unlock()
	if !ok {
		return
	}
	c.saveAttention()
	c.jumpToAttention(item)
}

func (c *Console) jumpToAttention(item couchcore.AttentionItem) {
	c.mu.Lock()
	target := c.switchTargetForAddressLocked(item.Thread)
	c.mu.Unlock()
	if target == "" {
		c.setNotice(item.Label + " is not attached; open it from the switcher (Ctrl+Space).")
		c.repaint()
		return
	}
	c.switchTo(target, false, arrivalNotification)
}

// dismissAttention removes the panel's item n.
func (c *Console) dismissAttention(n int) {
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	if !c.attn.open || n < 0 || n >= len(c.attn.ids) {
		c.attn.mu.Unlock()
		return
	}
	c.attn.list = c.attn.list.Dismiss(c.attn.ids[n])
	c.layoutAttentionLocked(cols, rows)
	c.attn.mu.Unlock()
	c.saveAttention()
	c.repaint()
}

// routeAttentionKey takes the keyboard while the panel is open: Up/Down (or
// k/j) move, Enter jumps, d or Backspace dismisses, Escape closes. Any other
// key goes on to its usual place. It reports whether it took the key.
func (c *Console) routeAttentionKey(raw, canonical []byte) bool {
	c.attn.mu.Lock()
	open, sel := c.attn.open, c.attn.sel
	c.attn.mu.Unlock()
	if !open {
		return false
	}
	if len(canonical) == 1 && canonical[0] == 0x1b {
		c.closeAttention()
		return true
	}
	keys, held := DecodePanelKeys(raw)
	if len(keys) != 1 || len(held) != 0 {
		return false
	}
	move := 0
	switch k := keys[0]; {
	case k.Kind == KeyUp || k.Kind == KeyRune && k.Rune == 'k':
		move = -1
	case k.Kind == KeyDown || k.Kind == KeyRune && k.Rune == 'j':
		move = 1
	case k.Kind == KeyEnter:
		c.selectAttention(sel)
		return true
	case k.Kind == KeyBackspace || k.Kind == KeyRune && k.Rune == 'd':
		c.dismissAttention(sel)
		return true
	case k.Kind == KeyEscape:
		c.closeAttention()
		return true
	default:
		return false
	}
	c.attn.mu.Lock()
	if n := len(c.attn.ids); n > 0 {
		c.attn.sel = min(max(c.attn.sel+move, 0), n-1)
	}
	c.attn.mu.Unlock()
	c.repaint()
	return true
}

// routeAttentionPanel takes the mouse over the open panel: a click on an item
// jumps to it, on the close mark closes, and a click elsewhere closes the
// panel and goes on. Motion over it moves the selection.
func (c *Console) routeAttentionPanel(raw []byte) bool {
	hit, _, _, ok := mouseinput.ParsePrefix(raw)
	if !ok {
		return false
	}
	x, y := hit.X-1, hit.Y-1
	c.attn.mu.Lock()
	open, box := c.attn.open, c.attn.box
	c.attn.mu.Unlock()
	if !open {
		return false
	}
	if !box.Contains(x, y) {
		if !hit.Release && hit.Button&32 == 0 {
			c.closeAttention()
		}
		return false
	}
	if hit.Release {
		return true
	}
	id := box.ItemAt(x, y)
	switch {
	case hit.Button&32 != 0:
		if id > 0 {
			c.attn.mu.Lock()
			changed := c.attn.sel != id-1
			c.attn.sel = id - 1
			c.attn.mu.Unlock()
			if changed {
				c.repaint()
			}
		}
	case mouseinput.BaseButton(hit.Button) == 0 && box.CloseAt(x, y):
		c.closeAttention()
	case mouseinput.BaseButton(hit.Button) == 0 && id > 0:
		c.selectAttention(id - 1)
	}
	return true
}

// attentionLayer draws the open panel. Presenter goroutine; leaf lock only.
func (c *Console) attentionLayer(f terminal.Frame) terminal.Frame {
	c.pnotes.mu.Lock()
	style := c.drawStyleLocked()
	c.pnotes.mu.Unlock()
	c.attn.mu.Lock()
	defer c.attn.mu.Unlock()
	if !c.attn.open {
		return f
	}
	return c.attn.box.Draw(f, c.attn.sel+1, style)
}

// attentionStatusLocked fills the status model's badge. Caller holds c.mu.
func (c *Console) attentionStatusLocked(model *StatusModel) {
	c.attn.mu.Lock()
	model.Attention, model.AttentionPrompt = c.attn.list.Len(), c.attn.list.Has(couchcore.AttentionPrompt)
	c.attn.mu.Unlock()
}
