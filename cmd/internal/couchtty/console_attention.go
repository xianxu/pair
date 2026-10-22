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

// attentionState is a leaf: its lock is taken under c.mu (the status model)
// and on the presenter goroutine (the layer), and never takes another.
type attentionState struct {
	mu    sync.Mutex
	list  couchcore.AttentionList
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

// ObserveAttention folds one source fact into the list. It never blocks on
// the disk: the list is ephemeral and folds entirely in memory.
func (c *Console) ObserveAttention(f couchcore.AttentionFact) {
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	c.attn.list = c.attn.list.Apply(f, c.now())
	c.layoutAttentionLocked(cols, rows)
	c.attn.mu.Unlock()
	c.repaint()
}

// expireAttention shares the existing activity tick and never touches IO.
func (c *Console) expireAttention() {
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	before := c.attn.list.Len()
	c.attn.list = c.attn.list.Expire(c.now())
	changed := before != c.attn.list.Len()
	c.layoutAttentionLocked(cols, rows)
	c.attn.mu.Unlock()
	if changed {
		c.repaint()
	}
}

func (c *Console) clearAttention() {
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	c.attn.list = c.attn.list.Clear()
	c.layoutAttentionLocked(cols, rows)
	c.attn.mu.Unlock()
	c.repaint()
}

// attentionRows is the panel's rows, newest first: the slot, what it waits
// on, and when, with a chip of its kind's color.
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
		rows = append(rows, attentionRow{item: item, bg: bg,
			text: item.Since.Local().Format("15:04") + "  " + item.Label + "  " + text})
	}
	return rows
}

// attentionScreen is the whole screen, status row included, read before
// c.attn.mu is taken (c.mu is never taken under it).
func (c *Console) attentionScreen() (cols, rows int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int(c.size.Cols), int(c.size.Rows) - c.statusRows + 2
}

// layoutAttentionLocked lays the open panel out again over a cols×rows
// screen; with nothing left in the list the panel closes. Caller holds
// c.attn.mu.
func (c *Console) layoutAttentionLocked(cols, screenRows int) {
	if !c.attn.open {
		return
	}
	rows := attentionRows(c.attn.list.Items())
	c.attn.shown, c.attn.ids = rows, nil
	var items []menuItem
	if len(rows) > 0 {
		c.attn.ids = append(c.attn.ids, "")
		items = append(items, menuItem{ID: 1, Text: "Clear all (c)"})
	}
	for i, r := range rows {
		c.attn.ids = append(c.attn.ids, r.item.ID)
		items = append(items, menuItem{ID: i + 2, Text: r.text, Chip: true, Bg: r.bg})
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
	c.expireAttention()
	cols, rows := c.attentionScreen()
	c.attn.mu.Lock()
	c.attn.open = !c.attn.open && c.attn.list.Len() > 0
	c.attn.sel = 1
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
	if c.attn.ids[n] == "" {
		c.attn.mu.Unlock()
		c.clearAttention()
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
	c.repaint()
}

// routeAttentionKey takes the keyboard while the panel is open: Up/Down (or
// k/j) move, Enter jumps, d or Backspace dismisses, c clears all, Escape closes. Any other
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
	case k.Kind == KeyRune && k.Rune == 'c':
		c.clearAttention()
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
	if !c.chromeConfirmed() {
		return false
	}
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
		if hit.Release || hit.Button&32 != 0 {
			return false
		}
		c.closeAttention()
		// A click on 🔔N toggles: it closed the panel, so it must not go on
		// to the badge's handler and open it again.
		c.mu.Lock()
		onBadge := hit.Y == int(c.size.Rows)-c.statusRows+1 && c.statusAttention.Contains(x)
		c.mu.Unlock()
		return onBadge && mouseinput.BaseButton(hit.Button) == 0
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
	c.layoutAttentionLocked(f.Geometry.Cols, f.Geometry.Rows-f.ChromeRows+2)
	return c.attn.box.Draw(f, c.attn.sel+1, style)
}

// attentionStatusLocked fills the status model's badge. Caller holds c.mu.
func (c *Console) attentionStatusLocked(model *StatusModel) {
	c.attn.mu.Lock()
	model.Attention, model.AttentionPrompt = c.attn.list.Len(), c.attn.list.Has(couchcore.AttentionPrompt)
	c.attn.mu.Unlock()
}
