package couchtty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var (
	ana = recording.Author{ID: "v1", Name: "Ana"}
	bo  = recording.Author{ID: "v2", Name: "Bo"}
	cy  = recording.Author{ID: "v3", Name: "Cy"}
)

// The span, the menu rows and what Couch says, for every control state.
func TestDriveSpanRowsAndNotices(t *testing.T) {
	for _, tc := range []struct {
		name  string
		c     recording.Control
		span  string
		texts []string
	}{
		{"nobody", recording.Control{}, "", nil},
		{"one asks", recording.Control{Pending: []recording.Author{bo}}, "✋Bo", []string{"Bo asks to drive. Click to let Bo drive; right-click to say no."}},
		{"two ask", recording.Control{Pending: []recording.Author{bo, cy}}, "✋2", []string{"Bo asks", "Cy asks"}},
		{"a driver", recording.Control{Driver: &ana}, "", []string{"Ana is driving: uploads and downloads off while Ana drives. Click to take control back."}},
		{"a driver and an asker", recording.Control{Driver: &ana, Pending: []recording.Author{bo}}, "✋Bo", []string{"Ana is driving", "Bo asks"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := driveSpan(tc.c); got != tc.span {
				t.Errorf("span %q, want %q", got, tc.span)
			}
			n := 0
			rows, items := driveMenuRows(tc.c, func(string) int { n++; return n })
			if len(rows) != len(tc.texts) || len(items) != len(tc.texts) {
				t.Fatalf("rows %d, want %d", len(rows), len(tc.texts))
			}
			for i, want := range tc.texts {
				if !strings.HasPrefix(items[i].Text, want) || rows[i].menuID != items[i].ID {
					t.Errorf("row %d: %q (id %d/%d), want %q", i, items[i].Text, rows[i].menuID, items[i].ID, want)
				}
			}
		})
	}
	notices := driveNotices(recording.Control{Pending: []recording.Author{bo}}, recording.Control{Driver: &bo, Pending: []recording.Author{cy}})
	if len(notices) != 2 || !strings.Contains(notices[0], "Bo is driving the cast tab. Uploads and downloads are off") || !strings.Contains(notices[1], "Cy asks") {
		t.Errorf("grant and a new asker: %q", notices)
	}
	if n := driveNotices(recording.Control{Driver: &bo}, recording.Control{}); len(n) != 1 || !strings.Contains(n[0], "back with you") {
		t.Errorf("control back: %q", n)
	}
	if n := driveNotices(recording.Control{Driver: &bo, Pending: []recording.Author{cy}}, recording.Control{Driver: &bo, Pending: []recording.Author{cy}}); len(n) != 0 {
		t.Errorf("nothing changed: %q", n)
	}
}

// The span sits after the Chrome cell and its 💭N, plain text, its own
// click target.
func TestRenderDriveSpan(t *testing.T) {
	row := RenderStatusRow(80, StatusModel{Webcast: WebcastLive, WebDrivingOn: true, WebDriverName: "Ana", WebNotes: 2, WebDrive: "✋Bo"})
	if row.WebDrive == (ColumnSpan{}) || row.WebDrive.Start <= row.WebNotes.End || row.WebNotes.Start <= row.WebDriveLink.End {
		t.Fatalf("spans: cell %+v notes %+v drive %+v", row.Webcast, row.WebNotes, row.WebDrive)
	}
	if !strings.Contains(row.Body, "✋Bo") {
		t.Fatalf("body %q", row.Body)
	}
	if row := RenderStatusRow(80, StatusModel{Webcast: WebcastLive, WebDrive: "✋A\x1b[2Jna"}); strings.Contains(row.Body, "\x1b[2J") {
		t.Fatalf("a name reached the terminal raw: %q", row.Body)
	}
	if row := RenderStatusRow(80, StatusModel{WebDrive: "✋Ana"}); row.WebDrive != (ColumnSpan{}) {
		t.Fatal("a drive span without Chrome")
	}
}

// driveCast is a console with a recorded, drivable web cast (FakeDriven)
// and viewers that take seats and ask over the cast link.
type driveCast struct {
	*webcastRig
	d              *webcast.FakeDriven
	link, viewLink string
}

