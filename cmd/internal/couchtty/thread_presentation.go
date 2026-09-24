package couchtty

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// ThreadPresentation is a display-only projection of one actionable inventory row.
// Row retains valid targets; malformed targets fall back to the row's native
// address before either display or selection can consume them.
type ThreadPresentation struct {
	Row        couchcore.ActionableThreadSummary
	GroupKey   string
	Label      string
	Path       string
	Indent     int
	SlotNumber int
	// Glyph is the slot quick-status glyph (pair#317), derived here once so the
	// switcher and the tab bar cannot disagree. Empty without an observation.
	Glyph string
}

type threadPresentationGroup struct {
	key, root, name string
	hasSlots        bool
	rows            []ThreadPresentation
}

// PresentThreads groups and orders inventory rows without filesystem discovery or
// synthetic headers. Root recovery is best-effort: immutable starting paths and
// their lexical ancestors can prove a scope match without consulting current cwd.
// git holds the last slot git observations by checkout path; nil shows no glyphs.
func PresentThreads(rows []couchcore.ActionableThreadSummary, git map[string]couchcore.SlotGitStatus) []ThreadPresentation {
	groups := make(map[string]*threadPresentationGroup)
	for _, row := range rows {
		row = presentationRow(row)
		p := ThreadPresentation{Row: row, GroupKey: row.Address.RepoScope}
		root := ""
		validSlot := row.Target.Kind == couchcore.ThreadTargetSlot
		if validSlot {
			slot := row.Target.Slot
			scope, _ := launcher.ResolveRepoScope(slot.PrimaryRoot)
			p.GroupKey = scope.Key
			root = slot.PrimaryRoot
			p.SlotNumber = slot.Number
			p.Indent = 2
			p.Path = slot.WorktreeRoot
			p.Glyph = slotGlyphAt(git, p.Path, slot.Number)
		} else {
			if p.GroupKey == "" {
				p.GroupKey = "native:" + fmt.Sprintf("%q", row.Address)
			}
			root = presentationRoot(row.StartingPath, row.Address.RepoScope)
		}
		g := groups[p.GroupKey]
		if g == nil {
			g = &threadPresentationGroup{key: p.GroupKey}
			groups[p.GroupKey] = g
		}
		if root != "" && (g.root == "" || root < g.root) {
			g.root = root
		}
		g.hasSlots = g.hasSlots || validSlot
		g.rows = append(g.rows, p)
	}
	ordered := make([]*threadPresentationGroup, 0, len(groups))
	names := make(map[string]int)
	for _, g := range groups {
		g.name = g.key
		if g.root != "" {
			g.name = filepath.Base(g.root)
		}
		names[strings.ToLower(g.name)]++
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := strings.ToLower(ordered[i].name), strings.ToLower(ordered[j].name)
		if a != b {
			return a < b
		}
		return ordered[i].key < ordered[j].key
	})
	ordinary := make([]couchcore.LabelRow, 0, len(rows))
	for _, g := range ordered {
		for i := range g.rows {
			p := &g.rows[i]
			if p.SlotNumber > 0 {
				p.Label = g.name + ":" + strconv.Itoa(p.SlotNumber)
			} else {
				p.Label = p.Row.Label()
				if g.hasSlots {
					p.Label = g.name
				}
				p.Path = g.root
				if p.Path == "" {
					p.Path = p.Row.WorkingPath
				}
				if p.Path == "" {
					p.Path = p.Row.StartingPath
				}
				if p.Path == "" {
					p.Path = "(path unavailable)"
				}
				// The primary checkout is :0 of a slot group; an ordinary repo
				// without slots has no resting branch and so no glyph.
				if g.hasSlots && p.Path == g.root {
					p.Glyph = slotGlyphAt(git, p.Path, 0)
				}
				ordinary = append(ordinary, couchcore.LabelRow{Address: p.Row.Address, Label: p.Label})
			}
		}
	}
	labels := couchcore.DisambiguateLabels(ordinary)
	out := make([]ThreadPresentation, 0, len(rows))
	for _, g := range ordered {
		qualifier := ""
		if names[strings.ToLower(g.name)] > 1 {
			detail := g.root
			if detail == "" {
				detail = g.key
			}
			qualifier = " [" + detail + "]"
		}
		for i := range g.rows {
			p := &g.rows[i]
			suffix := ""
			if p.SlotNumber == 0 {
				suffix = strings.TrimPrefix(labels[p.Row.Address], p.Label)
			}
			p.Label += qualifier + suffix
		}
		if len(g.rows) > 1 {
			sort.Slice(g.rows, func(i, j int) bool {
				a, b := g.rows[i], g.rows[j]
				if a.SlotNumber != b.SlotNumber {
					return a.SlotNumber < b.SlotNumber
				}
				return presentationLess(a.Row, b.Row)
			})
		}
		out = append(out, g.rows...)
	}
	return out
}

func slotGlyphAt(git map[string]couchcore.SlotGitStatus, path string, slot int) string {
	status, ok := git[path]
	if !ok {
		return ""
	}
	return couchcore.SlotGlyph(status, couchcore.RestingBranch(slot))
}

func presentationRoot(start, scopeKey string) string {
	if scopeKey == "" || !filepath.IsAbs(start) {
		return ""
	}
	for candidate := filepath.Clean(start); ; candidate = filepath.Dir(candidate) {
		scope, err := launcher.ResolveRepoScope(candidate)
		if err == nil && scope.Key == scopeKey {
			return candidate
		}
		if filepath.Dir(candidate) == candidate {
			return ""
		}
	}
}

func presentationLess(a, b couchcore.ActionableThreadSummary) bool {
	if a.Address.RepoScope != b.Address.RepoScope {
		return a.Address.RepoScope < b.Address.RepoScope
	}
	if a.Address.Tag != b.Address.Tag {
		return a.Address.Tag < b.Address.Tag
	}
	ak, bk := menuRowKey(a), menuRowKey(b)
	if ak.Kind != bk.Kind {
		return ak.Kind < bk.Kind
	}
	if ak.Address.RepoScope != bk.Address.RepoScope {
		return ak.Address.RepoScope < bk.Address.RepoScope
	}
	if ak.Address.Tag != bk.Address.Tag {
		return ak.Address.Tag < bk.Address.Tag
	}
	if ak.SlotPath != bk.SlotPath {
		return ak.SlotPath < bk.SlotPath
	}
	return a.Target.Slot.WorktreeRoot < b.Target.Slot.WorktreeRoot
}

// presentationRow validates the enclosing sum type, not just its slot payload.
// Normalize once at projection/ingestion so rendering and every reducer action
// agree about fallback identity. The source snapshot is never mutated.
func presentationRow(row couchcore.ActionableThreadSummary) couchcore.ActionableThreadSummary {
	target := row.Target
	// Legacy summaries have no typed target and already route by native address.
	if target == (couchcore.ThreadTarget{}) {
		return row
	}
	if target.Validate() == nil && (target.Kind != couchcore.ThreadTargetOrdinary || target.Address == row.Address) {
		return row
	}
	target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: row.Address}
	key, err := target.RowKey()
	if err != nil {
		row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonUnreadable
		row.Recovery = nil
		key = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: row.Address}
	}
	row.Target, row.RowKey = target, key
	return row
}
