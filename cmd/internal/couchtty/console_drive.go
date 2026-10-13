package couchtty

import (
	"errors"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Remote driving in Couch (#467). The web cast's controller tells Couch who
// drives the cast tab and who asks; the tab bar shows it after the Chrome
// cell (👽 Ana on the enabled drive control, ✋Bo for one asker, ✋2
// for more). The driver chip toggles driving off or copies its link;
// the asking span opens the drive menu: left grants, right declines.
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

// driveSpan shows pending requests separately from the 👽 driver control.
func driveSpan(c recording.Control) string {
	switch n := len(c.Pending); {
	case n == 1:
		return "✋" + c.Pending[0].Name
	case n > 1:
		return "✋" + strconv.Itoa(n)
	default:
		return ""
	}
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

// webDriveLabels derives both row labels from one delivered control snapshot.
func (c *Console) webDriveLabels() (driver, asking string) {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	ctl := c.pnotes.drive.control
	if ctl.Driver != nil {
		driver = ctl.Driver.Name
	}
	return driver, driveSpan(ctl)
}

// driveActionLocked is what a click on menu row id does (nil: nothing):
// right is a right click. Callers hold pnotes.mu; the action runs after.
func (c *Console) driveActionLocked(id int, right bool) func() {
	cast := c.pnotes.drive.cast
	if cast == nil || cast.Drive() == nil {
		return nil
	}
	for _, r := range c.pnotes.drive.rows {
		if r.menuID != id {
			continue
		}
		who := r.who
		switch {
		case r.driver:
			return func() { c.startDriveOperation(cast, recording.CtlRevoke, who) }
		case right:
			return func() { c.startDriveOperation(cast, recording.CtlDeny, who) }
		default:
			return func() { c.startDriveOperation(cast, recording.CtlGrant, who) }
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
	c.startDriveOperation(cast, recording.CtlRevoke, recording.Author{})
}

// One slot belongs to the console, not the cast's replaceable menu state.
// It stays occupied through ACK and terminal completion; repeated clicks never
// create a queue or detached workers.
type driveOperation struct{ op string }

func (c *Console) startDriveOperation(cast *webcast.Cast, op string, who recording.Author) {
	c.mu.Lock()
	if !c.started || c.lifetime.Err() != nil || c.wcast.cast != cast || c.wcast.phase != webcastRunning {
		c.mu.Unlock()
		return
	}
	if c.drivePending != nil {
		c.mu.Unlock()
		c.setNotice("Remote driving: wait for the current action.")
		return
	}
	pending := &driveOperation{op: op}
	c.drivePending = pending
	c.mu.Unlock()
	generation := cast.Drive().DriveCapability().Generation
	admission, err := cast.Drive().AdmitDriveGeneration(generation)
	if err != nil {
		c.mu.Lock()
		c.drivePending = nil
		c.mu.Unlock()
		c.driveResult(err, "")
		return
	}
	c.GoTracked(func() {
		defer func() {
			c.mu.Lock()
			if c.drivePending == pending {
				c.drivePending = nil
			}
			c.mu.Unlock()
		}()
		// GoTracked runs inline after teardown begins. Never dispatch in that case.
		if c.lifetime.Err() != nil {
			return
		}
		var err error
		done := ""
		switch pending.op {
		case recording.CtlGrant:
			err = admission.Grant(who.ID)
		case recording.CtlDeny:
			err = admission.Deny(who.ID)
			done = "Said no to " + who.Name + "."
		case recording.CtlRevoke:
			err = admission.Revoke()
		}
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.mu.Lock()
			current := c.lifetime.Err() == nil && c.wcast.cast == cast && c.wcast.phase == webcastRunning && c.drivePending == pending
			c.mu.Unlock()
			if current && cast.Drive().DriveCapability().Generation == generation {
				c.driveResult(err, done)
			}
			return nil
		})
	})
}

// driveResult tells the operator a control change that didn't happen;
// one that did is told by applyDrive, from the state that results.
func (c *Console) driveResult(err error, done string) {
	if err != nil {
		c.setNotice(driveFailure(err))
	} else if done != "" {
		c.setNotice(done)
	}
}

// driveFailure is what the operator is told when a control change fails:
// a race (the viewer gave up or left, the state moved under the click) is
// said as one; anything else is a failure, with its cause, and control is
// as it was.
func driveFailure(err error) string {
	switch {
	case errors.Is(err, drive.ErrNoSeat), errors.Is(err, drive.ErrControlOp), errors.Is(err, drive.ErrDriveUnavailable):
		return "Remote driving: that changed meanwhile."
	case errors.Is(err, drive.ErrClosed):
		return "Remote driving: the cast has ended."
	case errors.Is(err, drive.ErrNotLive):
		return "Remote driving: go live first; control changes once the cast is recording."
	}
	return "Remote driving failed, control is unchanged: " + err.Error()
}

// Shares is every link out, by kind, as `couch --broadcast-list` lists them
// (#467): the screen broadcast, the web cast's view-only and pointer links
// (counted together as `web view`, pair#506) and its drive link. Never a link or a token. Counts are read after c.mu is released.
func (c *Console) Shares() []couchmessage.ShareStatus {
	var out []couchmessage.ShareStatus
	if b, ok := c.BroadcastStatus(); ok {
		out = append(out, couchmessage.ShareStatus{Kind: couchmessage.ShareScreen, State: b.State, Viewers: b.Viewers})
	}
	c.mu.Lock()
	cast, running, u := c.wcast.cast, c.wcast.phase == webcastRunning, c.wcast.update
	c.mu.Unlock()
	if !running || cast == nil || u.Link == "" {
		return out
	}
	state := "idle"
	if u.State.Phase == webcast.PhaseLive {
		state = "live"
	}
	view, drive := cast.Viewers()
	out = append(out, couchmessage.ShareStatus{Kind: couchmessage.ShareWebView, State: state, Viewers: view})
	if u.DriveLink != "" && cast.Drive() != nil {
		ctl := cast.Drive().Control()
		s := couchmessage.ShareStatus{Kind: couchmessage.ShareWebDrive, State: state, Viewers: drive, Asking: len(ctl.Pending)}
		if ctl.Driver != nil {
			s.Driver = ctl.Driver.Name
		}
		out = append(out, s)
	}
	return out
}
