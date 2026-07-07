package launcher

import (
	"encoding/json"
	"fmt"
	"strings"
)

const minSessionComponentRunes = 4

type SessionNameEntry struct {
	SessionName string `json:"session_name"`
	ScopeKey    string `json:"scope_key"`
	RepoRoot    string `json:"repo_root"`
	RepoName    string `json:"repo_name"`
	Tag         string `json:"tag"`
}

type SessionNameIndex struct {
	Entries []SessionNameEntry
}

func BuildSessionNameIndexLine(entry SessionNameEntry) (string, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseSessionNameIndex(raw string) SessionNameIndex {
	var index SessionNameIndex
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry SessionNameEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		index.Entries = append(index.Entries, entry)
	}
	return index
}

type SessionNameExhausted struct {
	RepoName string
	Tag      string
}

func (e SessionNameExhausted) Error() string {
	return fmt.Sprintf("repo/tag too long for zellij socket path; choose a shorter tag: %s/%s", e.RepoName, e.Tag)
}

func PublicSessionBase(scope RepoScope, tag string) string {
	return "pair-" + NormalizeDisplayComponent(scope.DisplayName) + "-" + NormalizeDisplayComponent(tag)
}

func BuildSessionNameCandidates(scope RepoScope, tag string, suffix int) []string {
	repo := []rune(NormalizeDisplayComponent(scope.DisplayName))
	tagPart := []rune(NormalizeDisplayComponent(tag))
	var out []string
	seen := map[string]bool{}
	for {
		name := publicSessionName(repo, tagPart, suffix)
		if !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
		if len(repo) <= minSessionComponentRunes && len(tagPart) <= minSessionComponentRunes {
			break
		}
		if len(repo) >= len(tagPart) && len(repo) > minSessionComponentRunes {
			repo = repo[:len(repo)-1]
			continue
		}
		if len(tagPart) > minSessionComponentRunes {
			tagPart = tagPart[:len(tagPart)-1]
			continue
		}
	}
	return out
}

func publicSessionName(repo, tag []rune, suffix int) string {
	name := "pair-" + string(repo) + "-" + string(tag)
	if suffix > 1 {
		name += fmt.Sprintf("-%d", suffix)
	}
	return name
}

func AssignSessionName(index SessionNameIndex, live []Session, scope RepoScope, tag string, accepts func(string) bool) (string, SessionNameIndex, error) {
	if accepts == nil {
		accepts = func(string) bool { return true }
	}
	if prior, ok := index.latestFor(scope.Key, tag); ok && accepts(prior.SessionName) {
		return prior.SessionName, index, nil
	}
	for suffix := 1; suffix <= 100; suffix++ {
		for _, candidate := range BuildSessionNameCandidates(scope, tag, suffix) {
			if !accepts(candidate) || index.ownedByOther(candidate, scope.Key, tag) || liveOwnedByOther(candidate, live, index, scope.Key, tag) {
				continue
			}
			entry := SessionNameEntry{
				SessionName: candidate,
				ScopeKey:    scope.Key,
				RepoRoot:    scope.Root,
				RepoName:    scope.DisplayName,
				Tag:         tag,
			}
			index.Entries = append(index.Entries, entry)
			return candidate, index, nil
		}
	}
	return "", index, SessionNameExhausted{RepoName: scope.DisplayName, Tag: tag}
}

func (i SessionNameIndex) latestFor(scopeKey, tag string) (SessionNameEntry, bool) {
	for n := len(i.Entries) - 1; n >= 0; n-- {
		e := i.Entries[n]
		if e.ScopeKey == scopeKey && e.Tag == tag {
			return e, true
		}
	}
	return SessionNameEntry{}, false
}

func (i SessionNameIndex) ownerOf(sessionName string) (SessionNameEntry, bool) {
	for n := len(i.Entries) - 1; n >= 0; n-- {
		e := i.Entries[n]
		if e.SessionName == sessionName {
			return e, true
		}
	}
	return SessionNameEntry{}, false
}

func (i SessionNameIndex) ownedByOther(sessionName, scopeKey, tag string) bool {
	e, ok := i.ownerOf(sessionName)
	return ok && (e.ScopeKey != scopeKey || e.Tag != tag)
}

func liveOwnedByOther(sessionName string, live []Session, index SessionNameIndex, scopeKey, tag string) bool {
	liveHere := false
	for _, s := range live {
		if s.Name == sessionName && s.State != SessionExited {
			liveHere = true
			break
		}
	}
	if !liveHere {
		return false
	}
	e, ok := index.ownerOf(sessionName)
	return !ok || e.ScopeKey != scopeKey || e.Tag != tag
}

func SessionsForScope(sessions []Session, index SessionNameIndex, scope RepoScope) []Session {
	var out []Session
	for _, session := range sessions {
		entry, ok := index.ownerOf(session.Name)
		if !ok || entry.ScopeKey != scope.Key {
			continue
		}
		session.Tag = entry.Tag
		session.RepoName = entry.RepoName
		out = append(out, session)
	}
	return out
}
