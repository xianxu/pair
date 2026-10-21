package mockserve

import (
	"net"
	"net/url"
	"testing"
)

func TestRealRedirectPreservesResolvedDestination(t *testing.T) {
	origin, _ := url.Parse("http://localhost:3000")
	front, _ := url.Parse("http://127.0.0.1:9000/current/page?old=1")
	for _, tc := range []struct{ input, path, query, fragment string }{
		{"http://localhost:3000", "/", "", ""},
		{"http://localhost:3000?next=1", "/", "next=1", ""},
		{"http://localhost:3000#section", "/", "", "section"},
		{"http://localhost:3000?next=1#section", "/", "next=1", "section"},
		{"http://localhost:3000//elsewhere.invalid/screen?x=1#section", "//elsewhere.invalid/screen", "x=1", "section"},
		{"//localhost:3000/detail", "/detail", "", ""},
		{"http://LOCALHOST:03000/detail", "/detail", "", ""},
		{"http://127.0.0.1:3000/detail", "/detail", "", ""},
	} {
		ref, err := url.Parse(rewriteRealLocation(tc.input, origin))
		if err != nil {
			t.Fatal(err)
		}
		resolved := front.ResolveReference(ref)
		if resolved.Host != front.Host || resolved.Path != tc.path || resolved.RawQuery != tc.query || resolved.Fragment != tc.fragment {
			t.Errorf("%s -> %s -> %s", tc.input, ref, resolved)
		}
	}
	for _, raw := range []string{"/relative", "?q=1", "#section", "http://elsewhere.invalid:3000/", "http://[::1]:3000/", "https://localhost:3000/"} {
		if got := rewriteRealLocation(raw, origin); got != raw {
			t.Errorf("unrelated reference changed: %s -> %s", raw, got)
		}
	}
}

func TestFrontSelfTargetUsesCompleteEndpoint(t *testing.T) {
	listener := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3000}
	for _, tc := range []struct {
		raw  string
		self bool
	}{
		{"http://localhost:3000", true}, {"http://127.0.0.1:03000", true},
		{"http://[::1]:3000", false}, {"http://127.0.0.2:3000", false}, {"http://127.0.0.1:3001", false},
	} {
		u, err := parseRealOrigin(tc.raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := targetsListener(u, listener); got != tc.self {
			t.Errorf("%s: self=%v want %v", tc.raw, got, tc.self)
		}
	}
}
