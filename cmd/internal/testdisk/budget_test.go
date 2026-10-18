package testdisk

import (
	"math"
	"testing"
)

func TestWriteBudgetVerdict(t *testing.T) {
	for _, tt := range []struct {
		name                        string
		before, after, cap, repeats uint64
		fail                        bool
	}{
		{"below", 10, 109, 100, 1, false}, {"equal", 10, 110, 100, 1, false},
		{"over", 10, 111, 100, 1, true}, {"backwards", 10, 9, 100, 1, true},
		{"two runs", 10, 210, 100, 2, false}, {"two runs over", 10, 211, 100, 2, true},
		{"zero cap", 0, 0, 0, 1, true}, {"zero repeats", 0, 0, 100, 0, true},
		{"overflow", 0, 1, math.MaxUint64, 2, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckBytes(tt.before, tt.after, tt.cap, tt.repeats)
			if (err != nil) != tt.fail {
				t.Fatalf("verdict=%+v err=%v want failure=%v", got, err, tt.fail)
			}
			if !tt.fail && (got.Bytes != tt.after-tt.before || got.Limit != tt.cap*tt.repeats) {
				t.Fatalf("verdict=%+v", got)
			}
		})
	}
}

func TestInvocationRunsIncludesCPURepetitions(t *testing.T) {
	for _, tt := range []struct {
		count int
		cpus  string
		want  uint64
	}{
		{1, "", 1}, {2, "", 2}, {2, "1,2,4", 6}, {3, "2,2", 6},
		{0, "", 1}, {-1, "", 1},
	} {
		got, err := invocationRuns(tt.count, tt.cpus)
		if err != nil || got != tt.want {
			t.Fatalf("count=%d cpu=%q got=%d err=%v want=%d", tt.count, tt.cpus, got, err, tt.want)
		}
	}
	if ^uint(0)>>63 != 0 {
		if _, err := invocationRuns(int(^uint(0)>>1), "1,2,3"); err == nil {
			t.Fatal("accepted repetition overflow")
		}
	}
}
