package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// driveFixture: a Server whose controller drives a fake injector over the
// contract's page (live, nobody driving), both following one stream.
type driveFixture struct {
	srv    *Server
	ctrl   *drive.MemController
	inj    *drive.FakeInjector
	stream *recording.MemSink
}

func newDriveFixture(t *testing.T, grace time.Duration) *driveFixture {
	t.Helper()
	stream := &recording.MemSink{}
	page := drive.DrivenPage(100)
	for _, r := range page {
		stream.Put(r)
	}
	inj, err := drive.NewFakeInjector(page, stream)
	if err != nil {
		t.Fatal(err)
	}
	clock := func() int64 { return inj.Page().T + 1 }
	ctrl := drive.NewMemController(recording.Tee(stream, inj), inj, clock, grace)
	t.Cleanup(func() { ctrl.Close() })
	return &driveFixture{srv: NewServer("tok", NewFakeBrowser(), time.Second, nil, nil).WithDrive(ctrl, "drv"), ctrl: ctrl, inj: inj, stream: stream}
}

// post sends one drive request straight to the server.
func (f *driveFixture) post(route, key, body string) (int, string) {
	r := httptest.NewRequest(http.MethodPost, "/drv/drive/"+route, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if key != "" {
		r.Header.Set(drive.SeatHeader, key)
	}
	w := httptest.NewRecorder()
	f.srv.ServeHTTP(w, r)
	return w.Code, strings.TrimSpace(w.Body.String())
}

func (f *driveFixture) hello(t *testing.T, name string) (viewer, key string) {
	t.Helper()
	code, body := f.post("hello", "", `{"name":"`+name+`"}`)
	var seat struct{ Viewer, Key, Name string }
	if code != http.StatusOK || json.Unmarshal([]byte(body), &seat) != nil || seat.Name != name || len(seat.Key) != 64 {
		t.Fatalf("hello: %d %q", code, body)
	}
	return seat.Viewer, seat.Key
}

func (f *driveFixture) click(id string) string {
	s := f.inj.Page()
	el := drive.ElementByID(s, id)
	box := s.Boxes[el.ID]
	b, _ := json.Marshal(drive.Batch{Seg: s.Seg, Acts: []drive.Act{{A: drive.ActClick, ID: el.ID, X: box[0] + 1, Y: box[1] + 1}}})
	return string(b)
}

// Every answer drive.Answer gives, through the real routes: status and
// sentence exactly as the viewer's shell will show them.
func TestDriveRoutesAnswer(t *testing.T) {
	f := newDriveFixture(t, time.Minute)
	ana, key := f.hello(t, "Ana")
	_, boKey := f.hello(t, "Bo")
	want := func(what string, code int, body string, status int, err error) {
		t.Helper()
		_, say := drive.Answer(err)
		if code != status || body != say {
			t.Errorf("%s: %d %q, want %d %q", what, code, body, status, say)
		}
	}
	code, body := f.post("input", "", f.click("q"))
	want("no seat key", code, body, 401, drive.ErrNoSeat)
	code, body = f.post("input", strings.Repeat("0", 64), f.click("q"))
	want("an unknown key", code, body, 401, drive.ErrNoSeat)
	code, body = f.post("input", key, f.click("q"))
	want("input before a grant", code, body, 403, drive.ErrNotDriver)
	code, body = f.post("control", key, `{"op":"request"}`)
	want("request", code, body, 204, nil)
	code, body = f.post("control", key, `{"op":"request"}`)
	want("asking twice", code, body, 409, drive.ErrControlOp)
	code, body = f.post("control", key, `{"op":"grant"}`)
	want("a viewer granting", code, body, 400, drive.ErrBatchShape)
	if err := f.ctrl.Grant(ana); err != nil {
		t.Fatal(err)
	}
	code, body = f.post("input", key, f.click("q"))
	want("the driver's click", code, body, 204, nil)
	var clicked bool
	for _, r := range f.stream.Records() {
		if c, ok := r.(*recording.Click); ok && c.By == ana {
			clicked = true
		}
	}
	if !clicked {
		t.Error("the driver's click isn't in the stream with by")
	}
	code, body = f.post("input", boKey, f.click("q"))
	want("another viewer's input", code, body, 403, drive.ErrNotDriver)
	stale := strings.Replace(f.click("q"), `"id":`+jsonID(f, "q"), `"id":`+jsonID(f, "save"), 1)
	code, body = f.post("input", key, stale)
	want("a stale target", code, body, 409, drive.ErrStale)
	paste, _ := json.Marshal(drive.Batch{Seg: f.inj.Page().Seg, Acts: []drive.Act{{A: drive.ActKey, Type: drive.KeyDown, Key: "v", Mods: []string{drive.ModMeta}}}})
	code, body = f.post("input", key, string(paste))
	want("paste", code, body, 422, drive.ErrRefused)
	gone, _ := json.Marshal(drive.Batch{Seg: f.inj.Page().Seg + 1, Acts: []drive.Act{{A: drive.ActText, V: "x"}}})
	code, body = f.post("input", key, string(gone))
	want("a gone segment", code, body, 409, drive.ErrNotLive)
	code, body = f.post("input", key, `{"seg":1,"acts":[],"x":1}`)
	want("malformed", code, body, 400, drive.ErrBatchShape)
	code, body = f.post("input", key, `{"seg":1,"acts":[{"a":"text","v":"`+strings.Repeat("x", drive.MaxBody)+`"}]}`)
	if code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized: %d %q", code, body)
	}
	code, body = f.post("control", key, `{"op":"release"}`)
	want("release", code, body, 204, nil)
	f.ctrl.Close()
	code, body = f.post("hello", "", `{"name":"Cy"}`)
	want("after the cast", code, body, 410, drive.ErrClosed)
}

