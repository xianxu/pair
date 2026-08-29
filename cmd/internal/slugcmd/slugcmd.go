// pair-slug — propose an orientation slug for a pair tab.
//
// Spawned (backgrounded) by pair-wrap at turn-end — pair's agent-agnostic
// notify point — so it works for claude/codex/agy alike (issue #000027 M3,
// replacing the earlier claude-only Stop hook). It reads the scanner-authorized
// transcript only after the shared inventory establishes the Pair owner, parses
// the native format into turns, derives the left segment from the
// git branch, asks a small model for the <focus> right segment over the recent
// transcript (with a KEEP gate), validates, and writes a candidate to
// exact proposed-slug binding. nvim applies it (see nvim/slug.lua).
//
// Inputs (all env / filesystem — no stdin):
//
//	PAIR_TAG, PAIR_DATA_DIR   required launch identity/scope
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
package slugcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/model"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

const (
	recentTurns  = 12  // baseline recency window fed to the model
	minUserTurns = 3   // extend back until the window holds this many user turns
	hardMaxTurns = 40  // cap on how far back the user-turn extension reaches
	perTurnChars = 500 // truncation per turn
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

func inventoryTranscript(runtime sessioninventory.Runtime, scopeKey, tag string, agent sessioninventory.Agent) ([]byte, sessioninventory.BindingStatus, error) {
	query, err := sessioninventory.QuerySession(runtime, scopeKey, tag, agent)
	if err != nil || query.Root == nil {
		return nil, query.Status, err
	}
	data, err := sessioninventory.ReadRootTranscript(runtime, *query.Root)
	return data, query.Status, err
}

// Run is the pair-slug body: env-driven (no args, no stdout/stderr — writes
// only to files + $PAIR_SLUG_LOG), tolerant (every path returns 0 so a hiccup
// never disturbs the agent). Shared by the bin/pair-slug shim and `pair slug`.
func Run() int {
	if os.Getenv("PAIR_SLUG_NESTED") != "" {
		logf("nested invocation (PAIR_SLUG_NESTED); skipping to avoid recursion")
		return 0
	}

	tag := os.Getenv("PAIR_TAG")
	dataDir := os.Getenv("PAIR_DATA_DIR")
	if tag == "" || dataDir == "" {
		logf("no PAIR_TAG/PAIR_DATA_DIR; not inside a pair session")
		return 0
	}
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		logf("unsafe artifact namespace: %v", err)
		return 0
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

	var data []byte
	transcriptPath := os.Getenv("PAIR_SLUG_TRANSCRIPT")
	if transcriptPath != "" {
		data, err = os.ReadFile(transcriptPath)
	} else {
		runtime := sessioninventory.NewOSRuntime(home, dataDir)
		var status sessioninventory.BindingStatus
		data, status, err = inventoryTranscript(runtime, os.Getenv("PAIR_SCOPE_KEY"), tag, sessioninventory.Agent(agent))
		if err == nil && status != sessioninventory.BindingEstablished {
			logf("native session is %s; slug waits for an established binding", status)
			return 0
		}
	}
	if err != nil {
		logf("read established transcript: %v", err)
		lg.Log(4, "slug-parse", adapt.Fail, "read transcript: "+err.Error())
		return 0
	}
	turns := windowTurns(parseTranscript(agent, data), recentTurns, minUserTurns, hardMaxTurns, perTurnChars)
	if len(turns) == 0 {
		logf("no turns extracted (agent=%s)", agent)
		lg.Log(4, "slug-parse", adapt.NearMiss, "transcript read but 0 turns extracted (agent="+agent+")")
		return 0
	}
	lg.Log(4, "slug-parse", adapt.Fired, fmt.Sprintf("%d turns", len(turns)))

	// prev is the effective slug nvim last wrote (includes user edits).
	prev := ""
	if b, err := os.ReadFile(paths.Slug()); err == nil {
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
		return 0
	}

	write, value := decide(branchLeft, prev, out)
	if !write {
		logf("no write (KEEP/invalid/unchanged): model=%q", modelLine(out))
		return 0
	}

	// Atomic write: nvim is a concurrent reader of slug-proposed-<tag>; write
	// to a temp sibling then rename so it never observes a torn file.
	proposed := paths.SlugProposed()
	tmp := proposed + ".tmp"
	if err := os.WriteFile(tmp, []byte(value+"\n"), 0o644); err != nil {
		logf("write %q: %v", tmp, err)
		return 0
	}
	if err := os.Rename(tmp, proposed); err != nil {
		logf("rename %q→%q: %v", tmp, proposed, err)
		return 0
	}
	logf("proposed: %s", value)
	return 0
}
