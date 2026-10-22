package wrapcmd

import (
	"context"
	"os"
	"strings"
	"time"
	"unicode"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// pair#428: a delivery pastes only while the operator can see the composer.
// Zellij keeps each pane's scroll position to itself; `dump-screen` prints
// the viewport as scrolled, and the wrapper's own model holds what the agent
// drew. Aligning the two gives the scroll shift without reading the agent.

// peerViewFresh is how long one viewport reading stands for the paste.
const peerViewFresh = 500 * time.Millisecond

// peerView is the latest viewport reading (in memory, one per delivery
// wrapper): when it was taken, and whether the cursor row was in view.
type peerView struct {
	at       time.Time
	visible  bool
	checking bool
}

// viewReadyLocked (d.mu held) answers from a fresh reading, else starts one
// off the dispatch locks and answers "not yet": a subprocess must not hold
// input admission.
func (d *peerDelivery) viewReadyLocked() bool {
	if d.viewProbe == nil {
		return true
	}
	if !d.view.at.IsZero() && d.now().Before(d.view.at.Add(peerViewFresh)) {
		return d.view.visible
	}
	if !d.view.checking {
		d.view.checking = true
		probe := d.viewProbe
		go func() {
			visible := probe()
			d.mu.Lock()
			d.view = peerView{at: d.now(), visible: visible}
			d.mu.Unlock()
			d.signal()
		}()
	}
	return false
}

// composerInView is the production viewport probe. Anything it cannot judge
// (no zellij, a failed dump, a view that matches no shift) reads as in view:
// the gate never holds a delivery on its own uncertainty.
func (p *proxy) composerInView() bool {
	pane := os.Getenv("ZELLIJ_PANE_ID")
	terminal := p.terminal
	if pane == "" || terminal == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	dump, err := (zellijquery.Source{}).Output(ctx, "action", "dump-screen", "-p", "terminal_"+pane)
	if err != nil {
		p.debug("PEER-view-fail", err.Error())
		return true
	}
	rows, height, cursorY := terminal.ViewRows()
	return cursorRowInView(rows, height, cursorY, string(dump))
}

// cursorRowInView: the composer counts as in view while the cursor's row is,
// so a small scroll that keeps it on screen doesn't hold a delivery.
func cursorRowInView(rows []string, height, cursorY int, dump string) bool {
	shift, ok := viewportShift(rows, height, strings.Split(strings.TrimRight(dump, "\n"), "\n"))
	return !ok || shift <= height-1-cursorY
}

// ViewRows renders the scrollback then the screen as text, one row each,
// with the screen's height and the cursor's screen row.
func (m *terminalModel) ViewRows() (rows []string, height, cursorY int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, 0, 0
	}
	if !m.emulator.IsAltScreen() {
		for _, line := range m.emulator.Scrollback().Lines() {
			rows = append(rows, cellsText([]uv.Cell(line)))
		}
	}
	width, height := m.emulator.Width(), m.emulator.Height()
	for y := 0; y < height; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			if cell := m.emulator.CellAt(x, y); cell != nil {
				b.WriteString(cell.Content)
			}
		}
		rows = append(rows, b.String())
	}
	return rows, height, m.emulator.CursorPosition().Y
}

func cellsText(cells []uv.Cell) string {
	var b strings.Builder
	for _, c := range cells {
		b.WriteString(c.Content)
	}
	return b.String()
}

// viewportShift is how many rows the view is scrolled up from the bottom of
// rows (scrollback then a screen of height rows): the smallest shift where
// all but two rows line up. The slack absorbs a row repainted between the
// two reads (a spinner); whitespace is ignored, since the two renderers pad
// and space wide cells differently.
func viewportShift(rows []string, height int, view []string) (int, bool) {
	if height <= 0 || len(rows) < height {
		return 0, false
	}
	norm := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}
	model := make([]string, len(rows))
	for i, row := range rows {
		model[i] = norm(row)
	}
	want := make([]string, height)
	for i := range want {
		if i < len(view) {
			want[i] = norm(view[i])
		}
	}
	for shift := 0; shift <= len(rows)-height; shift++ {
		top, misses := len(rows)-height-shift, 0
		for i := 0; i < height && misses <= 2; i++ {
			if model[top+i] != want[i] {
				misses++
			}
		}
		if misses <= 2 {
			return shift, true
		}
	}
	return 0, false
}
