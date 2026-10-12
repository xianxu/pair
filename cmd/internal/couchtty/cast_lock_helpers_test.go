package couchtty

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// Every cast link is locked (pair#495), so this package's tests reach casts
// as a viewer does: through castClient, whose cookie jar holds the sessions
// they unlocked. A session cookie is scoped to its own link's path, so one jar
// serves every cast; http.DefaultClient is left alone.
var castClient = func() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &http.Client{Jar: jar}
}()

// unlockCastLink enters s's code for which ("view" or "pointer") at link,
// through client, whose jar then holds the session.
func unlockCastLink(t *testing.T, client *http.Client, s *broadcast.Session, which, link string) {
	t.Helper()
	code := ""
	for _, c := range s.Codes() {
		if c.Link == which {
			code = c.Code
		}
	}
	resp, err := client.Post(link+"unlock", "application/json", strings.NewReader(`{"code":"`+code+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unlock %s: %d", which, resp.StatusCode)
	}
}

// unlockWebCast enters the running web cast's codes at its links, through
// castClient: the view link, and the drive link when there is one.
func unlockWebCast(t *testing.T, con *Console) {
	t.Helper()
	con.mu.Lock()
	cast, u := con.wcast.cast, con.wcast.update
	con.mu.Unlock()
	for _, code := range cast.Codes() {
		link := u.Link
		if code.Link == broadcast.LinkDrive {
			link = u.DriveLink
		}
		if link == "" {
			continue
		}
		resp, err := castClient.Post(link+"unlock", "application/json", strings.NewReader(`{"code":"`+code.Code+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unlock %s: %d", code.Link, resp.StatusCode)
		}
	}
}
