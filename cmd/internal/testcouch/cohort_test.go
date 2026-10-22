package testcouch

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func cohortKnown(resources ...Resource) OwnedRun {
	s := OwnedRun{Resources: map[string]Resource{}}
	for _, r := range resources {
		s.Resources[r.ID+"\x00"+r.Generation] = r
	}
	return s
}
func cohortPIDs(resources []Resource) map[string]string {
	result := map[string]string{}
	for _, r := range resources {
		result[r.ID] = r.Generation
	}
	return result
}
func TestSelectOwnedCohortIncludesAncestryTTYAndMarkedDaemon(t *testing.T) {
	// Native sh/sleep entries need neither an environment marker nor a name.
	rows := []procutil.Process{
		{PID: 12, PPID: 11, Identity: "grandchild"},
		{PID: 9, PPID: 1, Identity: "same-tty-reparented", TTY: "101"},
		{PID: 11, PPID: 10, Identity: "shell", TTY: "101"},
		{PID: 10, PPID: 1, Identity: "supervisor", TTY: "100"},
		{PID: 20, PPID: 1, Identity: "marked-daemon", TTY: "102"},
		{PID: 21, PPID: 1, Identity: "daemon-cohort", TTY: "102"},
		{PID: 22, PPID: 21, Identity: "cohort-child"},
		{PID: 30, PPID: 1, Identity: "foreign", TTY: "103"},
		{PID: 31, PPID: 1, Identity: "unmarked", TTY: ""},
		{PID: 32, PPID: 10, Identity: "zombie", TTY: "100", Zombie: true},
	}
	known := cohortKnown(Resource{ID: "10", Generation: "supervisor", Kind: "couch"})
	before := append([]procutil.Process(nil), rows...)
	marked := map[int]bool{20: true, 30: false}
	result, err := selectOwnedCohort(rows, known, marked)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"9": "same-tty-reparented", "10": "supervisor", "11": "shell", "12": "grandchild", "20": "marked-daemon", "21": "daemon-cohort", "22": "cohort-child"}
	if !reflect.DeepEqual(cohortPIDs(result), want) {
		t.Fatalf("cohort=%+v", result)
	}
	if !reflect.DeepEqual(rows, before) || !reflect.DeepEqual(marked, map[int]bool{20: true, 30: false}) || len(known.Resources) != 1 {
		t.Fatal("selector mutated observations")
	}
}
func TestSelectOwnedCohortRequiresPositiveCurrentGeneration(t *testing.T) {
	rows := []procutil.Process{
		{PID: 10, Identity: "new", TTY: "100"},
		{PID: 11, PPID: 10, Identity: "new-child", TTY: "100"},
		{PID: 12, Identity: "peer", TTY: "100"},
		{PID: 20, Identity: "old-zombie", TTY: "200", Zombie: true},
		{PID: 21, Identity: "zombie-peer", TTY: "200"},
		{PID: 30, Identity: "foreign-no-tty"},
	}
	known := cohortKnown(Resource{ID: "10", Generation: "old"}, Resource{ID: "20", Generation: "old-zombie"})
	result, err := selectOwnedCohort(rows, known, nil)
	if err != nil || len(result) != 0 {
		t.Fatalf("recycled or zombie ownership leaked: %+v %v", result, err)
	}
	result, err = selectOwnedCohort(rows, known, map[int]bool{11: true})
	if err != nil || len(result) != 3 {
		t.Fatalf("independent positive marker not honored: %+v %v", result, err)
	}
}
func TestSelectOwnedCohortUnknownRelevantIdentityIsError(t *testing.T) {
	for name, rows := range map[string][]procutil.Process{
		"known":      {{PID: 10}},
		"marked":     {{PID: 10, Identity: "owner"}, {PID: 20}},
		"descendant": {{PID: 10, Identity: "owner"}, {PID: 11, PPID: 10}},
		"tty":        {{PID: 10, Identity: "owner", TTY: "100"}, {PID: 11, TTY: "100"}},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := selectOwnedCohort(rows, cohortKnown(Resource{ID: "10", Generation: "owner"}), map[int]bool{20: true})
			if err == nil || len(result) != 0 {
				t.Fatalf("unknown identity silently admitted/omitted: %+v %v", result, err)
			}
		})
	}
	// A foreign unreadable row is not permission to interrupt the fixture.
	result, err := selectOwnedCohort([]procutil.Process{{PID: 10, Identity: "owner"}, {PID: 99}}, cohortKnown(Resource{ID: "10", Generation: "owner"}), nil)
	if err != nil || len(result) != 1 {
		t.Fatal(result, err)
	}
}
func TestSelectOwnedCohortIndexedClosureAtEightAndSixteen(t *testing.T) {
	for _, n := range []int{8, 16} {
		var rows []procutil.Process
		// Reverse rows ensure traversal does not depend on census ordering. All
		// members share a terminal: expanding its bucket for every member is waste.
		for pid := n; pid >= 1; pid-- {
			rows = append(rows, procutil.Process{PID: pid, PPID: pid - 1, Identity: fmt.Sprint(pid), TTY: "private"})
		}
		result, err := selectOwnedCohort(rows, cohortKnown(Resource{ID: "1", Generation: "1"}), nil)
		if err != nil || len(result) != n {
			t.Fatal(n, result, err)
		}
		seen := map[string]bool{}
		for _, r := range result {
			if seen[r.ID] || r.Generation != r.ID {
				t.Fatal("duplicate or wrong generation", result)
			}
			seen[r.ID] = true
		}
	}
}
func FuzzSelectOwnedCohortNeverSeedsRecycledPID(f *testing.F) {
	f.Add("old", "new", true)
	f.Add("old", "old", false)
	f.Fuzz(func(t *testing.T, old, current string, marked bool) {
		if old == "" || current == "" || len(old) > 128 || len(current) > 128 {
			return
		}
		rows := []procutil.Process{{PID: 1, Identity: current}, {PID: 2, PPID: 1, Identity: "child"}}
		result, err := selectOwnedCohort(rows, cohortKnown(Resource{ID: strconv.Itoa(1), Generation: old}), map[int]bool{1: marked})
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if marked || old == current {
			want = 2
		}
		if len(result) != want {
			t.Fatalf("false generation ownership: %+v", result)
		}
	})
}
