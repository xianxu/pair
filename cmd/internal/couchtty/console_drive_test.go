package couchtty

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast"
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
		{"a driver", recording.Control{Driver: &ana}, "🕹Ana", []string{"Ana is driving: uploads and downloads off while Ana drives. Click to take control back."}},
		{"a driver and an asker", recording.Control{Driver: &ana, Pending: []recording.Author{bo}}, "🕹Ana ✋Bo", []string{"Ana is driving", "Bo asks"}},
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
	row := RenderStatusRow(80, StatusModel{Webcast: WebcastLive, WebNotes: 2, WebDrive: "🕹Ana ✋Bo"})
	if row.WebDrive == (ColumnSpan{}) || row.WebDrive.Start <= row.WebNotes.End || row.WebNotes.Start <= row.Webcast.End {
		t.Fatalf("spans: cell %+v notes %+v drive %+v", row.Webcast, row.WebNotes, row.WebDrive)
	}
	if !strings.Contains(row.Body, "🕹Ana ✋Bo") {
		t.Fatalf("body %q", row.Body)
	}
	if row := RenderStatusRow(80, StatusModel{Webcast: WebcastLive, WebDrive: "🕹A\x1b[2Jna"}); strings.Contains(row.Body, "\x1b[2J") {
		t.Fatalf("a name reached the terminal raw: %q", row.Body)
	}
	if row := RenderStatusRow(80, StatusModel{WebDrive: "🕹Ana"}); row.WebDrive != (ColumnSpan{}) {
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

func newDriveCast(t *testing.T) *driveCast {
	t.Helper()
	var d *webcast.FakeDriven
	r := newWebcastRig(t, func(r *webcastRig) {
		r.wrap = func(f *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
			var err error
			if d, err = webcast.NewFakeDriven(f, drive.DrivenPage(100), &recording.MemSink{}); err != nil {
				t.Fatal(err)
			}
			return d
		}
	})
	r.child.Feed([]byte("\x1b[?1003;1006h"))
	waitFor(t, "child tracks the mouse", func() bool { return r.child.Endpoint().Modes().Tracking != 0 })
	r.open(t)
	d.Open()
	c := &driveCast{webcastRig: r, d: d}
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

func (c *driveCast) span(t *testing.T, text string) ColumnSpan {
	t.Helper()
	var span ColumnSpan
	waitFor(t, "the drive span "+text, func() bool {
		c.con.mu.Lock()
		span = c.con.statusWebDrive
		c.con.mu.Unlock()
		return span != ColumnSpan{} && strings.Contains(c.lastRow(), text)
	})
	return span
}

// clickRow opens the drive menu (if closed) and clicks the row whose text
// starts with prefix.
func (c *driveCast) clickRow(t *testing.T, span ColumnSpan, prefix string, button int) {
	t.Helper()
	// Couch has the controller's state (its delivery can lag a change).
	waitFor(t, "Couch caught up with control", func() bool {
		want := c.control() // takes c.mu: never under pnotes.mu, a leaf lock
		c.con.pnotes.mu.Lock()
		defer c.con.pnotes.mu.Unlock()
		return c.con.pnotes.drive.control.Equal(want)
	})
	if open, m, _ := c.notesMenu(); !open || !m.Open() {
		c.clickStatus(span.Start + 1)
	}
	// The menu on screen is the one the mouse hits: wait until the frame
	// drawn shows the rows delivered (a change redraws it), then click.
	var m NotesMenu
	y := -1
	waitFor(t, "the drive menu drawn with row "+prefix, func() bool {
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
	// Two links (operator, 10-12): right-click copies the view link,
	// Alt+right-click the drive link; only the drive link takes a seat.
	if c.link == c.viewLink || !strings.HasSuffix(c.link, "/") {
		t.Fatalf("links: view %q drive %q", c.viewLink, c.link)
	}
	c.clickCell(t, 2)
	waitFor(t, "view link copied", func() bool { return copies(c.consoleFixture, c.viewLink) == 1 })
	c.clickCell(t, 2|8)
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
		t.Fatalf("a seat by the view link: %d", resp.StatusCode)
	}
	v1 := c.viewer(t, "Ana")
	span := c.span(t, "✋Ana")
	waitFor(t, "the request said", func() bool { return strings.Contains(c.notice(), "Ana asks to drive") })

	// A left click on Ana's row grants.
	c.clickRow(t, span, "Ana asks", 0)
	waitFor(t, "Ana drives", func() bool { return c.control().Driving() == v1 })
	waitFor(t, "the grant's notice", func() bool { return strings.Contains(c.notice(), "Uploads and downloads are off") })
	waitFor(t, "Chrome told", func() bool { return c.d.Driving() == "Ana" })
	span = c.span(t, "🕹Ana")

	// Bo asks; a right click on Bo's row says no.
	v2 := c.viewer(t, "Bo")
	span = c.span(t, "🕹Ana ✋Bo")
	c.clickRow(t, span, "Bo asks", 2)
	waitFor(t, "Bo turned down", func() bool { return len(c.control().Pending) == 0 })
	waitFor(t, "the deny said", func() bool { return strings.Contains(c.notice(), "Said no to Bo.") })
	if c.control().Driving() != v1 {
		t.Fatal("saying no to Bo moved control")
	}

	// A left click on the driver's row takes control back.
	c.clickRow(t, c.span(t, "🕹Ana"), "Ana is driving", 0)
	waitFor(t, "control back", func() bool { return c.control().Driver == nil })
	waitFor(t, "back notice", func() bool { return strings.Contains(c.notice(), "back with you") })
	waitFor(t, "Chrome told nobody drives", func() bool { return c.d.Driving() == "" })

	// Ctrl+click on the span takes control back with the menu closed, and
	// with someone else asking.
	v1 = c.viewer(t, "Ana")
	span = c.span(t, "✋Ana")
	if err := c.cast().Drive().Grant(v1); err != nil {
		t.Fatal(err)
	}
	v2 = c.viewer(t, "Bo")
	span = c.span(t, "🕹Ana ✋Bo")
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
	if open, _, _ := c.notesMenu(); !open { // it stays open on Bo's driver row
		c.clickStatus(c.span(t, "🕹Bo").Start + 1)
	}
	waitFor(t, "menu open", func() bool { open, _, _ := c.notesMenu(); return open })
	sp := c.span(t, "🕹Bo")
	c.mouse(18, sp.Start, 23, false) // Ctrl+right
	c.mouse(18, sp.Start, 23, true)
	waitFor(t, "revoked with the menu open", func() bool { return c.control().Driver == nil })

	// A revoke that fails (the recorder stopped) is said as a failure,
	// not as a race, and control stays.
	c.viewer(t, "Ana")
	c.clickRow(t, c.span(t, "✋Ana"), "Ana asks", 0)
	waitFor(t, "Ana drives again", func() bool { return c.control().Driver != nil })
	c.d.FailEmits(errors.New("recorder stopped"))
	sp = c.span(t, "🕹Ana")
	c.mouse(16, sp.Start, 23, false)
	c.mouse(16, sp.Start, 23, true)
	waitFor(t, "the failure said", func() bool { return strings.Contains(c.notice(), "control is unchanged: recorder stopped") })
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
