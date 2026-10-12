package broadcast

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// bareClient has no cookies: a request from someone holding only the link.
func bareClient() *http.Client { return &http.Client{Timeout: 5 * time.Second} }

func jarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Timeout: 5 * time.Second, Jar: jar}
}

func do(t *testing.T, c *http.Client, method, url, contentType, body string, cookie *http.Cookie) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, string(b), resp.Header
}

func unlockWith(t *testing.T, c *http.Client, link, code string) (int, string, http.Header) {
	t.Helper()
	return do(t, c, http.MethodPost, link+"unlock", "application/json", `{"code":"`+code+`"}`, nil)
}

// lockedRoutes is every route a locked link has behind it.
func lockedRoutes() []struct{ method, rest string } {
	routes := []struct{ method, rest string }{
		{http.MethodGet, "events"},
		{http.MethodPost, "point"}, {http.MethodPost, "note"}, {http.MethodPost, "resolve"},
		{http.MethodGet, "no-such-path"}, {http.MethodPut, ""}, {http.MethodDelete, "events"},
	}
	for rest := range assets {
		if rest != "" {
			routes = append(routes, struct{ method, rest string }{http.MethodGet, rest})
		}
	}
	return routes
}

// TestLockedLinksRefuseEveryRouteWithoutASession: the view and pointer links
// answer nothing behind the lock -- not the event stream, not an asset, not a
// POST route -- to a request with no session, a forged one, or one revoked by
// rotation. The lock page itself is what an unlocked GET of the link gets.
func TestLockedLinksRefuseEveryRouteWithoutASession(t *testing.T) {
	p := startPointerSession(t)
	pointerLink, err := p.s.EnablePointer()
	if err != nil {
		t.Fatal(err)
	}
	// A session that rotation revoked, for each link.
	revoked := map[string]*http.Cookie{}
	for link, name := range map[string]string{p.s.Link(): LinkView, pointerLink: LinkPointer} {
		_, _, h := unlockWith(t, bareClient(), link, codeOf(t, p.s, name))
		cookies := (&http.Response{Header: h}).Cookies()
		if len(cookies) != 1 {
			t.Fatalf("%s: unlock set %d cookies", name, len(cookies))
		}
		revoked[link] = cookies[0]
		if err := p.s.RotateCode(name); err != nil {
			t.Fatal(err)
		}
	}
	for _, link := range []string{p.s.Link(), pointerLink} {
		for _, cookie := range []*http.Cookie{nil, {Name: sessionCookie, Value: "forged"}, revoked[link]} {
			for _, r := range lockedRoutes() {
				ct := ""
				if r.method == http.MethodPost {
					ct = "application/json"
				}
				status, body, _ := do(t, bareClient(), r.method, link+r.rest, ct, `{}`, cookie)
				if status != http.StatusUnauthorized || body != "locked\n" {
					t.Errorf("%s %s with %v: %d %q", r.method, r.rest, cookie != nil, status, body)
				}
			}
			status, body, _ := do(t, bareClient(), http.MethodGet, link, "", "", cookie)
			if status != http.StatusOK || !strings.Contains(body, `src="lock.js"`) || strings.Contains(body, "viewer.js") {
				t.Errorf("GET link without a session: %d, lock page %v", status, strings.Contains(body, "lock.js"))
			}
			for _, a := range []string{"lock.js", "lock.css"} {
				if status, _, _ := do(t, bareClient(), http.MethodGet, link+a, "", "", cookie); status != http.StatusOK {
					t.Errorf("%s: %d", a, status)
				}
			}
			if status, _, _ := do(t, bareClient(), http.MethodGet, link+"session", "", "", cookie); status != http.StatusUnauthorized {
				t.Errorf("session probe without a session: %d", status)
			}
		}
	}
}

