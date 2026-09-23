package launcher

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/resumeform"
)

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

// stripFlagAllForms removes a valued flag in both its space form (`flag value`)
// and its inline form (`flag=value`) — e.g. --session-id <uuid>, --resume <id>,
// and agy's --conversation <id> / --conversation=<id>. A trailing space-form flag
// with no value is dropped. Together with stripCodexResumeSubcommand +
// stripValuelessFlag, this is the single consolidation of the shell's several
// hand-rolled resume/binding strip loops (ARCH-DRY).
func stripFlagAllForms(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			i++ // also skip the space-form value
			continue
		}
		if strings.HasPrefix(args[i], flag+"=") {
			continue // inline form
		}
		out = append(out, args[i])
	}
	return out
}

// stripCodexResumeSubcommand drops `resume <id>` from Codex argv. Codex accepts
// global options before the command (`codex [OPTIONS] resume <id>`), so the
// command is position-sensitive only after those options have been consumed.
// Muse uses the same `resume <id>` subcommand shape (plus the same leading-
// resume fallback in extractExplicitResume).
func stripCodexResumeSubcommand(args []string) []string {
	if i := codexResumeCommandIndex(args); i >= 0 {
		out := append([]string(nil), args[:i]...)
		return append(out, args[i+2:]...)
	}
	// Fallback for `muse resume <id>` when codex-style globals don't cover
	// the prefix (muse has different globals like --provider/--model). The
	// canonical `composeResumeArgs` placement is always leading, so handle it.
	if len(args) >= 2 && args[0] == "resume" && args[1] != "" {
		return append([]string(nil), args[2:]...)
	}
	return args
}

func codexResumeCommandIndex(args []string) int {
	for i := 0; i < len(args); {
		if args[i] == "resume" && i+1 < len(args) && args[i+1] != "" {
			return i
		}
		if n := codexGlobalOptionWidth(args, i); n > 0 {
			i += n
			continue
		}
		return -1
	}
	return -1
}

func codexGlobalOptionWidth(args []string, i int) int {
	arg := args[i]
	if codexBoolGlobalOption(arg) {
		return 1
	}
	if codexValueGlobalOption(arg) {
		if strings.Contains(arg, "=") {
			return 1
		}
		if i+1 < len(args) {
			return 2
		}
		return 0
	}
	return 0
}

func codexBoolGlobalOption(arg string) bool {
	switch arg {
	case "--oss", "--strict-config", "--dangerously-bypass-approvals-and-sandbox",
		"--dangerously-bypass-hook-trust", "--search", "--no-alt-screen":
		return true
	default:
		return false
	}
}

func codexValueGlobalOption(arg string) bool {
	if strings.HasPrefix(arg, "--config=") ||
		strings.HasPrefix(arg, "--enable=") ||
		strings.HasPrefix(arg, "--disable=") ||
		strings.HasPrefix(arg, "--remote=") ||
		strings.HasPrefix(arg, "--remote-auth-token-env=") ||
		strings.HasPrefix(arg, "--image=") ||
		strings.HasPrefix(arg, "--model=") ||
		strings.HasPrefix(arg, "--local-provider=") ||
		strings.HasPrefix(arg, "--profile=") ||
		strings.HasPrefix(arg, "--sandbox=") ||
		strings.HasPrefix(arg, "--cd=") ||
		strings.HasPrefix(arg, "--add-dir=") ||
		strings.HasPrefix(arg, "--ask-for-approval=") {
		return true
	}
	switch arg {
	case "-c", "--config", "--enable", "--disable", "--remote",
		"--remote-auth-token-env", "-i", "--image", "-m", "--model",
		"--local-provider", "-p", "--profile", "-s", "--sandbox",
		"-C", "--cd", "--add-dir", "-a", "--ask-for-approval":
		return true
	default:
		return false
	}
}

// resumeToken is the per-agent surface for resuming a session id: claude uses
// `--resume <id>`, codex uses the `resume <id>` subcommand, agy uses
// `--conversation <id>`, muse uses `resume <id>` (like codex), qoder uses
// `--resume <id>` (like claude; a global flag, any position). Empty sid (or an
// unknown agent) yields no token.
func resumeToken(agent, sid string) []string {
	if sid == "" {
		return nil
	}
	switch agent {
	case "claude", "qoder":
		return []string{"--resume", sid}
	case "codex":
		return []string{"resume", sid}
	case "agy":
		return []string{"--conversation", sid}
	case "muse":
		return []string{"resume", sid}
	}
	return nil
}

// composeResumeArgs appends the resume token to the saved args in the order each
// agent needs. Codex's and muse's `resume` subcommand must sit at args[0] (inner
// pair + pair-session-watch detection assume that position), so its token goes
// first; claude's `--resume` flag works anywhere, so saved args keep their leading
// spot.
func composeResumeArgs(agent string, savedArgs []string, sid string) []string {
	token := resumeToken(agent, sid)
	if len(token) == 0 {
		return append([]string(nil), savedArgs...)
	}
	if agent == "codex" || agent == "muse" {
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

// MintsSessionID reports whether pair pins a caller-minted --session-id at
// launch: claude, whose jsonl is keyed by the id pair chooses (#20), and
// qoder, which honors the same flag (verified live at 1.1.60 — the transcript
// lands under ~/.qoder/projects/<slug>/<minted id>.jsonl). Every other agent's
// durable binding is established independently by the causal-round watcher.
func MintsSessionID(agent string) bool {
	return agent == "claude" || agent == "qoder"
}

// shouldMintSessionID decides whether the create path should pin a
// deterministic session id (via --session-id) instead of leaving it to the
// agent. Skip when a resume already pinned one, when the user passed
// their own --session-id, or when --fork-session lets the agent allocate
// internally.
func shouldMintSessionID(agent, explicitResume string, agentExtra []string) bool {
	return MintsSessionID(agent) && explicitResume == "" &&
		!hasFlag(agentExtra, "--session-id") && !hasFlag(agentExtra, "--fork-session")
}

// persistedConfigArgs strips every resume binding from saved launch
// parameters. Established inventory is the binding authority; leaving
// generated resume flags in compatibility config would compound them on
// every relaunch. The spellings live in one table (resumeform.Forms) shared
// with the extractors, the validator and sessionwatch; the deterministic
// --session-id pin is stripped alongside them. The strip is strictly
// per-agent (BR-22): args saved for agent A are only rewritten by A's own
// spellings — claude/qoder reuse the shared --resume/-r table, codex/muse
// the leading `resume <id>` subcommand — so a peer agent's legit token
// (e.g. a claude prompt word `resume`) is never eaten.
func persistedConfigArgs(agent string, args []string) []string {
	out := args
	if agent == "codex" || agent == "muse" {
		out = stripCodexResumeSubcommand(out)
	}
	out = resumeform.Strip(agent, out)
	out = stripFlagAllForms(out, "--session-id")
	return out
}

// FreshAgentArgs preserves user-authored launch options while removing every
// generated conversation-restoration binding.
func FreshAgentArgs(agent string, args []string) []string {
	return persistedConfigArgs(agent, args)
}