func jsonID(f *driveFixture, id string) string {
	b, _ := json.Marshal(drive.ElementByID(f.inj.Page(), id).ID)
	return string(b)
}

// Without a controller the routes are not found; a wrong content type is
// refused before anything is read.
func TestDriveRoutesShape(t *testing.T) {
	plain := NewServer("tok", NewFakeBrowser(), time.Second, nil, nil)
	for _, route := range []string{"hello", "control", "input"} {
		r := httptest.NewRequest(http.MethodPost, "/tok/drive/"+route, strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		plain.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s without a controller: %d", route, w.Code)
		}
	}
	f := newDriveFixture(t, time.Minute)
	r := httptest.NewRequest(http.MethodPost, "/drv/drive/hello", strings.NewReader(`{"name":"x"}`))
	w := httptest.NewRecorder()
	f.srv.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("no content type: %d", w.Code)
	}
}

// A seat's input over its budget is refused, and another seat's budget is
// its own.
func TestDriveRoutesRateLimitPerSeat(t *testing.T) {
	f := newDriveFixture(t, time.Minute)
	ana, key := f.hello(t, "Ana")
	_, boKey := f.hello(t, "Bo")
	f.ctrl.Request(key)
	f.ctrl.Grant(ana)
	busy := 0
	for i := 0; i < inputBurst+10; i++ {
		if code, _ := f.post("input", key, `{"seg":1,"acts":[{"a":"move","x":1,"y":1}]}`); code == http.StatusTooManyRequests {
			busy++
		}
	}
	if busy == 0 {
		t.Fatal("no input refused over the seat's budget")
	}
	if code, _ := f.post("control", boKey, `{"op":"request"}`); code != http.StatusNoContent {
		t.Fatalf("another seat was limited: %d", code)
	}
}

