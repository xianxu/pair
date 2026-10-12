package couchtty

import (
	"strconv"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Remote driving in Couch (#467). The web cast's controller tells Couch who
// drives the cast tab and who asks; the tab bar shows it after the Chrome
// cell (🕹Ana for the driver, ✋Bo for one asker, ✋2 for more), and a click
// there opens the drive menu: a left click on the driver takes control
// back, on an asker lets them drive; a right click turns an asker down.
// Ctrl+click on the span takes control back whatever is open: the operator
// always has the last word. While anyone drives, the cast tab's uploads and
// downloads are off (the operator's too), and Couch says so.

// driveState is the delivered control and its menu rows, under pnotes.mu.
type driveState struct {
	cast    *webcast.Cast
	control recording.Control
	rows    []driveRow
	items   []broadcast.Note
	// ids keeps one menu id per row key (driver:<id>, ask:<id>) while the
	// row shows, so a click acts on the row drawn even if the list moved.
	ids  map[string]int
	next int
}

// driveRow is one menu row: the driver's or an asker's.
type driveRow struct {
	menuID int
	who    recording.Author
	driver bool
}

// Menu chip colors: the palette's own indexes (#446).
const (
	driveChipDriver = 1
	driveChipAsker  = 3
)

// driveMenuRows lays control out as the menu's rows and items, with stable
// ids from menuID.
func driveMenuRows(c recording.Control, menuID func(key string) int) ([]driveRow, []broadcast.Note) {
	var rows []driveRow
	var items []broadcast.Note
	if d := c.Driver; d != nil {
		id := menuID("driver:" + d.ID)
		rows = append(rows, driveRow{menuID: id, who: *d, driver: true})
		items = append(items, broadcast.Note{ID: id, Color: driveChipDriver,
			Text: d.Name + " is driving: uploads and downloads off while " + d.Name + " drives. Click to take control back."})
	}
	for _, p := range c.Pending {
		id := menuID("ask:" + p.ID)
		rows = append(rows, driveRow{menuID: id, who: p})
		items = append(items, broadcast.Note{ID: id, Color: driveChipAsker,
			Text: p.Name + " asks to drive. Click to let " + p.Name + " drive; right-click to say no."})
	}
	return rows, items
}

// driveSpan is the tab bar's text for control: "" when nobody drives or
// asks.
func driveSpan(c recording.Control) string {
	var out string
	if c.Driver != nil {
		out = "🕹" + c.Driver.Name
	}
	switch n := len(c.Pending); {
	case n == 1:
		out = joinSpan(out, "✋"+c.Pending[0].Name)
	case n > 1:
		out = joinSpan(out, "✋"+strconv.Itoa(n))
	}
	return out
}

func joinSpan(a, b string) string {
	if a == "" {
		return b
	}
	return a + " " + b
}

// driveNotices is what Couch says when control moves from was to now: the
// operator learns of a grant's side effects and of every request.
func driveNotices(was, now recording.Control) []string {
	var out []string
	if d := now.Driver; d != nil && was.Driving() != d.ID {
		out = append(out, d.Name+" is driving the cast tab. Uploads and downloads are off until control comes back.")
	}
	if was.Driver != nil && now.Driver == nil {
		out = append(out, "Control of the cast tab is back with you.")
	}
	for _, p := range now.Pending {
		asked := false
		for _, q := range was.Pending {
			asked = asked || q.ID == p.ID
		}
		if !asked {
			out = append(out, p.Name+" asks to drive the cast tab: click ✋ in the tab bar.")
		}
	}
	return out
}

// applyDrive takes the cast's control state on the loop; a delivery from a
// cast that has since ended is dropped.
func (c *Console) applyDrive(cast *webcast.Cast, ctl recording.Control) {
	c.mu.Lock()
	current := c.wcast.cast == cast
	c.mu.Unlock()
	if !current {
		return
	}
	c.pnotes.mu.Lock()
	was := c.pnotes.drive.control
	d := &c.pnotes.drive
	d.cast, d.control = cast, ctl
	shown := map[string]bool{}
	d.rows, d.items = driveMenuRows(ctl, func(key string) int {
		shown[key] = true
		if id, ok := d.ids[key]; ok {
			return id
		}
		if d.ids == nil {
			d.ids = map[string]int{}
		}
		d.next++
		d.ids[key] = d.next
		return d.next
	})
	for key := range d.ids {
		if !shown[key] {
			delete(d.ids, key)
		}
	}
	if c.pnotes.source == driveMenu {
		c.settleMenuLocked()
	}
	c.pnotes.mu.Unlock()
	for _, n := range driveNotices(was, ctl) {
		c.setNotice(n)
	}
	c.repaint()
	c.terminalError(c.presenter.Refresh(c.lifetime))
}

// forgetDrive drops control when the cast ends.
func (c *Console) forgetDrive() {
	c.pnotes.mu.Lock()
	c.pnotes.drive = driveState{}
	if c.pnotes.source == driveMenu {
		c.settleMenuLocked()
	}
	c.pnotes.mu.Unlock()
}

// webDriveSpan is the span's text for the status row.
func (c *Console) webDriveSpan() string {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	return driveSpan(c.pnotes.drive.control)
}

// driveActionLocked is what a click on menu row id does (nil: nothing):
// right is a right click. Callers hold pnotes.mu; the action runs after.
func (c *Console) driveActionLocked(id int, right bool) func() {
	cast := c.pnotes.drive.cast
	if cast == nil || cast.Drive() == nil {
		return nil
	}
	ctrl := cast.Drive()
	for _, r := range c.pnotes.drive.rows {
		if r.menuID != id {
			continue
		}
		who := r.who
		switch {
		case r.driver:
			return func() { c.driveResult(ctrl.Revoke(), "") }
		case right:
			return func() { c.driveResult(ctrl.Deny(who.ID), "Said no to "+who.Name+".") }
		default:
			return func() { c.driveResult(ctrl.Grant(who.ID), "") }
		}
	}
	return nil
}

// revokeDrive is a Ctrl+click on the drive span: control back, from any
// state, menu open or not.
func (c *Console) revokeDrive() {
	c.pnotes.mu.Lock()
	cast := c.pnotes.drive.cast
	c.pnotes.mu.Unlock()
	if cast == nil || cast.Drive() == nil {
		return
	}
	c.driveResult(cast.Drive().Revoke(), "")
}

// driveResult tells the operator a control change that didn't happen (the
// state moved under the click: the viewer gave up or left); one that did
// is told by applyDrive, from the state that results.
func (c *Console) driveResult(err error, done string) {
	switch {
	case err != nil:
		c.setNotice("Remote driving: that changed meanwhile.")
	case done != "":
		c.setNotice(done)
	}
}
