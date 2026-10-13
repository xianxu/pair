package webcast

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// A cast's links are locked (pair#495), so this package's tests reach them as
// a viewer does: through castClient, whose cookie jar holds the sessions they
// unlocked. A session cookie is scoped to its own link's path, so one jar
// serves every cast; http.DefaultClient is left alone.
var castClient = func() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &http.Client{Jar: jar}
}()

// unlockCast enters c's codes at u's links: the view link, and the drive link
// when the cast has one.
func unlockCast(t *testing.T, c *Cast, u Update) {
	t.Helper()
	for _, code := range c.Codes() {
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
