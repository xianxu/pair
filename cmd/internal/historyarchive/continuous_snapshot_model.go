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
	Source                  SourceIdentity
	Kind, Path              string
	Status                  DatabaseBackupStatus
	CheckedAt, LastVerified time.Time
	Detail                  string
}

// DefaultDatabaseMaxAge is the grace period for failed or deferred backups.
const DefaultDatabaseMaxAge = 48 * time.Hour

// ParseDatabaseMaxAge accepts positive Go durations, defaulting an empty setting.
func ParseDatabaseMaxAge(raw string) (time.Duration, error) {
	if raw == "" {
		return DefaultDatabaseMaxAge, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("database max age: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("database max age must be positive")
	}
	return d, nil
}

// NeedsAttention applies the grace period only to unsuccessful acquisitions.
func (s DatabaseBackupSummary) NeedsAttention(now time.Time, maxAge time.Duration) bool {
	if s.Status != DatabaseFailed && s.Status != DatabaseDeferred {
		return false
	}
	return s.LastVerified.IsZero() || (!s.LastVerified.After(now) && now.Sub(s.LastVerified) > maxAge)
}

// Warning is a bounded operator diagnostic with one explicit recovery action.
func (s DatabaseBackupSummary) Warning(now time.Time, maxAge time.Duration) string {
	if !s.NeedsAttention(now, maxAge) {
		return ""
	}
	const recovery = "Ask the hub to repair this backup."
	clean := func(value string, limit int) string {
		// Format controls can reorder terminal text without producing a newline.
		value = strings.Map(func(r rune) rune {
			if unicode.Is(unicode.Cf, r) {
				return ' '
			}
			return r
		}, value)
		value = compactDatabaseText(value, limit)
		return strings.TrimSpace(strings.ReplaceAll(value, recovery, ""))
	}
	evidence := "never verified"
	if !s.LastVerified.IsZero() {
		evidence = "last verified age=" + now.Sub(s.LastVerified).String()
	}
	reason := clean(s.Detail, 240)
	if reason == "" {
		reason = string(s.Status)
	}
	return fmt.Sprintf("%s %s: %s; %s; %s", clean(s.Kind, 96), clean(s.Path, 320), evidence, reason, recovery)
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
	detail := compactDatabaseText(s.Detail, 240)
	if detail != "" {
		line += "; " + detail
	}
	return line
}

func compactDatabaseText(value string, limit int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if runes := []rune(value); len(runes) > limit {
		value = string(runes[:limit-3]) + "..."
	}
	return value
}
