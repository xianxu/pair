package couchcore

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

var (
	ErrRepositoryNotFound  = errors.New("no repository matches")
	ErrRepositoryAmbiguous = errors.New("repository reference is ambiguous")
)

// MaxRepositoryCandidateBytes bounds a miss's candidate list, which travels in
// CLI errors and message-broker responses.
const MaxRepositoryCandidateBytes = 1024

const maxRepositoryCandidates = 12

// RepositoryName is one addressable repository. Key is the caller's identity
// for it (a primary root, or a live message family); Dir is its directory
// name; Alias is the operator's optional short name.
type RepositoryName struct{ Key, Dir, Alias string }

type RepositoryMatch uint8

const (
	RepositoryNone RepositoryMatch = iota
	RepositoryPrefix
	RepositoryExact
)

func (r RepositoryName) display() string {
	if r.Alias == "" {
		return r.Dir
	}
	return r.Dir + " (" + r.Alias + ")"
}

// ResolveRepositoryName is the one rule every slot reference uses for its
// repository part: an exact directory name or alias wins, otherwise a unique
// prefix of either. Hits are counted per Key, so the directory and alias of
// one repository never compete, while two repositories sharing a name are
// refused rather than picked.
func ResolveRepositoryName(raw string, repos []RepositoryName) (RepositoryName, RepositoryMatch, error) {
	if !workspaceRepoName(raw) || strings.TrimSpace(raw) != raw {
		return RepositoryName{}, RepositoryNone, fmt.Errorf("%w %q; known: %s", ErrRepositoryNotFound, raw, FormatRepositoryCandidates(repos))
	}
	pick := func(match RepositoryMatch, hit func(string) bool) (RepositoryName, bool, error) {
		found := map[string]RepositoryName{}
		for _, r := range repos {
			if hit(r.Dir) || (r.Alias != "" && hit(r.Alias)) {
				found[r.Key] = r
			}
		}
		switch len(found) {
		case 0:
			return RepositoryName{}, false, nil
		case 1:
			for _, r := range found {
				return r, true, nil
			}
		}
		matches := make([]RepositoryName, 0, len(found))
		for _, r := range found {
			matches = append(matches, r)
		}
		return RepositoryName{}, false, fmt.Errorf("%w: %q matches %s", ErrRepositoryAmbiguous, raw, FormatRepositoryCandidates(matches))
	}
	for _, step := range []struct {
		match RepositoryMatch
		hit   func(string) bool
	}{
		{RepositoryExact, func(s string) bool { return s == raw }},
		{RepositoryPrefix, func(s string) bool { return strings.HasPrefix(s, raw) }},
	} {
		r, ok, err := pick(step.match, step.hit)
		if err != nil {
			return RepositoryName{}, RepositoryNone, err
		}
		if ok {
			return r, step.match, nil
		}
	}
	return RepositoryName{}, RepositoryNone, fmt.Errorf("%w %q; known: %s", ErrRepositoryNotFound, raw, FormatRepositoryCandidates(repos))
}

// FormatRepositoryCandidates lists repositories once each, sorted, bounded in
// both count and bytes.
func FormatRepositoryCandidates(repos []RepositoryName) string {
	seen := map[string]bool{}
	var names []string
	for _, r := range repos {
		if label := r.display(); !seen[label] {
			seen[label] = true
			names = append(names, label)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	sort.Strings(names)
	var b strings.Builder
	for i, name := range names {
		rest := fmt.Sprintf(" and %d more", len(names)-i)
		sep := ""
		if i > 0 {
			sep = ", "
		}
		if i == maxRepositoryCandidates || b.Len()+len(sep)+len(name)+len(rest) > MaxRepositoryCandidateBytes {
			b.WriteString(rest)
			break
		}
		b.WriteString(sep + name)
	}
	return b.String()
}

// ValidateRepositoryAlias admits an alias for the repository keyed self. The
// empty alias clears. An alias may not shadow any other repository's name or
// alias, nor restate its own directory name.
func ValidateRepositoryAlias(alias, self string, repos []RepositoryName) error {
	known := false
	for _, r := range repos {
		known = known || r.Key == self
	}
	if !known {
		return fmt.Errorf("%w: %s is not an enrolled repository", ErrRepositoryNotFound, self)
	}
	if alias == "" {
		return nil
	}
	// Stricter than a directory name: an alias must also be a message family.
	if !workspaceRepoName(alias) || strings.IndexFunc(alias, unicode.IsSpace) >= 0 {
		return fmt.Errorf("invalid alias %q: use a name without spaces, '/', '\\' or ':'", alias)
	}
	for _, r := range repos {
		if r.Key == self {
			if r.Dir == alias {
				return fmt.Errorf("alias %q is already the directory name", alias)
			}
			continue
		}
		if r.Dir == alias || r.Alias == alias {
			return fmt.Errorf("alias %q is already used by %s", alias, r.display())
		}
	}
	return nil
}
