package couchtty

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// Cast link codes in Couch (pair#495). Each cast link opens only with its
// code, which the operator says on the call. Couch shows the codes to the
// operator alone: hovering a cast cell shows its codes, Alt+click on it opens a
// Codes menu that rotates one link's code or all of them, and a code that
// changes flashes three times. All of it is drawn in the presenter's
// operator-only layer, which no broadcast frame carries; the status row and
// the notice line are broadcast, so a notice says that a code changed and
// never what it is.
//
// The layer runs on the Presenter goroutine and must take only leaf locks, so
// it draws from castCodes alone: a snapshot of the codes and the anchor's
// columns, taken on the Run loop when a hover starts, a menu opens, or a code
// rotates.

// castKind is which cast a code belongs to.
type castKind uint8

const (
	castNone   castKind = iota
	castScreen          // the screen broadcast: LIVE
	castWeb             // the web cast: Chrome
)

func (k castKind) name() string {
	if k == castWeb {
		return "Chrome"
	}
	return "LIVE"
}

const (
	codeFlashes     = 3
	codeFlashPhase  = 400 * time.Millisecond
	codeNoticeQuiet = 30 * time.Second // per link, between guess-rotation notices
)

// castCodes is what the operator-only layer draws. Its mutex is a leaf.
type castCodes struct {
	mu sync.Mutex
	// hover is the cast whose cell the pointer rests on, its codes and the
	// column its tooltip starts at.
	hover      castKind
	hoverCodes []broadcast.LinkCode
	hoverAt    int
	// The Codes menu: which cast, its layout, its items (by menu id) and the
	// hovered one.
	menu      castKind
	menuBox   NotesMenu
	menuItems map[int]string // menu id → link label; "" rotates every link
	menuHover int
	// flash is a code that just changed.
	flash *codeFlash
	// gens is the newest rotation seen per link (castlock.Rotation.Gen).
	gens map[string]uint64
	// noticed and quiet coalesce guess-rotation notices per link.
	noticed map[string]time.Time
	quiet   map[string]int
}

type codeFlash struct {
	cast  castKind
	link  string
	code  string
	at    int
	start time.Time
}

// linkKey names one link of one cast: its kind, the start attempt that made
// it, and its label. A rotation from an ended cast's attempt is never
// confused with the next cast's (BR-16).
func linkKey(k castKind, attempt uint64, link string) string {
	return strconv.Itoa(int(k)) + "/" + strconv.FormatUint(attempt, 10) + "/" + link
}

// castCodesOf is a cast's current codes; nil when it isn't running. On the Run
// loop (it reads c.mu).
func (c *Console) castCodesOf(k castKind) []broadcast.LinkCode {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch k {
	case castScreen:
		if c.bcast.phase == broadcastLive && c.bcast.session != nil {
			return c.bcast.session.Codes()
		}
	case castWeb:
		if c.wcast.phase == webcastRunning && c.wcast.cast != nil {
			return c.wcast.cast.Codes()
		}
	}
	return nil
}

// castAnchorAt is the cast whose status-row cell covers zero-based column x,
// and where that cell starts. LIVE's cell and its 👆 belong to the screen
// broadcast; the Chrome cell to the web cast.
func (c *Console) castAnchorAt(x int) (castKind, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case c.statusPointer.Contains(x):
		return castScreen, c.statusControl.Start
	case c.statusControl.Contains(x):
		return castScreen, c.statusControl.Start
	case c.statusWebcast.Contains(x):
		return castWeb, c.statusWebcast.Start
	}
	return castNone, 0
}

// routeCastCodes takes the mouse for the codes: the open Codes menu owns
// what happens over it, Alt+click on a cast cell opens the menu, and bare
// motion over a cast cell shows its tooltip (motion is never consumed). It
// reports whether it took the event.
func (c *Console) routeCastCodes(raw []byte) bool {
	hit, _, _, ok := mouseinput.ParsePrefix(raw)
	if !ok || hit.Release {
		return false
	}
	x, y := hit.X-1, hit.Y-1
	c.mu.Lock()
	rows := int(c.size.Rows)
	c.mu.Unlock()
	motion := hit.Button&32 != 0
	button := mouseinput.BaseButton(hit.Button)

	c.codes.mu.Lock()
	menu, box := c.codes.menu, c.codes.menuBox
	c.codes.mu.Unlock()
	if menu != castNone {
		if box.Contains(x, y) {
			if motion {
				c.setCodesMenuHover(box.ItemAt(x, y))
				return true
			}
			if button == 0 {
				if box.CloseAt(x, y) {
					c.closeCodesMenu()
				} else if id := box.ItemAt(x, y); id != 0 {
					c.chooseCodesItem(id)
				}
			}
			return true
		}
		if !motion {
			c.closeCodesMenu() // a click elsewhere closes it, and goes on
		}
	}

	onRow := y == rows-1
	if motion {
		kind, at := castNone, 0
		if onRow {
			kind, at = c.castAnchorAt(x)
		}
		c.setCodesHover(kind, at)
		return false
	}
	if onRow && button == 0 && hit.Button&mouseinput.ModAlt != 0 {
		if kind, _ := c.castAnchorAt(x); kind != castNone {
			c.openCodesMenu(kind)
			return true
		}
	}
	return false
}

