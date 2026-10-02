// Package couchsingleton owns the immutable local Couch selection and admission.
package couchsingleton

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
)

type Roots struct {
	StoreDir    string `json:"store_dir"`
	PairDataDir string `json:"pair_data_dir"`
	IdentityDir string `json:"identity_dir"`
}
type Selection struct {
	Version  int      `json:"version"`
	Roots    Roots    `json:"roots"`
	Excluded []string `json:"excluded,omitempty"`
}
type Request struct {
	Roots   Roots    `json:"roots"`
	Stores  []string `json:"stores,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}
type Candidate struct {
	Roots      Roots  `json:"roots"`
	State      string `json:"state"`
	Owner      string `json:"owner,omitempty"`
	Problem    string `json:"problem,omitempty"`
	Registered bool   `json:"registered,omitempty"`
}
type Report struct {
	Digest     string      `json:"digest"`
	Selection  Selection   `json:"selection"`
	Candidates []Candidate `json:"candidates"`
	Blockers   []string    `json:"blockers"`
	Status     string      `json:"status"`
}

func validateRoots(r Roots) error {
	for _, p := range []string{r.StoreDir, r.PairDataDir, r.IdentityDir} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return fmt.Errorf("root must be absolute and clean: %q", p)
		}
	}
	if r.StoreDir == r.PairDataDir || r.IdentityDir == r.StoreDir || r.IdentityDir == r.PairDataDir {
		return errors.New("store, Pair data and identity roots must be distinct")
	}
	return nil
}

// DecideAdoption admits only known inactive, unambiguous state. Exclusions keep
// their data intact and cannot turn unreadable or potentially live state into proof.
func DecideAdoption(q Request, defaults Roots, candidates []Candidate) (Selection, error) {
	r := defaults
	if q.Roots.StoreDir != "" {
		r.StoreDir = q.Roots.StoreDir
	}
	if q.Roots.PairDataDir != "" {
		r.PairDataDir = q.Roots.PairDataDir
	}
	if q.Roots.IdentityDir != "" {
		r.IdentityDir = q.Roots.IdentityDir
	}
	if e := validateRoots(r); e != nil {
		return Selection{}, e
	}
	excluded := append([]string(nil), q.Exclude...)
	slices.Sort(excluded)
	for i, p := range excluded {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || i > 0 && p == excluded[i-1] {
			return Selection{}, errors.New("invalid or duplicate exclusion")
		}
	}
	var populated []Candidate
	seen := map[string]bool{}
	for _, c := range candidates {
		if e := validateRoots(c.Roots); e != nil {
			return Selection{}, e
		}
		switch c.State {
		case "empty", "populated", "missing", "unknown":
		default:
			return Selection{}, fmt.Errorf("unknown candidate state %q", c.State)
		}
		p := c.Roots.StoreDir
		if seen[p] {
			return Selection{}, fmt.Errorf("duplicate candidate %s", p)
		}
		seen[p] = true
		if c.Owner != "" || c.State == "unknown" || c.Problem != "" {
			return Selection{}, fmt.Errorf("UNMIGRATED: %s: %s %s", p, c.Problem, c.Owner)
		}
		if slices.Contains(excluded, p) {
			continue
		}
		if c.State == "missing" && c.Registered {
			return Selection{}, fmt.Errorf("UNMIGRATED: missing registered store %s needs explicit disposition", p)
		}
		if c.State == "populated" {
			populated = append(populated, c)
		}
	}
	for _, p := range excluded {
		if !seen[p] {
			return Selection{}, fmt.Errorf("unknown exclusion %s", p)
		}
	}
	if len(populated) > 1 {
		return Selection{}, errors.New("UNMIGRATED: multiple populated stores; preserve backups and explicitly exclude retired inventories")
	}
	if len(populated) == 1 {
		chosen := populated[0].Roots
		if q.Roots.StoreDir != "" && q.Roots.StoreDir != chosen.StoreDir {
			return Selection{}, errors.New("UNMIGRATED: requested store conflicts with populated inventory")
		}
		r = chosen
	}
	if slices.Contains(excluded, r.StoreDir) {
		return Selection{}, errors.New("selected store cannot be excluded")
	}
	return Selection{Version: 1, Roots: r, Excluded: excluded}, nil
}
