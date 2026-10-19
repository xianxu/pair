package historyarchive

import (
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

func TestDatabaseFreshness(t *testing.T) {
	now := time.Date(2026, 10, 19, 4, 0, 0, 0, time.UTC)
	ages := []struct {
		name     string
		verified time.Time
		stale    bool
	}{
		{"never", time.Time{}, true},
		{"fresh", now.Add(-time.Hour), false},
		{"threshold", now.Add(-48 * time.Hour), false},
		{"over threshold", now.Add(-48*time.Hour - time.Nanosecond), true},
		{"future", now.Add(time.Hour), false},
	}
	for _, status := range []DatabaseBackupStatus{DatabaseArchived, DatabaseUnchanged, DatabaseFailed, DatabaseDeferred, "unknown", ""} {
		for _, age := range ages {
			t.Run(string(status)+"/"+age.name, func(t *testing.T) {
				s := DatabaseBackupSummary{Status: status, LastVerified: age.verified}
				want := (status == DatabaseFailed || status == DatabaseDeferred) && age.stale
				if got := s.NeedsAttention(now, DefaultDatabaseMaxAge); got != want {
					t.Fatalf("NeedsAttention() = %v, want %v", got, want)
				}
				if got := s.Warning(now, DefaultDatabaseMaxAge); (got != "") != want {
					t.Fatalf("Warning() = %q, want attention %v", got, want)
				}
			})
		}
	}
	for hours := 1; hours <= 100; hours++ {
		s := DatabaseBackupSummary{Status: DatabaseFailed, LastVerified: now.Add(-time.Duration(hours) * time.Hour)}
		for limit := 1; limit <= 100; limit++ {
			if got := s.NeedsAttention(now, time.Duration(limit)*time.Hour); got != (hours > limit) {
				t.Fatalf("age=%dh limit=%dh: attention=%v", hours, limit, got)
			}
		}
	}
	t.Run("future evidence never warns", func(t *testing.T) {
		s := DatabaseBackupSummary{Status: DatabaseFailed, LastVerified: now.Add(time.Hour)}
		if s.NeedsAttention(now, -2*time.Hour) {
			t.Fatal("future verification must never need attention")
		}
	})
}

func TestParseDatabaseMaxAge(t *testing.T) {
	for _, tt := range []struct {
		raw     string
		want    time.Duration
		invalid bool
	}{
		{"", 48 * time.Hour, false}, {"48h", 48 * time.Hour, false}, {"1h30m", 90 * time.Minute, false},
		{"1ns", time.Nanosecond, false}, {"9223372036854775807ns", time.Duration(1<<63 - 1), false},
		{"0", 0, true}, {"0s", 0, true}, {"-1h", 0, true}, {"24", 0, true},
		{"garbage", 0, true}, {" 48h ", 0, true}, {"9223372036854775808ns", 0, true},
		{"9999999999999999999999999h", 0, true}, {"\x0048h", 0, true},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseDatabaseMaxAge(tt.raw)
			if (err != nil) != tt.invalid || (!tt.invalid && got != tt.want) {
				t.Fatalf("ParseDatabaseMaxAge(%q) = %v, %v; want %v invalid=%v", tt.raw, got, err, tt.want, tt.invalid)
			}
		})
	}
}

func TestDatabaseWarning(t *testing.T) {
	now := time.Date(2026, 10, 19, 4, 0, 0, 0, time.UTC)
	s := DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseFailed, Detail: "publication failed"}
	const recovery = "Ask the hub to repair this backup."
	for _, tt := range []struct {
		verified time.Time
		evidence string
	}{
		{time.Time{}, "never verified"}, {now.Add(-49 * time.Hour), "49h0m0s"},
	} {
		s.LastVerified = tt.verified
		got := s.Warning(now, DefaultDatabaseMaxAge)
		for _, part := range []string{s.Kind, s.Path, tt.evidence, s.Detail} {
			if !strings.Contains(got, part) {
				t.Errorf("warning %q missing %q", got, part)
			}
		}
		if strings.Count(got, recovery) != 1 || !strings.HasSuffix(got, recovery) {
			t.Errorf("warning must end in exactly one recovery action: %q", got)
		}
	}
	controls := "\n\r\t\x00\x1b\u202e\u2066"
	s.Kind = controls + "codex" + strings.Repeat("界", 1000)
	s.Path = controls + "/state.sqlite" + strings.Repeat("界", 1000)
	s.Detail = controls + "busy" + strings.Repeat("界", 1000)
	got := s.Warning(now, DefaultDatabaseMaxAge)
	if !utf8.ValidString(got) || utf8.RuneCountInString(got) > 1024 {
		t.Fatalf("warning must be valid bounded Unicode: %d runes", utf8.RuneCountInString(got))
	}
	for _, r := range got {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			t.Errorf("unsafe rune %U in warning", r)
		}
	}
	if strings.Count(got, recovery) != 1 || !strings.HasSuffix(got, recovery) {
		t.Errorf("long warning lost recovery action: %q", got)
	}
	s.Kind, s.Path, s.Detail = recovery, recovery, "Ask the hub\nto repair this backup."
	got = s.Warning(now, DefaultDatabaseMaxAge)
	if strings.Count(got, recovery) != 1 || !strings.Contains(got, string(s.Status)) {
		t.Errorf("warning must give one recovery and a fallback reason: %q", got)
	}
}

func FuzzParseDatabaseMaxAge(f *testing.F) {
	for _, raw := range []string{"", "48h", "0", "-1ns", "9223372036854775808ns", "\x00", "界"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 256 {
			t.Skip()
		}
		got, err := ParseDatabaseMaxAge(raw)
		if raw == "" {
			if err != nil || got != 48*time.Hour {
				t.Fatalf("default = %v, %v", got, err)
			}
			return
		}
		want, parseErr := time.ParseDuration(raw)
		valid := parseErr == nil && want > 0
		if (err == nil) != valid || (valid && got != want) {
			t.Fatalf("raw=%q: got %v, %v; parsed=%v, %v", raw, got, err, want, parseErr)
		}
	})
}
