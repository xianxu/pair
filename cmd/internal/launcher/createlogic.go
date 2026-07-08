package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Pure create-flow logic behind RunLaunch's create path (#99 M2, ported from
// bin/pair-shell's create branch). Everything here is a deterministic transform
// of already-known values — the IO around it (config read/write, fzf present,
// uuid mint, agent-session stat) sits on the Runtime seam. RunLaunch stays a thin
// orchestrator by delegating every decision to these helpers + M1's agentargs.

// savedConfig is the on-disk config-<tag>-<agent>.json shape the launcher and the
// session-watcher both write ({agent, args, session_id}).
type savedConfig struct {
	Agent     string   `json:"agent"`
	Args      []string `json:"args"`
	SessionID string   `json:"session_id"`
}

// parseConfig decodes a config-<tag>-<agent>.json blob. A malformed/empty blob
// yields the zero value + error; callers treat that as "no usable saved config".
func parseConfig(raw string) (savedConfig, error) {
	var c savedConfig
	err := json.Unmarshal([]byte(raw), &c)
	return c, err
}

// buildConfigJSON renders the {agent, args, session_id} config the shell wrote
// via `jq -n --args`. SetEscapeHTML(false) matches jq/vim.json (no < for <>&)
// so a config a Go writer produces is byte-compatible with the shell readers;
// nil args serialize as [] (jq's $ARGS.positional is never null). The struct key
// order (agent, args, session_id) mirrors the jq object literal.
func buildConfigJSON(agent string, args []string, sid string) (string, error) {
	if args == nil {
		args = []string{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(savedConfig{Agent: agent, Args: args, SessionID: sid}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// extractExplicitResume returns the session id an explicit resume token on argv
// pins, or "" if none. Per-agent surface (shell create branch 2053-2075): claude
// `--resume <id>`, agy `--conversation <id>` / `--conversation=<id>`, codex the
// leading `resume <id>` subcommand. Drives both the tag-restart picker gate
// (a passed-in resume leaves the picker nothing to offer) and the pre-write of
// config-<tag>-<agent>.json so the id is captured from the start.
func extractExplicitResume(agent string, args []string) string {
	switch agent {
	case "codex":
		if i := codexResumeCommandIndex(args); i >= 0 {
			return args[i+1]
		}
	case "claude", "agy":
		prev := ""
		for _, tok := range args {
			if prev == "--resume" || prev == "--conversation" {
				return tok
			}
			// Only a non-empty inline value pins the id; a bare `--conversation=`
			// keeps scanning (the shell's `^--conversation=(.+)` needs ≥1 char).
			if v, ok := strings.CutPrefix(tok, "--conversation="); ok && v != "" {
				return v
			}
			prev = tok
		}
	}
	return ""
}

// configChoice is one row of the tag-restart config picker (#000016): a
// user-visible multi-line Label and the Action key it maps back to. The label
// carries the running number + resolved values so fzf --read0 shows the full
// launch without truncation; Action drives composeTagRestartArgs.
type configChoice struct {
	Label  string
	Action string
}

func displayOrNone(args []string) string {
	if s := strings.Join(args, " "); s != "" {
		return s
	}
	return "<none>"
}

// buildConfigChoices mirrors the shell's option/action matrix (create branch
// 1876-1927). savedArgsClean is the persisted args already stripped of resume
// bindings; agentExtra is what was typed this launch. Rows collapse when saved
// and current args are byte-identical (no point offering two indistinguishable
// launches), and the resume rows only appear when the agent's native session
// artifact for the saved id still exists (hasResumable).
func buildConfigChoices(hasResumable bool, savedArgsClean, agentExtra []string, savedSession string) []configChoice {
	savedDisplay := displayOrNone(savedArgsClean)
	currentDisplay := displayOrNone(agentExtra)
	argsMatch := savedDisplay == currentDisplay

	var choices []configChoice
	n := 1
	if hasResumable {
		choices = append(choices, configChoice{
			Label:  fmt.Sprintf("%d) use saved params + session\n     args=[%s]\n     resume=%s", n, savedDisplay, savedSession),
			Action: "saved+resume",
		})
		n++
	}
	choices = append(choices, configChoice{
		Label:  fmt.Sprintf("%d) use saved params\n     args=[%s]\n     fresh session", n, savedDisplay),
		Action: "saved",
	})
	n++
	// New params + resumed session is only meaningful when there ARE new params
	// that differ from what's saved; otherwise it collapses into "saved + session".
	if hasResumable && len(agentExtra) > 0 && !argsMatch {
		choices = append(choices, configChoice{
			Label:  fmt.Sprintf("%d) use new params + session\n     args=[%s]\n     resume=%s", n, currentDisplay, savedSession),
			Action: "new+resume",
		})
		n++
	}
	// Same dedup for the fresh-session row: when args match, "use new" is
	// byte-identical to "use saved" already in the list.
	if !argsMatch {
		choices = append(choices, configChoice{
			Label:  fmt.Sprintf("%d) use new params passed in\n     args=[%s]\n     fresh session", n, currentDisplay),
			Action: "new",
		})
	}
	return choices
}

// selectAction maps fzf's returned line back to a choice Action. Each Label is
// unique by construction, so an exact match is unambiguous; an unmatched
// selection (shouldn't happen) yields "".
func selectAction(choices []configChoice, selectedLabel string) string {
	for _, c := range choices {
		if c.Label == selectedLabel {
			return c.Action
		}
	}
	return ""
}

// composeTagRestartArgs resolves the final agent args for a tag-restart picker
// selection (shell create branch 1953-2016). The resume variants strip any
// existing resume binding from their base and re-append the canonical one from
// savedSession via composeResumeArgs (M1) — which also fixes codex's args[0]
// ordering. "new" keeps what was typed (the caller also removes the stale
// config); "saved" reuses the cleaned persisted args. persistedConfigArgs is the
// single strip (a superset of the shell's per-site resume-only strips — it also
// drops a stray --session-id, which is safer for a fresh/re-composed launch and
// keeps M1's ARCH-DRY consolidation).
func composeTagRestartArgs(action, agent string, savedArgsClean, agentExtra []string, savedSession string) []string {
	switch action {
	case "saved+resume":
		return composeResumeArgs(agent, savedArgsClean, savedSession)
	case "new+resume":
		return composeResumeArgs(agent, persistedConfigArgs(agentExtra), savedSession)
	case "saved":
		return append([]string(nil), savedArgsClean...)
	default: // "new" and any unmatched selection keep the typed args verbatim.
		return append([]string(nil), agentExtra...)
	}
}
