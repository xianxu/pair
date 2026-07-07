package launcher

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// The attach + quit-cleanup orchestrators behind RunLaunch's in-process restart
// loop (#99 M3, ported from bin/pair-shell's attach branch + cleanup_quit_marker).
// Both stay thin drivers over Runtime effects + the pure marker/config helpers;
// the loop itself (createflow.go) re-decides create-vs-attach each iteration.

// runAttach ports the shell's attach branch (1701-1723): re-attach to a live
// Pair session (`pair resume <tag>`). Like create it exports the tag +
// refreshes the outer-tty/title/cmux/poller, but it re-uses the existing pane's
// agent (no fresh spawn, no arg composition) and blocks on the attach handoff so
// the loop regains control for cleanup + restart. agent is the inferred title
// agent (the on-disk agent-<tag> record, resolved by the caller).
func runAttach(opts LaunchOptions, env Env, rt Runtime, tag, session, agent string) (int, error) {
	if session == "" {
		session = "pair-" + tag
	}
	// Export what the spawned poller inherits (pair-shell exports these globally
	// before the branch; the attach branch itself only re-exports PAIR_TAG).
	rt.SetEnv("PAIR_HOME", opts.PairHome)
	rt.SetEnv("PAIR_DATA_DIR", env.DataDir)
	rt.SetEnv("PAIR_TAG", tag)

	// zellij creates the draft on new-session but not on attach; ensure it.
	_ = rt.Touch(filepath.Join(env.DataDir, "draft-"+tag+".md"))
	rt.SetTerminalTitle(session)
	rt.RecordOuterTTY(tag)
	rt.CmuxRename(tag, session)
	// agent is already the on-disk record: attach is reached via `pair resume
	// <tag>` (ParseArgs leaves Agent=="") or a live-session pick (runOnce clears
	// Agent) — either way runOnce sets it via InferAgent(tag), so the poller
	// matches the running pane's agent regardless of any bare-`pair` default.
	rt.SpawnTitlePoller(tag, agent, session)

	return rt.AttachSession(session, filepath.Join(opts.PairHome, "zellij"))
}

