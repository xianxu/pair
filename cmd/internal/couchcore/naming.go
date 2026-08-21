package couchcore

import "strings"

// NameEntry is what a human or an agent has said about a tree.
//
// Tree carries the original-case path. The table is keyed on the folded form
// so lookups collide correctly, but a Worktree handed back to a caller must
// always be the unfolded path: it is fed to launcher.ResolveRepoScope, which
// hashes the raw string, and it is what gets displayed.
type NameEntry struct {
	Tree        Worktree `json:"tree"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
}

// NamingTable maps a worktree key to its labels.
//
// Labels attach to the TREE, not to the actor id: the id is per-incarnation
// and dies with it, so hanging names off it would re-impose on every revival
// exactly the memory load the naming layer exists to remove.
//
// Nothing structural depends on a label, so one may be wrong, duplicated or
// stale without corrupting anything. Lookup therefore returns every candidate
// and the caller disambiguates.
type NamingTable struct {
	byTree map[string]NameEntry
}

func NewNamingTable() NamingTable { return NamingTable{byTree: map[string]NameEntry{}} }

func (n NamingTable) Entry(w Worktree) NameEntry { return n.byTree[w.Key()] }

func (n NamingTable) SetName(w Worktree, name string) NamingTable {
	return n.with(w, func(e *NameEntry) { e.Name = name })
}

func (n NamingTable) SetDescription(w Worktree, desc string) NamingTable {
	return n.with(w, func(e *NameEntry) { e.Description = desc })
}

// Lookup matches ref case-insensitively as a substring of either label and
// returns every tree that matches.
func (n NamingTable) Lookup(ref string) []Worktree {
	needle := strings.ToLower(strings.TrimSpace(ref))
	if needle == "" {
		return nil
	}
	var out []Worktree
	for _, e := range n.byTree {
		if strings.Contains(strings.ToLower(e.Name), needle) ||
			strings.Contains(strings.ToLower(e.Description), needle) {
			out = append(out, e.Tree)
		}
	}
	return out
}

func (n NamingTable) All() map[string]NameEntry { return n.copyMap() }

func (n NamingTable) with(w Worktree, mutate func(*NameEntry)) NamingTable {
	next := NamingTable{byTree: n.copyMap()}
	e := next.byTree[w.Key()]
	e.Tree = w
	mutate(&e)
	next.byTree[w.Key()] = e
	return next
}

func (n NamingTable) copyMap() map[string]NameEntry {
	out := make(map[string]NameEntry, len(n.byTree))
	for k, v := range n.byTree {
		out[k] = v
	}
	return out
}
