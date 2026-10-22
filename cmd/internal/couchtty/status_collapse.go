package couchtty

import "strconv"

// prepareCollapsedStatusChips preserves presentation order and represents every
// quiet grouped member exactly once, in a synthetic count owned by that group.
// It never resolves ownership independently of the presentation model.
func prepareCollapsedStatusChips(actors []StatusActor, work *statusLayoutWork) []StatusActor {
	result := make([]StatusActor, 0, len(actors))
	if work != nil {
		work.Visits += len(actors)
	}
	for _, span := range chipGroups(actors) {
		first := actors[span[0]]
		if first.GroupKey == "" {
			result = append(result, actors[span[0]:span[1]]...)
			if work != nil {
				work.Visits += span[1] - span[0]
			}
			continue
		}
		kept, folded := 0, 0
		for _, actor := range actors[span[0]:span[1]] {
			if work != nil {
				work.Visits++
			}
			if actor.Active || actor.Bell || actor.Blocked || actor.SlotNumber == 0 && !actor.OtherHub {
				result = append(result, actor)
				kept++
			} else {
				folded++
			}
		}
		if folded > 0 {
			label := "+" + strconv.Itoa(folded)
			if kept == 0 {
				label = first.Label + " " + label
			}
			result = append(result, StatusActor{Label: label, GroupKey: first.GroupKey, OwnerHub: first.OwnerHub, OtherHub: first.OtherHub, Hub: kept == 0 && first.Hub, Folded: folded})
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
