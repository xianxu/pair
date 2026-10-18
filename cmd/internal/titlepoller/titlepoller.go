// Package titlepoller is the Go owner of the per-tag title poller (#93 M1,
// ported from bin/pair-title.sh). It owns two surfaces:
//
//  1. The zellij FRAME title of each agent pane — "<agent> (<count>)", where
//     <count> is the agent's context-window size (#71). No cwd: zellij composes
//     the tab as "<session name> | <pane title>" and the session half already
//     carries the folder (#133). Always-on.
//  2. The cmux WORKSPACE title — an activity heat-ramp emoji prefix. cmux-only.
//
// Each wrapper invocation owns a title through its pipe lease and isolated
// process group. EOF ends that group immediately; numeric PID publication is
// metadata only. The poll loop also exits when its zellij session disappears
// (miss-threshold debounced). Pure display decisions live here; IO/process
// interaction lives behind Runtime and the CLI lease boundary.
package titlepoller

import (
	"fmt"
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

// PrefixForAge buckets an activity age into a heat-ramp prefix (emoji + trailing
// space), or "" for cold (≥ 21 days). Mirrors pair-title.sh's prefix_for_age.
func PrefixForAge(age time.Duration) string {
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

// FrameTitle composes an agent pane's zellij frame title: "<agent> (<count>)", or
// just "<agent>" when no count resolved.
//
// It carries no cwd (#133). zellij renders the tab title as
// "<session name> | <focused pane title>", and since #130 the session half is
// "📁{repo}[-{tag}]" — so a cwd here would name the folder twice. The pane half
// carries only the pane's own identity.
func FrameTitle(agent, count string) string {
	if count != "" {
		return fmt.Sprintf("%s (%s)", agent, count)
	}
	return agent
}

// CmuxWorkspaceTitle builds the cmux workspace title from a heat prefix and the
// zellij session name. The name is used verbatim since #130 retired the
// word→emoji convention — the session name already carries its own 📁 glyph.
func CmuxWorkspaceTitle(prefix, session string) string {
	return prefix + session
}

// ShouldClaimWorkspace decides cmux workspace-title ownership (mirrors bin/pair's
// cmux_rename_workspace): claim if unowned or already ours; if another tag owns
// it, defer only while that owner's session is still alive — otherwise the owner
// crashed without cleanup and we reclaim.
func ShouldClaimWorkspace(owner, tag string, ownerSessionAlive bool) bool {
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

// Compatibility names keep the standalone poller and its existing tests on the
// same rendering decisions as supervisor callers.
func prefixForAge(age time.Duration) string            { return PrefixForAge(age) }
func frameTitle(agent, count string) string            { return FrameTitle(agent, count) }
func cmuxWorkspaceTitle(prefix, session string) string { return CmuxWorkspaceTitle(prefix, session) }
func shouldClaimWorkspace(owner, tag string, alive bool) bool {
	return ShouldClaimWorkspace(owner, tag, alive)
}
