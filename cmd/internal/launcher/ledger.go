package launcher

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type LedgerEntry struct {
	Agent        string    `json:"agent"`
	Args         []string  `json:"args"`
	SessionID    string    `json:"session_id"`
	Started      time.Time `json:"started"`
	LastActive   time.Time `json:"last_active"`
	RepoRoot     string    `json:"repo_root"`
	RepoName     string    `json:"repo_name"`
	LegacyImport bool      `json:"legacy_import,omitempty"`
}

func BuildLedgerLine(entry LedgerEntry) (string, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseLedger(raw string) []LedgerEntry {
	var entries []LedgerEntry
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func LatestLedgerEntry(entries []LedgerEntry) (LedgerEntry, bool) {
	if len(entries) == 0 {
		return LedgerEntry{}, false
	}
	latest := entries[0]
	for _, entry := range entries[1:] {
		if entry.LastActive.After(latest.LastActive) || (entry.LastActive.Equal(latest.LastActive) && entry.Started.After(latest.Started)) {
			latest = entry
		}
	}
	return latest, true
}

func LatestLedgerEntryForAgent(entries []LedgerEntry, agent string) (LedgerEntry, bool) {
	var latest LedgerEntry
	ok := false
	for _, entry := range entries {
		if entry.Agent != agent {
			continue
		}
		if !ok || entry.LastActive.After(latest.LastActive) || (entry.LastActive.Equal(latest.LastActive) && entry.Started.After(latest.Started)) {
			latest = entry
			ok = true
		}
	}
	return latest, ok
}

func CompactLedger(entries []LedgerEntry, keepRecent int) []LedgerEntry {
	if keepRecent < 0 {
		keepRecent = 0
	}
	keep := map[int]bool{}
	byRecent := make([]int, len(entries))
	for i := range entries {
		byRecent[i] = i
	}
	sort.SliceStable(byRecent, func(i, j int) bool {
		a, b := entries[byRecent[i]], entries[byRecent[j]]
		if a.LastActive.Equal(b.LastActive) {
			return a.Started.After(b.Started)
		}
		return a.LastActive.After(b.LastActive)
	})
	for i := 0; i < keepRecent && i < len(byRecent); i++ {
		keep[byRecent[i]] = true
	}
	latestByAgent := map[string]int{}
	for i, entry := range entries {
		prev, ok := latestByAgent[entry.Agent]
		if !ok || entry.LastActive.After(entries[prev].LastActive) || (entry.LastActive.Equal(entries[prev].LastActive) && entry.Started.After(entries[prev].Started)) {
			latestByAgent[entry.Agent] = i
		}
	}
	for _, i := range latestByAgent {
		keep[i] = true
	}
	var out []LedgerEntry
	for i, entry := range entries {
		if keep[i] {
			out = append(out, entry)
		}
	}
	return out
}
