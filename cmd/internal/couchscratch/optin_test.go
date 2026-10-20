package couchscratch

import "testing"

func TestChooseSlotScratch(t *testing.T) {
	ready := Probe{Root: DefaultRoot, Mounted: true, FreeBytes: Floor}
	for _, tc := range []struct {
		probe       Probe
		optIn, want bool
	}{
		{ready, true, true}, {ready, false, false}, {Probe{}, true, false}, {Probe{}, false, false},
		{Probe{Root: DefaultRoot, Mounted: true, FreeBytes: Floor - 1}, true, false},
	} {
		if got := ChooseSlotScratch(tc.probe, tc.optIn); got != tc.want {
			t.Fatalf("probe=%+v optIn=%v got=%v want=%v", tc.probe, tc.optIn, got, tc.want)
		}
	}
}

func TestScratchOptInPath(t *testing.T) {
	for _, tc := range []struct{ home, config, want string }{
		{"/home/operator", "", "/home/operator/.config/pair/sandbox-allowed"},
		{"/home/operator", "/xdg", "/xdg/pair/sandbox-allowed"},
		{"", "/xdg", "/xdg/pair/sandbox-allowed"},
		{"", "", ""}, {"relative", "", ""}, {"/home", "relative", ""},
		{"/home", "/bad\x00root", ""},
	} {
		if got := ScratchOptInPath(tc.home, tc.config); got != tc.want {
			t.Fatalf("home=%q config=%q got=%q want=%q", tc.home, tc.config, got, tc.want)
		}
	}
}
