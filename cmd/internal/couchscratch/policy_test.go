package couchscratch

import (
	"errors"
	"testing"
)

func TestChooseScratchLiteralPolicy(t *testing.T) {
	for _, tc := range []struct {
		p    Probe
		want bool
	}{{Probe{Root: "/missing/ram", Mounted: true, FreeBytes: Floor}, true}, {Probe{Root: "/missing/ram", Mounted: true, FreeBytes: Floor - 1}, false}, {Probe{Root: "/missing/ram", FreeBytes: Floor}, false}, {Probe{Root: "relative", Mounted: true, FreeBytes: Floor}, false}, {Probe{Root: "/missing/ram", Mounted: true, FreeBytes: Floor, Err: errors.New("probe")}, false}} {
		if got := ChooseScratch(tc.p); got != tc.want {
			t.Fatalf("%+v: %v", tc.p, got)
		}
	}
}
