// pair-slug — propose an orientation slug for a pair tab.
//
// Spawned (backgrounded) by pair-wrap at turn-end — pair's agent-agnostic
// notify point — so it works for claude/codex/gemini alike (issue #000027 M3,
// replacing the earlier claude-only Stop hook). It resolves its own transcript
// from $PAIR_DATA_DIR/config-<tag>-<agent>.json (session_id) + the per-agent
// path, parses the native format into turns, derives the left segment from the
// git branch, asks a small model for the <focus> right segment over the recent
// transcript (with a KEEP gate), validates, and writes a candidate to
// $PAIR_DATA_DIR/slug-proposed-<tag>. nvim applies it (see nvim/slug.lua).
//
// Inputs (all env / filesystem — no stdin):
//
//	PAIR_TAG, PAIR_DATA_DIR   required; identify the session
//	PAIR_AGENT                agent name (claude|codex|gemini); default claude
//	PAIR_SLUG_MODEL           small-model override; default claude-haiku-4-5
//	PAIR_SLUG_TRANSCRIPT      explicit transcript path, bypassing resolution
//	                          (tests; also lets pair-wrap pass it directly)
//	PAIR_SLUG_NESTED          set by the model child — makes pair-slug no-op
//	cwd                       the repo (inherited from pair-wrap) — branch left
//
// Failure mode: any error is non-fatal — logs to $PAIR_SLUG_LOG when set and
// exits 0 without writing, so a hiccup never disturbs the agent or the draft.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	recentTurns  = 12  // baseline recency window fed to the model
	minUserTurns = 3   // extend back until the window holds this many user turns
	hardMaxTurns = 40  // cap on how far back the user-turn extension reaches
	perTurnChars = 500 // truncation per turn
	defaultModel = "claude-haiku-4-5"
	modelTimeout = 30 * time.Second // a hung model must not leave pair-slug resident
)

// logf writes a diagnostic line to $PAIR_SLUG_LOG if set; otherwise silent.
func logf(format string, a ...any) {
	path := os.Getenv("PAIR_SLUG_LOG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", a...)
}

// gitBranch returns the current branch in dir, or "" on any failure.
func gitBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// repoBase returns the repo's toplevel basename, falling back to the cwd
// basename when dir isn't a git repo.
func repoBase(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err == nil {
		if top := strings.TrimSpace(string(out)); top != "" {
			return filepath.Base(top)
		}
	}
	return filepath.Base(dir)
}

// sessionID reads the session id pair recorded for (tag, agent) — written by
// bin/pair-session-watch.sh once the agent's session file is discovered.
func sessionID(dataDir, tag, agent string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "config-"+tag+"-"+agent+".json"))
	if err != nil {
		return ""
	}
	var c struct {
		SessionID string `json:"session_id"`
	}
	if json.Unmarshal(b, &c) != nil {
		return ""
	}
	return c.SessionID
}

// claudePathEncoder mirrors nvim's `cwd:gsub('[./]', '-')` for the
// ~/.claude/projects/<encoded-cwd>/ directory name.
var claudePathEncoder = strings.NewReplacer(".", "-", "/", "-")

// resolveTranscript returns the on-disk transcript path for (agent, sid), or
// "" if it can't be located. Mirrors nvim/init.lua's session_age_hint.
func resolveTranscript(agent, sid, cwd, home string) string {
	switch agent {
	case "codex":
		// ~/.codex/sessions/YYYY/MM/DD/rollout-...-<sid>.jsonl
		matches, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "rollout-*"+sid+"*.jsonl"))
		if len(matches) > 0 {
			return matches[0]
		}
		return ""
	case "gemini":
		// ~/.gemini/tmp/<project>/chats/session-*.json whose .sessionId == sid
		var found string
		root := filepath.Join(home, ".gemini", "tmp")
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || found != "" {
				return nil
			}
			if d.IsDir() || !strings.HasPrefix(d.Name(), "session-") || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			b, e := os.ReadFile(p)
			if e != nil {
				return nil
			}
			var g struct {
				SessionID string `json:"sessionId"`
			}
			if json.Unmarshal(b, &g) == nil && g.SessionID == sid {
				found = p
			}
			return nil
		})
		return found
	default: // claude
		return filepath.Join(home, ".claude", "projects", claudePathEncoder.Replace(cwd), sid+".jsonl")
	}
}

// runModel invokes the small model: `claude -p --model <m> <prompt>` with
// input on stdin, returning raw stdout. The child inherits the env with
// PAIR_SLUG_NESTED=1 — a breaker against any recursion through the agent.
func runModel(model, prompt, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), modelTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "-p", "--model", model, prompt)
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	out, err := cmd.Output()
	return string(out), err
}

func main() {
	if os.Getenv("PAIR_SLUG_NESTED") != "" {
		logf("nested invocation (PAIR_SLUG_NESTED); skipping to avoid recursion")
		return
	}

	tag := os.Getenv("PAIR_TAG")
	dataDir := os.Getenv("PAIR_DATA_DIR")
	if tag == "" || dataDir == "" {
		logf("no PAIR_TAG/PAIR_DATA_DIR; not inside a pair session")
		return
	}
	agent := os.Getenv("PAIR_AGENT")
	if agent == "" {
		agent = "claude"
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	transcript := os.Getenv("PAIR_SLUG_TRANSCRIPT")
	if transcript == "" {
		sid := sessionID(dataDir, tag, agent)
		if sid == "" {
			logf("no session_id in config-%s-%s.json", tag, agent)
			return
		}
		transcript = resolveTranscript(agent, sid, cwd, home)
	}
	if transcript == "" {
		logf("could not resolve transcript for agent %q", agent)
		return
	}

	data, err := os.ReadFile(transcript)
	if err != nil {
		logf("read transcript %q: %v", transcript, err)
		return
	}
	turns := windowTurns(parseTranscript(agent, data), recentTurns, minUserTurns, hardMaxTurns, perTurnChars)
	if len(turns) == 0 {
		logf("no turns extracted (agent=%s, transcript=%s)", agent, transcript)
		return
	}

	// prev is the effective slug nvim last wrote (includes user edits).
	prev := ""
	if b, err := os.ReadFile(filepath.Join(dataDir, "slug-"+tag)); err == nil {
		prev = strings.TrimSpace(string(b))
	}

	model := os.Getenv("PAIR_SLUG_MODEL")
	if model == "" {
		model = defaultModel
	}
	branchLeft := normalizeBranch(gitBranch(cwd), repoBase(cwd))

	out, err := runModel(model, buildPrompt(branchLeft), buildModelInput(prev, turns))
	if err != nil {
		logf("model %q failed: %v", model, err)
		return
	}

	write, value := decide(branchLeft, prev, out)
	if !write {
		logf("no write (KEEP/invalid/unchanged): model=%q", modelLine(out))
		return
	}

	// Atomic write: nvim is a concurrent reader of slug-proposed-<tag>; write
	// to a temp sibling then rename so it never observes a torn file.
	proposed := filepath.Join(dataDir, "slug-proposed-"+tag)
	tmp := proposed + ".tmp"
	if err := os.WriteFile(tmp, []byte(value+"\n"), 0o644); err != nil {
		logf("write %q: %v", tmp, err)
		return
	}
	if err := os.Rename(tmp, proposed); err != nil {
		logf("rename %q→%q: %v", tmp, proposed, err)
		return
	}
	logf("proposed: %s", value)
}
