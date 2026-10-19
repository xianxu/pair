package historyarchive

import (
	"strings"
	"testing"
	"time"
)

func TestDatabaseBackupSummaryLine(t *testing.T) {
	now := time.Date(2026, 10, 19, 4, 0, 0, 0, time.UTC)
	checked := time.Date(2026, 10, 19, 3, 59, 0, 0, time.FixedZone("offset", 3600))
	for _, tt := range []struct {
		name    string
		summary DatabaseBackupSummary
		want    string
	}{
		{"archived", DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseArchived, CheckedAt: checked, LastVerified: now}, "codex-database /state.sqlite: archived; checked=2026-10-19T02:59:00Z; last verified=2026-10-19T04:00:00Z (age=0s)"},
		{"unchanged", DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseUnchanged, CheckedAt: now, LastVerified: now.Add(-2 * time.Hour)}, "codex-database /state.sqlite: unchanged; checked=2026-10-19T04:00:00Z; last verified=2026-10-19T02:00:00Z (age=2h0m0s)"},
		{"deferred without bundle", DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseDeferred, Detail: "deadline exceeded"}, "codex-database /state.sqlite: deferred; last verified=never; deadline exceeded"},
		{"failed retains bundle", DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseFailed, LastVerified: now.Add(-48 * time.Hour), Detail: "publication failed"}, "codex-database /state.sqlite: failed; last verified=2026-10-17T04:00:00Z (age=48h0m0s); publication failed"},
		{"future verified time", DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseArchived, LastVerified: now.Add(time.Hour)}, "codex-database /state.sqlite: archived; last verified=2026-10-19T05:00:00Z (age=0s)"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.Line(now); got != tt.want {
				t.Fatalf("Line() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDatabaseBackupSummaryLineSanitizesBoundedDetail(t *testing.T) {
	s := DatabaseBackupSummary{Kind: "codex-database", Path: "/state.sqlite", Status: DatabaseFailed, Detail: "\n busy\twith\x00control\x1b[31m\r\n" + strings.Repeat("界", 300)}
	got := s.Line(time.Time{})
	prefix := "codex-database /state.sqlite: failed; last verified=never; "
	if !strings.HasPrefix(got, prefix+"busy with control [31m ") {
		t.Fatalf("detail not sanitized: %q", got)
	}
	if strings.ContainsAny(got, "\n\t\x00\x1b\r") {
		t.Fatalf("control characters in report: %q", got)
	}
	if n := len([]rune(strings.TrimPrefix(got, prefix))); n > 240 {
		t.Fatalf("detail has %d runes, maximum 240", n)
	}
}
