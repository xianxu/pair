package couchcore

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// OrphanNvim is one untagged embedded nvim the sweep could not attribute to a
// live slot. Reporting it is unconditional; reaping it needs --confirm (#583).
type OrphanNvim struct {
	Key      ResourceProcessKey `json:"key"`
	Age      time.Duration      `json:"age"`
	Cwd      string             `json:"cwd"`
	Root     string             `json:"root"`
	RSSBytes *uint64            `json:"rss_bytes"`
	Reason   string             `json:"reason"`
}

// ResourceOrphanProbe is what the sampler's IO proved about one candidate beyond
// its usage row and argv. A missing claim refuses the candidate, never widens it.
type ResourceOrphanProbe struct {
	Cwd      string
	CwdKnown bool
	Age      time.Duration
	AgeKnown bool
}

// ManagedRoots are the fleet's own checkouts. An unattributed process inside one
// is this fleet's business; anywhere else on the host is not.
func ManagedRoots(slots []ResourceSlot) []string {
	seen := map[string]bool{}
	roots := []string{}
	for _, slot := range slots {
		root := slot.Workspace.WorktreeRoot
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots
}

// WithinManagedRoot names the managed checkout cwd is at or inside. The longest
// match wins, so a checkout nested in another reports the specific one.
func WithinManagedRoot(cwd string, roots []string) (string, bool) {
	match := ""
	for _, root := range roots {
		if _, err := RelativeFamilyPath(root, cwd); err != nil {
			continue
		}
		if len(root) > len(match) {
			match = root
		}
	}
	return match, match != ""
}

// ClassifyOrphanNvim decides whether one candidate is an untagged embedded nvim
// whose slot is gone. It is pure: the sampler owns the probes and hands over
// their results, so an unproven claim never reaches for the host.
func ClassifyOrphanNvim(row procutil.Process, args []string, claimed map[ResourceProcessKey]bool, probe ResourceOrphanProbe, rss *uint64, roots []string, minAge time.Duration) (OrphanNvim, bool) {
	if filepath.Base(row.Name) != "nvim" || !UnattributedCandidate(row, claimed, args) {
		return OrphanNvim{}, false
	}
	if !probe.CwdKnown || probe.Cwd == "" {
		return OrphanNvim{}, false
	}
	if !probe.AgeKnown || probe.Age < 0 || probe.Age < minAge {
		return OrphanNvim{}, false
	}
	root, ok := WithinManagedRoot(probe.Cwd, roots)
	if !ok {
		return OrphanNvim{}, false
	}
	return OrphanNvim{Key: resourceKey(row), Age: probe.Age, Cwd: probe.Cwd, Root: root, RSSBytes: rss, Reason: "unattributed embedded nvim"}, true
}

// ClassifyOrphanNvims sweeps one sample's table. A row the sampler never probed
// is unproven and stays out of the report; probes must have been
// generation-fenced by the caller around the cwd IO, as with commands.
func ClassifyOrphanNvims(table []procutil.Process, owned []ResourceProvenance, observations []ResourceUsageObservation, commands map[ResourceProcessKey][]string, probes map[ResourceProcessKey]ResourceOrphanProbe, roots []string, minAge time.Duration) []OrphanNvim {
	claimed := map[ResourceProcessKey]bool{}
	for _, p := range owned {
		claimed[p.Key] = true
	}
	obs := map[ResourceProcessKey]ResourceUsageObservation{}
	for _, o := range observations {
		obs[o.Key] = o
	}
	out := []OrphanNvim{}
	for i, p := range table {
		if i >= ResourceMaxTableRows || len(out) >= ResourceMaxProcesses {
			break
		}
		k := resourceKey(p)
		probe, probed := probes[k]
		if !probed {
			continue
		}
		var rss *uint64
		if o := obs[k]; o.RSSKnown {
			bytes := o.RSSBytes
			rss = &bytes
		}
		if orphan, ok := ClassifyOrphanNvim(p, commands[k], claimed, probe, rss, roots, minAge); ok {
			out = append(out, orphan)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key.PID < out[j].Key.PID })
	return out
}
