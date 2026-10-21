// Package testcouch owns disposable native Couch fleets for opt-in acceptance.
package testcouch

import (
	"fmt"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/pathscope"
)

type RootsPlan struct {
	Root    string
	Mutable map[string]string
	Denied  []string
}

// ValidateRoots decides confinement over already-physical path facts. It does
// no discovery, creation, or cleanup, so refusal precedes all process effects.
func ValidateRoots(p RootsPlan) error {
	clean := func(s string) bool {
		return filepath.IsAbs(s) && filepath.Clean(s) == s && s != string(filepath.Separator)
	}
	if !clean(p.Root) || len(p.Mutable) == 0 {
		return fmt.Errorf("fresh canonical owned root and mutable paths required")
	}
	for _, live := range p.Denied {
		if !clean(live) {
			return fmt.Errorf("invalid live deny path %q", live)
		}
		if pathscope.Within(live, p.Root) || pathscope.Within(p.Root, live) {
			return fmt.Errorf("owned root overlaps live path %q", live)
		}
	}
	for name, path := range p.Mutable {
		if name == "" || !clean(path) || !pathscope.Within(p.Root, path) {
			return fmt.Errorf("%s path escapes owned root: %q", name, path)
		}
	}
	return nil
}

func physicalRoots(p RootsPlan) (RootsPlan, error) {
	if err := ValidateRoots(p); err != nil {
		return RootsPlan{}, err
	}
	root, err := pathscope.CanonicalFuture(p.Root)
	if err != nil {
		return RootsPlan{}, err
	}
	result := RootsPlan{Root: root, Mutable: make(map[string]string, len(p.Mutable))}
	for key, path := range p.Mutable {
		result.Mutable[key], err = pathscope.CanonicalFuture(path)
		if err != nil {
			return RootsPlan{}, err
		}
	}
	for _, path := range p.Denied {
		physical, e := pathscope.CanonicalFuture(path)
		if e != nil {
			return RootsPlan{}, e
		}
		result.Denied = append(result.Denied, physical)
	}
	return result, ValidateRoots(result)
}