// TestDirectEventStreamRequestIsRefused is the done-when's named case: a
// client that skips the page and asks for the stream gets nothing.
func TestDirectEventStreamRequestIsRefused(t *testing.T) {
	s := startSession(t, &FakeTunnel{}, HubOptions{})
	s.Offer(live(t, "secret screen"), terminal.FramePublic)
	status, body, _ := do(t, bareClient(), http.MethodGet, s.Link()+"events", "", "", nil)
	if status != http.StatusUnauthorized || strings.Contains(body, "secret") || strings.Contains(body, "event:") {
		t.Fatalf("direct stream: %d %q", status, body)
	}
}

func TestUnlockIssuesAPathScopedHttpOnlyStrictSessionCookie(t *testing.T) {
	s := startSession(t, &FakeTunnel{}, HubOptions{})
	c := jarClient(t)
	code := codeOf(t, s, LinkView)
	if status, body, _ := unlockWith(t, c, s.Link(), wrongDigits(code)); status != http.StatusForbidden || !strings.Contains(body, refuseWrongCode) {
		t.Fatalf("wrong code: %d %q", status, body)
	}
	if status, body, _ := unlockWith(t, c, s.Link(), "abcd"); status != http.StatusBadRequest || !strings.Contains(body, refuseMalformed) {
		t.Fatalf("malformed code: %d %q", status, body)
	}
	if status, _, _ := do(t, c, http.MethodPost, s.Link()+"unlock", "text/plain", `{"code":"`+code+`"}`, nil); status != http.StatusUnsupportedMediaType {
		t.Fatalf("non-JSON unlock: %d", status)
	}
	status, _, h := unlockWith(t, c, s.Link(), code)
	if status != http.StatusOK {
		t.Fatalf("right code: %d", status)
	}
	cookies := (&http.Response{Header: h}).Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies %v", cookies)
	}
	ck := cookies[0]
	token := strings.TrimSuffix(strings.TrimPrefix(s.Link(), s.mount.Base()), "/")
	if ck.Name != sessionCookie || ck.Path != token+"/" || !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode ||
		!ck.Expires.IsZero() || ck.MaxAge != 0 || ck.Secure {
		t.Fatalf("cookie %+v (local-only link: not Secure)", ck)
	}
	if status, body, _ := do(t, c, http.MethodGet, s.Link(), "", "", nil); status != http.StatusOK || !strings.Contains(body, "viewer.js") {
		t.Fatalf("unlocked link: %d, viewer %v", status, strings.Contains(body, "viewer.js"))
	}
	if status, _, _ := do(t, c, http.MethodGet, s.Link()+"session", "", "", nil); status != http.StatusNoContent {
		t.Fatalf("session probe: %d", status)
	}
}

func wrongDigits(code string) string {
	b := []byte(code)
	b[0] = '0' + (b[0]-'0'+1)%10
	return string(b)
}

func TestUnlockAttemptsAreRateLimited(t *testing.T) {
	s := startSession(t, &FakeTunnel{}, HubOptions{})
	wrong := wrongDigits(codeOf(t, s, LinkView))
	limited := 0
	for range 7 {
		if status, _, h := unlockWith(t, bareClient(), s.Link(), wrong); status == http.StatusTooManyRequests {
			limited++
			if h.Get("Retry-After") == "" {
				t.Fatal("429 without Retry-After")
			}
		}
	}
	if limited == 0 {
		t.Fatal("7 rapid attempts were never limited")
	}
}

