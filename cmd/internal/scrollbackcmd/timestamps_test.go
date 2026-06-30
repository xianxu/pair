package scrollbackcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDateOf(t *testing.T) {
	if got := dateOf("2026-06-14T10:30:00Z"); got != "2026-06-14" {
		t.Fatalf("dateOf = %q, want 2026-06-14", got)
	}
	if got := dateOf("garbage"); got != "" {
		t.Fatalf("dateOf(garbage) = %q, want empty", got)
	}
}

func TestParseEventsKeepsTimeEvents(t *testing.T) {
	p := filepath.Join(t.TempDir(), "e.jsonl")
	body := `{"type":"resize","offset":0,"cols":80,"rows":24}` + "\n" +
		`{"type":"time","offset":120,"ts":"2026-06-14T10:00:00Z"}` + "\n" +
		`{"type":"bogus","offset":5}` + "\n"
	os.WriteFile(p, []byte(body), 0o644)
	got, err := parseEvents(p)
	if err != nil {
		t.Fatal(err)
	}
	// resize + time kept; unknown type dropped.
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(got), got)
	}
	if got[1].Type != "time" || got[1].Ts != "2026-06-14T10:00:00Z" {
		t.Fatalf("time event not parsed: %+v", got[1])
	}
	// resize-only consumers still see the right initial size.
	if c, r := initialSize(got); c != 80 || r != 24 {
		t.Fatalf("initialSize = %d,%d", c, r)
	}
}

func TestInterleaveDateMarkers(t *testing.T) {
	lines := []string{"a", "b", "c", "d"}
	got := interleaveDateMarkers(lines, []dateMark{{0, "2026-06-13"}, {2, "2026-06-14"}})
	want := []string{
		"⟦pair:ts 2026-06-13⟧", "a", "b",
		"⟦pair:ts 2026-06-14⟧", "c", "d",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}

	// Leading undated region: first mark at line 1 → "a" stays bare.
	got2 := interleaveDateMarkers(lines, []dateMark{{1, "2026-06-14"}})
	if got2[0] != "a" || got2[1] != "⟦pair:ts 2026-06-14⟧" {
		t.Fatalf("leading undated region mishandled: %q", got2)
	}

	// Consecutive same-date marks collapse to one marker.
	got3 := interleaveDateMarkers(lines, []dateMark{{0, "2026-06-14"}, {2, "2026-06-14"}})
	if n := strings.Count(strings.Join(got3, "\n"), "⟦pair:ts"); n != 1 {
		t.Fatalf("same-date marks should collapse, got %d markers: %q", n, got3)
	}

	// No marks → unchanged.
	if got4 := interleaveDateMarkers(lines, nil); !reflect.DeepEqual(got4, lines) {
		t.Fatalf("no marks altered lines: %q", got4)
	}
}

// End-to-end at the render layer: a time event mid-stream yields a marker line in
// --with-timestamps mode, and NO marker without the flag (scrollback view stays
// clean). Confirms feedSegments snapshots + interleave wire together (#59).
func TestRenderWithTimestamps(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "r.raw")
	evPath := filepath.Join(dir, "r.events.jsonl")
	outPath := filepath.Join(dir, "o.txt")

	var raw strings.Builder
	var off35 int
	for i := 0; i < 40; i++ {
		if i == 35 {
			off35 = raw.Len() // time event lands after 35 lines (≈11 in scrollback)
		}
		fmt.Fprintf(&raw, "line %02d\r\n", i)
	}
	os.WriteFile(rawPath, []byte(raw.String()), 0o644)
	events := `{"type":"resize","offset":0,"cols":80,"rows":24}` + "\n" +
		fmt.Sprintf(`{"type":"time","offset":%d,"ts":"2026-06-14T10:00:00Z"}`, off35) + "\n"
	os.WriteFile(evPath, []byte(events), 0o644)

	if err := render(rawPath, evPath, outPath, true, 0, true); err != nil {
		t.Fatal(err)
	}
	withTs, _ := os.ReadFile(outPath)
	if !strings.Contains(string(withTs), "⟦pair:ts 2026-06-14⟧") {
		t.Fatalf("--with-timestamps: day marker missing:\n%s", withTs)
	}

	if err := render(rawPath, evPath, outPath, true, 0, false); err != nil {
		t.Fatal(err)
	}
	noTs, _ := os.ReadFile(outPath)
	if strings.Contains(string(noTs), "⟦pair:ts") {
		t.Fatalf("no flag: marker leaked into scrollback render:\n%s", noTs)
	}
}
