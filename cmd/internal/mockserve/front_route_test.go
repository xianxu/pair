package mockserve

import (
	"net/url"
	"strings"
	"testing"
)

func TestDecideFrontRoute(t *testing.T) {
	c, err := ParseCatalogue([]byte(testCatalogue))
	if err != nil {
		t.Fatal(err)
	}
	home, _ := c.ResolveRoute("/")
	runs, _ := c.ResolveRoute("/runs")
	detail, _ := c.ResolveRoute("/runs/detail")
	for _, tc := range []struct {
		target string
		want   frontRoute
	}{
		{"/", frontRoute{Kind: routeReal, Screen: home}},
		{"/runs?sort=recent", frontRoute{Kind: routeMockRedirect, Screen: runs, Version: "mock-2", Relative: "runs.html"}},
		{"/runs/detail", frontRoute{Kind: routeMockRedirect, Screen: detail, Version: "mock-2", Relative: "nested/detail.html"}},
		{"/mock-2/runs.html", frontRoute{Kind: routeMockDocument, Screen: runs, Version: "mock-2", Relative: "runs.html"}},
		{"/mock-1/nested/detail.html", frontRoute{Kind: routeMockRedirect, Screen: detail, Version: "mock-2", Relative: "nested/detail.html"}},
		{"/mock-9/index.html", frontRoute{Kind: routeRealRedirect, Screen: home, Version: "mock-9", Relative: "index.html"}},
		{"/mock-2/tokens.css", frontRoute{Kind: routeAsset, Version: "mock-2", Relative: "tokens.css"}},
		{"/mock-1/nested/app.js", frontRoute{Kind: routeAsset, Version: "mock-1", Relative: "nested/app.js"}},
		{statePath, frontRoute{Kind: routeInternal}},
		{reloadPath, frontRoute{Kind: routeInternal}},
		{"/fixtures.json", frontRoute{Kind: routeFixtures}},
		{"/framework/asset.js", frontRoute{Kind: routeReal}},
		{"/api/save?x=y", frontRoute{Kind: routeReal}},
		{"/unknown.html", frontRoute{Kind: routeReal}},
		{"/mockish/normal", frontRoute{Kind: routeReal}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			u, err := url.Parse(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			if got := decideFrontRoute(u, c, "mock-2"); got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
	for _, target := range []string{"", "relative", "//runs", "/a/../b", "/a/./b", "/a//b", "/a/", "/a\\b", "/a\x00b", "/mock-2/%2fruns.html", "/mock-2/a%5cb.html", "/mock-2/%2e%2e/private", "/__couch_mock", "/__couch_mock/unknown", "/__couch_mockish", "/__couch_mock/state/child", "/fixtures.json/child", "/screens.json", "/screens.json/child", "/design.md", "/.secret", "/.secret/child", "/mock-1", "/mock-1/", "/mock-other/file.js", "/mock-0/file.js", "/mock-01/file.js", "/mock-1aA/file.js", "/mock-2/unknown.html", "/mock-2/nested/UNKNOWN.HTML"} {
		t.Run(target, func(t *testing.T) {
			u, err := url.Parse(target)
			if err != nil {
				return
			}
			if got := decideFrontRoute(u, c, "mock-2"); got.Kind != routeNotFound {
				t.Fatalf("accepted %q: %+v", target, got)
			}
		})
	}
	if got := decideFrontRoute(nil, c, "mock-2"); got.Kind != routeNotFound {
		t.Fatal(got)
	}
}

func TestDecideFrontRouteWithoutCatalogue(t *testing.T) {
	for _, target := range []string{"/", "/mock-2/index.html", "/api/save"} {
		u, _ := url.Parse(target)
		if got := decideFrontRoute(u, nil, "mock-2"); got.Kind != routeUnavailable {
			t.Fatal(target, got)
		}
	}
	for target, want := range map[string]frontRouteKind{statePath: routeInternal, reloadPath: routeInternal, "/fixtures.json": routeFixtures} {
		u, _ := url.Parse(target)
		if got := decideFrontRoute(u, nil, "mock-2"); got.Kind != want {
			t.Fatal(target, got)
		}
	}
}

func FuzzDecideFrontRoute(f *testing.F) {
	for _, target := range []string{"/", "/runs", "/mock-2/runs.html", "/mock-1/index.html", "/mock-2/tokens.css", "/__couch_mockish", "/fixtures.json/child", "/mock-2/%2fruns.html", "/a/../b", "/.private", "relative"} {
		f.Add(target)
	}
	c, err := ParseCatalogue([]byte(testCatalogue))
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, target string) {
		u, err := url.Parse(target)
		if err != nil {
			return
		}
		got := decideFrontRoute(u, c, "mock-2")
		if got != decideFrontRoute(u, c, "mock-2") {
			t.Fatal("nondeterministic route")
		}
		if (!validFrontPath(u) || privateFrontPath(u.Path)) && got.Kind != routeNotFound {
			t.Fatal("unsafe path admitted", target, got)
		}
		if strings.HasPrefix(u.Path, "/__couch_mock") && u.Path != statePath && u.Path != reloadPath && got.Kind != routeNotFound {
			t.Fatal("reserved namespace escaped", target, got)
		}
		if strings.HasPrefix(u.Path, "/fixtures.json/") && got.Kind != routeNotFound {
			t.Fatal("fixture child escaped", target, got)
		}
		if got.Kind == routeMockDocument || got.Kind == routeMockRedirect {
			if got.Screen.State != "mock" || got.Relative != got.Screen.Mock || got.Version != "mock-2" {
				t.Fatal("incorrect mock target", got)
			}
		}
		if got.Kind == routeRealRedirect && got.Screen.State != "real" {
			t.Fatal("mock screen redirected to app", got)
		}
	})
}
