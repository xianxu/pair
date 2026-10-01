package couchmessage

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Candidate is one current observation. Known means both the branch and
// activity observation are available, never an inference from missing data.
type Candidate struct {
	Binding        Binding
	Resting, Known bool
	LastActivity   time.Time
	Pending        bool
	Remaining      int
	Supported      bool
}

func validFamily(s string) bool {
	return s != "" && s != "." && s != ".." && utf8.ValidString(s) && !strings.ContainsAny(s, "/\\:") && strings.IndexFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}
func parseSlot(s string) (string, int, error) {
	ref, recognized, err := couchcore.ParseWorkspaceReference(s)
	if err != nil || !recognized || ref.Repo == "" {
		return "", 0, ErrInvalidTarget
	}
	return ref.Repo, ref.Number, nil
}

// ResolveRecipient does no IO or reservation. Its caller must perform selection
// and pending-slot reservation under one lock against unchanged observations.
func ResolveRecipient(target string, candidates []Candidate, now time.Time) (Binding, error) {
	family := target
	exact := strings.Contains(target, ":")
	if exact {
		var err error
		family, _, err = parseSlot(target)
		if err != nil {
			return Binding{}, err
		}
	} else if !validFamily(family) {
		return Binding{}, ErrInvalidTarget
	}
	var matches []Candidate
	repository := ""
	for _, c := range candidates {
		f, _, err := parseSlot(c.Binding.Slot)
		if err != nil || f != family {
			continue
		}
		if c.Binding.Repository == "" {
			continue
		}
		if repository != "" && repository != c.Binding.Repository {
			return Binding{}, ErrAmbiguous
		}
		repository = c.Binding.Repository
		if !exact || c.Binding.Slot == target {
			matches = append(matches, c)
		}
	}
	if exact {
		if len(matches) == 0 {
			return Binding{}, ErrUnavailable
		}
		if len(matches) > 1 {
			return Binding{}, fmt.Errorf("%w: multiple live bindings for %s", ErrAmbiguous, target)
		}
		c := matches[0]
		if !c.Supported {
			return Binding{}, ErrUnsupported
		}
		if c.Pending {
			return Binding{}, ErrRecipientBusy
		}
		if c.Remaining <= 0 {
			return Binding{}, ErrBudgetExhausted
		}
		return c.Binding, nil
	}
	sort.Slice(matches, func(i, j int) bool {
		_, a, _ := parseSlot(matches[i].Binding.Slot)
		_, b, _ := parseSlot(matches[j].Binding.Slot)
		return a < b
	})
	for i, c := range matches {
		if i > 0 && matches[i-1].Binding.Slot == c.Binding.Slot {
			return Binding{}, ErrAmbiguous
		}
	}
	for _, c := range matches {
		if c.Supported && c.Known && c.Resting && !c.Pending && c.Remaining > 0 && !c.LastActivity.IsZero() && !now.Before(c.LastActivity.Add(QuietInterval)) {
			return c.Binding, nil
		}
	}
	return Binding{}, ErrNoRecipient
}
