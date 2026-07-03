package launcher

import (
	"io"
	"strings"
)

// runRestart is the in-process port of bin/pair-restart.sh (#94 M1): resolve the
// live session/tag/agent, write the restart marker (carrying new_session +
// optional rename_to), touch the quit marker, then exec kill-session. It reuses
// the exact Runtime seam runCompaction drives (compaction.go:54-68) — the effects
// already live on OSRuntime, so nothing new is added to the seam. ExecKillSession
// is terminal on the real runtime (syscall.Exec replaces the process), so the
// return is reached only when the kill binary is missing or under the fake.
func runRestart(rt Runtime, args LaunchArgs, session string, stderr io.Writer) int {
	if session == "" {
		_, _ = io.WriteString(stderr, "pair restart: ZELLIJ_SESSION_NAME unset; cannot restart cleanly.\n")
		return 1
	}
	tag := strings.TrimPrefix(session, "pair-")
	rt.WriteRestartMarker(session, RestartMarker{
		Tag: tag,
		// InferAgent reads agent-<tag> (always present when the keybind fires —
		// cleanup removes it only AFTER the restart). Its config-<tag>-*.json
		// fallback is broader than the shell's plain `cat agent-<tag>`, but it
		// only ever fills an otherwise-empty agent=, never contradicts it —
		// a deliberate, safe divergence from the byte-faithful shell.
		Agent:      rt.InferAgent(tag),
		NewSession: args.NewSession,
		RenameTo:   args.RenameTo,
	})
	rt.TouchQuitMarker(session)
	rt.ExecKillSession(session)
	return 0
}

// runQuit is the in-process port of bin/pair-quit.sh (#94 M1): touch the quit
// marker so the outer loop's cleanup fires, then exec kill-session. No restart
// marker — Alt+x is a full quit, not a reload.
func runQuit(rt Runtime, session string, stderr io.Writer) int {
	if session == "" {
		_, _ = io.WriteString(stderr, "pair quit: ZELLIJ_SESSION_NAME unset; cannot quit cleanly.\n")
		return 1
	}
	rt.TouchQuitMarker(session)
	rt.ExecKillSession(session)
	return 0
}
