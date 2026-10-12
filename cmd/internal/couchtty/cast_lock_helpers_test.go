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

// unlockCastLink enters s's code for which ("view" or "pointer") at link.
func unlockCastLink(t *testing.T, s *broadcast.Session, which, link string) {
	t.Helper()
	code := ""
	for _, c := range s.Codes() {
		if c.Link == which {
			code = c.Code
		}
	}
	resp, err := castClient.Post(link+"unlock", "application/json", strings.NewReader(`{"code":"`+code+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unlock %s: %d", which, resp.StatusCode)
	}
}
