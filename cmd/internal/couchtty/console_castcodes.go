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
// operator alone: hovering a cast's cell (LIVE ⏸ or Chrome ⏸) shows all its
// codes, and Alt+click on it opens a Codes menu that copies its links (drawn
// blue) and rotates one link's code or all of them, and a code that
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
	codesPress     = 180 * time.Millisecond // a pressed word's feedback
	codeFlashes    = 3
	codeFlashPhase = 400 * time.Millisecond
)

// castCodes is what the operator-only layer draws. Its mutex is a leaf.
type castCodes struct {
	mu sync.Mutex
	// hover is the cast whose cell the pointer rests on, its codes and the
	// column its tooltip starts at.
	hover      castKind
	hoverCodes []broadcast.LinkCode
	hoverAt    int
	// The Codes menu: which cast, and its layout and selection. pressed is
	// the word just run, drawn pressed until pressedUntil (a click's
	// feedback; the menu stays open).
	menu         castKind
	menuBox      codesMenu
	pressed      codesMenuSel
	pressedUntil time.Time
	// flash is a code that just changed.
	flash *codeFlash
	// gens is the newest rotation seen per link (castlock.Rotation.Gen).
	gens map[string]uint64
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

// castCellAt is the cast whose cell -- LIVE ⏸ or Chrome ⏸, the main
// entrance -- covers column x, and where that cell starts. Its link controls
// (the 👆 and 👽 beside it) are not the cell: they copy their links.
func (c *Console) castCellAt(x int) (castKind, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case c.statusControl.Contains(x):
		return castScreen, c.statusControl.Start
	case c.statusWebcast.Contains(x):
		return castWeb, c.statusWebcast.Start
	}
	return castNone, 0
}

// codeAnchorLocked is where a link's code flashes: over its cast's cell,
// where the hover shows the codes. Callers hold c.mu.
func (c *Console) codeAnchorLocked(kind castKind) int {
	if kind == castWeb {
		return c.statusWebcast.Start
	}
	return c.statusControl.Start
}

