package couchtty

import (
	"math/bits"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestCollapseStatusFoldsQuietMembersAtWideWidth(t *testing.T) {
	actors := []StatusActor{hubActor("alpha", "a", 0), hubActor("alpha:1", "a", 1), hubActor("alpha:2", "a", 2), hubActor("alpha:3", "a", 3)}
	actors[0].Active = true
	actors[2].Bell = true
	b := RenderStatusBar(100, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors})
	if b.Height() != 1 || rowPlain(b.Rows[0]) != "[alpha]  :2  +2" || len(b.Chips) != 2 {
		t.Fatalf("collapse=%q chips=%+v", b.Body(), b.Chips)
	}
}

func TestCollapsePreparationConservesActorsAndCountsWork(t *testing.T) {
	for _, n := range []int{128, 256} {
		actors := make([]StatusActor, n)
		for i := range actors {
			actors[i] = hubActor("repo:"+strconv.Itoa(i), "g", i)
			actors[i].OwnerHub = "owner"
			actors[i].Bell = i%7 == 0
		}
		work := &statusLayoutWork{}
		prepared := prepareCollapsedStatusChips(actors, work)
		weight := 0
		for _, a := range prepared {
			weight += statusActorWeight(a)
		}
		if weight != n {
			t.Fatalf("actor conservation: %d != %d", weight, n)
		}
		bar := renderStatusBar(50, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors}, work)
		if bar.Height() != 1 || work.Visits > 4*n*(1+bits.Len(uint(n))) || work.Bytes > 2*collapseInputBytes(actors) {
			t.Fatalf("n%d work=%+v height%d", n, work, bar.Height())
		}
	}
}

func collapseInputBytes(actors []StatusActor) int {
	total := 0
	for _, a := range actors {
		total += len(a.Label) + len(a.Glyph)
	}
	return total
}

func TestCollapseAllAttentionWorkBound(t *testing.T) {
	for _, n := range []int{128, 256} {
		actors := make([]StatusActor, n)
		for i := range actors {
			actors[i] = hubActor("repo:"+strconv.Itoa(i), "g", i)
			actors[i].Bell = true
		}
		work := &statusLayoutWork{}
		renderStatusBar(50, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors}, work)
		if work.Visits > 4*n*(1+bits.Len(uint(n))) || work.Bytes > 2*collapseInputBytes(actors) {
			t.Fatalf("n%d work=%+v", n, work)
		}
	}
}

func TestCollapseCountsCarryExactDrawnScope(t *testing.T) {
	a := hubActor("same", "repo", 0)
	a.OwnerHub, a.OtherHub = "first", true
	b := a
	b.OwnerHub = "second"
	bar := RenderStatusBar(80, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: []StatusActor{a, b}})
	if len(bar.GroupCounts) != 2 || bar.GroupCounts[0].Group.OwnerHub != "first" || bar.GroupCounts[1].Group.OwnerHub != "second" {
		t.Fatalf("count identities: %+v (%q)", bar.GroupCounts, bar.Body())
	}
	for width := 1; width < 20; width++ {
		bar = RenderStatusBar(width, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: []StatusActor{a, b}})
		for _, span := range bar.GroupCounts {
			if span.Start < 0 || span.End > width || span.End <= span.Start {
				t.Fatalf("phantom span width%d: %+v", width, span)
			}
		}
	}
}

func TestCollapseVisibleAndHiddenCountsConserveOriginalActors(t *testing.T) {
	actors := make([]StatusActor, 12)
	for i := range actors {
		group := i / 6
		actors[i] = hubActor("repo"+strconv.Itoa(group)+":"+strconv.Itoa(i%6), strconv.Itoa(group), i%6)
		actors[i].Bell = i == 3 || i == 8
	}
	actors[1].Active = true
	count := regexp.MustCompile(`\+([0-9]+)`)
	for width := 1; width <= 120; width++ {
		bar := RenderStatusBar(width, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors})
		if len(bar.GroupCounts) == 0 {
			continue
		} // an impossible width can omit the entire marker
		total := len(bar.Chips)
		for _, span := range bar.GroupCounts {
			text := []rune(rowPlain(bar.Rows[span.Row]))[span.Start:span.End]
			match := count.FindStringSubmatch(string(text))
			if len(match) != 2 {
				t.Fatalf("incomplete drawn count width%d: %q", width, text)
			}
			n, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			total += n
		}
		if total != len(actors) {
			t.Fatalf("width%d counted%d actors, want%d: %q", width, total, len(actors), bar.Body())
		}
	}
}

func TestCollapseStatusInactiveHubAndWorkersFoldTogether(t *testing.T) {
	active := hubActor("ops", "o", 0)
	active.Active, active.OwnerHub = true, "ops"
	hub, worker := hubActor("other", "g", 0), hubActor("other:1", "g", 1)
	hub.Hub, hub.OtherHub, hub.OwnerHub = true, true, "other"
	worker.OtherHub, worker.OwnerHub = true, "other"
	b := RenderStatusBar(100, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: []StatusActor{active, hub, worker}})
	if b.Height() != 1 || len(b.Chips) != 1 || !strings.Contains(rowPlain(b.Rows[0]), "other +2") {
		t.Fatalf("quiet inactive hub remained visible: %q %+v", b.Body(), b.Chips)
	}
}

func TestCollapseHubRoleAndIdentityIndependentOfSlotNumber(t *testing.T) {
	for _, hubNumber := range []int{0, 1, 4} {
		for _, inactive := range []bool{false, true} {
			t.Run(strconv.Itoa(hubNumber)+"/inactive="+strconv.FormatBool(inactive), func(t *testing.T) {
				hub := hubActor("repo:"+strconv.Itoa(hubNumber), "repo", hubNumber)
				hub.Hub, hub.OwnerHub, hub.OtherHub = true, "owner", inactive
				worker := hubActor("repo:5", "repo", 5)
				worker.OwnerHub, worker.OtherHub, worker.Active = "owner", inactive, !inactive
				bar := RenderStatusBar(100, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: []StatusActor{hub, worker}})
				if inactive {
					if len(bar.Chips) != 0 || len(bar.GroupCounts) != 1 || !strings.Contains(rowPlain(bar.Rows[0]), "+2") {
						t.Fatalf("inactive hub%d split identity: %q %+v", hubNumber, bar.Body(), bar.GroupCounts)
					}
				} else {
					if len(bar.Chips) != 2 || len(bar.GroupCounts) != 0 {
						t.Fatalf("current hub%d lost anchor: %q %+v", hubNumber, bar.Body(), bar.Chips)
					}
				}
			})
		}
	}
}

func TestCollapseStatusAlwaysOneBoundedRowWithActiveSpan(t *testing.T) {
	actors := []StatusActor{hubActor("active界", "a", 0), hubActor("bell", "b", 0), hubActor("blocked", "c", 0), hubActor("quiet", "d", 0)}
	actors[0].Active, actors[1].Bell, actors[2].Blocked = true, true, true
	for width := 1; width < 80; width++ {
		b := RenderStatusBar(width, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors})
		if b.Height() != 1 || textwidth.Width(rowPlain(b.Rows[0])) > width {
			t.Fatalf("width%d: %q", width, b.Body())
		}
		found := false
		for _, span := range b.Chips {
			found = found || span.Thread == actors[0].Thread
		}
		if !found {
			t.Fatalf("width%d lost active span: %q", width, b.Body())
		}
	}
}
