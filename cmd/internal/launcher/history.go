package launcher

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HistorySource scans Pair draft/log/ledger sidecars under the data directory.
type HistorySource struct {
	DataDir       string
	LegacyDataDir string
}

func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
	latest := map[string]time.Time{}
	for _, pattern := range []string{"draft-*.md", "log-*.md", "ledger-*.jsonl"} {
		matches, err := filepath.Glob(filepath.Join(s.DataDir, pattern))
		if err != nil {
			return nil, err
		}
		for _, path := range matches {
			tag, ok := tagFromSidecar(filepath.Base(path))
			if !ok {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			mtime := info.ModTime()
			if strings.HasPrefix(filepath.Base(path), "ledger-") {
				if entry, ok := s.latestLedgerEntry(tag); ok && !entry.LastActive.IsZero() {
					mtime = entry.LastActive
				}
			}
			if mtime.Before(cutoff) {
				continue
			}
			if mtime.After(latest[tag]) {
				latest[tag] = mtime
			}
		}
	}
	tags := make([]string, 0, len(latest))
	for tag := range latest {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	out := make([]HistoricalTag, 0, len(tags))
	for _, tag := range tags {
		row := HistoricalTag{Tag: tag, MTime: latest[tag], QueueCount: s.queueCount(tag)}
		if entry, ok := s.latestLedgerEntry(tag); ok {
			row.Agent = entry.Agent
			row.RepoName = entry.RepoName
			if entry.LastActive.After(row.MTime) {
				row.MTime = entry.LastActive
			}
		}
		out = append(out, row)
	}
	if s.LegacyDataDir != "" && s.LegacyDataDir != s.DataDir {
		legacy, err := s.scanLegacy(base, cutoff, latest)
		if err != nil {
			return nil, err
		}
		out = append(out, legacy...)
	}
	sortHistoricalRows(out)
	return out, nil
}

func (s HistorySource) scanLegacy(base string, cutoff time.Time, scoped map[string]time.Time) ([]HistoricalTag, error) {
	latest := map[string]time.Time{}
	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
		matches, err := filepath.Glob(filepath.Join(s.LegacyDataDir, pattern))
		if err != nil {
			return nil, err
		}
		for _, path := range matches {
			tag, ok := tagFromSidecar(filepath.Base(path))
			if !ok || !matchesHistoryBase(tag, base) {
				continue
			}
			if _, exists := scoped[tag]; exists {
				continue
			}
			info, err := os.Stat(path)
			if err != nil || info.ModTime().Before(cutoff) {
				continue
			}
			if info.ModTime().After(latest[tag]) {
				latest[tag] = info.ModTime()
			}
		}
	}
	tags := make([]string, 0, len(latest))
	for tag := range latest {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	out := make([]HistoricalTag, 0, len(tags))
	for _, tag := range tags {
		out = append(out, HistoricalTag{Tag: tag, MTime: latest[tag], QueueCount: s.legacyQueueCount(tag), LegacyUnscoped: true})
	}
	return out, nil
}

// queueCount counts the queued prompts nvim parked under queue-<tag>/ — each is a
// <digits>.md file (queue_count_for, shell 1335). Surfaced as the picker's amber
// badge so a forgotten queue is visible before resuming.
func (s HistorySource) queueCount(tag string) int {
	matches, err := filepath.Glob(filepath.Join(s.DataDir, "queue-"+tag, "[0-9]*.md"))
	if err != nil {
		return 0
	}
	return len(matches)
}

func (s HistorySource) legacyQueueCount(tag string) int {
	matches, err := filepath.Glob(filepath.Join(s.LegacyDataDir, "queue-"+tag, "[0-9]*.md"))
	if err != nil {
		return 0
	}
	return len(matches)
}

func tagFromSidecar(name string) (string, bool) {
	switch {
	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
	case strings.HasPrefix(name, "ledger-") && strings.HasSuffix(name, ".jsonl"):
		return strings.TrimSuffix(strings.TrimPrefix(name, "ledger-"), ".jsonl"), true
	default:
		return "", false
	}
}

func matchesHistoryBase(tag, base string) bool {
	return tag == base || strings.HasPrefix(tag, base+"-")
}

func (s HistorySource) latestLedgerEntry(tag string) (LedgerEntry, bool) {
	raw, err := os.ReadFile(filepath.Join(s.DataDir, "ledger-"+tag+".jsonl"))
	if err != nil {
		return LedgerEntry{}, false
	}
	return LatestLedgerEntry(ParseLedger(string(raw)))
}

func sortHistoricalRows(rows []HistoricalTag) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].MTime.Equal(rows[j].MTime) {
			return rows[i].Tag < rows[j].Tag
		}
		return rows[i].MTime.After(rows[j].MTime)
	})
}
