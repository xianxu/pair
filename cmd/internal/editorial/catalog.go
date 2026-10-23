// Package editorial compiles the document settings for an editorial round.
package editorial

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/ariadne/pkg/vocab"
)

// Scope names the portion of a document a processing brief may change.
type Scope string

const (
	WholeDocument Scope = "whole-doc"
	MarkersOnly   Scope = "markers-only"
)

// Mode is a processing brief and its pinned metadata. Brief preserves the
// source body bytes, including its final newline.
type Mode struct {
	Name     string
	Order    int
	Scope    Scope
	Frontier bool
	Brief    string
}

// Catalog is an immutable, ordered collection of processing modes.
type Catalog struct{ modes []Mode }

//go:embed modes/*.md
var embeddedModes embed.FS

// DefaultCatalog reads the canonical mode files embedded in this binary.
func DefaultCatalog() (*Catalog, error) { return NewCatalog(embeddedModes) }

// NewCatalog reads modes/*.md from source. It accepts the flat scalar metadata
// contract of the canonical mode files, rejecting unsupported syntax.
func NewCatalog(source fs.FS) (*Catalog, error) {
	paths, err := fs.Glob(source, "modes/*.md")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("editorial catalog has no modes")
	}
	c := &Catalog{}
	names, orders := map[string]bool{}, map[int]bool{}
	for _, path := range paths {
		raw, err := fs.ReadFile(source, path)
		if err != nil {
			return nil, err
		}
		m, err := parseMode(string(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if names[m.Name] || orders[m.Order] {
			return nil, fmt.Errorf("%s: duplicate mode name or order", path)
		}
		names[m.Name], orders[m.Order] = true, true
		c.modes = append(c.modes, m)
	}
	sort.Slice(c.modes, func(i, j int) bool { return c.modes[i].Order < c.modes[j].Order })
	return c, nil
}

func parseMode(raw string) (Mode, error) {
	var m Mode
	if !strings.HasPrefix(raw, "---\n") {
		return m, fmt.Errorf("missing metadata delimiter")
	}
	header, body, ok := strings.Cut(raw[4:], "\n---\n")
	if !ok {
		return m, fmt.Errorf("missing metadata closing delimiter")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(header, "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if !ok || value == "" || strings.TrimSpace(value) != value {
			return m, fmt.Errorf("malformed metadata line %q", line)
		}
		switch key {
		case "name", "order", "scope", "frontier":
		default:
			return m, fmt.Errorf("unknown metadata key %q", key)
		}
		if _, exists := fields[key]; exists {
			return m, fmt.Errorf("duplicate metadata key %q", key)
		}
		fields[key] = value
	}
	if len(fields) != 4 {
		return m, fmt.Errorf("missing required metadata")
	}
	m.Name = fields["name"]
	for i, ch := range m.Name {
		if (ch < 'a' || ch > 'z') && (i == 0 || ch != '-') {
			return Mode{}, fmt.Errorf("invalid mode name %q", m.Name)
		}
	}
	var err error
	m.Order, err = strconv.Atoi(fields["order"])
	if err != nil || m.Order < 1 {
		return Mode{}, fmt.Errorf("invalid mode order %q", fields["order"])
	}
	m.Scope = Scope(fields["scope"])
	if m.Scope != WholeDocument && m.Scope != MarkersOnly {
		return Mode{}, fmt.Errorf("invalid mode scope %q", m.Scope)
	}
	switch fields["frontier"] {
	case "on":
		m.Frontier = true
	case "off":
	default:
		return Mode{}, fmt.Errorf("invalid mode frontier %q", fields["frontier"])
	}
	if strings.TrimSpace(body) == "" {
		return Mode{}, fmt.Errorf("missing mode brief")
	}
	m.Brief = body
	return m, nil
}

// Modes returns a copy of the modes in metadata order.
func (c *Catalog) Modes() []Mode { return append([]Mode(nil), c.modes...) }

// Lookup finds an exact canonical processing mode name.
func (c *Catalog) Lookup(name string) (Mode, bool) {
	for _, m := range c.modes {
		if m.Name == name {
			return m, true
		}
	}
	return Mode{}, false
}

// ResolveProcessing accepts an exact name or an unambiguous nonempty prefix.
// Composite legacy selectors are parsed by the source compiler, not here.
func (c *Catalog) ResolveProcessing(selector string) (Mode, error) {
	if m, ok := c.Lookup(selector); ok {
		return m, nil
	}
	var match Mode
	count := 0
	if selector != "" {
		for _, m := range c.modes {
			if strings.HasPrefix(m.Name, selector) {
				match = m
				count++
			}
		}
	}
	if count == 1 {
		return match, nil
	}
	if count > 1 {
		return Mode{}, fmt.Errorf("ambiguous processing mode %q", selector)
	}
	return Mode{}, fmt.Errorf("unknown processing mode %q", selector)
}

// ResolveCollaboration validates an exact collaboration value against its
// vocabulary owner. It supplies no document default.
func ResolveCollaboration(value string) (string, error) {
	for _, name := range vocab.CollaborationModes {
		if name == value {
			return name, nil
		}
	}
	return "", fmt.Errorf("unknown collaboration mode %q", value)
}

// ResolvePresentation validates an exact presentation value. Document fallback
// and diagnostics are the source compiler's responsibility.
func ResolvePresentation(value string) (string, error) {
	switch value {
	case "highlight", "direct", "propose":
		return value, nil
	}
	return "", fmt.Errorf("unknown presentation mode %q", value)
}
