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
// Row retains the original target and identity used by selection and dispatch.
type ThreadPresentation struct {
	Row        couchcore.ActionableThreadSummary
	GroupKey   string
	Label      string
	Path       string
	Indent     int
	SlotNumber int
}

type threadPresentationGroup struct {
	key, root, name string
	hasSlots        bool
	rows            []ThreadPresentation
}

// PresentThreads groups and orders inventory rows without filesystem discovery or
// synthetic headers. Root recovery is best-effort: immutable starting paths and
// their lexical ancestors can prove a scope match without consulting current cwd.
func PresentThreads(rows []couchcore.ActionableThreadSummary) []ThreadPresentation {
	groups := make(map[string]*threadPresentationGroup)
	for _, row := range rows {
		p := ThreadPresentation{Row: row, GroupKey: row.Address.RepoScope}
		root := ""
		validSlot := row.Target.Kind == couchcore.ThreadTargetSlot && row.Target.Slot.Validate() == nil
		if validSlot {
			slot := row.Target.Slot
			scope, _ := launcher.ResolveRepoScope(slot.PrimaryRoot)
			p.GroupKey = scope.Key
			root = slot.PrimaryRoot
			p.SlotNumber = slot.Number
			p.Indent = 2
			p.Path = slot.WorktreeRoot
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
