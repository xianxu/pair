package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strconv"
	"strings"
)

// prepareCollapsedStatusChips preserves presentation order and represents every
// quiet grouped member exactly once, in a synthetic count owned by that group.
// It never resolves ownership independently of the presentation model.
func prepareCollapsedStatusChips(actors []StatusActor, work *statusLayoutWork) []StatusActor {
	// Multiline display runs deliberately split after a numbered hub. Collapse
	// aggregates identities instead, keeping one count even across those runs.
	groups := make([][]StatusActor, 0, len(actors))
	byIdentity := make(map[StatusGroupID]int)
	for _, actor := range actors {
		if work != nil {
			work.Visits++
		}
		identity := StatusGroupID{OwnerHub: actor.OwnerHub, GroupKey: actor.GroupKey}
		index, found := byIdentity[identity]
		if !found || actor.GroupKey == "" {
			index = len(groups)
			groups = append(groups, nil)
			if actor.GroupKey != "" {
				byIdentity[identity] = index
			}
		}
		groups[index] = append(groups[index], actor)
	}
	result := make([]StatusActor, 0, len(actors))
	for _, members := range groups {
		if work != nil {
			work.Visits++
		}
		first := members[0]
		if first.GroupKey == "" {
			result = append(result, members...)
			continue
		}
		kept, folded := 0, 0
		for _, actor := range members {
			if work != nil {
				work.Visits++
			}
			if actor.Active || actor.Bell || actor.Blocked || actor.CollapseAnchor || actor.Placeholder {
				kept++
			} else {
				folded++
			}
		}
		// A singleton or pair is clearer as its real addresses than a menu count.
		if folded < 3 {
			result = append(result, members...)
			continue
		}
		var strip strings.Builder
		for _, actor := range members {
			if work != nil {
				work.Visits++
			}
			if actor.Active || actor.Bell || actor.Blocked || actor.CollapseAnchor || actor.Placeholder {
				result = append(result, actor)
			} else {
				if work != nil {
					work.Bytes += len(actor.Glyph)
				}
				strip.WriteString(couchcore.CompactSlotGlyph(actor.Glyph))
			}
		}
		if folded > 0 {
			label := "+" + strconv.Itoa(folded)
			if kept == 0 {
				label = first.Label + " " + label
			}
			result = append(result, StatusActor{Label: label, GroupKey: first.GroupKey, OwnerHub: first.OwnerHub, OtherHub: first.OtherHub, Hub: kept == 0 && first.Hub, Folded: folded, FoldedStateStrip: strip.String()})
		}
	}
	return result
}

func statusActorWeight(a StatusActor) int {
	if a.Folded > 0 {
		return a.Folded
	}
	return 1
}
