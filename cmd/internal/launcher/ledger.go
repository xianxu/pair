package launcher

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type LedgerEntry struct {
	Agent         string    `json:"agent"`
	Args          []string  `json:"args"`
	SessionID     string    `json:"session_id"`
	Started       time.Time `json:"started"`
	LastActive    time.Time `json:"last_active"`
	RepoRoot      string    `json:"repo_root"`
	RepoName      string    `json:"repo_name"`
	LegacyImport  bool      `json:"legacy_import,omitempty"`
	Typed         bool      `json:"-"`
	SourceOrdinal uint64    `json:"-"`
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
	parsed := sessionledger.ParseLedger([]byte(raw))
	compatibility := make(map[uint64]bool, len(parsed.CompatibilityOrdinals))
	for _, ordinal := range parsed.CompatibilityOrdinals {
		compatibility[ordinal] = true
	}
	for index, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !compatibility[uint64(index+1)] {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Agent == "" {
			continue
		}
		entry.SourceOrdinal = uint64(index + 1)
		entries = append(entries, entry)
	}
	owners := map[sessionledger.Owner]bool{}
	for _, record := range parsed.Records {
		if record.Kind == sessionledger.RecordLaunch {
			owners[sessionledger.Owner{ScopeKey: record.ScopeKey, Tag: record.Tag, Agent: record.Agent}] = true
		}
	}
	for owner := range owners {
		current, ok := sessionledger.CurrentLaunch(parsed.Records, owner)
		if !ok {
			continue
		}
		entry := LedgerEntry{Agent: owner.Agent, Typed: true, SourceOrdinal: current.Launch.Ordinal}
		if current.Binding != nil {
			entry.SessionID = current.Binding.RootNativeID
		}
		entries = append(entries, MergeAuthorityMetadata(entry, entries))
	}
	return entries
}

// MergeAuthorityMetadata overlays only presentation fields from the newest
// same-agent compatibility row. Typed root authority and source identity are
// never sourced from compatibility JSON.
// pair:156-concept pure new final
func MergeAuthorityMetadata(typed LedgerEntry, compatibility []LedgerEntry) LedgerEntry {
	if !typed.Typed || typed.Agent == "" {
		return typed
	}
	var newest LedgerEntry
	found := false
	for _, candidate := range compatibility {
		if candidate.Typed || candidate.Agent != typed.Agent {
			continue
		}
		if !found || ledgerEntryNewer(candidate, newest) {
			newest = candidate
			found = true
		}
	}
	if !found {
		return typed
	}
	typed.Args = append([]string(nil), newest.Args...)
	typed.Started = newest.Started
	typed.LastActive = newest.LastActive
	typed.RepoRoot = newest.RepoRoot
	typed.RepoName = newest.RepoName
	return typed
}

func LatestLedgerEntry(entries []LedgerEntry) (LedgerEntry, bool) {
	if len(entries) == 0 {
		return LedgerEntry{}, false
	}
	latest := entries[0]
	for _, entry := range entries[1:] {
		if ledgerEntryNewer(entry, latest) {
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
		if !ok || ledgerEntryNewer(entry, latest) {
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
		return ledgerEntryNewer(a, b)
	})
	for i := 0; i < keepRecent && i < len(byRecent); i++ {
		keep[byRecent[i]] = true
	}
	latestByAgent := map[string]int{}
	for i, entry := range entries {
		prev, ok := latestByAgent[entry.Agent]
		if !ok || ledgerEntryNewer(entry, entries[prev]) {
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

func ledgerEntryNewer(candidate, current LedgerEntry) bool {
	if candidate.Typed != current.Typed {
		return candidate.Typed
	}
	if candidate.Typed && candidate.SourceOrdinal != current.SourceOrdinal {
		return candidate.SourceOrdinal > current.SourceOrdinal
	}
	if candidate.LastActive.Equal(current.LastActive) {
		if candidate.Started.Equal(current.Started) {
			return candidate.SourceOrdinal > current.SourceOrdinal
		}
		return candidate.Started.After(current.Started)
	}
	return candidate.LastActive.After(current.LastActive)
}
