// Package titlepoller is the Go owner of the per-tag title poller (#93 M1,
// ported from bin/pair-title.sh). It owns two surfaces:
//
//  1. The zellij FRAME title of each agent pane — "<agent> (<count>) [<cwd>]",
//     where <count> is the agent's context-window size (#71). Always-on.
//  2. The cmux WORKSPACE title — an activity heat-ramp emoji prefix. cmux-only.
//
// Single-instance per tag via a pidfile whose liveness is identity-checked (not
// a bare kill -0), so a recycled PID left by a dead poller can't wedge the
// respawn. Self-terminates when the pair-<tag> zellij session disappears
// (miss-threshold debounced). Following the #78 sessionwatch template, the pure
// decisions live here and the IO/process interaction is behind the Runtime seam.
package titlepoller

import (
	"fmt"
	"strings"
	"time"
)

// Heat-ramp thresholds + CJK-wide emoji prefixes for the cmux workspace title
// (hottest first). All four emoji are CJK-wide so the cmux sidebar alignment
// stays uniform across buckets.
const (
	oneDay        = 24 * time.Hour
	threeDays     = 3 * oneDay
	tenDays       = 10 * oneDay
	twentyOneDays = 21 * oneDay
)

const (
	prefixHot      = "🔴" // < 1 day
	prefixWarm     = "🟠" // < 3 days
	prefixLukewarm = "🟡" // < 10 days
	prefixCool     = "🔵" // < 21 days
)

// prefixForAge buckets an activity age into a heat-ramp prefix (emoji + trailing
// space), or "" for cold (≥ 21 days). Mirrors pair-title.sh's prefix_for_age.
func prefixForAge(age time.Duration) string {
	switch {
	case age < oneDay:
		return prefixHot + " "
	case age < threeDays:
		return prefixWarm + " "
	case age < tenDays:
		return prefixLukewarm + " "
	case age < twentyOneDays:
		return prefixCool + " "
	default:
		return ""
	}
}

// abbrevCwd abbreviates a raw cwd to ~ on a path boundary (mirrors bin/pair's
// abbrev_cwd): exactly $HOME → "~"; under $HOME/ → "~/rest"; else unchanged.
func abbrevCwd(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}

// frameTitle composes an agent pane's zellij frame title: "<agent> (<count>)
// [<cwd>]", or "<agent> [<cwd>]" when no count resolved.
func frameTitle(agent, count, cwdDisp string) string {
	if count != "" {
		return fmt.Sprintf("%s (%s) [%s]", agent, count, cwdDisp)
	}
	return fmt.Sprintf("%s [%s]", agent, cwdDisp)
}

// cmuxWorkspaceTitle builds the cmux workspace title from a heat prefix and the
// zellij session name, applying the personal display convention (brain→🧠,
// book→📗, pair→♋ anywhere in the title). Mirrors bin/pair's cmux_rename_workspace.
func cmuxWorkspaceTitle(prefix, session string) string {
	title := prefix + session
	title = strings.ReplaceAll(title, "brain", "🧠")
	title = strings.ReplaceAll(title, "book", "📗")
	title = strings.ReplaceAll(title, "pair", "♋")
	return title
}

// pollerArgvMatches is the single-instance identity guard: true iff argv is a
// live pair-title poller for THIS tag. The Go poller runs as
// "<…>/pair-title <tag> <agent> …" (the shim re-execs the Go binary), so we
// match the substring "pair-title <tag> " — the trailing space keeps tag 21
// from matching 211, exactly as pair-title.sh's `*"pair-title.sh $TAG "*` did.
func pollerArgvMatches(argv, tag string) bool {
	if argv == "" || tag == "" {
		return false
	}
	return strings.Contains(argv, "pair-title "+tag+" ")
}

// shouldClaimWorkspace decides cmux workspace-title ownership (mirrors bin/pair's
// cmux_rename_workspace): claim if unowned or already ours; if another tag owns
// it, defer only while that owner's session is still alive — otherwise the owner
// crashed without cleanup and we reclaim.
func shouldClaimWorkspace(owner, tag string, ownerSessionAlive bool) bool {
	if owner == "" || owner == tag {
		return true
	}
	return !ownerSessionAlive
}

// frameCache tracks the last title written per pane id so redundant zellij
// renames are skipped (macOS bash 3.2 had no assoc arrays; Go gives us a map).
type frameCache map[string]string

// changed reports whether paneID's title differs from the last one written, and
// records the new title when it does (so the caller renames then caches).
func (c frameCache) changed(paneID, title string) bool {
	if c[paneID] == title {
		return false
	}
	c[paneID] = title
	return true
}
