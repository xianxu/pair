package terminal

import "testing"

func TestComposeTwoChromeRows(t *testing.T) {
	child := testFrame(8, 3)
	bottom, err := StyledRows("first\nsecond", 8, 2)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Compose(child, Geometry{8, 5}, bottom)
	if err != nil {
		t.Fatalf("two-row chrome refused: %v", err)
	}
	if len(f.Rows) != 5 || f.Rows[3] != rowMetadata(bottom[:8]) || f.Rows[4] != rowMetadata(bottom[8:]) {
		t.Fatalf("chrome metadata: %+v", f.Rows)
	}
}

func TestChildGeometryAndChromeMetadata(t *testing.T) {
	host := Geometry{8, 5}
	for _, rows := range []int{0, 1, 2, 4} {
		bottom := make([]Cell, 8*rows)
		got, err := ChildGeometry(host, bottom)
		if err != nil || got != (Geometry{8, 5 - rows}) {
			t.Fatalf("%d rows: %v %v", rows, got, err)
		}
		f, err := Compose(testFrame(8, 5-rows), host, bottom)
		if err != nil || f.ChromeRows != rows {
			t.Fatalf("compose %d rows: %+v %v", rows, f, err)
		}
		if f.Clone().ChromeRows != rows {
			t.Fatal("clone lost metadata")
		}
	}
	for _, n := range []int{1, 7, 9, 40, 48} {
		if _, err := ChildGeometry(host, make([]Cell, n)); err == nil {
			t.Fatalf("accepted %d cells", n)
		}
	}
	bad := make([]Cell, 8)
	bad[0] = Cell{Content: "\x1b", Width: 1}
	if _, err := ChildGeometry(host, bad); err == nil {
		t.Fatal("accepted unsafe chrome")
	}
	f := testFrame(8, 5)
	for _, rows := range []int{-1, 5, 6} {
		f.ChromeRows = rows
		if f.Validate() == nil {
			t.Fatalf("accepted %d chrome rows", rows)
		}
	}
	child := testFrame(8, 4)
	child.ChromeRows = 1
	if _, err := Compose(child, host, make([]Cell, 8)); err == nil {
		t.Fatal("accepted nested chrome")
	}
}

func TestChromeMetadataParticipatesInHistoryIdentity(t *testing.T) {
	a := testFrame(8, 5)
	b := a.Clone()
	b.ChromeRows = 2
	if sameHistoryFrame(a, b) {
		t.Fatal("history identity ignored chrome reservation")
	}
}
