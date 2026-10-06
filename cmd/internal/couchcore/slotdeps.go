package couchcore

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/xianxu/ariadne/pkg/layergraph"
)

// DeclaredDep is one dependency a slot's construct/deps graph declares
// (pair#387), read through ariadne's layergraph so weave and Couch share one
// parser and one resolver.
type DeclaredDep struct {
	// Rel is the path relative to the slot environment ("" when Outside).
	Rel string
	// Path is the physical, absolute directory.
	Path string
	// Owner is the checkout whose construct/deps declares it.
	Owner string
	// Source is the row's clone source, "" when the row has none.
	Source string
	// Present: the directory exists.
	Present bool
	// Outside: the dependency does not live directly in the slot
	// environment, so it is not slot state; reconcile never converges it.
	Outside bool
}

// DeclaredDeps is a slot's dependency graph as far as it could be walked.
type DeclaredDeps struct {
	Deps []DeclaredDep
	// NotLayer is a present dependency without construct/base.manifest (an
	// interrupted or gutted clone). The walk stops there, so Deps is empty:
	// what lies beyond it is not known until it is repaired.
	NotLayer *DeclaredDep
}

// DeclaredDepsOf walks the construct/deps graph of the host checkout in the
// slot environment env. An error (a malformed row, an unreadable declaration)
// means the declaration is unknown, never that there are no dependencies.
func DeclaredDepsOf(env, host string) (DeclaredDeps, error) {
	physicalEnv, err := filepath.EvalSymlinks(env)
	if err != nil {
		return DeclaredDeps{}, fmt.Errorf("resolve slot environment: %w", err)
	}
	declared, err := layergraph.DeclaredSubstrates(layergraph.OSFS{}, host)
	var notLayer *layergraph.NotLayerError
	if errors.As(err, &notLayer) {
		d := declaredDep(physicalEnv, notLayer.Path, notLayer.Owner, "", true)
		return DeclaredDeps{NotLayer: &d}, nil
	}
	if err != nil {
		return DeclaredDeps{}, err
	}
	out := DeclaredDeps{Deps: make([]DeclaredDep, 0, len(declared))}
	for _, d := range declared {
		out.Deps = append(out.Deps, declaredDep(physicalEnv, d.Path, d.Owner, d.Source, d.Present))
	}
	return out, nil
}

func declaredDep(env, path, owner, source string, present bool) DeclaredDep {
	d := DeclaredDep{Path: path, Owner: owner, Source: source, Present: present}
	if filepath.Dir(path) == env && filepath.Base(path) != "." {
		d.Rel = filepath.Base(path)
	} else {
		d.Outside = true
	}
	return d
}
