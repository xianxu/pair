package broadcast

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

type pointerSession struct {
	s      *Session
	mu     sync.Mutex
	got    []PointBatch
	offs   int
	onHook func(PointBatch)
}

func (p *pointerSession) batches() []PointBatch {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]PointBatch(nil), p.got...)
}

func startPointerSession(t *testing.T) *pointerSession {
	t.Helper()
	p := &pointerSession{}
	s, err := Start(context.Background(), Config{Tunnel: &FakeTunnel{}, Ping: 20 * time.Millisecond,
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
	resp, err := http.Post(url, contentType, strings.NewReader(body))
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
	link, err := p.s.EnablePointer()
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Split(strings.TrimSuffix(link, "/"), "/")
	ptoken := token[len(token)-1]
	if !tokenShape.MatchString(ptoken) || ptoken == p.s.token {
		t.Fatalf("pointer token %q (view token %q)", ptoken, p.s.token)
	}
	p.s.DisablePointer()
	again, err := p.s.EnablePointer()
	if err != nil || again != link {
		t.Fatalf("re-enabled link %q, want the same %q", again, link)
	}
	if code, _ := post(t, p.s.Link()+"point", "application/json", tapBody); code != http.StatusMethodNotAllowed {
		t.Fatalf("view link accepted a POST: %d", code)
	}
}

func TestPointerPostLands(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.s.EnablePointer()
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
			link, _ := p.s.EnablePointer()
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
	link, _ := p.s.EnablePointer()
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
// link's point route reads a body.
func TestPointerRouteTable(t *testing.T) {
	p := startPointerSession(t)
	plink, _ := p.s.EnablePointer()
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
		resp, err := http.DefaultClient.Do(req)
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
	link, _ := p.s.EnablePointer()
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
	p := startPointerSession(t)
	link, _ := p.s.EnablePointer()
	p.showPointer(t)
	limited := 0
	for range PointRatePerSec + 20 {
		if code, _ := post(t, link+"point", "application/json", tapBody); code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited < 15 {
		t.Fatalf("only %d of %d rapid posts were limited", limited, PointRatePerSec+20)
	}
}

// A request that never finishes its body can't hold one of the few in-flight
// slots forever; other posts still get through after the read deadline.
func TestPointerSlowBodyDoesNotStarve(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.s.EnablePointer()
	p.showPointer(t)
	for range MaxPointInFlight {
		pr, pw := io.Pipe()
		defer pw.Close()
		req, _ := http.NewRequest(http.MethodPost, link+"point", pr)
		req.Header.Set("Content-Type", "application/json")
		go func() {
			if resp, err := http.DefaultClient.Do(req); err == nil {
				resp.Body.Close()
			}
		}()
		pw.Write([]byte(`{"cols":`))
	}
	time.Sleep(100 * time.Millisecond)
	if code, _ := post(t, link+"point", "application/json", tapBody); code != http.StatusTooManyRequests {
		t.Fatalf("with every slot held, a post got %d, want 429", code)
	}
	deadline := time.Now().Add(pointReadBudget + 5*time.Second)
	for {
		code, _ := post(t, link+"point", "application/json", tapBody)
		if code == http.StatusNoContent {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("slow bodies held the slots past the read budget (last %d)", code)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// OnPoints may call back into the session (Couch turning pointing off on a
// batch it judges late): no lock is held across the call.
func TestPointerCallbackMayReenter(t *testing.T) {
	p := startPointerSession(t)
	link, _ := p.s.EnablePointer()
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
	link, _ := p.s.EnablePointer()
	p.showPointer(t)
	resp, err := http.Get(link + "events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	pr := bufio.NewReader(resp.Body)
	if got := readEvent(t, pr, "caps"); got != `{"pointer":true}` {
		t.Fatalf("caps on join %q", got)
	}
	p.s.DisablePointer()
	if got := readEvent(t, pr, "caps"); got != `{"pointer":false}` {
		t.Fatalf("caps after off %q", got)
	}
	p.s.EnablePointer()
	if got := readEvent(t, pr, "caps"); got != `{"pointer":true}` {
		t.Fatalf("caps after on %q", got)
	}
	// A view stream never hears of the pointer.
	vresp, err := http.Get(p.s.Link() + "events")
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
	link, _ := p.s.EnablePointer()
	p.showPointer(t)
	resp, err := http.Get(link + "events")
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