// routeCastCodes takes the mouse for the codes: the open Codes menu owns
// what happens over it, Alt+click on a cast cell opens the menu, and bare
// motion over a cast cell shows its tooltip (motion is never consumed). It
// reports whether it took the event.
func (c *Console) routeCastCodes(raw []byte) bool {
	if !c.chromeConfirmed() {
		return false
	}
	hit, _, _, ok := mouseinput.ParsePrefix(raw)
	if !ok || hit.Release {
		return false
	}
	x, y := hit.X-1, hit.Y-1
	c.mu.Lock()
	rows := int(c.size.Rows) - c.statusRows + 1
	c.mu.Unlock()
	motion := hit.Button&32 != 0
	button := mouseinput.BaseButton(hit.Button)

	c.codes.mu.Lock()
	menu, box := c.codes.menu, c.codes.menuBox
	c.codes.mu.Unlock()
	if menu != castNone {
		if box.Contains(x, y) {
			sel, onWord := box.WordAt(x, y)
			if motion {
				c.selectCodesWord(sel)
				return true
			}
			switch {
			case button != 0:
			case box.CloseAt(x, y):
				c.closeCodesMenu()
			case onWord:
				c.selectCodesWord(sel)
				c.runCodesSelection()
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
			kind, at = c.castCellAt(x)
		}
		c.setCodesHover(kind, at)
		return false
	}
	if onRow && button == 0 && hit.Button&mouseinput.ModAlt != 0 {
		if kind, _ := c.castCellAt(x); kind != castNone {
			c.openCodesMenu(kind)
			return true
		}
	}
	return false
}

// setCodesHover shows all of kind's codes at column at above the status
// row, or hides them.
func (c *Console) setCodesHover(kind castKind, at int) {
	var codes []broadcast.LinkCode
	if kind != castNone {
		if codes = c.castCodesOf(kind); len(codes) == 0 {
			kind = castNone
		}
	}
	c.codes.mu.Lock()
	changed := c.codes.hover != kind || (kind != castNone && (c.codes.hoverAt != at || len(c.codes.hoverCodes) != len(codes)))
	c.codes.hover, c.codes.hoverCodes, c.codes.hoverAt = kind, codes, at
	c.codes.mu.Unlock()
	if changed {
		c.repaint()
	}
}

// openCodesMenu opens kind's Codes menu: a row per link -- its code, then
// rotate and copy-link -- and rotate all. The screen broadcast's pointer link
// is a row of its label alone until pointing first makes it (pair#495: the
// menu never turns pointing on as a side effect of a copy). Chrome also has
// an annotation row using the same word layout and hit-testing.
func (c *Console) openCodesMenu(kind castKind) {
	box, ok := c.buildCodesMenu(kind)
	if !ok {
		return
	}
	c.codes.mu.Lock()
	c.codes.menu, c.codes.menuBox = kind, box
	c.codes.mu.Unlock()
	c.repaint()
}

// buildCodesMenu lays out kind's menu from its current codes.
func (c *Console) buildCodesMenu(kind castKind) (codesMenu, bool) {
	codes := c.castCodesOf(kind)
	if len(codes) == 0 {
		return codesMenu{}, false
	}
	c.mu.Lock()
	cols, rows := int(c.size.Cols), int(c.size.Rows)-c.statusRows+1
	c.mu.Unlock()
	var menuRows []codesMenuRow
	hasPointer := false
	for _, code := range codes {
		hasPointer = hasPointer || code.Link == broadcast.LinkPointer
		menuRows = append(menuRows, codesMenuRow{label: code.Link + " " + code.Code, actions: []codesMenuWord{
			{text: "rotate", action: codesAction{link: code.Link}},
			{text: "copy-link", action: codesAction{copy: true, link: code.Link}},
		}})
	}
	if kind == castScreen && !hasPointer {
		menuRows = append(menuRows, codesMenuRow{label: "pointer (not made yet)"})
	}
	if len(codes) > 1 {
		menuRows = append(menuRows, codesMenuRow{label: "", actions: []codesMenuWord{{text: "rotate all", action: codesAction{}}}})
	}
	if kind == castWeb {
		menuRows = append(menuRows, codesMenuRow{label: "Annotation", actions: []codesMenuWord{
			{text: "Off", action: codesAction{annotation: "watch"}},
			{text: "Pen", action: codesAction{annotation: "point"}},
			{text: "Note", action: codesAction{annotation: "note"}},
		}})
	}
	box := layoutCodesMenu(kind.name()+" codes", menuRows, cols, rows)
	return box, box.Open()
}

// refreshCodesMenu redraws the open menu with kind's current codes, keeping
// its selection: a rotated code changes in place.
func (c *Console) refreshCodesMenu(kind castKind) {
	box, ok := c.buildCodesMenu(kind)
	c.codes.mu.Lock()
	if c.codes.menu == kind && ok {
		c.codes.menuBox = box.Select(c.codes.menuBox.sel)
	}
	c.codes.mu.Unlock()
	c.repaint()
}

func (c *Console) closeCodesMenu() {
	c.codes.mu.Lock()
	open := c.codes.menu != castNone
	c.codes.menu, c.codes.menuBox = castNone, codesMenu{}
	c.codes.mu.Unlock()
	if open {
		c.repaint()
	}
}

// selectCodesWord moves the menu's selection to sel (a mouse over a word).
func (c *Console) selectCodesWord(sel codesMenuSel) {
	c.codes.mu.Lock()
	before := c.codes.menuBox.sel
	c.codes.menuBox = c.codes.menuBox.Select(sel)
	changed := c.codes.menuBox.sel != before
	c.codes.mu.Unlock()
	if changed {
		c.repaint()
	}
}

// routeCodesMenuKey takes the keyboard while the Codes menu is open:
// Up/Down pick the row, Left/Right the action, Enter runs it, Escape closes.
// Any other key goes on, to its usual place. It reports whether it took raw.
func (c *Console) routeCodesMenuKey(raw, canonical []byte) bool {
	c.codes.mu.Lock()
	open := c.codes.menu != castNone
	c.codes.mu.Unlock()
	if !open {
		return false
	}
	if len(canonical) == 1 && canonical[0] == 0x1b {
		// A lone Escape, once the decoder's ambiguity wait has passed.
		c.closeCodesMenu()
		return true
	}
	keys, held := DecodePanelKeys(raw)
	if len(keys) != 1 || len(held) != 0 {
		return false
	}
	dy, dx := 0, 0
	switch keys[0].Kind {
	case KeyUp:
		dy = -1
	case KeyDown:
		dy = 1
	case KeyLeft:
		dx = -1
	case KeyRight:
		dx = 1
	case KeyEnter:
		c.runCodesSelection()
		return true
	case KeyEscape:
		c.closeCodesMenu()
		return true
	default:
		return false
	}
	c.codes.mu.Lock()
	c.codes.menuBox = c.codes.menuBox.Move(dy, dx)
	c.codes.mu.Unlock()
	c.repaint()
	return true
}

// runCodesSelection does the selected word: selects annotation, copies its
// link, or rotates its code (or every link's).
func (c *Console) runCodesSelection() {
	c.codes.mu.Lock()
	kind := c.codes.menu
	action, ok := c.codes.menuBox.Selected()
	if ok {
		// The menu stays open (Escape or × closes it); the word shows it
		// was pressed.
		c.codes.pressed, c.codes.pressedUntil = c.codes.menuBox.sel, c.now().Add(codesPress)
	}
	c.codes.mu.Unlock()
	if !ok {
		return
	}
	c.repaint()
	c.GoTracked(func() {
		select {
		case <-time.After(codesPress):
			c.repaint()
		case <-c.lifetime.Done():
		}
	})
	if action.annotation != "" {
		c.mu.Lock()
		cast, running := c.wcast.cast, c.wcast.phase == webcastRunning
		c.mu.Unlock()
		if kind != castWeb || !running || cast == nil {
			c.setNotice("Chrome: annotation is unavailable; the cast is not running")
		} else if err := cast.SetAnnotationMode(action.annotation); err != nil {
			c.setNotice("Chrome: couldn't change annotation: " + err.Error())
		}
		return
	}
	if action.copy {
		c.copyCastLink(kind, action.link)
		return
	}
	defer c.refreshCodesMenu(kind)
	var links []string
	if action.link == "" {
		for _, code := range c.castCodesOf(kind) {
			links = append(links, code.Link)
		}
	} else {
		links = []string{action.link}
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

// copyCastLink copies one of a cast's links, as its control's click does.
func (c *Console) copyCastLink(kind castKind, link string) {
	switch {
	case kind == castScreen && link == broadcast.LinkView:
		c.copyViewLink()
	case kind == castScreen && link == broadcast.LinkPointer:
		c.onPointerClick(true)
	case kind == castWeb && link == broadcast.LinkView:
		c.copyWebcastLink()
	case kind == castWeb && link == broadcast.LinkPointer:
		c.copyWebcastPointerLink()
	case kind == castWeb && link == broadcast.LinkDrive:
		c.copyWebcastDriveLink()
	}
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
// wins; every current guessing rotation is told in a notice without the code.
func (c *Console) onCodeRotated(kind castKind, attempt uint64, link string, r castlock.Rotation) {
	key := linkKey(kind, attempt, link)
	now := c.now()
	// No caller holds the console's locks while a code rotates (the gate's
	// unlock, or the menu's RotateCode), so the console can be read here.
	// The rotation counts only while the cast that made it is live: the
	// same start attempt AND its session or cast still running. A cast that
	// ended keeps its attempt number until the next start, so the attempt
	// alone can't tell (BR-23). c.mu is held until the flash is set, so the
	// check and the write are one step against the end's reset, which runs
	// under c.mu. c.mu before codes.mu is the order everywhere.
	c.mu.Lock()
	cell, current := c.codeAnchorLocked(kind), c.bcast.attempt
	live := c.bcast.session != nil && c.bcast.phase == broadcastLive
	if kind == castWeb {
		current = c.wcast.attempt
		live = c.wcast.cast != nil && c.wcast.phase == webcastRunning
	}
	if attempt != current || !live {
		c.mu.Unlock()
		return // a late report from a cast that has ended
	}
	c.codes.mu.Lock()
	if c.codes.gens == nil {
		c.codes.gens = map[string]uint64{}
	}
	if r.Gen <= c.codes.gens[key] {
		c.codes.mu.Unlock()
		c.mu.Unlock()
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
		if r.Capped {
			notice = fmt.Sprintf("Too many wrong codes on the %s %s link: its code changed; the link is locked until you rotate its code.", kind.name(), link)
		} else {
			notice = fmt.Sprintf("Too many wrong codes on the %s %s link: its code changed. Hover %s to see it.", kind.name(), link, kind.name())
		}
	}
	c.codes.mu.Unlock()
	// Publish before releasing the generation guard: an older callback must
	// never overwrite the notice from a newer, capped rotation.
	paintNotice := false
	if notice != "" {
		paintNotice = c.publishNoticeLocked(Notice{Kind: "status", Body: notice})
	}
	c.mu.Unlock() // before GoTracked, which takes c.mu
	if paintNotice {
		c.repaint()
	}
	c.GoTracked(c.flashCodes)
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
		c.codes.menu, c.codes.menuBox = castNone, codesMenu{}
	}
	if c.codes.flash != nil && c.codes.flash.cast == kind {
		c.codes.flash = nil
	}
	for key := range c.codes.gens {
		if strings.HasPrefix(key, strconv.Itoa(int(kind))+"/") {
			delete(c.codes.gens, key)
		}
	}
	c.codes.mu.Unlock()
}

// reflowCodesMenu reuses the retained semantic rows and selection against the
// exact frame being composed, without reading Console or scheduling a repaint.
func reflowCodesMenu(menu codesMenu, cols, rows int) codesMenu {
	source := make([]codesMenuRow, len(menu.rows))
	for i, row := range menu.rows {
		source[i].label = row.label
		for _, word := range row.words {
			source[i].actions = append(source[i].actions, word.codesMenuWord)
		}
	}
	return layoutCodesMenu(menu.title, source, cols, rows).Select(menu.sel)
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
	if f.ChromeRows == 0 {
		c.codes.menu, c.codes.menuBox = castNone, codesMenu{}
		c.codes.hover, c.codes.hoverCodes = castNone, nil
		c.codes.flash = nil
		return f
	}
	cols, rows := f.Geometry.Cols, f.Geometry.Rows-f.ChromeRows+1
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
		c.codes.menuBox = reflowCodesMenu(c.codes.menuBox, cols, rows)
		if !c.codes.menuBox.Open() {
			c.codes.menu = castNone
			return f
		}
		press := codesMenuSel{row: -1}
		if now.Before(c.codes.pressedUntil) {
			press = c.codes.pressed
		}
		f = c.codes.menuBox.Draw(f, press)
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
	return NotesMenu{X: x, Y: rows - 2, W: w, H: 1, lines: []notesMenuLine{{text: line, tip: true}}}
}

// installCodesLayer puts the codes, and the attention list over them
// (pair#673), on the operator's screen only: the presenter has one
// operator-only layer.
func (c *Console) installCodesLayer() error {
	return c.presenter.SetOperatorLayer(context.Background(), func(f terminal.Frame, class terminal.FrameClass) terminal.Frame {
		return c.attentionLayer(c.codesLayer(c.remoteSessionLayer(f), class))
	})
}
