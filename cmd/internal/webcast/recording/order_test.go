package recording

import "testing"

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
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := OrderedTime(tc.t, tc.lastTime, tc.last, tc.next); got != tc.want {
				t.Fatalf("time=%d want %d", got, tc.want)
			}
		})
	}
}
