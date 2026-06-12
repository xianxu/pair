// pair-slug — propose an orientation slug for a pair tab.
//
// Spawned (backgrounded) by pair-wrap at turn-end — pair's agent-agnostic
// notify point — so it works for claude/codex/agy alike (issue #000027 M3,
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
//	PAIR_AGENT                agent name (claude|codex|agy); default claude
//	PAIR_SLUG_MODEL           small-model override; default depends on agent
//	PAIR_SLUG_TRANSCRIPT      explicit transcript path, bypassing resolution
//	                          (tests; also lets pair-wrap pass it directly)
//	PAIR_SLUG_NESTED          set by the model child — makes pair-slug no-op
//	OPENAI_API_KEY            optional for Codex's direct OpenAI API path
//	cwd                       the repo (inherited from pair-wrap) — branch left
//
// Failure mode: any error is non-fatal — logs to $PAIR_SLUG_LOG when set and
// exits 0 without writing, so a hiccup never disturbs the agent or the draft.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/model"
)

const (
	recentTurns        = 12  // baseline recency window fed to the model
	minUserTurns       = 3   // extend back until the window holds this many user turns
	hardMaxTurns       = 40  // cap on how far back the user-turn extension reaches
	perTurnChars       = 500 // truncation per turn
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

var codexRolloutRE = regexp.MustCompile(`^(.*/\.codex/sessions/.*/rollout-.*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl)$`)

func processChildren() map[string][]string {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return nil
	}
	children := make(map[string][]string)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, ppid := fields[0], fields[1]
		children[ppid] = append(children[ppid], pid)
	}
	return children
}

func descendantPIDs(root string, children map[string][]string) []string {
	if root == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{root: true}
	queue := []string{root}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		out = append(out, pid)
		for _, child := range children[pid] {
			if seen[child] {
				continue
			}
			seen[child] = true
			queue = append(queue, child)
		}
	}
	return out
}

func lsofNames(pid string) []string {
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return nil
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			names = append(names, line[1:])
		}
	}
	return names
}

func resolveLiveCodexTranscript(dataDir, tag, home string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "agent-pid-"+tag))
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(b))
	if root == "" {
		return ""
	}
	prefix := filepath.Join(home, ".codex", "sessions") + string(os.PathSeparator)
	for _, pid := range descendantPIDs(root, processChildren()) {
		for _, name := range lsofNames(pid) {
			if strings.HasPrefix(name, prefix) && codexRolloutRE.MatchString(name) {
				return name
			}
		}
	}
	return ""
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
	case "agy":
		// ~/.gemini/antigravity-cli/brain/<sid>/.system_generated/logs/transcript.jsonl
		return filepath.Join(home, ".gemini", "antigravity-cli", "brain", sid, ".system_generated", "logs", "transcript.jsonl")
	default: // claude
		return filepath.Join(home, ".claude", "projects", claudePathEncoder.Replace(cwd), sid+".jsonl")
	}
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
	// Aspect 4 flight recorder: slug-parse fires on a successful parse,
	// near-misses when a transcript is read but yields no turns (schema drift),
	// fails when no transcript resolves at all. See atlas §3.
	lg := adapt.Open("pair-slug", agent)
	defer lg.Close()
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	transcript := os.Getenv("PAIR_SLUG_TRANSCRIPT")
	if transcript == "" {
		sid := sessionID(dataDir, tag, agent)
		if sid != "" {
			transcript = resolveTranscript(agent, sid, cwd, home)
		}
		if transcript == "" && agent == "codex" {
			transcript = resolveLiveCodexTranscript(dataDir, tag, home)
			if transcript != "" {
				logf("resolved live codex transcript without config: %s", transcript)
			}
		}
		if transcript == "" && sid == "" {
			// No session id yet — normal early in a session before the watcher
			// has written the config. Not a drift signal, so don't log `fail`
			// (it would fire on every turn-end until the id resolves).
			logf("no session_id in config-%s-%s.json", tag, agent)
			return
		}
	}
	if transcript == "" {
		logf("could not resolve transcript for agent %q", agent)
		lg.Log(4, "slug-parse", adapt.Fail, "could not resolve transcript for agent "+agent)
		return
	}

	data, err := os.ReadFile(transcript)
	if err != nil {
		logf("read transcript %q: %v", transcript, err)
		lg.Log(4, "slug-parse", adapt.Fail, "read transcript: "+err.Error())
		return
	}
	turns := windowTurns(parseTranscript(agent, data), recentTurns, minUserTurns, hardMaxTurns, perTurnChars)
	if len(turns) == 0 {
		logf("no turns extracted (agent=%s, transcript=%s)", agent, transcript)
		lg.Log(4, "slug-parse", adapt.NearMiss, "transcript read but 0 turns extracted (agent="+agent+")")
		return
	}
	lg.Log(4, "slug-parse", adapt.Fired, fmt.Sprintf("%d turns", len(turns)))

	// prev is the effective slug nvim last wrote (includes user edits).
	prev := ""
	if b, err := os.ReadFile(filepath.Join(dataDir, "slug-"+tag)); err == nil {
		prev = strings.TrimSpace(string(b))
	}

	modelName := os.Getenv("PAIR_SLUG_MODEL")
	if modelName == "" {
		modelName = model.DefaultModel(agent)
	}
	branchLeft := normalizeBranch(gitBranch(cwd), repoBase(cwd))

	out, err := model.Run(model.Request{
		Agent:           agent,
		Model:           modelName,
		Prompt:          buildPrompt(branchLeft),
		Input:           buildModelInput(prev, turns),
		MaxOutputTokens: 64,
		Verbosity:       "low",
	})
	if err != nil {
		logf("model %q failed: %v", modelName, err)
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
