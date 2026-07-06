package titlepoller

import (
	"path/filepath"
	"strings"
	"time"
)

// Options are the poller inputs after CLI/env resolution.
type Options struct {
	Tag             string
	Agent           string
	DataDir         string
	Home            string
	CmuxWorkspaceID string // CMUX_WORKSPACE_ID; empty ⇒ skip the cmux surface

	// Tunables (defaults applied in Run). PollInterval is the loop cadence;
	// StartupGrace covers the create-path race (poller spawned right before
	// `zellij --new-session-with-layout`); MissThreshold debounces transient
	// `zellij list-sessions` failures before deciding the session is gone.
	PollInterval  time.Duration
	StartupGrace  time.Duration
	MissThreshold int
}

// PaneInfo is one decoded pane-<tag>-<agent>.json (the fields the frame meter needs).
type PaneInfo struct {
	Agent      string
	PaneID     string
	Cwd        string
	CwdDisplay string
}

// Runtime is the IO/process boundary for the poller. The pure decisions live in
// titlepoller.go; everything here that touches zellij/cmux/fs/clock/ps is a seam
// method so the loop is unit-testable with a fake.
type Runtime interface {
	Now() time.Time
	Sleep(time.Duration)
	Getpid() string
	ProcessAlive(pid string) bool
	ProcessCommand(pid string) string

	SessionAlive(session string) bool
	RenamePane(session, paneID, title string) error
	CmuxAvailable() bool
	CmuxRenameWorkspace(title string) error

	ReadFile(path string) (string, error)
	WriteFile(path, data string) error
	Remove(path string)
	ModTime(path string) (time.Time, bool)

	PaneFiles(dataDir, tag string) []PaneInfo
	ContextCount(tag, agent string) string
	TranscriptPath(tag, agent string) string
}

// Run drives the poller until the pair-<tag> session disappears (or a startup
// race is lost). Returns a process exit code (always 0 — like the shell poller,
// it exits cleanly on session-gone and never surfaces an error).
func Run(opts Options, rt Runtime) int {
	if opts.Tag == "" || opts.Agent == "" {
		return 0
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 60 * time.Second
	}
	if opts.StartupGrace <= 0 {
		opts.StartupGrace = 30 * time.Second
	}
	if opts.MissThreshold <= 0 {
		opts.MissThreshold = 5
	}

	session := "pair-" + opts.Tag
	pidfile := filepath.Join(opts.DataDir, "title-pid-"+opts.Tag)

	// Single-instance: bail only if a prior poller for this tag is genuinely
	// still running. Identity-checked (not a bare liveness check) so a recycled
	// PID left by a dead poller can't wedge the respawn.
	if raw, err := rt.ReadFile(pidfile); err == nil {
		old := strings.TrimSpace(raw)
		if rt.ProcessAlive(old) && pollerArgvMatches(rt.ProcessCommand(old), opts.Tag) {
			return 0
		}
	}
	_ = rt.WriteFile(pidfile, rt.Getpid()+"\n")
	defer rt.Remove(pidfile)

	// Wait for the zellij session to appear (create-path race). After this,
	// "session missing" reliably means the user ended the session.
	graceDeadline := rt.Now().Add(opts.StartupGrace)
	seen := false
	for rt.Now().Before(graceDeadline) {
		if rt.SessionAlive(session) {
			seen = true
			break
		}
		rt.Sleep(1 * time.Second)
	}
	if !seen {
		return 0
	}

	cache := frameCache{}
	lastPrefix := "__init__" // sentinel so the first real bucket always fires
	misses := 0
	for {
		// Self-terminate when the session is gone, debounced across misses so a
		// single flaky IPC read (common right after sleep/wake) doesn't kill us.
		if rt.SessionAlive(session) {
			misses = 0
		} else {
			misses++
			if misses >= opts.MissThreshold {
				return 0
			}
			rt.Sleep(opts.PollInterval)
			continue
		}

		latest := activityMTime(opts, rt)
		if latest.IsZero() {
			// No activity source resolved yet (config not written, agent
			// crashed pre-startup). Try again next tick.
			rt.Sleep(opts.PollInterval)
			continue
		}
		age := rt.Now().Sub(latest)

		// Frame meter (#71): refresh each agent pane's zellij FRAME title while
		// active. Gated on recent activity so idle sessions stop re-rendering;
		// the per-pane unchanged-skip cache prevents churn during an
		// active-but-stable stretch. MUST be outside the cmux bucket guard — an
		// active session keeps one heat bucket for a day, so gating the meter on
		// the bucket would refresh once then freeze.
		if age < 2*opts.PollInterval {
			updateFrameTitles(opts, rt, cache, session)
		}

		// cmux WORKSPACE title (cmux-only): the heat-ramp emoji prefix.
		if opts.CmuxWorkspaceID != "" && rt.CmuxAvailable() {
			lastPrefix = updateWorkspaceTitle(opts, rt, age, session, lastPrefix)
		}

		rt.Sleep(opts.PollInterval)
	}
}

