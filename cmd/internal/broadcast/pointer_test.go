package broadcast

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type pointerSession struct {
	s      *Session
	mu     sync.Mutex
	got    []PointBatch
	offs   int
	onHook func(PointBatch)
	// notes records each OnNotes delivery (#447).
	notes []notesCall
}

type notesCall struct {
	notes   []Note
	version uint64
}

func (p *pointerSession) notesCalls() []notesCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]notesCall(nil), p.notes...)
}

func (p *pointerSession) batches() []PointBatch {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]PointBatch(nil), p.got...)
}

func startPointerSession(t *testing.T) *pointerSession {
	t.Helper()
	return startPointerSessionWith(t, HubOptions{})
}

func startPointerSessionWith(t *testing.T, hub HubOptions) *pointerSession {
	return startPointerSessionAt(t, hub, nil)
}

func startPointerSessionAt(t *testing.T, hub HubOptions, now func() time.Time) *pointerSession {
	t.Helper()
	p := &pointerSession{}
	s, err := Start(context.Background(), Config{Tunnel: &FakeTunnel{}, Ping: 20 * time.Millisecond, Hub: hub, Now: now,
		OnNotes: func(_ *Session, notes []Note, version uint64) {
			p.mu.Lock()
			p.notes = append(p.notes, notesCall{notes, version})
			p.mu.Unlock()
		},
		OnPoints: func(b PointBatch) {
			p.mu.Lock()
			p.got = append(p.got, b)
			hook := p.onHook
			p.mu.Unlock()
			if hook != nil {
				hook(b)
			}
		},
		OnPointerOff: func() { p.mu.Lock(); p.offs++; p.mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(nil); <-s.Done() })
	unlockView(t, s)
	p.s = s
	return p
}

// The operator's screen as Couch draws it while pointing is on: 40x4, LIVE
// and the active pointer marker on the status row.
func (p *pointerSession) showPointer(t *testing.T) {
	t.Helper()
	p.s.Offer(livePointer(t, "operator screen"), terminal.FramePublic)
	p.s.hub.sync()
}

const tapBody = `{"cols":40,"rows":4,"down":false,"points":[[3,1]]}`

func post(t *testing.T, url, contentType, body string) (int, string) {
	t.Helper()
	resp, err := castClient.Post(url, contentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestPointerLinkLifecycle(t *testing.T) {
	p := startPointerSession(t)
	if p.s.PointerLink() != "" {
		t.Fatal("pointer link before EnablePointer")
	}
	link, err := p.enablePointer(t)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Split(strings.TrimSuffix(link, "/"), "/")
	ptoken := token[len(token)-1]
	if !tokenShape.MatchString(ptoken) || ptoken == p.s.token {
		t.Fatalf("pointer token %q (view token %q)", ptoken, p.s.token)
	}
	p.s.DisablePointer()
	again, err := p.enablePointer(t)
	if err != nil || again != link {
		t.Fatalf("re-enabled link %q, want the same %q", again, link)
	}
	if code, _ := post(t, p.s.Link()+"point", "application/json", tapBody); code != http.StatusMethodNotAllowed {
		t.Fatalf("view link accepted a POST: %d", code)
	}
}

func TestPointerPostLands(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	if code, _ := post(t, link+"point", "application/json", tapBody); code != http.StatusNoContent {
		t.Fatalf("POST %d", code)
	}
	got := p.batches()
	if len(got) != 1 || got[0].Points[0] != [2]int{3, 1} {
		t.Fatalf("batches %+v", got)
	}
}

// Points land only on what the operator sees now; the page gets the same
// answer either way.
func TestPointerPostDropped(t *testing.T) {
	cases := map[string]struct {
		frame func(*testing.T) terminal.Frame
		class terminal.FrameClass
		body  string
	}{
		"stale grid":     {func(t *testing.T) terminal.Frame { return livePointer(t, "x") }, terminal.FramePublic, `{"cols":80,"rows":24,"down":false,"points":[[3,1]]}`},
		"private screen": {func(t *testing.T) terminal.Frame { return livePointer(t, "x") }, terminal.FramePrivate, tapBody},
		"marker hidden":  {func(t *testing.T) terminal.Frame { return live(t, "x") }, terminal.FramePublic, tapBody},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := startPointerSession(t)
			link, _ := p.enablePointer(t)
			p.s.Offer(c.frame(t), c.class)
			p.s.hub.sync()
			if code, _ := post(t, link+"point", "application/json", c.body); code != http.StatusNoContent {
				t.Fatalf("POST %d", code)
			}
			if got := p.batches(); len(got) != 0 {
				t.Fatalf("dropped batch landed: %+v", got)
			}
		})
	}
}

func TestPointerPostWhileOff(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	p.s.DisablePointer()
	if code, _ := post(t, link+"point", "application/json", tapBody); code != http.StatusForbidden {
		t.Fatalf("POST while off: %d", code)
	}
	if len(p.batches()) != 0 {
		t.Fatal("a batch landed while pointing was off")
	}
	// Still a view link.
	if code, err := get(t, link); err != nil || code != http.StatusOK {
		t.Fatalf("pointer link as view-only: %d %v", code, err)
	}
}

// Every method on every route for every token: only POST to the pointer
// link's point, note and resolve routes reads a body.
func TestPointerRouteTable(t *testing.T) {
	p := startPointerSession(t)
	plink, _ := p.enablePointer(t)
	p.showPointer(t)
	base := strings.TrimSuffix(p.s.Link(), p.s.token+"/")
	vlink := p.s.Link()
	wrong := base + strings.Repeat("A", 43) + "/"
	type row struct {
		method, url string
		want        int
	}
	var rows []row
	for _, m := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions} {
		rows = append(rows, row{m, plink + "point", 405}, row{m, vlink + "point", 405})
	}
	rows = append(rows,
		row{http.MethodPost, vlink + "point", 405},
		row{http.MethodPost, vlink + "note", 405},
		row{http.MethodPost, vlink + "resolve", 405},
		row{http.MethodPost, wrong + "note", 405},
		row{http.MethodGet, plink + "note", 404},
		row{http.MethodGet, plink + "resolve", 404},
		row{http.MethodPost, wrong + "point", 405},
		row{http.MethodPost, plink, 405},
		row{http.MethodPost, plink + "events", 405},
		row{http.MethodPost, plink + "point/", 405},
		row{http.MethodPost, plink + "../" + p.s.token + "/point", 405},
		row{http.MethodGet, plink + "point", 404},
		row{http.MethodGet, plink, 200},
		row{http.MethodGet, plink + "viewer.js", 200},
		row{http.MethodGet, wrong, 404},
	)
	for _, r := range rows {
		req, _ := http.NewRequest(r.method, r.url, strings.NewReader(tapBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := castClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != r.want {
			t.Errorf("%s %s = %d, want %d", r.method, strings.TrimPrefix(r.url, base), resp.StatusCode, r.want)
		}
	}
	if n := len(p.batches()); n != 0 {
		t.Fatalf("%d batches landed through the wrong routes", n)
	}
}

// Rejections are fixed text: nothing from the request is echoed.
func TestPointerPostRejects(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	const marker = "PWNED<script>"
	cases := []struct {
		name, contentType, body string
		want                    int
	}{
		{"content type", "text/plain", tapBody, 415},
		{"form", "application/x-www-form-urlencoded", "cols=1", 415},
		{"oversize", "application/json", `{"cols":40,"rows":4,"down":false,"points":[[1,1]],"x":"` + strings.Repeat(marker, 600) + `"}`, 413},
		{"unknown field", "application/json", `{"cols":40,"rows":4,"down":false,"points":[[1,1]],"x":"` + marker + `"}`, 400},
		{"not json", "application/json", marker, 400},
		{"out of grid", "application/json", `{"cols":40,"rows":4,"down":false,"points":[[40,1]]}`, 400},
	}
	for _, c := range cases {
		code, body := post(t, link+"point", c.contentType, c.body)
		if code != c.want {
			t.Errorf("%s: %d, want %d", c.name, code, c.want)
		}
		if strings.Contains(body, "PWNED") || strings.Contains(body, "<script>") {
			t.Errorf("%s: response echoed the request: %q", c.name, body)
		}
	}
	if n := len(p.batches()); n != 0 {
		t.Fatalf("%d rejected batches landed", n)
	}
}

func TestPointerRateLimit(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(testwait.HostState())
		}
	})
	var clockMu sync.Mutex
	now := time.Unix(100, 0)
	p := startPointerSessionAt(t, HubOptions{}, func() time.Time { clockMu.Lock(); defer clockMu.Unlock(); return now })
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	limited := 0
	for range PointRatePerSec + 20 {
		if code, _ := post(t, link+"point", "application/json", tapBody); code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 20 {
		t.Fatalf("limited %d of %d same-time posts, want20; %s", limited, PointRatePerSec+20, testwait.HostState())
	}
	clockMu.Lock()
	now = now.Add(time.Second)
	clockMu.Unlock()
	if code, _ := post(t, link+"point", "application/json", tapBody); code != http.StatusNoContent {
		t.Fatalf("refilled request=%d; %s", code, testwait.HostState())
	}
}

