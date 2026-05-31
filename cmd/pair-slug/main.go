// pair-slug — propose an orientation slug for a pair tab.
//
// Invoked (backgrounded) from a Claude Code `Stop` hook. Reads the hook
// JSON on stdin (transcript_path, cwd), derives the left segment from the
// current git branch, asks a small model for the <focus> right segment over
// the recent transcript (with a KEEP gate), and writes a validated
// candidate to $PAIR_DATA_DIR/slug-proposed-<tag>. nvim is the consumer
// that applies it to the draft (see nvim/slug.lua, issue #000027).
//
// Single writer of slug-proposed-<tag>; never touches the draft or the
// effective slug-<tag> (nvim owns that, and it is this tool's `prev`).
//
// Failure mode: any error (no tag, no transcript, model failure, invalid
// output) is non-fatal — it logs to $PAIR_SLUG_LOG when set and exits 0
// without writing, so a hiccup never disturbs the agent or the draft.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type hookInput struct {
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
}

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

// runModel invokes the small model: `claude -p --model <m> <prompt>` with
// input on stdin, returning raw stdout. The child inherits the env with
// PAIR_SLUG_NESTED=1 set — a breaker so that if headless `claude -p` ever
// fires Stop hooks (it does not today), the nested pair-slug no-ops instead
// of recursing into an unbounded fork/cost loop.
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

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		logf("read stdin: %v", err)
		return
	}
	var hin hookInput
	if err := json.Unmarshal(raw, &hin); err != nil {
		logf("parse hook JSON: %v", err)
		return
	}
	if hin.Cwd == "" {
		hin.Cwd = "."
	}

	branchLeft := normalizeBranch(gitBranch(hin.Cwd), repoBase(hin.Cwd))

	transcript, err := os.ReadFile(hin.TranscriptPath)
	if err != nil {
		logf("read transcript %q: %v", hin.TranscriptPath, err)
		return
	}
	turns := extractTurns(transcript, recentTurns, minUserTurns, hardMaxTurns, perTurnChars)
	if len(turns) == 0 {
		logf("no turns extracted")
		return
	}

	// prev is the effective slug nvim last wrote (includes user edits).
	// Missing on a fresh tab → cold start, "(none)".
	prev := ""
	if b, err := os.ReadFile(filepath.Join(dataDir, "slug-"+tag)); err == nil {
		prev = strings.TrimSpace(string(b))
	}

	model := os.Getenv("PAIR_SLUG_MODEL")
	if model == "" {
		model = defaultModel
	}

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

	// Atomic write: nvim (M2) is a concurrent reader of slug-proposed-<tag>;
	// write to a temp sibling then rename so it never observes a torn file.
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
