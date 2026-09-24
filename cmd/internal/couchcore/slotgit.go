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
	// Behind is as fresh as the checkout's last fetch: the probe reads the
	// local remote-tracking ref and never touches the network.
	Behind int
}

// slotGlyphBranch is the Powerline branch symbol (U+E0A0, Nerd Font).
const slotGlyphBranch = ""

// SlotGlyphDiverged marks a resting branch with commits on both sides of its
// upstream. Presentation draws it in the alert colour (pair#319).
const SlotGlyphDiverged = "±"

// SlotGlyphDirty marks a dirty working tree on any branch.
const SlotGlyphDirty = "*"

// RestingBranch is the branch a checkout rests on: main for :0, main-slotN for
// slot N. sdlc workspace asserts the same convention, which validate() checks
// through this helper.
func RestingBranch(n int) string {
	if n == 0 {
		return "main"
	}
	return "main-slot" + strconv.Itoa(n)
}

// SlotGlyph is two independent parts (pair#319). The branch part says where the
// checkout is: off its resting branch (issue work), or on it and diverged from
// its upstream both ways (±), ahead only (+, unpublished) or behind only (-,
// needs a pull). Without an upstream there is no divergence evidence, and an
// issue branch against its upstream is not compared. The dirty part (*) follows
// on any branch, because many operations are only safe on a clean tree.
func SlotGlyph(s SlotGitStatus, resting string) string {
	glyph := slotBranchGlyph(s, resting)
	if s.Dirty {
		glyph += SlotGlyphDirty
	}
	return glyph
}

func slotBranchGlyph(s SlotGitStatus, resting string) string {
	switch {
	case s.Detached || s.Branch != resting:
		return slotGlyphBranch
	case !s.HasUpstream:
		return ""
	case s.Ahead > 0 && s.Behind > 0:
		return SlotGlyphDiverged
	case s.Ahead > 0:
		return "+"
	case s.Behind > 0:
		return "-"
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
	sawHead, sawAB := false, false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case line == "# branch.head" || strings.HasPrefix(line, "# branch.head "):
			head := strings.TrimPrefix(line, "# branch.head ")
			if sawHead || head == "" || strings.ContainsAny(head, " \t\r") {
				return SlotGitStatus{}, fmt.Errorf("malformed branch.head %q", line)
			}
			sawHead = true
			if head == "(detached)" {
				s.Detached = true
			} else {
				s.Branch = head
			}
		case line == "# branch.upstream" || strings.HasPrefix(line, "# branch.upstream "):
			upstream := strings.TrimPrefix(line, "# branch.upstream ")
			if s.HasUpstream || upstream == "" || strings.ContainsAny(upstream, " \t\r") {
				return SlotGitStatus{}, fmt.Errorf("malformed branch.upstream %q", line)
			}
			s.HasUpstream = true
		case line == "# branch.ab" || strings.HasPrefix(line, "# branch.ab "):
			fields := strings.Split(strings.TrimPrefix(line, "# branch.ab "), " ")
			if sawAB || len(fields) != 2 || !strings.HasPrefix(fields[0], "+") || !strings.HasPrefix(fields[1], "-") {
				return SlotGitStatus{}, fmt.Errorf("malformed branch.ab %q", line)
			}
			ahead, aheadErr := strconv.ParseUint(fields[0][1:], 10, strconv.IntSize-1)
			behind, behindErr := strconv.ParseUint(fields[1][1:], 10, strconv.IntSize-1)
			if aheadErr != nil || behindErr != nil {
				return SlotGitStatus{}, fmt.Errorf("malformed branch.ab %q", line)
			}
			sawAB = true
			s.Ahead, s.Behind = int(ahead), int(behind)
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
