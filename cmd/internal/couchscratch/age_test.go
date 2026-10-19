package couchscratch

import (
	"os"
	"testing"
	"time"
)

func TestAgeStrictBoundary(t *testing.T) {
	now := time.Unix(200000, 0)
	for _, mode := range []os.FileMode{0600, os.ModeDir | 0700, os.ModeSymlink, os.ModeSocket, os.ModeNamedPipe, os.ModeDevice} {
		for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
			mt := now.Add(-24 * time.Hour).Add(delta)
			want := delta < 0 && (mode.IsRegular() || mode.IsDir())
			if ageEligible(mode, mt, now) != want {
				t.Fatalf("%v %v", mode, delta)
			}
		}
	}
}
func FuzzAgePolicy(f *testing.F) {
	f.Add(int64(0), uint32(0))
	f.Fuzz(func(t *testing.T, offset int64, bits uint32) {
		now := time.Unix(200000, 0)
		mode := os.FileMode(bits)
		mt := time.Unix(offset, 0)
		got := ageEligible(mode, mt, now)
		want := (mode.IsRegular() || mode.IsDir()) && mt.Before(now.Add(-24*time.Hour))
		if got != want {
			t.Fatal("age policy mismatch")
		}
	})
}