// A request that never finishes its body can't hold one of the few in-flight
// slots forever; other posts still get through after the read deadline.
func TestPointerSlowBodyDoesNotStarve(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	for range MaxPointInFlight {
		pr, pw := io.Pipe()
		defer pw.Close()
		req, _ := http.NewRequest(http.MethodPost, link+"point", pr)
		req.Header.Set("Content-Type", "application/json")
		go func() {
			if resp, err := castClient.Do(req); err == nil {
				resp.Body.Close()
			}
		}()
		pw.Write([]byte(`{"cols":`))
	}
	// Wait on the slots themselves, not a guess at how long the posts take
	// to reach the server (#480).
	inFlight := func() int { return len(p.s.pointer.inFlight) }
	for inFlight() < MaxPointInFlight {
		time.Sleep(time.Millisecond)
	}
	if code, _ := post(t, link+"point", "application/json", tapBody); code != http.StatusTooManyRequests {
		t.Fatalf("with every slot held, a post got %d, want 429", code)
	}
	// The read deadline frees the slots; until it does, the slow bodies hold
	// them all. A deadline that never fired would hang here.
	for inFlight() > 0 {
		time.Sleep(time.Millisecond)
	}
	if code, _ := post(t, link+"point", "application/json", tapBody); code != http.StatusNoContent {
		t.Fatalf("after the read budget freed the slots, a post got %d, want 204", code)
	}
}

