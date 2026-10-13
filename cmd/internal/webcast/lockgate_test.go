package webcast

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// lockedCast mounts a web cast's server, with its view and drive links
// locked, on a local Host: what a viewer reaches through the tunnel.
type lockedCast struct {
	view, drive *castlock.Lock
	base        string
}

func newLockedCast(t *testing.T) *lockedCast {
	t.Helper()
	f := newDriveFixture(t, time.Minute)
	bd := board.New(notes.Structural{}, func(recording.Record, int) error { return nil })
	t.Cleanup(func() { _ = bd.Close() })
	locks := &broadcast.LinkLocks{}
	if err := locks.Mint(broadcast.LinkView, "tok", castlock.DefaultDigits, nil); err != nil {
		t.Fatal(err)
	}
	if err := locks.Mint(broadcast.LinkDrive, "drv", broadcast.DriveCodeDigits, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(locks.End)
	view, drv := locks.LockFor("tok"), locks.LockFor("drv")
	srv := f.srv.WithBoard(bd).WithLocks(locks)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcast.LocalOnly{}, ProbeTimeout: 5 * time.Second})
	m, err := host.Mount(context.Background(), srv, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Unmount)
	return &lockedCast{view: view, drive: drv, base: m.Base()}
}

func request(t *testing.T, c *http.Client, method, url, body string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(b)
}

// Every web cast route answers 401 before a session, on both links: the
// event stream, the recorded assets, the palette, the viewer's files and
// every POST -- the board's and remote driving's.
func TestWebCastLinksRefuseEveryRouteWithoutASession(t *testing.T) {
	lc := newLockedCast(t)
	routes := []struct{ method, rest string }{
		{http.MethodGet, "events"}, {http.MethodGet, "events?viewer=v1"},
		{http.MethodGet, "assets/0123456789abcdef"}, {http.MethodGet, "palette.json"},
		{http.MethodPost, "stroke"}, {http.MethodPost, "note"}, {http.MethodPost, "note-state"},
		{http.MethodPost, "drive/hello"}, {http.MethodPost, "drive/control"}, {http.MethodPost, "drive/input"},
	}
	for rest := range assets {
		if rest != "" {
			routes = append(routes, struct{ method, rest string }{http.MethodGet, rest})
		}
	}
	bare := &http.Client{Timeout: 5 * time.Second}
	for _, link := range []string{lc.base + "/tok/", lc.base + "/drv/"} {
		for _, r := range routes {
			if status, body := request(t, bare, r.method, link+r.rest, `{}`); status != http.StatusUnauthorized || body != "locked\n" {
				t.Errorf("%s %s%s: %d %q", r.method, link, r.rest, status, body)
			}
		}
		if status, body := request(t, bare, http.MethodGet, link, ""); status != http.StatusOK || !strings.Contains(body, `src="lock.js"`) {
			t.Errorf("%s without a session: %d, lock page %v", link, status, strings.Contains(body, "lock.js"))
		}
	}
}

// The drive code is six digits and its own: the view code never opens the
// drive link, and a view session is not a drive session.
func TestWebCastDriveCodeIsSeparate(t *testing.T) {
	lc := newLockedCast(t)
	if len(lc.view.Code()) != 4 || len(lc.drive.Code()) != 6 {
		t.Fatalf("codes %q %q", lc.view.Code(), lc.drive.Code())
	}
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Timeout: 5 * time.Second, Jar: jar}
	if status, _ := request(t, c, http.MethodPost, lc.base+"/tok/unlock", `{"code":"`+lc.view.Code()+`"}`); status != http.StatusOK {
		t.Fatalf("view unlock: %d", status)
	}
	if status, _ := request(t, c, http.MethodPost, lc.base+"/drv/unlock", `{"code":"`+lc.view.Code()+`"}`); status == http.StatusOK {
		t.Fatal("the view code opened the drive link")
	}
	if status, _ := request(t, c, http.MethodPost, lc.base+"/drv/drive/hello", `{"name":"Ada"}`); status != http.StatusUnauthorized {
		t.Fatalf("drive hello with a view session: %d", status)
	}
	if status, _ := request(t, c, http.MethodPost, lc.base+"/drv/unlock", `{"code":"`+lc.drive.Code()+`"}`); status != http.StatusOK {
		t.Fatal("the drive code did not open the drive link")
	}
	if status, body := request(t, c, http.MethodPost, lc.base+"/drv/drive/hello", `{"name":"Ada"}`); status != http.StatusOK {
		t.Fatalf("drive hello with a drive session: %d %q", status, body)
	}
}

// A cast's codes: the view code always, the drive code only when the cast can
// be driven; rotation gives a new code and revokes the link's sessions.
func TestCastCodesAndRotation(t *testing.T) {
	c, _ := startCast(t, &broadcast.FakeTunnel{})
	u := waitUpdate(t, c, "link", func(u Update) bool { return u.Link != "" })
	codes := c.Codes()
	if len(codes) != 1 || codes[0].Link != broadcast.LinkView || len(codes[0].Code) != 4 {
		t.Fatalf("an undriven cast's codes: %+v", codes)
	}
	if err := c.RotateCode(broadcast.LinkDrive); err == nil {
		t.Fatal("rotated a drive code the cast doesn't have")
	}
	unlockCast(t, c, u)
	s := openSSE(t, u.Link+"events")
	s.nextState()
	old := codes[0].Code
	if err := c.RotateCode(broadcast.LinkView); err != nil {
		t.Fatal(err)
	}
	if c.Codes()[0].Code == old {
		t.Fatal("rotation kept the code")
	}
	ended := make(chan struct{})
	go func() { io.Copy(io.Discard, s.resp.Body); close(ended) }()
	select {
	case <-ended:
	case <-testwait.After(t):
		t.Fatal("a revoked stream kept running")
	}
}

func TestNoteAuthorRoundTripsOnTheBoardRule(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool
	}{
		{"  Ada ", "Ada", true},
		{"", "", true},
		{strings.Repeat("x", broadcast.MaxViewerNameRunes), strings.Repeat("x", broadcast.MaxViewerNameRunes), true},
		{strings.Repeat("x", broadcast.MaxViewerNameRunes+1), "", false},
		{"A\nB", "", false},
		{"A​B", "", false},
	} {
		got, ok := broadcast.ViewerName(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%q → %q %v", tc.in, got, ok)
		}
	}
	if board.MaxNameRunes != broadcast.MaxViewerNameRunes {
		t.Fatal("the board's name bound drifted from every cast's")
	}
}
