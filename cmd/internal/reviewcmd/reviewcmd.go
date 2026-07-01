// Package reviewcmd is the Go owner of pair's three review-start orchestrators
// (#93 M3, ported from bin/pair-review-target / -open / -readiness):
//
//   - target:    stamp review-target-<tag>.json with the session-scoped target.
//   - open:      spawn the full-screen floating nvim review pane (single pane).
//   - readiness: gather git facts, classify the review-start action (via the
//     pure nvim/review/readiness.lua — single source), and either
//     emit JSON or perform the deterministic --prepare git effects.
//
// nvim/review/*.lua stays native (#95). Pure decisions live here; git / nvim
// -classify / zellij-spawn / codex-sid sit behind the Runtime seam.
package reviewcmd

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

// ReadinessFacts are the git facts the pure classifier (readiness.lua) consumes.
type ReadinessFacts struct {
	IsGit          bool
	IsTracked      bool
	OnReviewBranch bool
	FileMatches    bool
	IsClean        bool
}

// targetDoc is the review-target-<tag>.json shape (session-scoped target seam #6).
type targetDoc struct {
	File    string `json:"file"`
	Status  string `json:"status"`
	Session string `json:"session"`
}

func targetJSON(file, status, session string) string {
	b, _ := json.Marshal(targetDoc{File: file, Status: status, Session: session})
	return string(b)
}

// readinessDoc is the JSON-mode output of pair-review-readiness.
type readinessDoc struct {
	Case           string `json:"case"`
	IsGit          bool   `json:"is_git"`
	IsTracked      bool   `json:"is_tracked"`
	Branch         string `json:"branch"`
	OnReviewBranch bool   `json:"on_review_branch"`
	ScopedFile     string `json:"scoped_file"`
	FileMatches    bool   `json:"file_matches"`
	IsClean        bool   `json:"is_clean"`
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns a path into a review-branch slug: basename without extension,
// lowercased, non-alphanumerics collapsed to single dashes, trimmed. Mirrors the
// shell's `tr`/`sed` pipeline in pair-review-readiness.
func slugify(path string) string {
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	base = strings.ToLower(base)
	base = nonAlnum.ReplaceAllString(base, "-")
	return strings.Trim(base, "-")
}
