package launcher

import (
	"fmt"
	"io"
	"strings"
)

// The fzf session picker (#99 M5a, ported from bin/pair-shell 1428-1508). When a
// bare `pair` finds detached and/or historical Pair sessions, fzf offers
// them plus a "+ new" row. The row *build* is a pure function over the decision
// snapshot (detached-first ordering, age grey-grading, the queued badge); only
// the fzf call itself (resolvePick) is a Runtime effect. Picking an existing tag
// is resume-by-name — it re-enters DecideLaunch under a ForcedTag, so attach-vs-
// create + agent inference match `pair resume <tag>`.

// pickSelection is what a chosen picker row resolves to: a specific tag, or the
// "+ new" row (create a fresh free-slot tag with the name prompt).
type pickSelection struct {
	tag               string
	sessionName       string
	isNew             bool
	legacy            bool
	disabled          bool
	disabledReason    string
	needsContinuation bool
	sourceAgent       string
}

// PickPolicy records launch intent that changes picker row classification. The
// zero value preserves bare `pair`: detached live rows + historical rows.
type PickPolicy struct {
	RequestedAgent string
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
// live Pair sessions (any state) are deduped out — they already show as their own
// live row (shell 1387).
func buildPickRows(snap SessionSnapshot, base string, nowEpoch int64) (display []string, byPlain map[string]pickSelection) {
	return buildPickRowsWithPolicy(snap, base, nowEpoch, PickPolicy{})
}

func buildPickRowsWithPolicy(snap SessionSnapshot, base string, nowEpoch int64, policy PickPolicy) (display []string, byPlain map[string]pickSelection) {
	type row struct {
		plain, colored string
		selection      pickSelection
	}
	var rows []row
	live := map[string]bool{}
	for _, s := range snap.Sessions {
		live[sessionTag(s)] = true
	}

	add := func(plain, colored string, sel pickSelection) {
		rows = append(rows, row{plain: plain, colored: colored, selection: sel})
	}

	for _, s := range snap.Sessions {
		if policy.RequestedAgent == "" && s.State != SessionDetached {
			continue
		}
		if policy.RequestedAgent != "" && s.State != SessionAttached && s.State != SessionDetached {
			continue
		}
		plain := livePickLabel(s, policy)
		sel := pickSelection{tag: sessionTag(s), sessionName: s.Name}
		color := ansiGreen
		if policy.RequestedAgent != "" && s.Agent != policy.RequestedAgent {
			sel.disabled = true
			sel.sessionName = ""
			sel.disabledReason = fmt.Sprintf("pair: session '%s' is driven by %s; detach/exit it before continuing with %s.\n", sessionTag(s), firstNonEmpty(s.Agent, "unknown agent"), policy.RequestedAgent)
			color = AgeColor(14)
		}
		add(plain, color+plain+ansiReset, sel)
	}

	for _, h := range snap.Historical {
		if live[h.Tag] {
			continue // already surfaced as a live row
		}
		if h.LegacyUnscoped {
			plain := fmt.Sprintf("legacy unscoped %s  (manual import)", h.Tag)
			add(plain, AgeColor(int((nowEpoch-h.MTime.Unix())/secondsPerDay))+plain+ansiReset, pickSelection{tag: h.Tag, legacy: true})
			continue
		}
		baseRow := historicalPickLabel(h, nowEpoch)
		badgePlain, badgeColored := "", ""
		if h.QueueCount > 0 {
			badgePlain = fmt.Sprintf("   [⏎ %d queued]", h.QueueCount)
			badgeColored = fmt.Sprintf("   %s[⏎ %d queued]%s", ansiAmber, h.QueueCount, ansiReset)
		}
		days := int((nowEpoch - h.MTime.Unix()) / secondsPerDay)
		sel := pickSelection{tag: h.Tag}
		if policy.RequestedAgent != "" && h.Agent != "" && h.Agent != policy.RequestedAgent {
			sel.needsContinuation = true
			sel.sourceAgent = h.Agent
		}
		add(baseRow+badgePlain, AgeColor(days)+baseRow+ansiReset+badgeColored, sel)
	}

	newLabel := fmt.Sprintf("+ new %s session", base)
	add(newLabel, newLabel, pickSelection{isNew: true})

	byPlain = make(map[string]pickSelection, len(rows))
	for _, row := range rows {
		display = append(display, row.colored)
		byPlain[row.plain] = row.selection
	}
	return display, byPlain
}

// sessionTag is the picker's per-row tag. SessionsForScope populates s.Tag from
// the ledger; the fallbacks below only run for a session it could not resolve.
//
// This one must never return "" (#130). It feeds both the live-dedup key
// (`live[sessionTag(s)]`) and the row's pickSelection, so an empty tag would
// collapse every unresolved row into one dedup bucket and make selection resolve
// to no tag at all. Falling back to the full name keeps the row distinct and
// selectable even when the tag is unrecoverable — which is exactly the case for
// a 📁 name missing from the ledger, since that scheme has no string inverse.
func sessionTag(s Session) string {
	if s.Tag != "" {
		return s.Tag
	}
	if tag, ok := strings.CutPrefix(s.Name, legacySessionPrefix); ok && tag != "" {
		return tag
	}
	return s.Name
}

func livePickLabel(s Session, policy PickPolicy) string {
	state := string(s.State)
	if state == "" {
		state = string(SessionDetached)
	}
	if policy.RequestedAgent != "" && s.Agent != policy.RequestedAgent {
		state = fmt.Sprintf("%s, unavailable for %s", state, policy.RequestedAgent)
	}
	if s.RepoName != "" || s.Agent != "" {
		agent := s.Agent
		if agent == "" {
			agent = "?"
		}
		repo := s.RepoName
		if repo == "" {
			repo = "?"
		}
		return fmt.Sprintf("%s/%s  %s  (%s)", repo, sessionTag(s), agent, state)
	}
	return s.Name
}

func historicalPickLabel(h HistoricalTag, nowEpoch int64) string {
	age := FormatAge(nowEpoch, h.MTime.Unix())
	if h.RepoName != "" || h.Agent != "" {
		agent := h.Agent
		if agent == "" {
			agent = "?"
		}
		repo := h.RepoName
		if repo == "" {
			repo = "?"
		}
		return fmt.Sprintf("%s/%s  %s  (%s, no live session)", repo, h.Tag, agent, age)
	}
	// Bare tag, not a spelled-out session name (#130): under the 📁 scheme this
	// row would otherwise read `pair-work` right next to a live `📁repo-work`
	// row for the same tag.
	return fmt.Sprintf("%s  (%s, no live session)", h.Tag, age)
}

// resolvePick presents the picker and maps the choice into a concrete launch
// decision. aborted=true means the user dismissed fzf (ESC/empty) — the caller
// exits 0. "+ new" builds a fresh free-slot create (with the name prompt); an
// existing tag re-enters DecideLaunch under a ForcedTag (attach if live, else
// create-by-name), the resume-by-name path.
func resolvePick(rt Runtime, snap SessionSnapshot, base string, nowEpoch int64) (LaunchDecision, bool) {
	d, aborted, _ := resolvePickWithPolicy(rt, snap, base, nowEpoch, PickPolicy{}, io.Discard)
	return d, aborted
}

func resolvePickWithPolicy(rt Runtime, snap SessionSnapshot, base string, nowEpoch int64, policy PickPolicy, stderr io.Writer) (LaunchDecision, bool, int) {
	display, byPlain := buildPickRowsWithPolicy(snap, base, nowEpoch, policy)
	picked := rt.PickFromList("pick a pair session", display, 10)
	if picked == "" {
		return LaunchDecision{}, true, 0
	}
	sel, ok := byPlain[picked]
	if !ok {
		return LaunchDecision{}, true, 0 // fzf returned an unmapped line — abort safely.
	}
	if sel.disabled {
		if sel.disabledReason != "" {
			fmt.Fprint(stderr, sel.disabledReason)
		}
		return LaunchDecision{}, true, 1
	}
	if sel.isNew {
		tag := nextFreeTag(base, snap)
		return createDecision(tag, sessionNameForTag(snap, tag), true), false, 0
	}
	if sel.sessionName != "" {
		return LaunchDecision{Action: ActionAttach, Tag: sel.tag, SessionName: sel.sessionName}, false, 0
	}
	continueDoc := ""
	sourceAgent := ""
	if sel.needsContinuation {
		path, _, ok := rt.ResolveContinuationDoc(sel.tag)
		if ok {
			continueDoc = path
		} else {
			sourceAgent = sel.sourceAgent
		}
	}
	d, _ := DecideLaunch(LaunchArgs{ForcedTag: sel.tag}, snap) // never errors (no pick recursion)
	if d.Action == ActionCreate {
		d.SessionName = ""
	}
	d.LegacyImport = sel.legacy
	d.ContinueDoc = continueDoc
	d.SourceAgent = sourceAgent
	return d, false, 0
}
