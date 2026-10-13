package webcast

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
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

// unlockCast enters each of c's codes at its own link in u (pair#506): the
// view-only link, and the pointer and drive links when the cast has them.
func unlockCast(t *testing.T, c *Cast, u Update) {
	t.Helper()
	for _, code := range c.Codes() {
		link := u.LinkFor(code.Link)
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
