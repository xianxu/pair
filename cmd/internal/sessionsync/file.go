// Package sessionsync is the one mutation path for a `type: session-sync`
// file (pair#437): a turn-based markdown file shared by the operator and the
// agent named in its `owner:` frontmatter (parley.nvim#313).
//
// The rules the agent used to keep from its prompt are gates here: a write
// refuses while the operator holds the lock, replaces a section only if it
// still has the hash the agent read (pair#439: pessimistic for the human,
// optimistic for the agent), and finds sections by whole heading lines only. Couch records every
// delivery to a slot in a ledger (ledger.go), so the owner can be shown what
// it still owes the file (owed.go).
package sessionsync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Paths are a session-sync file's companions. The lock is parley's; the
// shadow (the file as the agent last wrote it), the state sidecar and the
// flock file (serializing couch's own commands) are the agent's, and die
// with the file.
type Paths struct {
	File, Lock, Shadow, State, Flock string
}

func PathsOf(file string) Paths {
	dir, base := filepath.Split(file)
	return Paths{
		File:   file,
		Lock:   file + ".lock",
		Shadow: filepath.Join(dir, "."+base+".shadow"),
		State:  filepath.Join(dir, "."+base+".sync"),
		Flock:  filepath.Join(dir, "."+base+".flock"),
	}
}

// Holder is who holds the lock: the operator, or nobody. The agent never
// holds one (pair#439).
type Holder string

const (
	HolderNone     Holder = ""
	HolderOperator Holder = "operator"
)

// ParseHolder reads a lock body. A legacy `holder: agent` lock (parley
// before parley.nvim#315) is no lock; any other lock blocks, because a lock
// nobody can read is not permission to write.
func ParseHolder(lock string, exists bool) Holder {
	if !exists {
		return HolderNone
	}
	for _, line := range strings.Split(lock, "\n") {
		if v, ok := strings.CutPrefix(line, "holder:"); ok && strings.TrimSpace(v) == "agent" {
			return HolderNone
		}
	}
	return HolderOperator
}

// Hash names one version of a section body or a whole file, for `read` to
// hand out and `write --if-hash` to compare.
func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// ParseOwner returns the frontmatter's `owner:` (the owning agent's Couch
// slot), or "" when the file has no frontmatter or no owner.
func ParseOwner(s string) string { return frontmatter(s, "owner") }

// IsSessionSync reports whether the frontmatter says `type: session-sync`.
func IsSessionSync(s string) bool { return frontmatter(s, "type") == "session-sync" }

func frontmatter(s, key string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return ""
		}
		v, ok := strings.CutPrefix(line, key+":")
		if !ok {
			continue
		}
		if i := strings.Index(v, "#"); i >= 0 {
			v = v[:i]
		}
		return strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return ""
}

var stamp = regexp.MustCompile(`Updated \d{4}-\d\d-\d\d \d\d:\d\d`)

// Restamp rewrites the first "Updated YYYY-MM-DD HH:MM" to at.
func Restamp(s string, at time.Time) string {
	done := false
	return stamp.ReplaceAllStringFunc(s, func(m string) string {
		if done {
			return m
		}
		done = true
		return "Updated " + at.Format("2006-01-02 15:04")
	})
}

// section finds the one `## heading` line i and the end j of its body
// (the next `## ` line, or len(lines)). Headings match whole lines only, so
// a heading quoted in prose can never misplace a write; zero or several
// matches refuse.
func section(lines []string, heading string) (i, j int, err error) {
	target := "## " + heading
	var starts []int
	for k, l := range lines {
		if strings.TrimRight(l, " \t") == target {
			starts = append(starts, k)
		}
	}
	if len(starts) != 1 {
		return 0, 0, fmt.Errorf("expected exactly one `%s` heading line, found %d", target, len(starts))
	}
	i, j = starts[0], len(lines)
	for k := i + 1; k < len(lines); k++ {
		if strings.HasPrefix(lines[k], "## ") {
			j = k
			break
		}
	}
	return i, j, nil
}

// SectionBody is the body of the one `## heading` section as stored: every
// line after the heading up to the next section, so any edit to it, even
// whitespace, changes its Hash.
func SectionBody(s, heading string) (string, error) {
	lines := strings.Split(s, "\n")
	i, j, err := section(lines, heading)
	if err != nil {
		return "", err
	}
	return strings.Join(lines[i+1:j], "\n"), nil
}

// ReplaceSection replaces the body of the one `## heading` section.
func ReplaceSection(s, heading, body string) (string, error) {
	lines := strings.Split(s, "\n")
	i, j, err := section(lines, heading)
	if err != nil {
		return "", err
	}
	tail := lines[j:]
	if j == len(lines) {
		// The last section runs to EOF; keep the file's final newline.
		tail = []string{""}
	} else {
		tail = append([]string{""}, tail...)
	}
	out := append(append([]string{}, lines[:i+1]...), "")
	out = append(out, strings.Split(strings.Trim(body, "\n"), "\n")...)
	return strings.Join(append(out, tail...), "\n"), nil
}