// activityMTime returns the most recent mtime across the poller's activity
// sources — the nvim draft and the agent's native transcript (resolved via the
// same path `pair context` uses). Zero time ⇒ nothing resolved yet.
func activityMTime(opts Options, rt Runtime) time.Time {
	var latest time.Time
	candidates := []string{filepath.Join(opts.DataDir, "draft-"+opts.Tag+".md")}
	if tp := rt.TranscriptPath(opts.Tag, opts.Agent); tp != "" {
		candidates = append(candidates, tp)
	}
	for _, f := range candidates {
		if m, ok := rt.ModTime(f); ok && m.After(latest) {
			latest = m
		}
	}
	return latest
}

// updateFrameTitles renames the active agent's zellij frame to
// "<agent> (<count>) [<cwd>]", skipping panes whose title is unchanged.
//
// PaneFiles globs pane-<tag>-*.json, which can match a STALE twin left by a
// prior session that paired this tag with a different agent (nothing cleaned it
// up before #97's runCleanup fix, and a crash still bypasses that cleanup). The
// two-pane invariant means exactly one agent pane is live per tag, and opts.Agent
// authoritatively names it (the poller is respawned each entry with the agent
// resolved fresh from agent-<tag>), so we render only that pane. Without this a
// stale twin sharing the live pane_id makes the pane_id-keyed frameCache render a
// different title for the same pane every tick → alphabetical last-wins + churn
// (the #97 bug: "claude" pane labelled "codex").
func updateFrameTitles(opts Options, rt Runtime, cache frameCache, session string) {
	for _, pane := range rt.PaneFiles(opts.DataDir, opts.Tag) {
		if pane.PaneID == "" || pane.Agent != opts.Agent {
			continue
		}
		cwdDisp := pane.CwdDisplay
		if cwdDisp == "" {
			cwdDisp = abbrevCwd(pane.Cwd, opts.Home)
		}
		title := frameTitle(pane.Agent, rt.ContextCount(opts.Tag, pane.Agent), cwdDisp)
		if !cache.changed(pane.PaneID, title) {
			continue
		}
		_ = rt.RenamePane(session, pane.PaneID, title)
	}
}

// updateWorkspaceTitle applies the cmux heat-ramp workspace title when the
// bucket changes, honoring workspace-title ownership. Returns the prefix to
// carry as lastPrefix (unchanged when we defer to a live owner).
func updateWorkspaceTitle(opts Options, rt Runtime, age time.Duration, session, lastPrefix string) string {
	prefix := prefixForAge(age)
	if prefix == lastPrefix {
		return lastPrefix
	}
	ownerPath := filepath.Join(opts.DataDir, "cmux-owner-"+opts.CmuxWorkspaceID)
	owner := ""
	if raw, err := rt.ReadFile(ownerPath); err == nil {
		owner = strings.TrimSpace(raw)
	}
	if owner != "" && owner != opts.Tag {
		if !shouldClaimWorkspace(owner, opts.Tag, rt.SessionAlive("pair-"+owner)) {
			return lastPrefix // another live pair owns it; leave the title alone
		}
	}
	_ = rt.WriteFile(ownerPath, opts.Tag+"\n")
	_ = rt.CmuxRenameWorkspace(cmuxWorkspaceTitle(prefix, session))
	return prefix
}
