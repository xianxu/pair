package couchschedule

import (
	"testing"
	"time"
)

func instant(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCalendarStrictParser(t *testing.T) {
	for _, expr := range []string{"", "@daily", "@every 1h", "0 0 7 * * *", "TZ=UTC 0 7 * * *", "CRON_TZ=UTC 0 7 * * *", "61 * * * *", "* * * * *\x00", "* * * * * trailing"} {
		if _, err := NewCalendar(expr, time.UTC); err == nil {
			t.Errorf("accepted %q", expr)
		}
	}
	if _, err := NewCalendar("* * * * *", nil); err == nil {
		t.Fatal("nil location accepted")
	}
	c, err := NewCalendar("*/5 * * * *", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	after := instant(t, "2026-01-02T07:10:00Z")
	if got := c.Next(after); !got.Equal(after.Add(5 * time.Minute)) {
		t.Fatalf("strict next: %s", got)
	}
	c, err = NewCalendar("0 0 31 2 *", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Next(after); !got.IsZero() {
		t.Fatalf("impossible schedule: %s", got)
	}
	var zero Calendar
	if got := zero.Next(after); !got.IsZero() {
		t.Fatalf("zero calendar: %s", got)
	}
}

func TestCalendarInjectedLocalIgnoresInputZone(t *testing.T) {
	c, err := NewCalendar("0 7 * * *", time.Local)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 6, 1, 6, 0, 0, 0, time.Local)
	_, offset := after.Zone()
	after = after.In(time.FixedZone("different", offset+3*3600))
	want := time.Date(2026, 6, 1, 7, 0, 0, 0, time.Local)
	if got := c.Next(after); !got.Equal(want) {
		t.Fatalf("injected local ignored: got %s want %s", got, want)
	}
}

func TestCalendarDSTOncePerLocalTime(t *testing.T) {
	for _, tc := range []struct{ name, zone, cron, after, want string }{
		{"spring-skip", "America/Los_Angeles", "30 2 * * *", "2026-03-08T00:00:00-08:00", "2026-03-09T02:30:00-07:00"},
		{"fall-first", "America/Los_Angeles", "30 1 * * *", "2026-11-01T00:00:00-07:00", "2026-11-01T01:30:00-07:00"},
		{"fall-after-first", "America/Los_Angeles", "30 1 * * *", "2026-11-01T01:30:00-07:00", "2026-11-02T01:30:00-08:00"},
		{"startup-second-fold", "America/Los_Angeles", "30 1 * * *", "2026-11-01T01:05:00-08:00", "2026-11-02T01:30:00-08:00"},
		{"fall-minute", "America/Los_Angeles", "* * * * *", "2026-11-01T01:59:00-07:00", "2026-11-01T02:00:00-08:00"},
		{"half-hour-fold", "Australia/Lord_Howe", "45 1 * * *", "2026-04-05T01:45:00+11:00", "2026-04-06T01:45:00+10:30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			c, err := NewCalendar(tc.cron, loc)
			if err != nil {
				t.Fatal(err)
			}
			got := c.Next(instant(t, tc.after))
			if !got.Equal(instant(t, tc.want)) {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func FuzzCalendar(f *testing.F) {
	f.Add("*/5 * * * *", int64(1700000000))
	f.Add("0 0 31 2 *", int64(0))
	f.Fuzz(func(t *testing.T, expression string, seconds int64) {
		if seconds < 0 || seconds > 4102444800 {
			return
		}
		c, err := NewCalendar(expression, time.UTC)
		if err != nil {
			return
		}
		after := time.Unix(seconds, 0)
		next := c.Next(after)
		if !next.IsZero() && !next.After(after) {
			t.Fatalf("next %s is not after %s", next, after)
		}
	})
}
