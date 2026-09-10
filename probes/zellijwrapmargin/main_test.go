package main

import "testing"

// The verdict table, including every case the first parser turned into
// "fixed": an unparsed reading must be INCONCLUSIVE, never a finding about
// zellij (#223 BR-1, pair#208's rule).
func TestVerdictRefusesToReadAnUnparsedMeasurementAsFixed(t *testing.T) {
	const head = "pane rows=22 cols=78 region=1..21\n"
	for _, tt := range []struct {
		name     string
		readings string
		want     wrapVerdict
	}{
		{"escaped onto the reserved row", head + "after-wrapping-line row=22 col=11\nDONE\n", wrapEscapes},
		{"stayed on the margin", head + "after-wrapping-line row=21 col=11\nDONE\n", wrapScrolls},
		{"empty CPR reply", head + "after-wrapping-line row= col=\nDONE\n", inconclusive},
		{"failed tput: no height", "pane rows= cols= region=1..-1\nafter-wrapping-line row=21 col=11\n", inconclusive},
		{"wrap line missing", head + "DONE\n", inconclusive},
		{"row above the margin: setup never reached it", head + "after-wrapping-line row=5 col=11\n", inconclusive},
		{"row past the screen", head + "after-wrapping-line row=40 col=11\n", inconclusive},
		{"nothing at all", "", inconclusive},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := verdict(tt.readings); got != tt.want {
				t.Fatalf("verdict = %d, want %d", got, tt.want)
			}
		})
	}
}