// OnPoints may call back into the session (Couch turning pointing off on a
// batch it judges late): no lock is held across the call.
func TestPointerCallbackMayReenter(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	p.mu.Lock()
	p.onHook = func(PointBatch) { p.s.DisablePointer(); _ = p.s.PointerLink() }
	p.mu.Unlock()
	done := make(chan int, 1)
	go func() { code, _ := post(t, link+"point", "application/json", tapBody); done <- code }()
	select {
	case code := <-done:
		if code != http.StatusNoContent {
			t.Fatalf("POST %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("re-entrant callback deadlocked")
	}
}

func readEvent(t *testing.T, r *bufio.Reader, name string) string {
	t.Helper()
	events := &sseReader{t: t, r: r}
	for {
		ev := events.next()
		if ev.name == name {
			return ev.data
		}
	}
}

func TestPointerCapsEvents(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	resp, err := castClient.Get(link + "events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	pr := bufio.NewReader(resp.Body)
	palette := `"palette":["#ffaf00","#56b4e9","#009e73","#f0e442","#d55e00","#cc79a7"]`
	if got := readEvent(t, pr, "caps"); got != `{"pointer":true,"color":0,`+palette+`}` {
		t.Fatalf("caps on join %q", got)
	}
	p.s.DisablePointer()
	if got := readEvent(t, pr, "caps"); got != `{"pointer":false,"color":0,`+palette+`}` {
		t.Fatalf("caps after off %q", got)
	}
	p.enablePointer(t)
	if got := readEvent(t, pr, "caps"); got != `{"pointer":true,"color":0,`+palette+`}` {
		t.Fatalf("caps after on %q", got)
	}
	// A view stream never hears of the pointer.
	vresp, err := castClient.Get(p.s.Link() + "events")
	if err != nil {
		t.Fatal(err)
	}
	defer vresp.Body.Close()
	go func() { time.Sleep(50 * time.Millisecond); p.s.DisablePointer() }()
	vr := &sseReader{t: t, r: bufio.NewReader(vresp.Body)}
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if ev := vr.next(); ev.name == "caps" {
			t.Fatal("a view stream got caps")
		}
	}
}

func TestPointerLinkEndsWithBroadcast(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	resp, err := castClient.Get(link + "events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	p.s.Stop(nil)
	readEvent(t, bufio.NewReader(resp.Body), "end")
	<-p.s.Done()
	if _, err := get(t, link); err == nil {
		t.Fatal("pointer link still answers after the broadcast ended")
	}
}

func TestPointerContentTypeParsing(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	for _, ct := range []string{"Application/JSON", "application/json; charset=utf-8"} {
		if code, _ := post(t, link+"point", ct, tapBody); code != http.StatusNoContent {
			t.Errorf("%q: %d", ct, code)
		}
	}
	for _, ct := range []string{"application/jsonx", "text/json", "application/json;;;"} {
		if code, _ := post(t, link+"point", ct, tapBody); code != http.StatusUnsupportedMediaType {
			t.Errorf("%q: %d, want 415", ct, code)
		}
	}
}

// A body that doesn't arrive in time is a bad request, not "too large".
func TestPointerSlowBodyIsBadRequest(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	pr, pw := io.Pipe()
	defer pw.Close()
	req, _ := http.NewRequest(http.MethodPost, link+"point", pr)
	req.Header.Set("Content-Type", "application/json")
	codes := make(chan int, 1)
	go func() {
		resp, err := castClient.Do(req)
		if err != nil {
			codes <- 0
			return
		}
		resp.Body.Close()
		codes <- resp.StatusCode
	}()
	pw.Write([]byte(`{"cols":`))
	select {
	case code := <-codes:
		if code != http.StatusBadRequest && code != 0 {
			t.Fatalf("slow body: %d, want 400", code)
		}
	case <-time.After(pointReadBudget + 5*time.Second):
		t.Fatal("slow body never timed out")
	}
}

func TestPointerAfterStop(t *testing.T) {
	p := startPointerSession(t)
	p.s.Stop(nil)
	<-p.s.Done()
	if link, err := p.enablePointer(t); err == nil {
		t.Fatalf("EnablePointer after Stop minted %q", link)
	}
}

// A watch report from before the operator flipped pointing off and on again
// doesn't turn the new pointing off.
func TestPointerLateHiddenReportIgnored(t *testing.T) {
	p := startPointerSession(t)
	p.enablePointer(t)
	stale := p.s.pointer.generation()
	p.s.DisablePointer()
	p.enablePointer(t)
	p.s.pointerHidden(stale)
	if on, _ := p.s.pointer.On(); !on {
		t.Fatal("a stale watch report turned re-enabled pointing off")
	}
	p.mu.Lock()
	offs := p.offs
	p.mu.Unlock()
	if offs != 0 {
		t.Fatal("OnPointerOff called for a stale report")
	}
	p.s.pointerHidden(p.s.pointer.generation())
	if on, _ := p.s.pointer.On(); on {
		t.Fatal("a current watch report didn't turn pointing off")
	}
}

// Each pointer stream is assigned the color fewest live streams hold (#446):
// a second viewer gets the next color, and a color comes back when its
// stream ends.
func TestPointerStreamsGetDistinctColors(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.enablePointer(t)
	p.showPointer(t)
	join := func() (int, func()) {
		t.Helper()
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, link+"events", nil)
		resp, err := castClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var caps struct{ Color int }
		if err := json.Unmarshal([]byte(readEvent(t, bufio.NewReader(resp.Body), "caps")), &caps); err != nil {
			t.Fatal(err)
		}
		return caps.Color, func() { cancel(); resp.Body.Close() }
	}
	a, leaveA := join()
	b, leaveB := join()
	defer leaveB()
	if a != 0 || b != 1 {
		t.Fatalf("two streams got colors %d and %d, want 0 and 1", a, b)
	}
	// Each viewer points in its assigned color; both land, and the marks
	// draw them apart.
	for i, c := range []int{a, b} {
		body := fmt.Sprintf(`{"cols":40,"rows":4,"down":false,"color":%d,"points":[[%d,1]]}`, c, 3+10*i)
		if code, _ := post(t, link+"point", "application/json", body); code != http.StatusNoContent {
			t.Fatalf("POST %d", code)
		}
	}
	got := p.batches()
	if len(got) != 2 || got[0].Color != a || got[1].Color != b {
		t.Fatalf("landed %+v", got)
	}
	m := NewMarks()
	for _, batch := range got {
		m.Add(batch.Points, batch.Cols, batch.Rows, batch.ChromeRows, batch.Color, t0)
	}
	frame := m.Overlay(livePointer(t, "operator screen"), t0)
	if x, y := frame.Cells[40+3].Style.Bg, frame.Cells[40+13].Style.Bg; x == y || x != color.Color(markPalette[a].index) {
		t.Fatalf("two viewers' marks drew %v and %v", x, y)
	}
	leaveA()
	deadline := time.Now().Add(3 * time.Second)
	for {
		c, leaveC := join()
		leaveC()
		if c == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("a freed color was not reassigned: got %d", c)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestClaimColorPrefersLeastHeld(t *testing.T) {
	p := newPointerState(nil)
	for i := range PointerColors {
		if c := p.claimColor(); c != i {
			t.Fatalf("claim %d got color %d", i, c)
		}
	}
	if c := p.claimColor(); c != 0 {
		t.Fatalf("with every color held once, got %d, want 0", c)
	}
	p.releaseColor(3)
	if c := p.claimColor(); c != 3 {
		t.Fatalf("after releasing 3, got %d", c)
	}
}

func TestEnsurePointerLinkStaysOffAndStable(t *testing.T) {
	p := startPointerSession(t)
	before := p.s.pointer.generation()
	link, err := p.s.EnsurePointerLink()
	if err != nil || link == "" {
		t.Fatalf("link %q: %v", link, err)
	}
	if on, _ := p.s.pointer.On(); on {
		t.Fatal("copy enabled pointing")
	}
	if p.s.pointer.generation() != before {
		t.Fatal("copy changed admission generation")
	}
	again, err := p.s.EnsurePointerLink()
	if err != nil || again != link {
		t.Fatal("unstable copy", again, err)
	}
	enabled, err := p.s.EnablePointer()
	if err != nil || enabled != link {
		t.Fatal("enable replaced link", enabled, err)
	}
	p.s.DisablePointer()
	again, err = p.s.EnsurePointerLink()
	if err != nil || again != link {
		t.Fatal("off copy replaced link", again, err)
	}
}

func TestEnsurePointerFailureDoesNotPublishUnlockedLink(t *testing.T) {
	for _, lockFails := range []bool{false, true} {
		failure := fmt.Errorf("mint failed")
		p := newPointerState(func(string) error {
			if lockFails {
				return failure
			}
			return nil
		})
		mint := func() (string, error) {
			if !lockFails {
				return "", failure
			}
			return "token", nil
		}
		if token, err := p.ensure(mint); err != failure || token != "" {
			t.Fatalf("%q %v", token, err)
		}
		if p.link() != "" || p.generation() != 0 {
			t.Fatal("failed mint published or changed admission")
		}
		if on, _ := p.On(); on {
			t.Fatal("failed mint enabled")
		}
	}
}
