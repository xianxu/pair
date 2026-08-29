package sessioninventory

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type normalizationGolden struct {
	SchemaVersion int `json:"schema_version"`
	Cases         []struct {
		Name  string `json:"name"`
		Input string `json:"input"`
		Want  string `json:"want"`
	} `json:"cases"`
}

func TestPairLogAppendIDRoundTrips(t *testing.T) {
	t.Parallel()
	raw := EncodePairLogEntryWithID([]byte("authored"), time.Date(2026, 8, 28, 1, 2, 3, 0, time.UTC), "attempt-a")
	parsed := ParsePairLog(raw, 0)
	if len(parsed.MalformedOffsets) != 0 || len(parsed.Facts) != 1 || len(parsed.Entries) != 1 || parsed.Facts[0].AppendID != "attempt-a" || parsed.Entries[0].AuthoredText != "authored" {
		t.Fatalf("parsed=%#v raw=%q", parsed, raw)
	}
}

func TestParsePairLogExcludesPreparedEntriesFromCorrelationFacts(t *testing.T) {
	prepared := EncodePairLogPreparedEntry([]byte("not sent"), time.Time{}, "attempt-prepared")
	submitted := EncodePairLogEntryWithID([]byte("sent"), time.Time{}, "attempt-submitted")
	parsed := ParsePairLog(append(prepared, submitted...), 0)
	if len(parsed.MalformedOffsets) != 0 || len(parsed.Entries) != 2 {
		t.Fatalf("parsed=%#v", parsed)
	}
	if len(parsed.Facts) != 1 || parsed.Facts[0].AuthoredText != "sent" {
		t.Fatalf("prepared entry became correlation evidence: %#v", parsed.Facts)
	}
}

func TestNormalizePairTextGolden(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/normalization/v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden normalizationGolden
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	if golden.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", golden.SchemaVersion)
	}
	for _, test := range golden.Cases {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizePairText(test.Input); got != test.Want {
				t.Fatalf("NormalizePairText(%q) = %q, want %q", test.Input, got, test.Want)
			}
		})
	}
}

func TestParsePairLogSuffix(t *testing.T) {
	t.Parallel()
	first := "## 2026-08-28 01:02:03\n\nold\n\n---\n\n"
	second := "## 2026-08-28 01:02:04\n\n=== focus ===\r\nnew body  \r\n\n\n---\n\n"
	result := ParsePairLog([]byte(first+second), uint64(len(first)))
	if len(result.MalformedOffsets) != 0 || len(result.Facts) != 1 {
		t.Fatalf("result=%#v", result)
	}
	if result.Facts[0].Position != uint64(len(first)) || result.Facts[0].Text != "new body" {
		t.Fatalf("fact=%#v", result.Facts[0])
	}
}

func TestParsePairLogFailsClosedOnMalformedOrMidEntryOffset(t *testing.T) {
	t.Parallel()
	valid := []byte("## 2026-08-28 01:02:03\n\nbody\n\n---\n\n")
	for _, test := range []struct {
		name   string
		raw    []byte
		offset uint64
	}{
		{name: "truncated", raw: valid[:len(valid)-2]},
		{name: "bad header", raw: []byte("## nope\n\nbody\n\n---\n\n")},
		{name: "mid entry", raw: valid, offset: 3},
		{name: "past end", raw: valid, offset: uint64(len(valid) + 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := ParsePairLog(test.raw, test.offset)
			if len(result.Facts) != 0 || len(result.MalformedOffsets) == 0 {
				t.Fatalf("result=%#v", result)
			}
		})
	}
}
