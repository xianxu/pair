package recording

import (
	"errors"
	"strings"
	"testing"
)

var (
	ana = Author{ID: "v1", Name: "Ana"}
	bo  = Author{ID: "v2", Name: "Bo"}
)

func TestNextControl(t *testing.T) {
	none := Control{}
	asked := Control{Pending: []Author{ana}}
	drives := Control{Driver: &ana}
	other := Control{Driver: &bo, Pending: []Author{ana}}
	cases := []struct {
		name string
		from Control
		op   string
		want *Control // nil: refused
	}{
		{"request", none, CtlRequest, &asked},
		{"request twice", asked, CtlRequest, nil},
		{"request while driving", drives, CtlRequest, nil},
		{"grant", asked, CtlGrant, &drives},
		{"grant unasked", none, CtlGrant, nil},
		{"grant transfers", other, CtlGrant, &drives},
		{"deny", asked, CtlDeny, &none},
		{"deny unasked", none, CtlDeny, nil},
		{"revoke", drives, CtlRevoke, &none},
		{"revoke someone else", other, CtlRevoke, nil},
		{"release driving", drives, CtlRelease, &none},
		{"release withdraws a request", other, CtlRelease, &Control{Driver: &bo}},
		{"release uninvolved", none, CtlRelease, nil},
		{"leave driving", drives, CtlLeave, &none},
		{"leave pending", asked, CtlLeave, &none},
		{"leave uninvolved", none, CtlLeave, nil},
		{"unknown op", asked, "steal", nil},
	}
	for _, c := range cases {
		got, err := NextControl(c.from, c.op, ana)
		switch {
		case c.want == nil && err == nil:
			t.Errorf("%s: allowed, got %+v", c.name, got)
		case c.want == nil && !errors.Is(err, ErrControl):
			t.Errorf("%s: error %v isn't ErrControl", c.name, err)
		case c.want != nil && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want != nil && !got.Equal(*c.want):
			t.Errorf("%s: got driver %q pending %v, want %q %v", c.name, got.Driving(), ids(got.Pending), c.want.Driving(), ids(c.want.Pending))
		}
	}
	// It never changes its input.
	from := Control{Pending: []Author{bo, ana}}
	NextControl(from, CtlGrant, ana)
	if len(from.Pending) != 2 || from.Pending[1].ID != "v1" {
		t.Errorf("NextControl changed its input: %v", ids(from.Pending))
	}
}

// ctl is a ctl record stating what the op gives from c.
func ctl(t *testing.T, c *Control, op string, who Author) *Ctl {
	t.Helper()
	next, err := NextControl(*c, op, who)
	if err != nil {
		t.Fatal(err)
	}
	*c = next
	return &Ctl{Op: op, Who: who, Control: next}
}

func TestAttributionNeedsTheDriver(t *testing.T) {
	b := NewBuilder("attr")
	btn := b.El("button", nil, b.Txt("Go"))
	b.Page(SegLive, "http://x/", btn)
	var c Control
	click := func(by string) Record { return &Click{Source: Source{By: by}, ID: btn.ID, X: 1, Y: 1} }

	b.At(10).Add(ctl(t, &c, CtlRequest, ana))
	pending := len(b.Recs)
	b.Add(click("v1")) // asked, not granted
	if _, err := Reduce(b.Recs, 10); err == nil || !strings.Contains(err.Error(), `driver is ""`) {
		t.Fatalf("a click by a pending viewer reduced: %v", err)
	}
	b.Recs = b.Recs[:pending]

	b.At(20).Add(ctl(t, &c, CtlGrant, ana)).Add(click("v1")).Add(click(""))
	s, err := Reduce(b.Recs, 20)
	if err != nil {
		t.Fatal(err)
	}
	if s.LastClick.By != "" || s.Control.Driving() != "v1" {
		t.Fatalf("operator's click after the driver's: by %q, driver %q", s.LastClick.By, s.Control.Driving())
	}

	// Control survives a pause, a resume and a navigation.
	b.At(30).Add(&Stop{Why: StopPause})
	b.At(40).Page(SegNavigate, "http://x/2", b.El("p", nil))
	b.Add(&Focus{Source: Source{By: "v1"}})
	if s, err = Reduce(b.Recs, 40); err != nil || s.Control.Driving() != "v1" {
		t.Fatalf("after pause and navigate: %v, driver %q", err, s.Control.Driving())
	}

	b.At(50).Add(ctl(t, &c, CtlRevoke, ana)).Add(&Focus{Source: Source{By: "v1"}})
	if _, err := Reduce(b.Recs, 50); err == nil {
		t.Fatal("a focus by a revoked driver reduced")
	}
}

