// Package catalogue is a read-only development consumer of mocks/screens.json.
package catalogue

import (
	"encoding/json"
	"example.com/mock-to-real/internal/app"
	"example.com/mock-to-real/internal/jsonvalue"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"
)

const MaxCatalogueBytes = 256 * 1024
const MaxScreens = 256

func Read(r io.Reader) ([]app.Screen, error) {
	b, e := io.ReadAll(io.LimitReader(r, MaxCatalogueBytes+1))
	if e != nil {
		return nil, e
	}
	if len(b) > MaxCatalogueBytes {
		return nil, fmt.Errorf("screens.json exceeds %d bytes", MaxCatalogueBytes)
	}
	v, e := jsonvalue.Decode(b)
	if e != nil {
		return nil, e
	}
	root, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("catalogue must be object")
	}
	version, ok := root["version"].(json.Number)
	if !ok || string(version) != "1" {
		return nil, fmt.Errorf("version must be 1")
	}
	rows, ok := root["screens"].([]any)
	if !ok {
		return nil, fmt.Errorf("screens must be array")
	}
	if len(rows) == 0 || len(rows) > MaxScreens {
		return nil, fmt.Errorf("screens.json requires 1 to %d screens", MaxScreens)
	}
	result := make([]app.Screen, 0, len(rows))
	ids := map[string]bool{}
	routes := map[string]bool{}
	mocks := map[string]bool{}
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("screen must be object")
		}
		fields := map[string]string{}
		for _, k := range []string{"id", "route", "mock", "state"} {
			s, ok := m[k].(string)
			if !ok || s == "" {
				return nil, fmt.Errorf("%s must be nonempty string", k)
			}
			fields[k] = s
		}
		id, route, mock, state := fields["id"], fields["route"], fields["mock"], fields["state"]
		if len(id) > 128 || strings.IndexFunc(id, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 || !canonicalRoute(route) || !mockPath(mock) || (route == "/" && mock != "index.html") || (state != "mock" && state != "real") || ids[id] || routes[route] || mocks[mock] {
			return nil, fmt.Errorf("invalid or ambiguous screen %q", id)
		}
		ids[id] = true
		routes[route] = true
		mocks[mock] = true
		result = append(result, app.Screen{ID: id, Route: route, State: state})
	}
	if !routes["/"] {
		return nil, fmt.Errorf("screens.json requires one root / screen using index.html")
	}
	return result, nil
}

// Keep these standalone convention checks aligned with the Couch catalogue wire
// contract. The worked sample intentionally has no production Pair dependency.
func canonicalRoute(route string) bool {
	if len(route) > 2048 || !strings.HasPrefix(route, "/") || path.Clean(route) != route || strings.ContainsAny(route, `\%?#:*{}`) || strings.IndexFunc(route, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return false
	}
	first := strings.SplitN(strings.TrimPrefix(route, "/"), "/", 2)[0]
	return !strings.HasPrefix(first, ".") && !strings.HasPrefix(route, "/__couch_mock") && !strings.HasPrefix(route, "/mock-") && first != "fixtures.json" && first != "screens.json" && first != "design.md"
}
func mockPath(name string) bool {
	if len(name) > 2048 || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.ContainsAny(name, "\\\x00?#") || !strings.HasSuffix(name, ".html") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." || part == ".html" {
			return false
		}
	}
	return true
}
