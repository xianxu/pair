package launcher

import (
	"fmt"
	"strings"
)

// The fzf session picker (#99 M5a, ported from bin/pair-shell 1428-1508). When a
// bare `pair` finds detached and/or historical pair-<tag> sessions, fzf offers
// them plus a "+ new" row. The row *build* is a pure function over the decision
// snapshot (detached-first ordering, age grey-grading, the queued badge); only
// the fzf call itself (resolvePick) is a Runtime effect. Picking an existing tag
// is resume-by-name — it re-enters DecideLaunch under a ForcedTag, so attach-vs-
// create + agent inference match `pair resume <tag>`.

// pickSelection is what a chosen picker row resolves to: a specific tag, or the
// "+ new" row (create a fresh free-slot tag with the name prompt).
type pickSelection struct {
	tag   string
	isNew bool
}

const (
	ansiGreen = "\033[32m"
	ansiReset = "\033[0m"
	ansiAmber = "\033[38;5;214m" // the queued badge (xterm 214)
)

// buildPickRows renders the picker's display rows (ANSI-colored for fzf --ansi)
// and a map from each row's PLAIN text (what fzf --ansi returns, color stripped)
// to its selection. Order mirrors the shell: live detached sessions (green)
// first, then historical "no live session" rows (age-graded grey + amber queued
// badge), then the "+ new <base> session" row. Historical tags that still have a
// live pair-<tag> (any state) are deduped out — they already show as their own
// live row (shell 1387).
func buildPickRows(snap SessionSnapshot, base string, nowEpoch int64) (display []string, byPlain map[string]pickSelection) {
	byPlain = map[string]pickSelection{}
	live := map[string]bool{}
	for _, s := range snap.Sessions {
		live[strings.TrimPrefix(s.Name, "pair-")] = true
	}

	add := func(plain, colored string, sel pickSelection) {
		display = append(display, colored)
		byPlain[plain] = sel
	}

	for _, s := range snap.Sessions {
		if s.State != SessionDetached {
			continue
		}
		add(s.Name, ansiGreen+s.Name+ansiReset, pickSelection{tag: strings.TrimPrefix(s.Name, "pair-")})
	}

	for _, h := range snap.Historical {
		if live[h.Tag] {
			continue // already surfaced as a live row
		}
		baseRow := fmt.Sprintf("pair-%s  (%s, no live session)", h.Tag, FormatAge(nowEpoch, h.MTime.Unix()))
		badgePlain, badgeColored := "", ""
		if h.QueueCount > 0 {
			badgePlain = fmt.Sprintf("   [⏎ %d queued]", h.QueueCount)
			badgeColored = fmt.Sprintf("   %s[⏎ %d queued]%s", ansiAmber, h.QueueCount, ansiReset)
		}
		days := int((nowEpoch - h.MTime.Unix()) / secondsPerDay)
		add(baseRow+badgePlain, AgeColor(days)+baseRow+ansiReset+badgeColored, pickSelection{tag: h.Tag})
	}

	newLabel := fmt.Sprintf("+ new %s session", base)
	add(newLabel, newLabel, pickSelection{isNew: true})
	return display, byPlain
}

// resolvePick presents the picker and maps the choice into a concrete launch
// decision. aborted=true means the user dismissed fzf (ESC/empty) — the caller
// exits 0. "+ new" builds a fresh free-slot create (with the name prompt); an
// existing tag re-enters DecideLaunch under a ForcedTag (attach if live, else
// create-by-name), the resume-by-name path.
func resolvePick(rt Runtime, snap SessionSnapshot, base string, nowEpoch int64) (LaunchDecision, bool) {
	display, byPlain := buildPickRows(snap, base, nowEpoch)
	picked := rt.PickFromList("pick a pair session", display, 10)
	if picked == "" {
		return LaunchDecision{}, true
	}
	sel, ok := byPlain[picked]
	if !ok {
		return LaunchDecision{}, true // fzf returned an unmapped line — abort safely.
	}
	if sel.isNew {
		return createDecision(nextFreeTag(base, snap), true), false
	}
	d, _ := DecideLaunch(LaunchArgs{ForcedTag: sel.tag}, snap) // never errors (no pick recursion)
	return d, false
}