func TestCtlMustStateWhatItsOpGives(t *testing.T) {
	b := NewBuilder("ctl")
	b.Page(SegLive, "http://x/")
	var c Control
	b.At(5).Add(ctl(t, &c, CtlRequest, ana))
	// Consistent in itself (bo asked), but it drops ana's request.
	b.Add(&Ctl{Op: CtlRequest, Who: bo, Control: Control{Pending: []Author{bo}}})
	if _, err := Reduce(b.Recs, 5); err == nil || !strings.Contains(err.Error(), "the op gives") {
		t.Fatalf("a ctl stating the wrong state reduced: %v", err)
	}
}

func TestCtlAppliesWhilePausedAndStopEndEndsControl(t *testing.T) {
	b := NewBuilder("ctl")
	b.Page(SegLive, "http://x/")
	var c Control
	b.At(5).Add(&Stop{Why: StopPause})
	b.Add(ctl(t, &c, CtlRequest, ana)).Add(ctl(t, &c, CtlGrant, ana))
	s, err := Reduce(b.Recs, 5)
	if err != nil || s.Control.Driving() != "v1" {
		t.Fatalf("ctl while paused: %v, driver %q", err, s.Control.Driving())
	}
	b.At(6).Add(&Seg{Seg: b.NextSeg(), Why: SegResume, Tab: Tab{ID: "tab1", URL: "http://x/"}, VP: Viewport{W: 1280, H: 800, DPR: 1}})
	b.Snapshot("http://x/", b.El("html", nil))
	b.At(7).Add(&Stop{Why: StopEnd})
	if s, err = Reduce(b.Recs, 7); err != nil || s.Control.Driver != nil {
		t.Fatalf("stop end left a driver: %v %+v", err, s.Control)
	}
	// End is terminal: a late joiner's catch-up can't start a new cast
	// with a control state from before the end.
	b.At(8).Add(&Seg{Seg: b.NextSeg(), Why: SegLive, Tab: Tab{ID: "tab1", URL: "http://x/"}, VP: Viewport{W: 1280, H: 800, DPR: 1}})
	if _, err := Reduce(b.Recs, 8); err == nil || !strings.Contains(err.Error(), "the cast is over") {
		t.Fatalf("a segment after stop end reduced: %v", err)
	}
	b.Recs = b.Recs[:len(b.Recs)-1]
	b.Add(&Ctl{Op: CtlRequest, Who: bo, Control: Control{Pending: []Author{bo}}})
	if _, err := Reduce(b.Recs, 8); err == nil || !strings.Contains(err.Error(), "the cast is over") {
		t.Fatalf("a ctl after stop end reduced: %v", err)
	}
}

func TestCtlRefusesADriverWithoutAnID(t *testing.T) {
	b := NewBuilder("ctl")
	b.Page(SegLive, "http://x/")
	b.At(5).Add(&Ctl{Op: CtlRelease, Who: ana, Control: Control{Driver: &Author{}}})
	if _, err := Reduce(b.Recs, 5); err == nil {
		t.Fatal("a ctl naming a driver without an id reduced")
	}
}

func TestFirstCtlIsABaselineExceptForAWholeRecording(t *testing.T) {
	b := NewBuilder("ctl")
	b.Page(SegLive, "http://x/")
	b.At(5).Add(&Ctl{Op: CtlGrant, Who: ana, Control: Control{Driver: &ana}}) // no request seen
	s, err := Reduce(b.Recs, 5)
	if err != nil || s.Control.Driving() != "v1" {
		t.Fatalf("a late joiner's first ctl: %v, driver %q", err, s.Control.Driving())
	}
	files := map[string][]byte{}
	for _, r := range b.Recs {
		line, _ := Encode(r)
		files[FileEvents] = append(append(files[FileEvents], line...), '\n')
	}
	if fs, _ := Validate(files); len(Errors(fs)) == 0 {
		t.Fatal("Validate took a grant nobody asked for")
	}
	// Even as a baseline, the stated state must fit the op.
	b.Recs[len(b.Recs)-1] = &Ctl{Header: Header{T: 5, K: "ctl"}, Op: CtlGrant, Who: ana, Control: Control{Pending: []Author{ana}}}
	if _, err := Reduce(b.Recs, 5); err == nil {
		t.Fatal("a grant stating its viewer still pending reduced")
	}
}
