package historyarchive

import (
	"testing"
	"time"
)

func TestFindQueryUsesUTCHalfOpenOverlap(t *testing.T) {
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, indexRefresh())
	if err != nil {
		t.Fatal(err)
	}
	start := indexLabel().From
	for _, tc := range []struct {
		q    FindQuery
		want int
	}{
		{FindQuery{}, 1},
		{FindQuery{Since: start}, 1},
		{FindQuery{Until: start}, 0},
		{FindQuery{Since: start.Add(time.Nanosecond)}, 0},
		{FindQuery{Kind: "claude-transcripts"}, 0},
	} {
		got, err := SelectLabels(i, tc.q)
		if err != nil || len(got) != tc.want {
			t.Fatalf("query %+v: %d %v", tc.q, len(got), err)
		}
	}
	for _, bad := range []FindQuery{{Kind: "unsupported"}, {Since: start, Until: start}, {Since: start.In(time.FixedZone("other", 3600))}} {
		if _, err := SelectLabels(i, bad); err == nil {
			t.Fatal("accepted invalid query", bad)
		}
	}
}

func TestFindTimeBoundsAcceptOnlyUTC(t *testing.T) {
	for _, raw := range []string{"2026-01-02", "2026-01-02T00:00:00Z"} {
		got, err := ParseFindTime(raw)
		if err != nil || !got.Equal(indexLabel().From) {
			t.Fatal(raw, got, err)
		}
	}
	for _, raw := range []string{"2026-01-02T00:00:00+01:00", "2026-02-30", "yesterday", ""} {
		if _, err := ParseFindTime(raw); err == nil {
			t.Fatal("accepted", raw)
		}
	}
}
