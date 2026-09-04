package couchcore

import (
	"strings"
	"testing"
)

// The vocabulary guards itself where the wording now lives, so all three
// renderers are covered by one check. Go cannot verify a switch is exhaustive;
// iterating the enumeration is what stands in for that.
// Distinctness is not string inequality. When `unreadable` was split out of
// `invalid`, ReasonInvalid kept the label "unreadable record" -- so the two
// states printed side by side in one listing, one of them wearing the other's
// defining word, and this guard passed because the strings differed.
//
// A label must not contain another reason's distinctive term.
func TestNoLabelBorrowsAnotherReasonsDefiningWord(t *testing.T) {
	// The word that identifies each state to the operator. A label containing
	// someone else's word is a collision even when the strings differ.
	defining := map[ThreadReason]string{
		ReasonBindingLost:      "binding",
		ReasonStaleIncarnation: "stale",
		ReasonUnrecordedChild:  "unrecorded",
		ReasonSessionGone:      "session gone",
		ReasonNeverStarted:     "never started",
		ReasonInvalid:          "validation",
		ReasonUnreadable:       "could not be read",
		ReasonPathMissing:      "path",
		ReasonProfileMissing:   "saved launch",
		ReasonAgentUnsupported: "agent",
		ReasonUnknown:          "checking",
	}
	for _, reason := range AllThreadReasons() {
		word, named := defining[reason]
		if !named {
			t.Fatalf("reason %q has no defining word -- add one when adding a reason", reason)
		}
		if !strings.Contains(strings.ToLower(reason.Label()), word) {
			t.Errorf("reason %q renders %q, which does not contain its own defining word %q",
				reason, reason.Label(), word)
		}
		for other, otherWord := range defining {
			if other == reason {
				continue
			}
			if strings.Contains(strings.ToLower(reason.Label()), otherWord) {
				t.Errorf("reason %q renders %q, which borrows %q's defining word %q",
					reason, reason.Label(), other, otherWord)
			}
		}
	}
}

func TestEveryReasonHasADistinctOperatorLabel(t *testing.T) {
	seen := map[string]ThreadReason{}
	for _, reason := range AllThreadReasons() {
		label := reason.Label()
		if label == "" || label == string(reason) {
			t.Errorf("reason %q has no operator label of its own (got %q)", reason, label)
			continue
		}
		if other, clash := seen[label]; clash {
			t.Errorf("reasons %q and %q both render %q", reason, other, label)
		}
		seen[label] = reason
	}
}
