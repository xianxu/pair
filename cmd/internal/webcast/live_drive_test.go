package webcast

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// drivePage: a search box, a button, a link to a new tab, a file input, a
// download, and a button that opens an alert.
const drivePage = `<!doctype html><title>drive</title><body style="margin:0;font:16px sans-serif">
<input id=q style="position:absolute;left:20px;top:20px;width:200px;height:30px">
<button id=save style="position:absolute;left:20px;top:70px;width:100px;height:30px" onclick="this.textContent='saved'">save</button>
<a id=pop href="/popped" target=_blank style="position:absolute;left:20px;top:120px;width:100px;height:30px;display:block">pop</a>
<input id=file type=file style="position:absolute;left:20px;top:170px;width:200px;height:30px">
<a id=dl href="/file.bin" style="position:absolute;left:20px;top:220px;width:100px;height:30px;display:block">download</a>
<button id=alert style="position:absolute;left:20px;top:270px;width:100px;height:30px" onclick="alert('hi')">alert</button>
</body>`

func serveDrive(t *testing.T) (string, *atomic.Int32) {
	t.Helper()
	var downloads atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, drivePage) })
	mux.HandleFunc("/popped", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "<title>popped</title>popped") })
	mux.HandleFunc("/file.bin", func(w http.ResponseWriter, _ *http.Request) {
		downloads.Add(1)
		w.Header().Set("Content-Disposition", `attachment; filename="file.bin"`)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte(strings.Repeat("x", 1<<20)))
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &http.Server{Handler: mux}
	go s.Serve(l)
	t.Cleanup(func() { s.Close() })
	return "http://" + l.Addr().String(), &downloads
}

// pagePoint is an element's centre in top-page CSS px.
func (h liveHarness) pagePoint(t *testing.T, sel string) (float64, float64) {
	t.Helper()
	var m map[string]float64
	h.eval(t, fmt.Sprintf(`(() => { const r = document.querySelector(%q).getBoundingClientRect(); return {x: r.left + r.width/2 + scrollX, y: r.top + r.height/2 + scrollY}; })()`, sel), &m)
	return m["x"], m["y"]
}