func TestSpentBudgetRotatesTheCodeAndTellsCouch(t *testing.T) {
	var mu sync.Mutex
	var got []string
	s, err := Start(context.Background(), Config{Tunnel: &FakeTunnel{}, OnCodeRotated: func(link string, r castlock.Rotation) {
		mu.Lock()
		defer mu.Unlock()
		if r.Reason == castlock.ReasonGuessing {
			got = append(got, link)
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(nil); <-s.Done() })
	viewer := jarClient(t)
	first := codeOf(t, s, LinkView)
	unlockWith(t, viewer, s.Link(), first)
	wrongs := 0
	deadline := testwait.After(t)
	for wrongs < castlock.DefaultBudget {
		status, _, _ := unlockWith(t, bareClient(), s.Link(), wrongDigits(first))
		switch status {
		case http.StatusForbidden:
			wrongs++
		case http.StatusTooManyRequests:
			select {
			case <-deadline:
				t.Fatal("budget never spent")
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	if codeOf(t, s, LinkView) == first {
		t.Fatal("budget spent but the code is unchanged")
	}
	mu.Lock()
	if len(got) != 1 || got[0] != LinkView {
		t.Fatalf("guess rotations told %v", got)
	}
	mu.Unlock()
	if status, _, _ := do(t, viewer, http.MethodGet, s.Link()+"session", "", "", nil); status != http.StatusNoContent {
		t.Fatal("a guess-rotation cut off a viewer")
	}
}

// TestRotationCutsLiveStreams: rotating a link's code ends the event streams
// admitted under the old sessions, and the page's probe then says locked.
func TestRotationCutsLiveStreams(t *testing.T) {
	s := startSession(t, &FakeTunnel{}, HubOptions{})
	s.Offer(live(t, "on air"), terminal.FramePublic)
	c := jarClient(t)
	unlockWith(t, c, s.Link(), codeOf(t, s, LinkView))
	resp, err := c.Get(s.Link() + "events")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("stream: %v %v", err, resp)
	}
	defer resp.Body.Close()
	ended := make(chan struct{})
	go func() { io.Copy(io.Discard, resp.Body); close(ended) }()
	if err := s.RotateCode(LinkView); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ended:
	case <-testwait.After(t):
		t.Fatal("a revoked stream kept running")
	}
	if status, _, _ := do(t, c, http.MethodGet, s.Link()+"session", "", "", nil); status != http.StatusUnauthorized {
		t.Fatalf("probe after rotation: %d", status)
	}
}

// TestViewSessionNeverOpensThePointerLink: each link has its own lock, so the
// view code's session is not the pointer link's.
func TestViewSessionNeverOpensThePointerLink(t *testing.T) {
	p := startPointerSession(t)
	pointerLink, err := p.s.EnablePointer()
	if err != nil {
		t.Fatal(err)
	}
	c := jarClient(t)
	_, _, h := unlockWith(t, c, p.s.Link(), codeOf(t, p.s, LinkView))
	view := (&http.Response{Header: h}).Cookies()[0]
	for _, ck := range []*http.Cookie{nil, {Name: sessionCookie, Value: view.Value}} {
		if status, _, _ := do(t, c, http.MethodGet, pointerLink+"events", "", "", ck); status != http.StatusUnauthorized {
			t.Fatalf("pointer stream with the view session: %d", status)
		}
	}
	if status, _, _ := unlockWith(t, bareClient(), pointerLink, codeOf(t, p.s, LinkView)); status == http.StatusOK && codeOf(t, p.s, LinkView) != codeOf(t, p.s, LinkPointer) {
		t.Fatal("the view code opened the pointer link")
	}
}

// TestBroadcastLinksAreLockedFromTheirFirstRequest is PQ-1: the lock exists
// before a token is routable, so even Mount's own probe met the lock page.
func TestBroadcastLinksAreLockedFromTheirFirstRequest(t *testing.T) {
	p := startPointerSession(t)
	server := NewServer(ServerOptions{Token: p.s.token, ViewLock: p.s.viewLock, Pointer: p.s.pointer})
	if server.LockFor(p.s.token) == nil {
		t.Fatal("the view link has no lock")
	}
	if p.s.pointer.Lock() != nil {
		t.Fatal("a pointer lock before the pointer link exists")
	}
	link, _ := p.s.EnablePointer()
	segment := strings.TrimSuffix(strings.TrimPrefix(link, p.s.base+"/"), "/")
	if server.LockFor(segment) == nil {
		t.Fatal("the pointer link was minted without its lock")
	}
}
