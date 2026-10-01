package couchmessage

import (
	"errors"
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
// Alias is display-only: the operator's short name for the repository.
type Candidate struct {
	Binding        Binding
	Resting, Known bool
	LastActivity   time.Time
	Pending        bool
	Remaining      int
	Supported      bool
	Alias          string `json:",omitempty"`
}

// Route is a sender's request: a repository family or exact repo:N, whose
// repository part may be an alias or unique prefix, optionally narrowed to
// slots running one agent.
type Route struct{ Target, Agent string }

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

// liveCandidates lists live slots for a miss, bounded like repository
// candidates so the response stays small.
func liveCandidates(candidates []Candidate) string {
	rows := make([]string, 0, len(candidates))
	for _, c := range candidates {
		rows = append(rows, c.Binding.Slot+" "+c.Binding.Agent)
	}
	return couchcore.FormatBoundedList(rows)
}

// canonicalTarget rewrites the route's repository part to a live family. The
// candidate set is every live binding's family plus its alias; ambiguity is
// refused, and a miss keeps the original target so the caller reports it.
func canonicalTarget(target string, candidates []Candidate, families map[string]string) (string, bool, error) {
	family, number, exact := target, 0, strings.Contains(target, ":")
	if exact {
		var err error
		if family, number, err = parseSlot(target); err != nil {
			return "", false, err
		}
	} else if !validFamily(family) {
		return "", false, ErrInvalidTarget
	}
	// Enrolled families join the live ones: an offline repository's exact name
	// must resolve to itself and miss, never prefix-route to a longer live name.
	var names []couchcore.RepositoryName
	for f, alias := range families {
		names = append(names, couchcore.RepositoryName{Key: f, Dir: f, Alias: alias})
	}
	for _, c := range candidates {
		if f, _, err := parseSlot(c.Binding.Slot); err == nil {
			names = append(names, couchcore.RepositoryName{Key: f, Dir: f, Alias: families[f]})
		}
	}
	found, _, err := couchcore.ResolveRepositoryName(family, names)
	switch {
	case errors.Is(err, couchcore.ErrRepositoryAmbiguous):
		return "", false, fmt.Errorf("%w: %v", ErrAmbiguous, err)
	case err != nil:
		return target, false, nil
	case exact:
		return couchcore.WorkspaceReference{Repo: found.Key, Number: number}.String(), true, nil
	}
	return found.Key, true, nil
}

// ResolveRecipient does no IO or reservation. Its caller must perform selection
// and pending-slot reservation under one lock against unchanged observations.
// families maps every enrolled family (directory name) to its operator alias,
// "" when it has none.
func ResolveRecipient(route Route, candidates []Candidate, families map[string]string, now time.Time) (Binding, error) {
	target, _, err := canonicalTarget(route.Target, candidates, families)
	if err != nil {
		return Binding{}, err
	}
	family := target
	exact := strings.Contains(target, ":")
	if exact {
		family, _, _ = parseSlot(target)
	}
	familyLive := false
	for _, c := range candidates {
		if f, _, err := parseSlot(c.Binding.Slot); err == nil && f == family {
			familyLive = true
		}
	}
	// An unknown or offline repository names the live slots, so the sender can
	// correct in one step; a live repository's busy or missing slot does not.
	miss := func(err error) error {
		if familyLive {
			return err
		}
		return fmt.Errorf("%w: no live slot matches %q; live: %s", err, route.Target, liveCandidates(candidates))
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
		// Repository identity is checked across the whole family before the
		// agent filter, so a filter can never hide an ambiguous family.
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
			return Binding{}, miss(ErrUnavailable)
		}
		if len(matches) > 1 {
			return Binding{}, fmt.Errorf("%w: multiple live bindings for %s", ErrAmbiguous, target)
		}
		c := matches[0]
		if route.Agent != "" && c.Binding.Agent != route.Agent {
			return Binding{}, fmt.Errorf("%w: %s runs %s, not %s", ErrUnavailable, target, c.Binding.Agent, route.Agent)
		}
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
	if len(matches) == 0 {
		return Binding{}, miss(ErrNoRecipient)
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
		if route.Agent != "" && c.Binding.Agent != route.Agent {
			continue
		}
		if c.Supported && c.Known && c.Resting && !c.Pending && c.Remaining > 0 && !c.LastActivity.IsZero() && !now.Before(c.LastActivity.Add(QuietInterval)) {
			return c.Binding, nil
		}
	}
	if route.Agent != "" {
		return Binding{}, fmt.Errorf("%w running %s in %s", ErrNoRecipient, route.Agent, family)
	}
	return Binding{}, ErrNoRecipient
}
