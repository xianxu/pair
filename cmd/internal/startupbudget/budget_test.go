package startupbudget

import (
	"strings"
	"testing"
	"time"
)

func TestSelectUsesCallerDeadlineWithoutClipping(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, remaining := range []time.Duration{time.Second, 25 * time.Second, 3 * time.Minute} {
		want := now.Add(remaining)
		got, err := Select(now, want, Default)
		if err != nil || got != want {
			t.Fatalf("remaining %v: got %v, %v; want exact %v", remaining, got, err, want)
		}
	}
	// An explicit valid deadline does not depend on the unused fallback.
	want := now.Add(time.Minute)
	if got, err := Select(now, want, -1); err != nil || got != want {
		t.Fatalf("explicit deadline: %v, %v", got, err)
	}
}

func TestSelectFallbackAndInvalidBudgets(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if Default != 2*time.Minute || DeadlineEnv != "PAIR_STARTUP_DEADLINE" {
		t.Fatal("startup compatibility contract changed")
	}
	got, err := Select(now, time.Time{}, Default)
	if err != nil || got != now.Add(2*time.Minute) {
		t.Fatalf("fallback: %v, %v", got, err)
	}
	cases := []struct {
		name          string
		now, deadline time.Time
		fallback      time.Duration
	}{
		{"expired", now, now.Add(-time.Nanosecond), Default},
		{"exactly exhausted", now, now, Default},
		{"zero fallback", now, time.Time{}, 0},
		{"negative fallback", now, time.Time{}, -time.Second},
		{"fallback outside timestamp range", time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), time.Time{}, Default},
		{"explicit outside timestamp range", now, time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC), Default},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Select(tt.now, tt.deadline, tt.fallback)
			if err == nil || !got.IsZero() {
				t.Fatalf("got %v, %v; want zero and error", got, err)
			}
		})
	}
}

func TestDeadlineCodecCanonicalRoundTrip(t *testing.T) {
	input := time.Date(2026, 10, 21, 12, 1, 2, 123400000, time.FixedZone("offset", 3600))
	raw := Encode(input)
	if raw != "2026-10-21T11:01:02.1234Z" {
		t.Fatalf("noncanonical encoding %q", raw)
	}
	got, err := Decode(raw)
	if err != nil || !got.Equal(input) || got.Location() != time.UTC {
		t.Fatalf("round trip: %v, %v", got, err)
	}
	// Decode validates representation only; the caller owns expiry checks.
	if _, err := Decode("2000-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
}

func TestDeadlineCodecRejectsNoncanonicalOrUnboundedInput(t *testing.T) {
	for _, raw := range []string{
		"", " ", "2026-10-21T12:00:00Z\n", " 2026-10-21T12:00:00Z",
		"2026-10-21T12:00:00+00:00", "2026-10-21T13:00:00+01:00",
		"2026-10-21T12:00:00.1000Z", "2026-10-21T12:00:00.000Z",
		"2026-10-21t12:00:00z", "2026-10-21T12:00:00,1Z",
		"2026-10-21T12:00:60Z", "2026-13-21T12:00:00Z",
		"2026-10-21T12:00:00.1234567891Z", strings.Repeat("x", 65),
	} {
		t.Run(raw, func(t *testing.T) {
			got, err := Decode(raw)
			if err == nil || !got.IsZero() {
				t.Fatalf("accepted %q as %v (%v)", raw, got, err)
			}
		})
	}
}
