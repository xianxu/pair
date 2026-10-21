package historyarchive

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDriveMonthLayoutUsesBundleUTCMonth(t *testing.T) {
	root, _ := ParseDriveDestination("backup", "archive/history")
	layout := DriveArchiveLayout{Root: root}
	label := indexLabel()
	label.From = time.Date(2026, 1, 1, 0, 30, 0, 0, time.FixedZone("east", 3600)).UTC()
	label.To = label.From.Add(3 * time.Hour)
	got, err := layout.MonthDestination(label)
	if err != nil || got.String() != "backup:archive/history/2025/12" {
		t.Fatalf("month=%+v %v", got, err)
	}
	b, _, err := BundleKeys(label.Kind, label.From, label.To, label.ID)
	if err != nil || !strings.Contains(b, "/2025/12/") {
		t.Fatal(b, err)
	}
	for _, locator := range []string{root.String(), got.String()} {
		parsed, err := layout.ValidateRecordedDriveLocation(label, locator)
		if err != nil || parsed.String() != locator {
			t.Fatalf("location %q: %v", locator, err)
		}
	}
	for _, locator := range []string{"backup:archive/history/2026/01", "backup:archive/history-more/2025/12", "other:archive/history", root.String() + "/../escape", root.String() + "/2025/12/x"} {
		if _, err := layout.ValidateRecordedDriveLocation(label, locator); err == nil {
			t.Errorf("accepted %q", locator)
		}
	}
}

func FuzzDriveMonthLocation(f *testing.F) {
	f.Add("archive/history", "backup:archive/history", int64(1790800000))
	f.Add("../escape", "backup:archive/history/2026/10", int64(0))
	f.Fuzz(func(t *testing.T, folder, locator string, unixTime int64) {
		root, err := ParseDriveDestination("backup", folder)
		if err != nil {
			return
		}
		label := indexLabel()
		label.From = time.Unix(unixTime, 0).UTC()
		label.To = label.From
		layout := DriveArchiveLayout{Root: root}
		month, err := layout.MonthDestination(label)
		if err != nil {
			return
		}
		want := fmt.Sprintf("%s/%s", root.String(), label.From.UTC().Format("2006/01"))
		if month.String() != want {
			t.Fatalf("month %q != %q", month.String(), want)
		}
		got, err := layout.ValidateRecordedDriveLocation(label, locator)
		if err == nil && (got.String() != root.String() && got.String() != want) {
			t.Fatalf("escaped: %q", got.String())
		}
	})
}