// The events stream holds a seat: with it the seat outlives the grace
// period; after it closes, the seat leaves.
func TestDriveSeatFollowsItsStream(t *testing.T) {
	f := newDriveFixture(t, 100*time.Millisecond)
	srv := httptest.NewServer(f.srv)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	viewer, key := f.hello(t, "Ana")
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/drv/events?viewer="+viewer, nil)
	resp, err := castClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	time.Sleep(300 * time.Millisecond) // past the grace period, streaming
	if err := f.ctrl.Request(key); err != nil {
		t.Fatalf("a streaming seat left: %v", err)
	}
	cancel()
	deadline := time.Now().Add(5 * time.Second)
	for f.ctrl.Release(key) != drive.ErrNoSeat {
		if time.Now().After(deadline) {
			t.Fatal("the seat outlived its stream")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if c := f.ctrl.Control(); len(c.Pending) != 0 {
		t.Fatalf("its request outlived it: %+v", c)
	}
}

// Whatever the body, an input route answers with one of drive.Answer's
// sentences (or admission's), never echoes it, and never reaches the page
// unless it is a valid batch.
func FuzzDriveInput(f *testing.F) {
	f.Add(`{"seg":1,"acts":[{"a":"click","id":3,"x":1,"y":1}]}`)
	f.Add(`{"seg":1,"acts":[{"a":"key","type":"down","key":"v","mods":["meta"]}]}`)
	f.Add(`{"seg":1,"acts":[{"a":"text","v":"<script>"}]}`)
	f.Add(`{`)
	f.Fuzz(func(t *testing.T, body string) {
		fx := newDriveFixture(t, time.Minute)
		ana, key := fx.hello(t, "Ana")
		fx.ctrl.Request(key)
		fx.ctrl.Grant(ana)
		n := len(fx.stream.Records())
		code, said := fx.post("input", key, body)
		if code >= 500 {
			t.Fatalf("%d for %q", code, body)
		}
		if len(body) > 8 && strings.Contains(said, body) {
			t.Fatalf("echoed %q", body)
		}
		if _, err := drive.ParseBatch([]byte(body)); err != nil && len(fx.stream.Records()) != n {
			t.Fatalf("an invalid body reached the page: %q", body)
		}
	})
}

// The per-seat table is bounded: a flood of unknown keys never grows it
// past maxSeatLimits (they leave as the controller refuses them), and a
// seat already budgeted keeps working through it.
func TestDriveSeatTableBounded(t *testing.T) {
	f := newDriveFixture(t, time.Minute)
	_, key := f.hello(t, "Ana")
	if code, _ := f.post("control", key, `{"op":"request"}`); code != http.StatusNoContent {
		t.Fatal(code)
	}
	for i := 0; i < 3*maxSeatLimits; i++ {
		f.post("control", strings.Repeat("f", 60)+fmt.Sprintf("%04d", i), `{"op":"request"}`)
	}
	f.srv.drive.mu.Lock()
	n := len(f.srv.drive.seats)
	f.srv.drive.mu.Unlock()
	if n > maxSeatLimits {
		t.Fatalf("the table grew to %d", n)
	}
	if code, _ := f.post("control", key, `{"op":"release"}`); code != http.StatusNoContent {
		t.Fatalf("a budgeted seat after the flood: %d", code)
	}
}

// Once full, the table sweeps keys idle past seatIdle. Until then a new key
// gets a budget that refuses at once, and the table doesn't grow; a key in
// use survives the sweep.
func TestDriveSeatTableSweepsIdleKeys(t *testing.T) {
	f := newDriveFixture(t, time.Minute)
	d := f.srv.drive
	t0 := time.Now()
	at := t0
	d.now = func() time.Time { return at }
	for i := 0; i < maxSeatLimits; i++ {
		d.limits(fmt.Sprintf("old%d", i))
	}
	at = t0.Add(30 * time.Second)
	refused := d.limits("hot")
	if d.seats["hot"] != nil || len(d.seats) != maxSeatLimits {
		t.Fatalf("a full table grew to %d", len(d.seats))
	}
	select {
	case refused.inFlight <- struct{}{}:
		t.Fatal("an unbudgeted key has a slot")
	default:
	}
	d.limits("old0") // in use at 30 s
	at = t0.Add(seatIdle + time.Second)
	d.limits("new")
	if d.seats["new"] == nil || d.seats["old1"] != nil || d.seats["old0"] == nil {
		t.Fatalf("after the sweep: new %v, idle old1 %v, used old0 %v", d.seats["new"] != nil, d.seats["old1"] != nil, d.seats["old0"] != nil)
	}
}

// Two links (operator, 10-12): the view link watches, points and notes;
// only the drive link takes a seat, control and input, and only its stream
// holds a seat.
func TestDriveOnlyUnderTheDriveLink(t *testing.T) {
	f := newDriveFixture(t, 100*time.Millisecond)
	if !f.srv.Owns("drv") || !f.srv.Owns("tok") || f.srv.Owns("nope") || f.srv.DriveToken() != "drv" {
		t.Fatal("the cast answers both links, and nothing else")
	}
	for _, route := range []string{"hello", "control", "input"} {
		r := httptest.NewRequest(http.MethodPost, "/tok/drive/"+route, strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		f.srv.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s under the view link: %d", route, w.Code)
		}
	}
	srv := httptest.NewServer(f.srv)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	viewer, key := f.hello(t, "Ana")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/tok/events?viewer="+viewer, nil)
	resp, err := castClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("watching by the view link: %v %v", resp, err)
	}
	defer resp.Body.Close()
	time.Sleep(300 * time.Millisecond) // past the grace: a view stream holds no seat
	if view, drive := f.srv.Viewers(); view != 1 || drive != 0 {
		t.Fatalf("viewers by link: view %d drive %d", view, drive)
	}
	if err := f.ctrl.Request(key); err != drive.ErrNoSeat {
		t.Fatalf("a seat held by a view-link stream: %v", err)
	}
}
