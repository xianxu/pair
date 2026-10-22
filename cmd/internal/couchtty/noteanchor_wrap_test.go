package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"testing"
)

func TestAnchorForUsesDeclaredChromeBoundary(t *testing.T) {
	f, l := anchorScreen()
	// A child's final row cannot become Couch chrome merely by resembling it.
	f.ChromeRows = 0
	if a := AnchorFor(cb(10, 9, 6, 1), f, l); a.Kind != broadcast.AnchorScreen {
		t.Fatalf("untrusted last row became chrome: %+v", a)
	}
}

func TestAnchorForTwoRowsMatchesChipRow(t *testing.T) {
	f := screenFrame(20, "child", "LIVE      first", "second")
	f.ChromeRows = 2
	l := ScreenLayout{Chips: []ChipSpan{{Row: 0, Start: 10, End: 15}, {Row: 1, Start: 0, End: 6}}, Slot: "active"}
	for _, tc := range []struct {
		box        broadcast.CellBox
		kind, text string
	}{
		{cb(10, 1, 5, 1), broadcast.AnchorTabBar, "first"},
		{cb(0, 2, 6, 1), broadcast.AnchorTabBar, "second"},
		{cb(0, 1, 4, 1), broadcast.AnchorStatusLine, "LIVE"},
		{cb(10, 2, 5, 1), broadcast.AnchorStatusLine, ""},
		{cb(0, 0, 5, 1), broadcast.AnchorScreen, "child"},
	} {
		a := AnchorFor(tc.box, f, l)
		if a.Kind != tc.kind || a.Text != tc.text {
			t.Fatalf("box %+v: %+v", tc.box, a)
		}
	}
}
