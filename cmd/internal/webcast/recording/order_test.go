package recording

import (
	"math"
	"testing"
)

func TestOrderedTimePreservesMergeOrder(t *testing.T) {
	for _, tc := range []struct {
		name, last, next  string
		t, lastTime, want int64
	}{
		{"first", "", "head", 0, 0, 0},
		{"same file", "note", "note-edit", 2, 2, 2},
		{"forward rank", "seg", "note", 2, 2, 2},
		{"backward rank", "note-edit", "stop", 2, 2, 3},
		{"pointer to note", "stroke", "note", 2, 2, 3},
		{"regressed clock", "note-edit", "stop", 0, 2, 3},
		{"regressed same file", "note", "note-edit", 0, 2, 2},
		{"later clock", "note-edit", "stop", 5, 2, 5},
		{"before sketch", "doc", "sketch-begin", 2, 2, 3},
		{"after sketch", "sketch-begin", "doc", 2, 2, 3},
		{"adjacent sketches", "sketch-begin", "sketch-begin", 2, 2, 3},
		{"clear", "doc", "annotation-clear", 2, 2, 3},
		{"first sketch", "", "sketch-begin", 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := OrderedTime(tc.t, tc.lastTime, tc.last, tc.next); err != nil || got != tc.want {
				t.Fatalf("time=%d want %d", got, tc.want)
			}
		})
	}
}

func TestOrderedTimeOverflow(t *testing.T) {
	for _, pair := range [][2]string{{"note", "stop"}, {"doc", "sketch-begin"}, {"sketch-begin", "doc"}, {"doc", "annotation-clear"}} {
		if _, err := OrderedTime(math.MaxInt64, math.MaxInt64, pair[0], pair[1]); err == nil {
			t.Fatalf("accepted overflowing fence %v", pair)
		}
	}
	if got, err := OrderedTime(math.MaxInt64, math.MaxInt64, "doc", "doc"); err != nil || got != math.MaxInt64 {
		t.Fatalf("unneeded fence: %d %v", got, err)
	}
}

func TestOrderedTimeAllFileRanks(t *testing.T) {
	kinds := []string{"doc", "note", "stroke"}
	for previousRank, previous := range kinds {
		for nextRank, next := range kinds {
			want := int64(7)
			if nextRank < previousRank {
				want++
			}
			got, err := OrderedTime(7, 7, previous, next)
			if err != nil || got != want {
				t.Fatalf("%s -> %s: %d %v; want %d", previous, next, got, err, want)
			}
		}
	}
}