func newDriveCast(t *testing.T, wrap ...func(*webcast.FakeDriven) webcast.Browser) *driveCast {
	t.Helper()
	var d *webcast.FakeDriven
	r := newWebcastRig(t, func(r *webcastRig) {
		r.wrap = func(f *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
			var err error
			if d, err = webcast.NewFakeDriven(f, drive.DrivenPage(100), &recording.MemSink{}); err != nil {
				t.Fatal(err)
			}
			if len(wrap) != 0 {
				return wrap[0](d)
			}
			return d
		}
	})
	r.child.Feed([]byte("\x1b[?1003;1006h"))
	waitFor(t, "child tracks the mouse", func() bool { return r.child.Endpoint().Modes().Tracking != 0 })
	r.open(t)
	d.Open()
	c := &driveCast{webcastRig: r, d: d}
	if err := c.cast().SetCapability(context.Background(), webcast.CapabilityDrive, true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "link", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		c.link, c.viewLink = r.con.wcast.update.DriveLink, r.con.wcast.update.Link
		return c.link != ""
	})
	unlockWebCast(t, r.con)
	return c
}

// viewer takes a seat and asks to drive, as a viewer's shell does.
func (c *driveCast) viewer(t *testing.T, name string) string {
	t.Helper()
	post := func(route, key, body string) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, c.link+"drive/"+route, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Couch-Drive-Generation", fmt.Sprint(c.cast().Capabilities().Drive.Generation))
		if key != "" {
			req.Header.Set(drive.SeatHeader, key)
		}
		resp, err := castClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := post("hello", "", fmt.Sprintf(`{"name":%q}`, name))
	var seat struct{ Viewer, Key string }
	json.NewDecoder(resp.Body).Decode(&seat)
	resp.Body.Close()
	if resp = post("control", seat.Key, `{"op":"request"}`); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("%s's request: %d", name, resp.StatusCode)
	}
	resp.Body.Close()
	return seat.Viewer
}

func (c *driveCast) control() recording.Control { return c.cast().Drive().Control() }

func (c *driveCast) cast() *webcast.Cast {
	c.con.mu.Lock()
	defer c.con.mu.Unlock()
	return c.con.wcast.cast
}

// waitDrive bounds fixture setup independently of the package's deadline.
// A lost menu click must fail here, not consume the full suite's ten minutes.
func (c *driveCast) waitDrive(t *testing.T, what string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			c.con.mu.Lock()
			span, phase, pending := c.con.statusWebDrive, c.con.wcast.phase, c.con.drivePending != nil
			c.con.mu.Unlock()
			c.con.pnotes.mu.Lock()
			menu, source, control := c.con.pnotes.menuOpen, c.con.pnotes.source, c.con.pnotes.drive.control
			c.con.pnotes.mu.Unlock()
			t.Fatalf("waiting for %s: span=%+v phase=%v pending=%v menu=%v source=%v control=%+v row=%q", what, span, phase, pending, menu, source, control, c.lastRow())
		}
		time.Sleep(2 * time.Millisecond)
	}
}
func awaitDrive(t *testing.T, what string, done <-chan struct{}) {
	t.Helper()
	if done == nil {
		t.Fatalf("%s: action was not accepted (nil completion channel)", what)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("waiting for %s", what)
	}
}

// Read the hit target only once its current control projection has committed.
// Sampling lastRow separately can pair a new visible row with an old hit span.
func (c *driveCast) span(t *testing.T, text string) ColumnSpan {
	t.Helper()
	var span ColumnSpan
	c.waitDrive(t, "committed drive target "+text, func() bool {
		want := c.control()
		c.con.pnotes.mu.Lock()
		current := c.con.pnotes.drive.control.Equal(want)
		c.con.pnotes.mu.Unlock()
		if !current {
			return false
		}
		c.con.mu.Lock()
		defer c.con.mu.Unlock()
		model := c.con.statusModelLocked()
		span = c.con.statusWebDrive
		row := RenderStatusRow(int(c.con.size.Cols), model)
		return c.con.drivePending == nil && span != (ColumnSpan{}) && span == row.WebDrive && (text == "" || model.WebDrive == text)
	})
	return span
}

