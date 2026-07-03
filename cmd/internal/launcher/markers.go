package launcher

import "strings"

// Restart/quit marker logic (#99 M3, ported from bin/pair-shell's
// handle_restart_marker + pair-restart.sh handshake). The markers live under
// ~/.cache/pair/{quit,restart}-<session>; parsing + the re-launch decision are
// pure here, the read/clear IO sits on the Runtime seam.

// RestartMarker is the parsed ~/.cache/pair/restart-<session> handshake dropped
// by pair-restart.sh (Alt+n / Shift+Alt+N) or the #55 compaction branch.
type RestartMarker struct {
	Tag        string
	Agent      string
	NewSession bool   // Shift+Alt+N / compaction: fresh agent conversation
	RenameTo   string // #22 inside-flow tag rename (native re-entry as of M5b)
	Continue   string // #55 compaction slug (native continue re-entry as of M5b)
}

// parseRestartMarker reads the `key=value` lines pair-restart.sh writes. Unknown
// keys are ignored; a missing marker is the caller's concern (empty content →
// zero value).
func parseRestartMarker(content string) RestartMarker {
	var m RestartMarker
	for _, line := range strings.Split(content, "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "tag":
			m.Tag = val
		case "agent":
			m.Agent = val
		case "new_session":
			m.NewSession = val == "1"
		case "rename_to":
			m.RenameTo = val
		case "continue":
			m.Continue = val
		}
	}
	return m
}

// restartPlan is the decision the in-process restart loop acts on: the next
// launch (Args), whether to drop the saved config first, and — for a #55
// compaction re-entry — the continuation slug to re-seed the draft from.
type restartPlan struct {
	Args         LaunchArgs
	DropConfig   bool   // Shift+Alt+N / compaction: drop the saved config first
	ContinueSlug string // #55 compaction re-entry: re-seed the draft from this slug
}

// planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
// into the next launch (#99 M5b makes rename/continue native). The caller has
// already applied the marker's tag/agent preference AND any rename_to move before
// calling this, so tag/agent here are final. Mirrors handle_restart_marker (shell
// 762-810): Shift+Alt+N / compaction drop the config and re-launch fresh; a plain
// Alt+n composes the canonical resume binding onto the saved args (codex's
// `resume` subcommand leads via composeResumeArgs).
func planRestart(m RestartMarker, tag, agent string, saved savedConfig) restartPlan {
	base := LaunchArgs{Agent: agent, ForcedTag: tag}
	if m.NewSession {
		// Fresh conversation: keep the saved args, drop the config so the create
		// path mints a new session id rather than resuming the prior one. A
		// Continue slug only ever rides new_session (shell 1055-1056), so re-seed
		// here (the loop resolves the slug → draft).
		base.AgentArgs = append([]string(nil), saved.Args...)
		return restartPlan{Args: base, DropConfig: true, ContinueSlug: m.Continue}
	}
	// Default Alt+n: resume the prior conversation via the saved id.
	base.AgentArgs = composeResumeArgs(agent, saved.Args, saved.SessionID)
	return restartPlan{Args: base}
}
