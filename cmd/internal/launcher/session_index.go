package launcher

import (
	"encoding/json"
	"fmt"
	"strings"
)

const minSessionComponentRunes = 4

// The session-name scheme (#130). A zellij session name is a socket filename
// with a small byte budget, so the prefix is spent carefully: sessionPrefix is
// 4 bytes and needs no separator, where legacySessionPrefix cost 5.
//
// The prefix is pair's ownership marker in zellij's GLOBAL namespace — it is
// what keeps `delete-session --force` off a stranger's abandoned session — so
// both spellings must be recognized even though only sessionPrefix is ever
// emitted.
const (
	sessionPrefix       = "📁"
	legacySessionPrefix = "pair-"
)

// isPairSessionName reports whether name is in pair's ownership namespace, in
// either scheme. This answers the OWNERSHIP question only: "may pair touch
// this?". It is deliberately NOT the inverse of the naming scheme — see
// tagFromLegacySessionName for why recovering a tag from a name works for
// legacySessionPrefix and cannot work for sessionPrefix.
func isPairSessionName(name string) bool {
	return strings.HasPrefix(name, sessionPrefix) || strings.HasPrefix(name, legacySessionPrefix)
}

// sessionNameParts decomposes a scope + tag into the two pieces the scheme is
// built from: the repo token (rule 2 — the first alphanumeric run of the
// normalized display name) and the residual tag tokens (rule 4 — the tag's
// tokens with any leading run matching the repo token dropped).
//
// This is the single implementation of rules 2-4; ComposeSessionName joins its
// output and the ladder shortens it, so neither re-derives the tokenization
// (ARCH-DRY). It composes on NormalizeDisplayComponent rather than re-deriving
// sanitisation — that already maps `parley.nvim` to `parley_nvim`.
func sessionNameParts(scope RepoScope, tag string) (repo string, residual []string) {
	repoTokens := alnumTokens(NormalizeDisplayComponent(scope.DisplayName))
	tagTokens := alnumTokens(NormalizeDisplayComponent(tag))

	repo = "pair" // NormalizeDisplayComponent's fallback, for a name with no alphanumerics at all
	if len(repoTokens) > 0 {
		repo = repoTokens[0]
	}

	// Rule 4 compares against the repo TOKEN, not the whole normalized name:
	// repo `parley` vs tag `[parley, nvim]` leaves `[nvim]`, which is what makes
	// `📁parley-nvim` rather than `📁parley`.
	//
	// Exactly one token is dropped, not the whole leading run. The repo side is
	// a single token, so "drop the tag's leading tokens that match the repo's"
	// is a one-token prefix. Dropping the run would also fold two distinct tags
	// onto one name — tag `pair-pair-x` and tag `pair-x` would both compose to
	// `📁pair-x`, and the collision would surface as an opaque numeric suffix.
	if len(tagTokens) > 0 && tagTokens[0] == repo {
		tagTokens = tagTokens[1:]
	}
	return repo, tagTokens
}

// ComposeSessionName builds a public session name from a scope and a tag:
// sessionPrefix + repo token + the residual tag tokens, hyphen-joined.
//
// The result is deliberately NOT invertible — rules 2 and 4 discard information
// — so recovering a tag from a name goes through SessionNameIndex, never through
// string surgery.
func ComposeSessionName(scope RepoScope, tag string) string {
	repo, residual := sessionNameParts(scope, tag)
	if len(residual) == 0 {
		return sessionPrefix + repo
	}
	return sessionPrefix + repo + "-" + strings.Join(residual, "-")
}

// alnumTokens splits an already-normalized component on its non-alphanumeric
// separators (NormalizeDisplayComponent leaves only `-` and `_`), dropping empty
// runs so `a__b` and `a-b` tokenize alike.
func alnumTokens(normalized string) []string {
	var out []string
	cur := make([]rune, 0, len(normalized))
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
		}
	}
	for _, r := range normalized {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cur = append(cur, r)
			continue
		}
		flush()
	}
	flush()
	return out
}

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
