package mockserve

import (
	"net/url"
	"strings"
)

type frontRouteKind uint8

const (
	routeNotFound frontRouteKind = iota
	routeInternal
	routeFixtures
	routeReal
	routeMockRedirect
	routeMockDocument
	routeRealRedirect
	routeAsset
	routeUnavailable
)

type frontRoute struct {
	Kind     frontRouteKind
	Screen   Screen
	Version  string
	Relative string
}

// decideFrontRoute selects an owner from one immutable catalogue snapshot.
// HTTP methods, project health and delivery effects belong to the caller.
func decideFrontRoute(u *url.URL, c *Catalogue, selected string) frontRoute {
	if u == nil || !validFrontPath(u) || privateFrontPath(u.Path) {
		return frontRoute{Kind: routeNotFound}
	}
	if strings.HasPrefix(u.Path, "/__couch_mock") {
		if u.Path == statePath || u.Path == reloadPath {
			return frontRoute{Kind: routeInternal}
		}
		return frontRoute{Kind: routeNotFound}
	}
	if u.Path == "/fixtures.json" {
		return frontRoute{Kind: routeFixtures}
	}
	if strings.HasPrefix(u.Path, "/fixtures.json/") {
		return frontRoute{Kind: routeNotFound}
	}
	if c == nil {
		return frontRoute{Kind: routeUnavailable}
	}
	if s, ok := c.ResolveRoute(u.Path); ok {
		if s.State == "real" {
			return frontRoute{Kind: routeReal, Screen: s}
		}
		return frontRoute{Kind: routeMockRedirect, Screen: s, Version: selected, Relative: s.Mock}
	}
	version, relative, slash := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	if strings.HasPrefix(version, "mock-") {
		if _, err := ParseVersion(version); err != nil || !slash || relative == "" {
			return frontRoute{Kind: routeNotFound}
		}
		if s, ok := c.ResolveMock(relative); ok {
			if s.State == "real" {
				return frontRoute{Kind: routeRealRedirect, Screen: s, Version: version, Relative: relative}
			}
			kind := routeMockDocument
			if version != selected {
				kind = routeMockRedirect
				version = selected
			}
			return frontRoute{Kind: kind, Screen: s, Version: version, Relative: relative}
		}
		if strings.HasSuffix(strings.ToLower(relative), ".html") {
			return frontRoute{Kind: routeNotFound}
		}
		return frontRoute{Kind: routeAsset, Version: version, Relative: relative}
	}
	return frontRoute{Kind: routeReal}
}