// runCleanup ports cleanup_quit_marker (shell 1520-1647): after a blocking
// handoff returns, if the Alt+x quit marker is present, tear the session down —
// delete the zellij record, reap nvim, offer to park the scrollback, remove the
// per-tag sidecars, print the resume hint, kill the title poller, and reset the
// cmux workspace. A detach (Alt+d) leaves no marker, so this is a no-op then.
// Runs after BOTH create and attach handoffs (either can leave a quit marker).
func runCleanup(env Env, rt Runtime, step launchStep, parkTimeout int, out io.Writer) {
	if !rt.TakeQuitMarker(step.session) {
		return
	}
	dataDir := env.DataDir
	// Resolve the agent this tag was paired with BEFORE the agent-<tag> record is
	// removed below, so the park path + resume hint name the right binary
	// (InferAgent reads agent-<tag> first, matching the shell, then falls back to
	// the config-filename agent, then the current run's agent).
	quitAgent := rt.InferAgent(step.tag)
	if quitAgent == "" {
		quitAgent = step.agent
	}

	rt.DeleteSession(step.session)
	rt.ReapNvim(step.tag)

	// Park-nudge (ariadne#91): the scrollback is the only on-disk record and
	// Alt+x is about to discard it — offer to preserve it. Gated on an
	// interactive tty with a non-empty raw capture, and skipped when a restart is
	// pending (a restart keeps the work, so re-asking is noise).
	sbBase := filepath.Join(dataDir, "scrollback-"+step.tag+"-"+quitAgent)
	parked := false
	if size, ok := rt.FileSize(sbBase + ".raw"); ok && size > 0 && rt.IsTTY() && !rt.RestartMarkerPresent(step.session) {
		if rt.ConfirmParkNudge(step.session, parkTimeout) {
			if pbase, ok := rt.ParkScrollback(step.tag, quitAgent, true); ok {
				parked = true
				fmt.Fprintf(out, "pair: scrollback preserved at\n        %s.raw\n      open a session and \"park %s\" to distill it into a continuation.\n", pbase, step.session)
			}
		}
	}

	// Remove the per-tag sidecars (shell 1583-1591). pane-<tag>-<quitAgent>.json
	// (written by the agent pane's zellij layout, main.kdl) was historically
	// omitted here — the leak behind #97: a surviving twin misled the frame
	// poller when the tag was later paired with a different agent. Cleaning it on
	// quit stops new twins at the source (the poller also filters defensively).
	for _, rel := range []string{
		"outer-tty-" + step.tag,
		"agent-" + step.tag,
		"agent-output-" + step.tag,
		"pair-wrap-pid-" + step.tag,
		"adapt-" + step.tag + ".jsonl",
		"image-capture-" + step.tag,
		"image-capture-" + step.tag + ".done",
		"pane-" + step.tag + "-" + quitAgent + ".json",
	} {
		rt.Remove(filepath.Join(dataDir, rel))
	}
	rt.Remove(sbBase + ".ansi")
	// Remove the raw capture only when it wasn't parked (preserved above).
	if !parked {
		rt.Remove(sbBase + ".raw")
		rt.Remove(sbBase + ".events.jsonl")
	}

	// Resume hint: a saved config for this (tag, agent) means the resume path
	// will work next time — surface the repo-local tag, not the public zellij
	// session name.
	if raw, err := rt.ReadFile(resolveConfigPath(rt, dataDir, step.tag, quitAgent)); err == nil {
		fmt.Fprintf(out, "pair: saved session config for tag \"%s\" (%s).\n", step.tag, quitAgent)
		fmt.Fprintf(out, "      resume with: pair resume %s\n", step.tag)
		if cfg, err := parseConfig(raw); err == nil && cfg.SessionID != "" {
			fmt.Fprintf(out, "      session id:  %s\n", cfg.SessionID)
		}
	}

	rt.KillTitlePoller(step.tag)

	// Reset the cmux workspace title to the shell cwd — this pair is dead — but
	// only when we own it, then release ownership so a remaining/next pair can
	// claim (shell 1640-1646). On a restart the follow-up create immediately
	// re-renames, so the cwd flash is invisible.
	if rt.PairOwnsCmuxWorkspace(step.tag) {
		reset := filepath.Base(env.Cwd)
		if reset == "" || reset == "." || reset == string(filepath.Separator) {
			reset = "shell"
		}
		rt.CmuxRename(step.tag, reset)
		rt.ClearCmuxOwner()
	}
}

// readSavedConfig loads config-<tag>-<agent>.json for the restart plan; a
// missing/unusable file yields the zero savedConfig (no resume, no saved args).
func readSavedConfig(rt Runtime, configPath string) savedConfig {
	raw, err := rt.ReadFile(configPath)
	if err != nil {
		return savedConfig{}
	}
	cfg, _ := parseConfig(raw)
	return cfg
}

// liveTagsForSweep projects the live pair session names to their bare tags for
// SweepOrphanNvim — every Pair session row (attached, detached, or exited) counts
// as live, so the sweep only reaps embeds with NO session record at all (matches
// the shell's all_pair, the full `list-sessions --short` list).
func liveTagsForSweep(sessions []Session) []string {
	tags := make([]string, 0, len(sessions))
	for _, s := range sessions {
		tags = append(tags, strings.TrimPrefix(s.Name, "pair-"))
	}
	return tags
}

// tagFromEmbedArgv recovers the pair tag from an `nvim --embed …` process argv by
// matching the draft-/scrollback- sidecar path under dataDir (sweep_orphan_nvim's
// ps-scan half, shell 1133-1149). "" when the argv references neither. Pure so
// the sweep's argv parsing is unit-testable; assumes the caller already filtered
// to nvim --embed lines.
func tagFromEmbedArgv(argv, dataDir string) string {
	if marker := dataDir + "/draft-"; strings.Contains(argv, marker) {
		rest := firstField(argv[strings.LastIndex(argv, marker)+len(marker):])
		return strings.TrimSuffix(rest, ".md")
	}
	if marker := dataDir + "/scrollback-"; strings.Contains(argv, marker) {
		rest := firstField(argv[strings.LastIndex(argv, marker)+len(marker):])
		rest = strings.TrimSuffix(rest, ".ansi")
		if i := strings.LastIndex(rest, "-"); i >= 0 {
			rest = rest[:i] // strip trailing -<agent> to recover <tag>
		}
		return rest
	}
	return ""
}

// firstField returns s up to its first space (the shell's `${x%% *}`).
func firstField(s string) string {
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}