// tryEnableDrive makes no attempt until cleanup reaches Off. Even then the
// previous Cast worker may still own its slot: nil means no action was admitted.
// A non-nil result owns one enable, independently of later notice delivery.
func (c *driveCast) tryEnableDrive() <-chan struct{} {
	if c.cast().Capabilities().Drive.Phase != capability.Off {
		return nil
	}
	return c.con.onWebCapabilityClick(webcast.CapabilityDrive, false)
}

func (c *driveCast) reenableDrive(t *testing.T) {
	t.Helper()
	var done <-chan struct{}
	c.waitDrive(t, "drive re-enable admitted", func() bool {
		done = c.tryEnableDrive()
		return done != nil
	})
	awaitDrive(t, "drive re-enable completed", done)
}

// heldDriveRelease pauses after reset has cleared control, before cleanup can
// finish releasing input. Ordinary driver changes remain untouched until armed.
type heldDriveRelease struct {
	*webcast.FakeDriven
	armed            atomic.Bool
	entered, release chan struct{}
}

func (b *heldDriveRelease) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	if err := b.FakeDriven.WithTab(ctx, by, fn); err != nil {
		return err
	}
	if by == "" && b.armed.CompareAndSwap(true, false) {
		close(b.entered)
		select {
		case <-b.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestDriveDisableCompletionOrdering(t *testing.T) {
	t.Run("control reset precedes cleanup completion", func(t *testing.T) {
		b := &heldDriveRelease{entered: make(chan struct{}), release: make(chan struct{})}
		var once sync.Once
		release := func() { once.Do(func() { close(b.release) }) }
		defer release()
		c := newDriveCast(t, func(d *webcast.FakeDriven) webcast.Browser {
			b.FakeDriven = d
			return b
		})
		v := c.viewer(t, "Ana")
		if err := c.cast().Drive().Grant(v); err != nil {
			t.Fatal(err)
		}
		waitFor(t, "Chrome told Ana drives", func() bool { return c.d.Driving() == "Ana" })
		b.armed.Store(true)
		c.con.mu.Lock()
		span := c.con.statusWebDriveLink
		c.con.mu.Unlock()
		c.mouse(0, span.End-1, 23, false)
		c.mouse(0, span.End-1, 23, true)
		awaitDrive(t, "input release entered", b.entered)
		waitFor(t, "control back before cleanup settles", func() bool { return c.control().Driver == nil })
		waitFor(t, "Chrome told nobody drives before cleanup settles", func() bool { return c.d.Driving() == "" })
		p := c.cast().Capabilities().Project(webcast.CapabilityDrive)
		if p.Enabled || p.Action != "wait" {
			t.Fatalf("unfinished cleanup: %+v", p)
		}
		if done := c.con.onWebCapabilityClick(webcast.CapabilityDrive, false); done != nil {
			t.Fatal("click during cleanup was accepted")
		}
		if done := c.tryEnableDrive(); done != nil {
			t.Fatal("fixture admitted enable before cleanup completed")
		}
		release()
		c.reenableDrive(t)
		if !c.cast().Capabilities().Project(webcast.CapabilityDrive).Enabled {
			t.Fatal("drive did not re-enable after cleanup")
		}
	})
	t.Run("authority off precedes terminal completion", func(t *testing.T) {
		c := newDriveCast(t)
		entered, release, joined := make(chan struct{}), make(chan struct{}), make(chan struct{})
		var once sync.Once
		unblock := func() { once.Do(func() { close(release) }) }
		defer func() { unblock(); awaitDrive(t, "terminal barrier joined", joined) }()
		go func() {
			defer close(joined)
			_ = c.con.runTerminalCommand(c.con.lifetime, func() error {
				close(entered)
				<-release
				return nil
			})
		}()
		awaitDrive(t, "terminal barrier entered", entered)
		done := c.con.onWebCapabilityClick(webcast.CapabilityDrive, false)
		waitFor(t, "authority off", func() bool {
			return c.cast().Capabilities().Drive.Phase == capability.Off
		})
		var enabled <-chan struct{}
		c.waitDrive(t, "enable admitted after Off while terminal blocked", func() bool {
			enabled = c.tryEnableDrive()
			return enabled != nil
		})
		select {
		case <-enabled:
			t.Fatal("enable completion preceded its terminal command")
		default:
		}
		unblock()
		awaitDrive(t, "disable terminal completion", done)
		awaitDrive(t, "enable terminal completion", enabled)
		if !c.cast().Capabilities().Project(webcast.CapabilityDrive).Enabled {
			t.Fatal("accepted enable did not complete")
		}
	})
	t.Run("completion precedes late reset presentation", func(t *testing.T) {
		c := newDriveCast(t)
		cast := c.cast()
		v := c.viewer(t, "Ana")
		if err := cast.Drive().Grant(v); err != nil {
			t.Fatal(err)
		}
		c.waitDrive(t, "grant projected before disable", func() bool {
			c.con.pnotes.mu.Lock()
			defer c.con.pnotes.mu.Unlock()
			return c.con.pnotes.drive.control.Driving() == v
		})
		awaitDrive(t, "disable completed before late presentation", c.con.onWebCapabilityClick(webcast.CapabilityDrive, false))
		c.waitDrive(t, "real reset projected before replay", func() bool {
			c.con.pnotes.mu.Lock()
			defer c.con.pnotes.mu.Unlock()
			return c.con.pnotes.drive.control.Driver == nil
		})
		// Replay the valid grant/reset delivery sequence after capability
		// completion, on the real terminal queue. This models its subscriber
		// lagging behind the independent capability-completion worker.
		if err := c.con.runTerminalCommand(c.con.lifetime, func() error {
			c.con.applyDrive(cast, recording.Control{Driver: &ana})
			c.con.applyDrive(cast, recording.Control{})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if got := c.notice(); got != "Control of the cast tab is back with you." {
			t.Fatalf("late reset did not replace completion notice: %q", got)
		}
		c.reenableDrive(t)
		if !cast.Capabilities().Project(webcast.CapabilityDrive).Enabled {
			t.Fatal("late reset presentation prevented re-enable")
		}
	})

}

// clickRow opens the drive menu (if closed) and clicks the row whose text
// starts with prefix.
func (c *driveCast) clickRow(t *testing.T, span ColumnSpan, prefix string, button int) {
	t.Helper()
	// Couch has the controller's state (its delivery can lag a change).
	c.waitDrive(t, "Couch caught up with control", func() bool {
		want := c.control() // takes c.mu: never under pnotes.mu, a leaf lock
		c.con.pnotes.mu.Lock()
		defer c.con.pnotes.mu.Unlock()
		return c.con.pnotes.drive.control.Equal(want)
	})
	if open, m, _ := c.notesMenu(); !open || !m.Open() {
		// The caller's span can predate control delivery (a driver's name moves
		// the asks target). Refresh it after the committed projection catches up.
		span = c.span(t, "")
		c.clickStatus(span.Start + 1)
	}
	// The menu on screen is the one the mouse hits: wait until the frame
	// drawn shows the rows delivered (a change redraws it), then click.
	var m NotesMenu
	y := -1
	c.waitDrive(t, "the drive menu drawn with row "+prefix, func() bool {
		c.con.pnotes.mu.Lock()
		defer c.con.pnotes.mu.Unlock()
		m = c.con.pnotes.menu
		items := c.con.pnotes.drive.items
		if !c.con.pnotes.menuOpen || !m.Open() || len(m.items) != len(items) {
			return false
		}
		for i, it := range items {
			if strings.HasPrefix(it.Text, prefix) && m.items[i].ID == it.ID {
				y = m.items[i].Top
				return true
			}
		}
		return false
	})
	c.mouse(button, m.X+8, y, false)
	c.mouse(button, m.X+8, y, true)
}

// Remote driving from Couch, through the real cast (#467): every gesture
// on the span and the drive menu ends in the control state it names; the
// operator takes control back from every state, with the menu open or
// closed; Couch says when someone asks and what a grant turns off.
func TestDriveFromCouch(t *testing.T) {
	c := newDriveCast(t)
	// Two links (operator, 10-12): right-click on the cell copies the view
	// link, a right-click on 👽 the drive link (pair#495's row rule); only
	// the drive link takes a seat.
	if c.link == c.viewLink || !strings.HasSuffix(c.link, "/") {
		t.Fatalf("links: view %q drive %q", c.viewLink, c.link)
	}
	c.clickCell(t, 2)
	waitFor(t, "view-only link copied", func() bool { return copies(c.consoleFixture, c.viewLink) == 1 })
	c.con.mu.Lock()
	alien := c.con.statusWebDriveLink.Start + 1
	c.con.mu.Unlock()
	_, _ = fmt.Fprintf(c.stdin, "\x1b[<2;%d;24M\x1b[<2;%d;24m", alien, alien)
	waitFor(t, "drive link copied", func() bool { return copies(c.consoleFixture, c.link) == 1 })
	waitFor(t, "the drive link's notice", func() bool { return strings.Contains(c.notice(), "drive link copied") })
	if strings.Contains(c.notice(), c.link) {
		t.Fatalf("the notice shows the link: %q", c.notice())
	}
	resp, err := castClient.Post(c.viewLink+"drive/hello", "application/json", strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("a seat by the view-only link: %d", resp.StatusCode)
	}
	v1 := c.viewer(t, "Ana")
	span := c.span(t, "✋Ana")
	waitFor(t, "the request said", func() bool { return strings.Contains(c.notice(), "Ana asks to drive") })

	// A left click on Ana's row grants.
	c.clickRow(t, span, "Ana asks", 0)
	waitFor(t, "Ana drives", func() bool { return c.control().Driving() == v1 })
	waitFor(t, "the grant's notice", func() bool { return strings.Contains(c.notice(), "Uploads and downloads are off") })
	waitFor(t, "Chrome told", func() bool { return c.d.Driving() == "Ana" })
	waitFor(t, "driver chip named", func() bool { return strings.Contains(c.lastRow(), "👽 Ana") })

	// Bo asks; a right click on Bo's row says no.
	v2 := c.viewer(t, "Bo")
	span = c.span(t, "✋Bo")
	c.clickRow(t, span, "Bo asks", 2)
	waitFor(t, "Bo turned down", func() bool { return len(c.control().Pending) == 0 })
	waitFor(t, "the deny said", func() bool { return strings.Contains(c.notice(), "Said no to Bo.") })
	if c.control().Driving() != v1 {
		t.Fatal("saying no to Bo moved control")
	}

	// Clicking the named capability control disables drive and takes control back.
	c.con.mu.Lock()
	driverSpan := c.con.statusWebDriveLink
	c.con.mu.Unlock()
	c.mouse(0, driverSpan.End-1, 23, false)
	c.mouse(0, driverSpan.End-1, 23, true)
	waitFor(t, "control back", func() bool { return c.control().Driver == nil })
	waitFor(t, "Chrome told nobody drives", func() bool { return c.d.Driving() == "" })
	c.reenableDrive(t)
	waitFor(t, "drive enabled", func() bool { return c.cast().Capabilities().Project("drive").Enabled })

	// Ctrl+click on the span takes control back with the menu closed, and
	// with someone else asking.
	v1 = c.viewer(t, "Ana")
	span = c.span(t, "✋Ana")
	if err := c.cast().Drive().Grant(v1); err != nil {
		t.Fatal(err)
	}
	v2 = c.viewer(t, "Bo")
	span = c.span(t, "✋Bo")
	if open, _, _ := c.notesMenu(); open {
		c.clickStatus(span.Start + 1) // close it
		waitFor(t, "menu closed", func() bool { open, _, _ := c.notesMenu(); return !open })
	}
	c.mouse(16, span.Start, 23, false) // Ctrl+left
	c.mouse(16, span.Start, 23, true)
	waitFor(t, "revoked by Ctrl+click", func() bool { return c.control().Driver == nil })
	if p := c.control().Pending; len(p) != 1 || p[0].ID != v2 {
		t.Fatalf("Bo's request survives a revoke: %+v", p)
	}
	// With only an asker, Ctrl+click on the span changes nothing and says
	// nothing went wrong.
	before := c.notice()
	c.mouse(16, c.span(t, "✋Bo").Start, 23, false)
	c.mouse(16, c.span(t, "✋Bo").Start, 23, true)
	time.Sleep(50 * time.Millisecond)
	if p := c.control().Pending; c.control().Driver != nil || len(p) != 1 || strings.Contains(c.notice(), "Remote driving") && c.notice() != before {
		t.Fatalf("Ctrl+click with only an asker: %+v, notice %q", c.control(), c.notice())
	}
	// Bo drives; with the menu open, Ctrl+right on the span revokes.
	c.clickRow(t, c.span(t, "✋Bo"), "Bo asks", 0)
	waitFor(t, "Bo drives", func() bool { return c.control().Driving() == v2 })
	c.viewer(t, "Cy")
	if open, _, _ := c.notesMenu(); !open {
		c.clickStatus(c.span(t, "✋Cy").Start + 1)
	}
	waitFor(t, "menu open", func() bool { open, _, _ := c.notesMenu(); return open })
	sp := c.span(t, "✋Cy")
	c.mouse(18, sp.Start, 23, false) // Ctrl+right
	c.mouse(18, sp.Start, 23, true)
	waitFor(t, "revoked with the menu open", func() bool { return c.control().Driver == nil })
	c.clickRow(t, c.span(t, "✋Cy"), "Cy asks", 2)
	waitFor(t, "Cy declined", func() bool { return len(c.control().Pending) == 0 })

	// A revoke that fails (the recorder stopped) is said as a failure,
	// not as a race, and control stays.
	c.viewer(t, "Ana")
	c.clickRow(t, c.span(t, "✋Ana"), "Ana asks", 0)
	waitFor(t, "Ana drives again", func() bool { return c.control().Driver != nil })
	c.viewer(t, "Bo")
	sp = c.span(t, "✋Bo")
	c.d.FailEmits(errors.New("recorder stopped"))
	c.mouse(16, sp.Start, 23, false)
	c.mouse(16, sp.Start, 23, true)
	waitFor(t, "the failure said", func() bool { return c.cast().Capabilities().Broken() && strings.Contains(c.statusRow(), "Cast broken") })
	if c.control().Driver == nil {
		t.Fatal("a failed revoke changed control")
	}
	c.d.FailEmits(nil)

	// The cast's end clears the span.
	c.clickCell(t, 16)
	waitFor(t, "span gone", func() bool {
		c.con.mu.Lock()
		defer c.con.mu.Unlock()
		return c.con.statusWebDrive == ColumnSpan{} && c.con.wcast.phase == webcastOff
	})
}

// Every class of control failure says what it is (the operator's side of
// drive.Answer's table).
func TestDriveFailure(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{drive.ErrNoSeat, "Remote driving: that changed meanwhile."},
		{fmt.Errorf("x: %w", drive.ErrControlOp), "Remote driving: that changed meanwhile."},
		{drive.ErrClosed, "Remote driving: the cast has ended."},
		{drive.ErrNotLive, "Remote driving: go live first; control changes once the cast is recording."},
		{errors.New("disk full"), "Remote driving failed, control is unchanged: disk full"},
	} {
		if got := driveFailure(tc.err); got != tc.want {
			t.Errorf("%v: %q, want %q", tc.err, got, tc.want)
		}
	}
}

// driveACKBrowser holds the real controller at its acknowledged writer seam.
type driveACKBrowser struct {
	*webcast.FakeDriven
	armed            atomic.Bool
	calls            atomic.Int32
	entered, release chan struct{}
	canceled, settle chan struct{}
	releaseGate      bool
}

func (b *driveACKBrowser) wait(ctx context.Context) error {
	if b.calls.Add(1) == 1 {
		close(b.entered)
	}
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		if b.canceled != nil {
			close(b.canceled)
			<-b.settle
		}
		return ctx.Err()
	}
}
func (b *driveACKBrowser) WriteControl(ctx context.Context, ctl *recording.Ctl) error {
	if b.armed.Load() && !b.releaseGate {
		if err := b.wait(ctx); err != nil {
			return err
		}
	}
	return b.FakeDriven.WriteControl(ctx, ctl)
}
func (b *driveACKBrowser) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	if b.armed.Load() && b.releaseGate && by == "" {
		if err := b.wait(ctx); err != nil {
			return err
		}
	}
	return b.FakeDriven.WithTab(ctx, by, fn)
}
func TestDriveActionsDoNotBlockTerminal(t *testing.T) {
	for _, action := range []string{"grant", "deny", "popup-revoke", "ctrl-revoke", "popup-release", "ctrl-release"} {
		t.Run(action, func(t *testing.T) {
			var b *driveACKBrowser
			r := newDriveCast(t, func(d *webcast.FakeDriven) webcast.Browser {
				b = &driveACKBrowser{FakeDriven: d, entered: make(chan struct{}), release: make(chan struct{}), releaseGate: strings.HasSuffix(action, "release")}
				return b
			})
			defer close(b.release)
			who := r.viewer(t, "Ana")
			if action != "grant" && action != "deny" {
				if err := r.cast().Drive().Grant(who); err != nil {
					t.Fatal(err)
				}
				r.viewer(t, "Bo")
			}
			b.armed.Store(true)
			if strings.HasPrefix(action, "ctrl-") {
				span := r.span(t, "✋Bo")
				r.mouse(16, span.Start, 23, false)
				r.mouse(16, span.Start, 23, true)
			} else {
				prefix, button := "Ana asks", 0
				if action == "deny" {
					button = 2
				}
				if strings.HasPrefix(action, "popup-") {
					prefix = "Ana is driving"
				}
				r.clickRow(t, r.span(t, map[bool]string{true: "✋Bo", false: "✋Ana"}[strings.HasPrefix(action, "popup-")]), prefix, button)
			}
			select {
			case <-b.entered:
			case <-time.After(time.Second):
				t.Fatal("action did not reach writer")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			if err := r.con.runTerminalCommand(ctx, func() error { return nil }); err != nil {
				t.Fatalf("terminal blocked awaiting %s ACK: %v", action, err)
			}
			// Unrelated keyboard input is also serviced while ACK is held.
			r.con.runTerminalCommand(ctx, func() error {
				r.con.pnotes.mu.Lock()
				r.con.pnotes.menuOpen = false
				r.con.pnotes.mu.Unlock()
				return nil
			})
			fmt.Fprint(r.stdin, "z")
			r.waitDrive(t, "typing while ACK held", func() bool { return bytes.Contains(bytes.Join(r.child.Writes(), nil), []byte("z")) })
			// Further clicks must not allocate another acknowledged operation.
			r.con.mu.Lock()
			pending := r.con.drivePending
			r.con.mu.Unlock()
			for i := 0; i < 10; i++ {
				r.con.revokeDrive()
			}
			if b.calls.Load() != 1 {
				t.Fatalf("unbounded operations: %d", b.calls.Load())
			}
			r.con.mu.Lock()
			same := pending != nil && r.con.drivePending == pending
			r.con.mu.Unlock()
			if !same || !strings.Contains(r.notice(), "wait for the current action") {
				t.Fatal("repeat click replaced the operation slot")
			}

		})
	}
}

func closeDriveBarrier(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}
func TestDriveActionStaleCompletion(t *testing.T) {
	for _, mode := range []string{"generation", "replacement"} {
		t.Run(mode, func(t *testing.T) {
			var b *driveACKBrowser
			r := newDriveCast(t, func(d *webcast.FakeDriven) webcast.Browser {
				b = &driveACKBrowser{FakeDriven: d, entered: make(chan struct{}), release: make(chan struct{})}
				return b
			})
			cast := r.cast()
			t.Cleanup(func() { closeDriveBarrier(b.release); r.con.mu.Lock(); r.con.wcast.cast = cast; r.con.mu.Unlock() })
			who := r.viewer(t, "Ana")
			b.armed.Store(true)
			r.con.startDriveOperation(cast, recording.CtlDeny, recording.Author{ID: who, Name: "Ana"})
			awaitDrive(t, "writer entered", b.entered)
			entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			t.Cleanup(func() { closeDriveBarrier(release) })
			go func() {
				r.con.runTerminalCommand(r.con.lifetime, func() error { close(entered); <-release; return nil })
				close(finished)
			}()
			awaitDrive(t, "terminal barrier entered", entered)
			// Hold the terminal callback after dispatch settles, then change its owner.
			close(b.release)
			r.waitDrive(t, "deny acknowledged", func() bool { return len(cast.Drive().Control().Pending) == 0 })
			r.con.mu.Lock()
			pending := r.con.drivePending
			r.con.mu.Unlock()
			if mode == "generation" {
				if err := cast.SetCapability(context.Background(), webcast.CapabilityDrive, false); err != nil {
					t.Fatal(err)
				}
				if err := cast.SetCapability(context.Background(), webcast.CapabilityDrive, true); err != nil {
					t.Fatal(err)
				}
			} else {
				r.con.mu.Lock()
				r.con.wcast.cast = nil
				r.con.mu.Unlock()
				r.con.forgetDrive()
				r.con.mu.Lock()
				if r.con.drivePending != pending {
					t.Error("forget released live operation slot")
				}
				r.con.mu.Unlock()
			}
			r.con.setNotice("newer state")
			close(release)
			awaitDrive(t, "terminal barrier finished", finished)
			r.waitDrive(t, "old worker finished", func() bool { r.con.mu.Lock(); defer r.con.mu.Unlock(); return r.con.drivePending == nil })
			if strings.Contains(r.notice(), "Said no") {
				t.Fatal("stale completion published success")
			}
			// Restore the fixture owner only after its old completion has been fenced.
			if mode == "replacement" {
				r.con.mu.Lock()
				r.con.wcast.cast = cast
				r.con.mu.Unlock()
			}
		})
	}
}

func TestDriveActionShutdownJoinsAndRejectsClicks(t *testing.T) {
	var b *driveACKBrowser
	r := newDriveCast(t, func(d *webcast.FakeDriven) webcast.Browser {
		b = &driveACKBrowser{FakeDriven: d, entered: make(chan struct{}), release: make(chan struct{}), canceled: make(chan struct{}), settle: make(chan struct{})}
		return b
	})
	t.Cleanup(func() { closeDriveBarrier(b.release); closeDriveBarrier(b.settle) })
	who := r.viewer(t, "Ana")
	cast := r.cast()
	b.armed.Store(true)
	r.con.startDriveOperation(cast, recording.CtlGrant, recording.Author{ID: who})
	awaitDrive(t, "writer entered", b.entered)
	// Keep HTTP transport teardown separate from the ACK-join assertion.
	castClient.CloseIdleConnections()
	r.con.Stop()
	select {
	case <-b.canceled:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel writer")
	}
	select {
	case <-r.done:
		t.Fatal("shutdown did not join ACK worker")
	default:
	}
	r.con.startDriveOperation(cast, recording.CtlGrant, recording.Author{ID: who})
	if b.calls.Load() != 1 {
		t.Fatal("shutdown admitted another click")
	}
	close(b.settle)
	select {
	case <-r.done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish after writer settled")
	}
	r.con.startDriveOperation(cast, recording.CtlGrant, recording.Author{ID: who})
	if b.calls.Load() != 1 {
		t.Fatal("stopped GoTracked dispatched inline")
	}
}

// Control ACK precedes the console's delivery and paint. A helper must refresh
// a coordinate captured before a grant adds the driver's name to the chip.
func TestDriveMenuClickRefreshesPreGrantCoordinate(t *testing.T) {
	r := newDriveCast(t)
	ana := r.viewer(t, "Ana")
	old := r.span(t, "✋Ana")
	if err := r.cast().Drive().Grant(ana); err != nil {
		t.Fatal(err)
	}
	r.viewer(t, "Bo")
	current := r.span(t, "✋Bo")
	if old.Start == current.Start {
		t.Fatal("fixture did not move drive target")
	}
	r.clickRow(t, old, "Ana is driving", 0)
	r.waitDrive(t, "revoked through refreshed coordinate", func() bool { return r.control().Driver == nil })
}