// setCodesHover shows kind's codes above its cell, or hides them.
func (c *Console) setCodesHover(kind castKind, at int) {
	var codes []broadcast.LinkCode
	if kind != castNone {
		if codes = c.castCodesOf(kind); len(codes) == 0 {
			kind = castNone
		}
	}
	c.codes.mu.Lock()
	changed := c.codes.hover != kind || (kind != castNone && c.codes.hoverAt != at)
	c.codes.hover, c.codes.hoverCodes, c.codes.hoverAt = kind, codes, at
	c.codes.mu.Unlock()
	if changed {
		c.repaint()
	}
}

func (c *Console) openCodesMenu(kind castKind) {
	codes := c.castCodesOf(kind)
	if len(codes) == 0 {
		return
	}
	c.mu.Lock()
	cols, rows := int(c.size.Cols), int(c.size.Rows)
	c.mu.Unlock()
	items := map[int]string{}
	var rowsText []broadcast.Note
	for i, code := range codes {
		items[i+1] = code.Link
		rowsText = append(rowsText, broadcast.Note{ID: i + 1, Text: fmt.Sprintf("%s %s · rotate", code.Link, code.Code)})
	}
	if len(codes) > 1 {
		all := len(codes) + 1
		items[all] = ""
		rowsText = append(rowsText, broadcast.Note{ID: all, Text: "rotate all"})
	}
	box := RenderTitledMenu(kind.name()+" codes", rowsText, cols, rows)
	c.codes.mu.Lock()
	c.codes.menu, c.codes.menuBox, c.codes.menuItems, c.codes.menuHover = kind, box, items, 0
	c.codes.mu.Unlock()
	c.repaint()
}

func (c *Console) closeCodesMenu() {
	c.codes.mu.Lock()
	open := c.codes.menu != castNone
	c.codes.menu, c.codes.menuBox, c.codes.menuItems, c.codes.menuHover = castNone, NotesMenu{}, nil, 0
	c.codes.mu.Unlock()
	if open {
		c.repaint()
	}
}

func (c *Console) setCodesMenuHover(id int) {
	c.codes.mu.Lock()
	changed := c.codes.menuHover != id
	c.codes.menuHover = id
	c.codes.mu.Unlock()
	if changed {
		c.repaint()
	}
}

// chooseCodesItem rotates the chosen link's code, or every link's.
func (c *Console) chooseCodesItem(id int) {
	c.codes.mu.Lock()
	kind, link, ok := c.codes.menu, "", false
	if c.codes.menuItems != nil {
		link, ok = c.codes.menuItems[id]
	}
	c.codes.mu.Unlock()
	c.closeCodesMenu()
	if !ok {
		return
	}
	var links []string
	if link == "" {
		for _, code := range c.castCodesOf(kind) {
			links = append(links, code.Link)
		}
	} else {
		links = []string{link}
	}
	if err := c.rotateCastCodes(kind, links); err != nil {
		c.setNotice(kind.name() + ": couldn't change the code: " + err.Error())
		return
	}
	what := links[0] + " code"
	if len(links) > 1 {
		what = "codes"
	}
	c.setNotice(fmt.Sprintf("New %s %s; its viewers were disconnected. Hover %s to see it.", kind.name(), what, kind.name()))
}

func (c *Console) rotateCastCodes(kind castKind, links []string) error {
	c.mu.Lock()
	var rotate func(string) error
	switch {
	case kind == castScreen && c.bcast.session != nil:
		rotate = c.bcast.session.RotateCode
	case kind == castWeb && c.wcast.cast != nil:
		rotate = c.wcast.cast.RotateCode
	}
	c.mu.Unlock()
	if rotate == nil {
		return fmt.Errorf("the cast isn't running")
	}
	for _, link := range links {
		if err := rotate(link); err != nil {
			return err
		}
	}
	return nil
}

