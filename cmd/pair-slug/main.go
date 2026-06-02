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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	recentTurns        = 12  // baseline recency window fed to the model
	minUserTurns       = 3   // extend back until the window holds this many user turns
	hardMaxTurns       = 40  // cap on how far back the user-turn extension reaches
	perTurnChars       = 500 // truncation per turn
	defaultClaudeModel = "claude-haiku-4-5"
	defaultOpenAIModel = "gpt-5.4-mini"
	modelTimeout       = 30 * time.Second // a hung model must not leave pair-slug resident
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
	case "agy":
		// ~/.gemini/antigravity-cli/brain/<sid>/.system_generated/logs/transcript.jsonl
		return filepath.Join(home, ".gemini", "antigravity-cli", "brain", sid, ".system_generated", "logs", "transcript.jsonl")
	default: // claude
		return filepath.Join(home, ".claude", "projects", claudePathEncoder.Replace(cwd), sid+".jsonl")
	}
}

func defaultModel(agent string) string {
	if agent == "codex" {
		return defaultOpenAIModel
	}
	return defaultClaudeModel
}

// runModel invokes the small model for the active agent family.
func runModel(agent, model, prompt, input string) (string, error) {
	if agent == "codex" {
		if os.Getenv("OPENAI_API_KEY") != "" {
			return runOpenAIModel(model, prompt, input)
		}
		return runCodexCLIModel(model, prompt, input)
	}
	if agent == "agy" {
		return runAgyModel(prompt, input)
	}
	return runClaudeModel(model, prompt, input)
}

// runClaudeModel invokes `claude -p --model <m> <prompt>` with
// input on stdin, returning raw stdout. The child inherits env with
// PAIR_SLUG_NESTED=1 — a breaker against any recursion through the agent.
func runClaudeModel(model, prompt, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), modelTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "-p", "--model", model, prompt)
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	out, err := cmd.Output()
	return string(out), err
}

// runAgyModel invokes `agy -p <prompt>` with the transcript input on stdin.
// Setting Dir to os.TempDir() is crucial: it forces the agy agent loop to execute in
// an empty sandbox directory, preventing it from discovering workspace files/projects
// or triggering slow background tool executions.
func runAgyModel(prompt, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), modelTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "agy", "-p", prompt)
	cmd.Stdin = strings.NewReader(input)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	out, err := cmd.Output()
	return string(out), err
}

// runCodexCLIModel invokes the authenticated Codex CLI path. This lets
// subscription-authenticated Codex sessions generate slugs even when no
// OPENAI_API_KEY is exported for direct API calls.
func runCodexCLIModel(model, prompt, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), modelTimeout)
	defer cancel()

	f, err := os.CreateTemp("", "pair-slug-codex-*.txt")
	if err != nil {
		return "", err
	}
	outPath := f.Name()
	_ = f.Close()
	defer os.Remove(outPath)

	cmd := exec.CommandContext(ctx, "codex", "exec",
		"--model", model,
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--ephemeral",
		"--output-last-message", outPath,
		"-",
	)
	cmd.Stdin = strings.NewReader(prompt + "\n\n" + input)
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codex exec failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if b, err := os.ReadFile(outPath); err == nil && strings.TrimSpace(string(b)) != "" {
		return string(b), nil
	}
	return string(out), nil
}

// runOpenAIModel invokes the Responses API directly. No SDK dependency: this
// binary is spawned in the hot turn-end path, and a tiny JSON POST is enough.
func runOpenAIModel(model, prompt, input string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), modelTimeout)
	defer cancel()
	body := map[string]any{
		"model":             model,
		"instructions":      prompt,
		"input":             input,
		"max_output_tokens": 64,
		"text": map[string]string{
			"verbosity": "low",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	baseURL := os.Getenv("PAIR_SLUG_OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	url := strings.TrimRight(baseURL, "/") + "/v1/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai responses status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return responseText(respBody)
}

func responseText(data []byte) (string, error) {
	var r struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", err
	}
	if strings.TrimSpace(r.OutputText) != "" {
		return r.OutputText, nil
	}
	var parts []string
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if (c.Type == "output_text" || c.Type == "text") && c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("openai response had no output text")
	}
	return strings.Join(parts, "\n"), nil
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
			logf("no session_id in config-%s-%s.json", tag, agent)
			return
		}
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
		model = defaultModel(agent)
	}
	branchLeft := normalizeBranch(gitBranch(cwd), repoBase(cwd))

	out, err := runModel(agent, model, buildPrompt(branchLeft), buildModelInput(prev, turns))
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
