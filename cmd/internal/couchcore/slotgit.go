package couchcore

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SlotGitStatus is one observation of a checkout, taken from a single porcelain
// v2 status read (pair#317). It is display evidence for the slot quick-status
// glyph, never mutation authority: the checkout can change the moment after.
type SlotGitStatus struct {
	Branch      string
	Detached    bool
	Dirty       bool
	HasUpstream bool
	Ahead       int
}

// slotGlyphBranch is the Powerline branch symbol (U+E0A0, Nerd Font).
const slotGlyphBranch = ""

// RestingBranch is the branch a checkout rests on: main for :0, main-slotN for
// slot N. sdlc workspace asserts the same convention, which validate() checks
// through this helper.
func RestingBranch(n int) string {
	if n == 0 {
		return "main"
	}
	return "main-slot" + strconv.Itoa(n)
}

// SlotGlyph applies the precedence: off the resting branch (issue work) beats a
// dirty tree, which beats commits not yet on the upstream. Without an upstream
// there is no evidence of unpublished work, so nothing is shown.
func SlotGlyph(s SlotGitStatus, resting string) string {
	switch {
	case s.Detached || s.Branch != resting:
		return slotGlyphBranch
	case s.Dirty:
		return "*"
	case s.HasUpstream && s.Ahead > 0:
		return "+"
	}
	return ""
}

// slotGitStatusArgs reads branch, upstream divergence and dirtiness in one
// call. --no-optional-locks keeps a background read from taking index.lock
// under the operator's own git command in that checkout.
var slotGitStatusArgs = []string{"--no-optional-locks", "status", "--porcelain=v2", "--branch"}

func ProbeSlotGit(ctx context.Context, git GitRunner, dir string) (SlotGitStatus, error) {
	out, err := git.RunContext(ctx, dir, slotGitStatusArgs...)
	if err != nil {
		return SlotGitStatus{}, err
	}
	return ParseSlotGitStatus(out)
}

// ParseSlotGitStatus reads `git status --porcelain=v2 --branch` as a closed
// grammar: branch.head is required, a malformed branch.ab or an unknown entry
// line is an error, and unknown `#` headers are future extensions to ignore.
func ParseSlotGitStatus(out string) (SlotGitStatus, error) {
	var s SlotGitStatus
	sawHead := false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			sawHead = true
			if head := strings.TrimPrefix(line, "# branch.head "); head == "(detached)" {
				s.Detached = true
			} else {
				s.Branch = head
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			s.HasUpstream = true
		case strings.HasPrefix(line, "# branch.ab "):
			var ahead, behind int
			if n, err := fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &ahead, &behind); err != nil || n != 2 || ahead < 0 {
				return SlotGitStatus{}, fmt.Errorf("malformed branch.ab %q", line)
			}
			s.Ahead = ahead
		case line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "! "):
		case len(line) > 2 && strings.IndexByte("12u?", line[0]) >= 0 && line[1] == ' ':
			s.Dirty = true
		default:
			return SlotGitStatus{}, fmt.Errorf("unknown status line %q", line)
		}
	}
	if !sawHead {
		return SlotGitStatus{}, fmt.Errorf("missing branch.head")
	}
	return s, nil
}
