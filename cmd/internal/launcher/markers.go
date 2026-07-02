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
	RenameTo   string // #22 inside-flow tag rename (M5-coupled → shell fallback)
	Continue   string // #55 compaction slug (continue re-entry, M5-coupled → shell fallback)
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

// restartPlan is the decision the in-process restart loop acts on: either a
// native re-launch (Args) or a hand-off to the shell for the M5-coupled
// rename/continue re-entries.
type restartPlan struct {
	Args          LaunchArgs
	DropConfig    bool // Shift+Alt+N / compaction: drop the saved config first
	ShellFallback bool // rename_to / continue: re-entry not yet native (M5)
}

// planRestart maps a restart marker + the current run's (tag, agent) + the saved
// config into the next launch. Mirrors handle_restart_marker (shell 735-810):
// the marker's tag/agent default to the current ones; Shift+Alt+N drops the
// config and re-launches fresh; a plain Alt+n composes the canonical resume
// binding onto the saved args (codex's `resume` subcommand leads via
// composeResumeArgs). rename_to / continue re-entries fall back to the shell.
func planRestart(m RestartMarker, curTag, curAgent string, saved savedConfig) restartPlan {
	rTag := m.Tag
	if rTag == "" {
		rTag = curTag
	}
	rAgent := m.Agent
	if rAgent == "" {
		rAgent = curAgent
	}
	if m.RenameTo != "" || m.Continue != "" {
		return restartPlan{ShellFallback: true}
	}
	base := LaunchArgs{Agent: rAgent, ForcedTag: rTag}
	if m.NewSession {
		// Fresh conversation: keep the saved args, drop the config so the create
		// path mints a new session id rather than resuming the prior one.
		base.AgentArgs = append([]string(nil), saved.Args...)
		return restartPlan{Args: base, DropConfig: true}
	}
	// Default Alt+n: resume the prior conversation via the saved id.
	base.AgentArgs = composeResumeArgs(rAgent, saved.Args, saved.SessionID)
	return restartPlan{Args: base}
}
