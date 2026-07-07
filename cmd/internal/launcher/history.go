package launcher

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HistorySource scans Pair draft/log sidecars under the data directory.
type HistorySource struct {
	DataDir string
}

func (s HistorySource) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
	latest := map[string]time.Time{}
	for _, pattern := range []string{"draft-*.md", "log-*.md"} {
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
			if info.ModTime().Before(cutoff) {
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
		out = append(out, HistoricalTag{Tag: tag, MTime: latest[tag], QueueCount: s.queueCount(tag)})
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

func tagFromSidecar(name string) (string, bool) {
	switch {
	case strings.HasPrefix(name, "draft-") && strings.HasSuffix(name, ".md"):
		return strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".md"), true
	case strings.HasPrefix(name, "log-") && strings.HasSuffix(name, ".md"):
		return strings.TrimSuffix(strings.TrimPrefix(name, "log-"), ".md"), true
	default:
		return "", false
	}
}

func matchesHistoryBase(tag, base string) bool {
	return tag == base || strings.HasPrefix(tag, base+"-")
}