// TestLiveDrive drives a real headless Chrome through the production pieces
// (MemController behind liveGate, TabInjector over the recorder, Chrome's
// guards) and reads the recording back:
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveDrive -v
func TestLiveDrive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	c := liveRecordingChrome(t, dir, -1)
	h := liveHarness{c}
	base, downloads := serveDrive(t)
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": base + "/"}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	var events sync.Map // method -> count
	c.do(func(s *loopState) {
		s.tap = func(ev cdp.Event) {
			n, _ := events.LoadOrStore(ev.Method, new(atomic.Int32))
			n.(*atomic.Int32).Add(1)
		}
	})
	seen := func(method string) int32 {
		if n, ok := events.Load(method); ok {
			return n.(*atomic.Int32).Load()
		}
		return 0
	}
	// The download's positive control: with downloads allowed into a
	// directory, the operator's click lands a file there.
	downloadDir := t.TempDir()
	if err := c.call("", "Browser.setDownloadBehavior", map[string]any{"behavior": "allow", "downloadPath": downloadDir}, nil); err != nil {
		t.Fatal(err)
	}
	files := func() int { m, _ := filepath.Glob(filepath.Join(downloadDir, "*")); return len(m) }
	{
		x, y := h.center(t, "#dl")
		h.mouse(t, "mousePressed", x, y)
		h.mouse(t, "mouseReleased", x, y)
	}
	for deadline := time.Now().Add(10 * time.Second); files() == 0; time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the positive control: the operator's download never landed")
		}
	}
	ctrl := drive.NewMemController(liveGate{c}, drive.NewTabInjector(c), func() int64 { return 0 }, time.Minute)
	ctrl.WithControlReset(c.ResetControl)
	if err := ctrl.SetDriveAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	defer ctrl.Close()
	go followDriving(ctrl, c)
	seat, err := ctrl.Join("Ana")
	if err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Request(seat.Key); !errors.Is(err, drive.ErrNotLive) {
		t.Fatalf("before go-live: %v", err)
	}
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for !c.Opened() {
		if time.Now().After(deadline) {
			t.Fatal("the recorder never opened")
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	if err := ctrl.Request(seat.Key); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Grant(seat.Viewer.ID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond) // the ctl reaches the recorder; Chrome is told

	// The page as the recording sees it: ids of the elements.
	ids := map[string]recording.NodeID{}
	var seg int
	if err := c.WithTab(context.Background(), "", func(tab drive.Tab) error {
		for _, id := range []string{"q", "save", "pop", "file", "dl", "alert"} {
			if n := drive.ElementByID(tab.Page(), id); n != nil {
				ids[id] = n.ID
			}
		}
		seg = tab.Page().Seg
		return nil
	}); err != nil || len(ids) != 6 {
		t.Fatalf("ids %v: %v", ids, err)
	}
	click := func(id string) drive.Act {
		x, y := h.pagePoint(t, "#"+id)
		return drive.Act{A: drive.ActClick, ID: ids[id], X: x, Y: y, Button: drive.ButtonLeft, Count: 1}
	}
	input := func(acts ...drive.Act) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return ctrl.Input(ctx, seat.Key, drive.Batch{Seg: seg, Acts: acts})
	}

	// Click the box, type, Enter; click save.
	if err := input(click("q"), drive.Act{A: drive.ActText, V: "invoice"}, drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "Enter"}, drive.Act{A: drive.ActKey, Type: drive.KeyUp, Key: "Enter"}); err != nil {
		t.Fatalf("click and type: %v", err)
	}
	if err := input(click("save")); err != nil {
		t.Fatalf("save: %v", err)
	}
	var value, saved string
	h.eval(t, `document.getElementById('q').value`, &value)
	h.eval(t, `document.getElementById('save').textContent`, &saved)
	if value != "invoice" || saved != "saved" {
		t.Fatalf("the page: value %q, save %q", value, saved)
	}
	// Refused and stale, nothing dispatched.
	if err := input(drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "v", Mods: []string{drive.ModMeta}}); !errors.Is(err, drive.ErrRefused) {
		t.Fatalf("paste: %v", err)
	}
	wrong := click("q")
	wrong.ID = ids["save"]
	if err := input(wrong); !errors.Is(err, drive.ErrStale) {
		t.Fatalf("stale: %v", err)
	}
	// A new tab opened by the cast tab is closed; the cast stays.
	if err := input(click("pop")); err != nil {
		t.Fatalf("pop: %v", err)
	}
	time.Sleep(1500 * time.Millisecond)
	var targets struct {
		TargetInfos []struct{ Type, URL string } `json:"targetInfos"`
	}
	if err := c.call("", "Target.getTargets", nil, &targets); err != nil {
		t.Fatal(err)
	}
	for _, ti := range targets.TargetInfos {
		if ti.Type == "page" && strings.HasSuffix(ti.URL, "/popped") {
			t.Fatalf("the opened tab is still there: %+v", targets.TargetInfos)
		}
	}
	if st := c.State(); !strings.HasPrefix(st.Tab.URL, base) || strings.HasSuffix(st.Tab.URL, "/popped") {
		t.Fatalf("the cast followed the new tab: %+v", st.Tab)
	}
	// A file chooser is intercepted (never shown); a download is denied.
	if err := input(click("file")); err != nil {
		t.Fatalf("file: %v", err)
	}
	for deadline := time.Now().Add(5 * time.Second); seen("Page.fileChooserOpened") == 0; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the file chooser wasn't intercepted")
		}
	}
	before := files()
	if err := input(click("dl")); err != nil {
		t.Fatalf("download: %v", err)
	}
	time.Sleep(2 * time.Second)
	if files() != before {
		t.Fatalf("a driver's download landed: %d files, was %d", files(), before)
	}
	// A dialog: the driver is told at once, and the operator answers it.
	start := time.Now()
	if err := input(click("alert")); !errors.Is(err, drive.ErrDialog) {
		t.Fatalf("alert: %v", err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("the dialog held the driver %s", d)
	}
	if err := c.call(h.session(t, ""), "Page.handleJavaScriptDialog", map[string]any{"accept": true}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := input(click("save")); err != nil {
		t.Fatalf("after the dialog: %v", err)
	}
	// Revoked: refused.
	if err := ctrl.Revoke(); err != nil {
		t.Fatal(err)
	}
	if err := input(click("save")); !errors.Is(err, drive.ErrNotDriver) {
		t.Fatalf("after a revoke: %v", err)
	}
	t.Logf("download requests reaching the server: %d (deny cancels the transfer)", downloads.Load())
	// Restored: with nobody driving, the operator's file chooser isn't
	// intercepted.
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		var driving string
		ran := make(chan struct{})
		c.do(func(s *loopState) { driving = s.driver; close(ran) })
		<-ran
		if driving == "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Chrome still thinks someone drives")
		}
	}
	time.Sleep(300 * time.Millisecond) // the intercept-off calls land
	choosers := seen("Page.fileChooserOpened")
	{
		x, y := h.center(t, "#file")
		h.mouse(t, "mousePressed", x, y)
		h.mouse(t, "mouseReleased", x, y)
	}
	time.Sleep(time.Second)
	if seen("Page.fileChooserOpened") != choosers {
		t.Fatal("the file chooser is still intercepted after driving")
	}

	time.Sleep(500 * time.Millisecond)
	c.SetLive(false)
	c.Close()
	<-c.Done()
	findings, _, err := recording.ValidateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("the driven recording: %+v", f)
	}
	r, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var clicks, typed, unattributedAfterGrant int
	driving := false
	for _, rec := range r.Records {
		switch x := rec.(type) {
		case *recording.Ctl:
			driving = x.Driving() == seat.Viewer.ID
		case *recording.Click:
			switch {
			case !driving:
				// the operator's, before the grant or after the revoke
			case x.By == seat.Viewer.ID:
				clicks++
			case x.ID == ids["alert"]:
				// The documented residual: the dialog stopped the page
				// before the take, and the expired hold released the
				// click without by.
			default:
				unattributedAfterGrant++
				t.Logf("a click without by: %+v", x)
			}
		case *recording.Input:
			if x.ID == ids["q"] && x.By == seat.Viewer.ID && x.V != nil && *x.V == "invoice" {
				typed++
			}
		}
	}
	if clicks < 5 || typed == 0 || unattributedAfterGrant != 0 {
		t.Fatalf("attribution: %d driver clicks, %d typed, %d clicks without by", clicks, typed, unattributedAfterGrant)
	}
}
