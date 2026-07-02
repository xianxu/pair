package launcher

// Per-agent launch-argument composition — the pure decisions behind the shell
// launcher's resume-token / --session-id / --no-alt-screen handling (#99 M1,
// ported from bin/pair-shell). The IO around them (uuidgen + collision stat,
// jq config read/write) lands on the Runtime seam in M2; here we own only the
// deterministic arg-vector transforms + the mint/skip decisions.

// hasFlag reports whether flag appears as its own token in args.
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// stripValuelessFlag removes every occurrence of a standalone flag (e.g.
// --no-alt-screen) from args, preserving order.
func stripValuelessFlag(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == flag {
			continue
		}
		out = append(out, a)
	}
	return out
}

// stripFlagWithValue removes a flag and the token that follows it (e.g.
// --session-id <uuid>, --resume <id>). A trailing flag with no value is dropped.
// This is the one helper that replaces the four hand-rolled strip loops the shell
// carried (explicit-resume, claude --session-id, codex --no-alt-screen, and the
// config-picker strip) — ARCH-DRY.
func stripFlagWithValue(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			i++ // also skip the value
			continue
		}
		out = append(out, args[i])
	}
	return out
}

// resumeToken is the per-agent surface for resuming a session id: claude uses
// `--resume <id>`, codex uses the `resume <id>` subcommand, agy uses
// `--conversation <id>`. Empty sid (or an unknown agent) yields no token.
func resumeToken(agent, sid string) []string {
	if sid == "" {
		return nil
	}
	switch agent {
	case "claude":
		return []string{"--resume", sid}
	case "codex":
		return []string{"resume", sid}
	case "agy":
		return []string{"--conversation", sid}
	}
	return nil
}

// composeResumeArgs appends the resume token to the saved args in the order each
// agent needs. Codex's `resume` subcommand must sit at args[0] (inner pair +
// pair-session-watch detection assume that position), so its token goes first;
// claude's `--resume` flag works anywhere, so saved args keep their leading spot.
func composeResumeArgs(agent string, savedArgs []string, sid string) []string {
	token := resumeToken(agent, sid)
	if len(token) == 0 {
		return append([]string(nil), savedArgs...)
	}
	if agent == "codex" {
		return append(append([]string(nil), token...), savedArgs...)
	}
	return append(append([]string(nil), savedArgs...), token...)
}

// codexAltScreenArgs forces codex into inline mode (--no-alt-screen) so its
// conversation flows through zellij's scrollback (alt-screen has none). Strips an
// existing --no-alt-screen first so repeated Alt+n restarts don't accumulate
// duplicates; optOut (PAIR_CODEX_ALT_SCREEN=1) leaves it off.
func codexAltScreenArgs(args []string, optOut bool) []string {
	stripped := stripValuelessFlag(args, "--no-alt-screen")
	if optOut {
		return stripped
	}
	return append(stripped, "--no-alt-screen")
}

// shouldMintClaudeSessionID decides whether the create path should pin a
// deterministic claude session id (via --session-id) instead of leaving it to
// the async watcher. Skip when a resume already pinned one, when the user passed
// their own --session-id, or when --fork-session lets claude allocate internally.
// Only claude supports the flag; codex/agy always fall back to the watcher.
func shouldMintClaudeSessionID(agent, explicitResume string, agentExtra []string) bool {
	return agent == "claude" && explicitResume == "" &&
		!hasFlag(agentExtra, "--session-id") && !hasFlag(agentExtra, "--fork-session")
}

// persistedConfigArgs strips the resume-binding flags from the args before they
// are saved to config-<tag>-<agent>.json: session_id is the canonical storage for
// the binding, so leaving --session-id / --resume in the saved args would compound
// on every relaunch through the picker.
func persistedConfigArgs(args []string) []string {
	out := stripFlagWithValue(args, "--session-id")
	out = stripFlagWithValue(out, "--resume")
	return out
}
