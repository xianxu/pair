package mockserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const MaxCatalogueBytes = 256 * 1024
const MaxScreens = 256

type Screen struct {
	ID    string `json:"id"`
	Route string `json:"route"`
	Mock  string `json:"mock"`
	State string `json:"state"`
}
type Catalogue struct {
	screens []Screen
	routes  map[string]Screen
	mocks   map[string]Screen
	missing []Screen
	real    int
}

// ParseCatalogue validates once and builds the immutable request lookup tables.
// Unknown additive fields are accepted, but are never rewritten by Couch.
func ParseCatalogue(raw []byte) (*Catalogue, error) {
	if len(raw) > MaxCatalogueBytes {
		return nil, fmt.Errorf("screens.json exceeds %d bytes", MaxCatalogueBytes)
	}
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("screens.json: %w", err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		return nil, fmt.Errorf("screens.json must be an object")
	}
	var version int
	if err := requiredCatalogueField(root, "version", &version); err != nil {
		return nil, err
	}
	if version != 1 {
		return nil, fmt.Errorf("screens.json unsupported version %d; use version 1", version)
	}
	var rows []map[string]json.RawMessage
	if err := requiredCatalogueField(root, "screens", &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 || len(rows) > MaxScreens {
		return nil, fmt.Errorf("screens.json requires 1 to %d screens", MaxScreens)
	}
	c := &Catalogue{routes: make(map[string]Screen, len(rows)), mocks: make(map[string]Screen, len(rows))}
	ids := make(map[string]bool, len(rows))
	for i, row := range rows {
		var s Screen
		for _, field := range []struct {
			name string
			out  *string
		}{{"id", &s.ID}, {"route", &s.Route}, {"mock", &s.Mock}, {"state", &s.State}} {
			if err := requiredCatalogueField(row, field.name, field.out); err != nil {
				return nil, fmt.Errorf("screen %d: %w", i+1, err)
			}
		}
		if s.ID == "" || len(s.ID) > 128 || strings.IndexFunc(s.ID, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
			return nil, fmt.Errorf("screen %d: invalid id", i+1)
		}
		if !canonicalScreenRoute(s.Route) {
			return nil, fmt.Errorf("screen %q: invalid canonical route %q", s.ID, s.Route)
		}
		if len(s.Mock) > 2048 {
			return nil, fmt.Errorf("screen %q: mock path too long", s.ID)
		}
		if _, err := ScreenFromPath(s.Mock); err != nil {
			return nil, fmt.Errorf("screen %q: %w", s.ID, err)
		}
		if s.State != "real" && s.State != "mock" {
			return nil, fmt.Errorf("screen %q: state must be real or mock", s.ID)
		}
		if s.Route == "/" && s.Mock != "index.html" {
			return nil, fmt.Errorf("screen %q: root route must use index.html", s.ID)
		}
		if ids[s.ID] {
			return nil, fmt.Errorf("duplicate screen id %q", s.ID)
		}
		if _, ok := c.routes[s.Route]; ok {
			return nil, fmt.Errorf("duplicate screen route %q", s.Route)
		}
		if _, ok := c.mocks[s.Mock]; ok {
			return nil, fmt.Errorf("duplicate screen mock %q", s.Mock)
		}
		ids[s.ID] = true
		c.routes[s.Route] = s
		c.mocks[s.Mock] = s
		c.screens = append(c.screens, s)
		if s.State == "real" {
			c.real++
		} else {
			c.missing = append(c.missing, s)
		}
	}
	if _, ok := c.routes["/"]; !ok {
		return nil, fmt.Errorf("screens.json requires one root / screen using index.html")
	}
	return c, nil
}

func requiredCatalogueField(fields map[string]json.RawMessage, name string, out any) error {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("screens.json field %s is required and cannot be null", name)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("screens.json field %s: %w", name, err)
	}
	return nil
}

func canonicalScreenRoute(route string) bool {
	if route == "" || len(route) > 2048 || !strings.HasPrefix(route, "/") || path.Clean(route) != route || strings.ContainsAny(route, "\\%?#:*{}") || strings.IndexFunc(route, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return false
	}
	first := strings.SplitN(strings.TrimPrefix(route, "/"), "/", 2)[0]
	return !strings.HasPrefix(first, ".") && !strings.HasPrefix(route, "/__couch_mock") && !strings.HasPrefix(route, "/mock-") && first != "fixtures.json" && first != "screens.json" && first != "design.md"
}

func (c *Catalogue) Screens() []Screen                        { return append([]Screen(nil), c.screens...) }
func (c *Catalogue) ResolveRoute(route string) (Screen, bool) { s, ok := c.routes[route]; return s, ok }
func (c *Catalogue) ResolveMock(mock string) (Screen, bool)   { s, ok := c.mocks[mock]; return s, ok }
func (c *Catalogue) Progress() (int, int)                     { return c.real, len(c.screens) }
func (c *Catalogue) Missing() []Screen                        { return append([]Screen(nil), c.missing...) }
func (c *Catalogue) Report() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Screen progress: %d/%d real\n", c.real, len(c.screens))
	for _, s := range c.missing {
		fmt.Fprintf(&b, "Missing: %s %s\n", s.ID, s.Route)
	}
	return b.String()
}
