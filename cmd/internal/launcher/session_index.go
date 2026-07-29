package launcher

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// minSessionRepoBytes floors the repo token when the ladder has nothing left to
// drop. Bytes, not runes: the budget being spent is zellij's socket-name budget.
const minSessionRepoBytes = 4

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
	return composeSessionName(sessionNameParts(scope, tag))
}

// composeSessionName joins already-decomposed parts. The ladder shortens parts
// and re-joins through here, so composition lives in exactly one place.
func composeSessionName(repo string, residual []string) string {
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

	// Superseded is the legacy `pair-…` name this entry migrated away from
	// (#130), or "". It rides with the entry to runCreate's commit point, where
	// the stale zellij record is reclaimed — the CALLER does that, never
	// AssignSessionName, which is pure and must not fire a destructive delete on
	// a launch the user goes on to abandon.
	//
	// Deliberately NOT persisted: the ledger records bindings, not the history
	// between them.
	Superseded string `json:"-"`
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

// TagForSessionName recovers the tag a session name is bound to.
//
// The ledger is the only correct inverse for the 📁 scheme: rules 2 and 4 discard
// information, so `📁parley-nvim` cannot be string-surgeried back into the tag
// `parley_nvim`. A 📁 name absent from the ledger therefore yields NO tag rather
// than a plausible-looking wrong one.
//
// The legacy fallback is not a shortcut, it is a different scheme: unindexed
// `pair-<tag>` names predate scoping and genuinely are invertible.
func TagForSessionName(index SessionNameIndex, name string) (string, bool) {
	if entry, ok := index.ownerOf(name); ok && entry.Tag != "" {
		return entry.Tag, true
	}
	if tag, ok := strings.CutPrefix(name, legacySessionPrefix); ok && tag != "" {
		return tag, true
	}
	return "", false
}

// sessionReclaimable reports whether the superseded record named by a migration
// may be force-deleted: only when zellij has no such session, or has one that
// already EXITED.
//
// A DETACHED session is deliberately excluded. It is resumable and may hold real
// work, and migration must never be the thing that destroys it — the user can
// still reach it through the picker or `pair resume`, and it migrates for real
// once they quit it. Only an already-exited record is dead weight, and leaving
// that behind would strand it in `zellij list-sessions` forever, since after
// migration nothing ever names it again.
func sessionReclaimable(live []Session, name string) bool {
	for _, s := range live {
		if s.Name == name {
			return s.State == SessionExited
		}
	}
	return true
}

// BuildSessionNameCandidates yields the names to try for one collision suffix,
// longest first. It carries NO byte budget of its own: the zellij probe
// (`accepts`) is the acceptance oracle, because the real limit is the socket
// path's, which varies by username and platform. The ladder's job is only to
// offer progressively shorter names for that oracle to judge.
func BuildSessionNameCandidates(scope RepoScope, tag string, suffix int) []string {
	repo, residual := sessionNameParts(scope, tag)
	return sessionNameLadder(repo, residual, suffix)
}

// sessionNameLadder is the order of sacrifice when a name must shrink:
//
//  1. the full name;
//  2. residual tag tokens dropped WHOLE, from the right;
//  3. only once no residual is left, the repo token truncated a rune at a time
//     down to minSessionRepoBytes.
//
// Dropping whole tokens before truncating anything is the point of the ladder.
// The old scheme shortened both components a rune at a time and produced
// `pair-parley_nv-parley_nv` — a name whose owner could not tell why it said
// `parley_nv`. A dropped token is at least a name the user recognizes.
//
// Lengths are measured in BYTES (zellij's socket-name budget is bytes) while
// truncation steps by RUNE, so a candidate never contains half a character. The
// two units agreeing for ASCII is what hid this bug: sessionPrefix is 1 rune and
// 4 bytes, so a rune-denominated ladder would be 3 bytes off on every candidate.
func sessionNameLadder(repo string, residual []string, suffix int) []string {
	var out []string
	seen := map[string]bool{}
	add := func(base string) {
		name := withCollisionSuffix(base, suffix)
		if !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}

	for n := len(residual); n >= 0; n-- {
		add(composeSessionName(repo, residual[:n]))
	}
	for short := repo; len(short) > minSessionRepoBytes; {
		short = trimOneRune(short)
		add(composeSessionName(short, nil))
	}
	return out
}

// withCollisionSuffix appends the `-N` disambiguator AssignSessionName walks when
// a composed name is already owned by another (scope, tag). Suffix 1 is the bare
// name. Candidates are measured with their suffix applied, so the ladder needs
// no fixed reservation for it.
func withCollisionSuffix(base string, suffix int) string {
	if suffix > 1 {
		return base + fmt.Sprintf("-%d", suffix)
	}
	return base
}

// trimOneRune drops the last rune, keeping the result valid UTF-8.
func trimOneRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}

// AssignSessionName picks the session name for a (scope, tag), consulting the
// ledger first and walking the candidate ladder otherwise.
//
// Full migration (#130): a ledger row pins the name only once it is ALREADY in
// the 📁 scheme. A legacy `pair-…` row deliberately falls through and re-mints —
// that is the only way the new scheme reaches the tags that motivated the issue,
// since `accepts` asks whether zellij tolerates the name's LENGTH, not whether a
// session exists, so an unconditional short-circuit would pin every existing tag
// to its old name forever.
//
// The prefix clause is load-bearing rather than cosmetic. Drop the short-circuit
// entirely and each create recomposes the same name, clears ownedByOther (same
// scope+tag) and liveOwnedByOther (not live), and appends an identical ledger
// row — once per create, without bound.
func AssignSessionName(index SessionNameIndex, live []Session, scope RepoScope, tag string, accepts func(string) bool) (string, SessionNameIndex, error) {
	if accepts == nil {
		accepts = func(string) bool { return true }
	}
	superseded := ""
	if prior, ok := index.latestFor(scope.Key, tag); ok && accepts(prior.SessionName) {
		if strings.HasPrefix(prior.SessionName, sessionPrefix) {
			return prior.SessionName, index, nil
		}
		superseded = prior.SessionName
	}
	for suffix := 1; suffix <= 100; suffix++ {
		for _, candidate := range BuildSessionNameCandidates(scope, tag, suffix) {
			// The ladder resolves LENGTH, never ownership. Keep walking while
			// candidates are too long for zellij...
			if !accepts(candidate) {
				continue
			}
			// ...but the moment one fits, it is this suffix's answer. If it is
			// taken, bump the suffix rather than shortening further: a shorter
			// name is a DIFFERENT tag's natural name (drop `work` from
			// `📁pair-work` and you have `📁pair`, which belongs to tag `pair`),
			// so shortening to dodge a collision hands out someone else's name.
			if index.ownedByOther(candidate, scope.Key, tag) || liveOwnedByOther(candidate, live, index, scope.Key, tag) {
				break
			}
			entry := SessionNameEntry{
				SessionName: candidate,
				ScopeKey:    scope.Key,
				RepoRoot:    scope.Root,
				RepoName:    scope.DisplayName,
				Tag:         tag,
				Superseded:  superseded,
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
