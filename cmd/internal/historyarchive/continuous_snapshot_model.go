package historyarchive

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// DatabaseBackupStatus describes the result of a database in one continuous pass.
// pair:m5-concept pure
type DatabaseBackupStatus string

const (
	DatabaseArchived  DatabaseBackupStatus = "archived"
	DatabaseUnchanged DatabaseBackupStatus = "unchanged"
	DatabaseDeferred  DatabaseBackupStatus = "deferred"
	DatabaseFailed    DatabaseBackupStatus = "failed"
)

// DatabaseBackupSummary projects runtime acquisition and publication results.
// LastVerified is catalog evidence; CheckedAt only describes this pass's check.
// pair:m5-concept pure
type DatabaseBackupSummary struct {
	Kind, Path              string
	Status                  DatabaseBackupStatus
	CheckedAt, LastVerified time.Time
	Detail                  string
}

// Line renders a deterministic operator report using the caller's clock.
func (s DatabaseBackupSummary) Line(now time.Time) string {
	line := fmt.Sprintf("%s %s: %s", s.Kind, s.Path, s.Status)
	if !s.CheckedAt.IsZero() {
		line += "; checked=" + s.CheckedAt.UTC().Format(time.RFC3339)
	}
	if s.LastVerified.IsZero() {
		line += "; last verified=never"
	} else {
		age := now.Sub(s.LastVerified)
		if age < 0 {
			age = 0
		}
		line += fmt.Sprintf("; last verified=%s (age=%s)", s.LastVerified.UTC().Format(time.RFC3339), age)
	}
	detail := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s.Detail)
	detail = strings.Join(strings.Fields(detail), " ")
	if runes := []rune(detail); len(runes) > 240 {
		detail = string(runes[:237]) + "..."
	}
	if detail != "" {
		line += "; " + detail
	}
	return line
}
