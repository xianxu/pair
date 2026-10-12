package broadcast

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

// Every link is locked (pair#495), so this package's tests talk to casts the
// way a viewer does: through a cookie jar holding the sessions they unlocked.
// A session cookie is scoped to its own link's path, so one jar serves every
// session in the process.
func init() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	http.DefaultClient.Jar = jar
}

// unlockLink enters code at link (a URL ending in "/") and fails the test
// unless the session is issued.
func unlockLink(t *testing.T, link, code string) {
	t.Helper()
	if resp, err := http.Get(link + "session"); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusNoContent {
			return // this jar already holds the link's session
		}
	}
	status, body := post(t, link+"unlock", "application/json", `{"code":"`+code+`"}`)
	if status != http.StatusOK || !strings.Contains(body, `"ok":true`) {
		t.Fatalf("unlock %s: %d %q", link, status, body)
	}
}

// unlockView unlocks s's view link.
func unlockView(t *testing.T, s *Session) {
	t.Helper()
	unlockLink(t, s.Link(), codeOf(t, s, LinkView))
}

func codeOf(t *testing.T, s *Session, link string) string {
	t.Helper()
	for _, c := range s.Codes() {
		if c.Link == link {
			return c.Code
		}
	}
	t.Fatalf("no %s code", link)
	return ""
}

// enablePointer turns pointing on and unlocks the pointer link.
func (p *pointerSession) enablePointer(t *testing.T) (string, error) {
	t.Helper()
	link, err := p.s.EnablePointer()
	if err == nil {
		unlockLink(t, link, codeOf(t, p.s, LinkPointer))
	}
	return link, err
}