// onCodeRotated is a cast reporting a new code, on whichever goroutine
// rotated it; attempt is the start that made the cast. A rotation from an
// earlier attempt (a cast that has ended) is dropped. The newest Gen per link
// wins; a guessing rotation is told in a notice, coalesced per link, without
// the code.
func (c *Console) onCodeRotated(kind castKind, attempt uint64, link string, r castlock.Rotation) {
	key := linkKey(kind, attempt, link)
	now := c.now()
	// No caller holds the console's locks while a code rotates (the gate's
	// unlock, or the menu's RotateCode), so the console can be read here.
	c.mu.Lock()
	cell, current := c.statusControl.Start, c.bcast.attempt
	if kind == castWeb {
		cell, current = c.statusWebcast.Start, c.wcast.attempt
	}
	c.mu.Unlock()
	if attempt != current {
		return // a late report from a cast that has ended
	}
	c.codes.mu.Lock()
	if c.codes.gens == nil {
		c.codes.gens, c.codes.noticed, c.codes.quiet = map[string]uint64{}, map[string]time.Time{}, map[string]int{}
	}
	if r.Gen <= c.codes.gens[key] {
		c.codes.mu.Unlock()
		return // an older rotation delivered late
	}
	c.codes.gens[key] = r.Gen
	at := cell
	if c.codes.hover == kind {
		for i := range c.codes.hoverCodes {
			if c.codes.hoverCodes[i].Link == link {
				c.codes.hoverCodes[i].Code = r.Code
			}
		}
	}
	c.codes.flash = &codeFlash{cast: kind, link: link, code: r.Code, at: at, start: now}
	notice := ""
	if r.Reason == castlock.ReasonGuessing {
		if last, ok := c.codes.noticed[key]; ok && now.Sub(last) < codeNoticeQuiet {
			c.codes.quiet[key]++
		} else {
			times := ""
			if n := c.codes.quiet[key]; n > 0 {
				times = fmt.Sprintf(" (%d more times since)", n)
			}
			c.codes.noticed[key], c.codes.quiet[key] = now, 0
			notice = fmt.Sprintf("Too many wrong codes on the %s %s link: its code changed%s. Hover %s to see it.", kind.name(), link, times, kind.name())
		}
	}
	c.codes.mu.Unlock()
	c.GoTracked(func() {
		if notice != "" {
			c.setNotice(notice)
		}
		c.flashCodes()
	})
}

// flashCodes repaints at each phase of a code's flash; the layer decides
// from the clock whether the code shows.
func (c *Console) flashCodes() {
	for i := 0; i <= 2*codeFlashes; i++ {
		c.repaint()
		select {
		case <-time.After(codeFlashPhase):
		case <-c.lifetime.Done():
			return
		}
	}
}

// forgetCastCodes drops what the layer shows of a cast that ended.
func (c *Console) forgetCastCodes(kind castKind) {
	c.codes.mu.Lock()
	if c.codes.hover == kind {
		c.codes.hover, c.codes.hoverCodes = castNone, nil
	}
	if c.codes.menu == kind {
		c.codes.menu, c.codes.menuBox, c.codes.menuItems, c.codes.menuHover = castNone, NotesMenu{}, nil, 0
	}
	if c.codes.flash != nil && c.codes.flash.cast == kind {
		c.codes.flash = nil
	}
	for key := range c.codes.gens {
		if strings.HasPrefix(key, strconv.Itoa(int(kind))+"/") {
			delete(c.codes.gens, key)
			delete(c.codes.noticed, key)
			delete(c.codes.quiet, key)
		}
	}
	c.codes.mu.Unlock()
}

// codesLayer is the presenter's operator-only layer: the codes tooltip, the
// Codes menu and a code's flash. Presenter goroutine; leaf locks only.
func (c *Console) codesLayer(f terminal.Frame, _ terminal.FrameClass) terminal.Frame {
	c.pnotes.mu.Lock()
	style := c.drawStyleLocked()
	c.pnotes.mu.Unlock()
	now := c.now()
	c.codes.mu.Lock()
	defer c.codes.mu.Unlock()
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	if fl := c.codes.flash; fl != nil {
		phase := int(now.Sub(fl.start) / codeFlashPhase)
		switch {
		case phase >= 2*codeFlashes:
			c.codes.flash = nil
		case phase%2 == 0:
			f = codeTip(fmt.Sprintf("%s %s %s", fl.cast.name(), fl.link, fl.code), fl.at, cols, rows).Draw(f, 0, style)
		}
	}
	if c.codes.hover != castNone && len(c.codes.hoverCodes) > 0 && (c.codes.flash == nil || c.codes.flash.cast != c.codes.hover) {
		parts := make([]string, 0, len(c.codes.hoverCodes))
		for _, code := range c.codes.hoverCodes {
			parts = append(parts, code.Link+" "+code.Code)
		}
		f = codeTip(strings.Join(parts, " · "), c.codes.hoverAt, cols, rows).Draw(f, 0, style)
	}
	if c.codes.menu != castNone {
		f = c.codes.menuBox.Draw(f, c.codes.menuHover, style)
	}
	return f
}

// codeTip lays a one-line box over the row above the status row, starting
// at column at and kept on screen.
func codeTip(text string, at, cols, rows int) NotesMenu {
	line := " " + text + " "
	w := len([]rune(line))
	if w > cols || rows < 2 {
		return NotesMenu{}
	}
	x := min(max(at, 0), cols-w)
	return NotesMenu{X: x, Y: rows - 2, W: w, H: 1, lines: []notesMenuLine{{text: line}}}
}

// installCodesLayer puts the codes on the operator's screen only.
func (c *Console) installCodesLayer() error {
	return c.presenter.SetOperatorLayer(context.Background(), c.codesLayer)
}
