package vt

import "testing"

func TestPairMouseEpoch(t *testing.T) {
	e := NewEmulator(8, 2)
	defer e.Close()
	last := e.MouseEpoch()
	for _, tc := range []struct {
		wire   string
		change bool
	}{
		{"\x1b[?1002h", true}, {"\x1b[?1002h", false},
		{"\x1b[?1002l\x1b[?1002h", true}, {"\x1b[?1006h", true},
		{"\x1b[?1006h", false}, {"\x1b[?1004h", false},
		{"\x1b[?1003h", true}, {"\x1bc", true}, {"\x1bc", false},
		{"\x1b[?1006h", true}, {"\x1bc", true},
	} {
		e.WriteString(tc.wire)
		got := e.MouseEpoch()
		if (got > last) != tc.change || got < last {
			t.Fatalf("%q: %d -> %d change=%v", tc.wire, last, got, tc.change)
		}
		last = got
	}
}
