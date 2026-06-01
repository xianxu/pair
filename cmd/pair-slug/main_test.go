package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles pair-slug into a temp dir once per integration test.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-slug")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// fakeClaude writes a PATH-shimmed `claude` that ignores its args/stdin and
// prints body — a process-level fake, not a function mock.
func fakeClaude(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' " + shellQuote(body) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// runSlug runs the built binary with a fake claude on PATH, a fake
// transcript, and an isolated PAIR_DATA_DIR. cwd is a non-git temp dir so
// the branch left is deterministically the dir basename.
func runSlug(t *testing.T, bin, claudeDir, modelOut string) (dataDir, cwd string) {
	t.Helper()
	dataDir = t.TempDir()
	cwd = t.TempDir()

	transcript := filepath.Join(t.TempDir(), "t.jsonl")
	lines := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"add the keep gate"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"on it"}]}}`,
	}, "\n")
	if err := os.WriteFile(transcript, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin)
	cmd.Dir = cwd // os.Getwd() in the binary → branch left = basename(cwd)
	cmd.Env = append(os.Environ(),
		"PATH="+claudeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"PAIR_TAG=testtag",
		"PAIR_DATA_DIR="+dataDir,
		"PAIR_AGENT=claude",
		"PAIR_SLUG_MODEL=fake-model",
		"PAIR_SLUG_TRANSCRIPT="+transcript,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run pair-slug: %v\n%s", err, out)
	}
	return dataDir, cwd
}

func TestIntegrationProposesValidSlug(t *testing.T) {
	bin := buildBinary(t)
	claudeDir := fakeClaude(t, "=== whatever | doing tests ===")
	dataDir, cwd := runSlug(t, bin, claudeDir, "")

	got, err := os.ReadFile(filepath.Join(dataDir, "slug-proposed-testtag"))
	if err != nil {
		t.Fatalf("expected slug-proposed-testtag: %v", err)
	}
	want := "=== " + filepath.Base(cwd) + " | doing tests ===\n"
	if string(got) != want {
		t.Errorf("proposed = %q, want %q (left stomped with branch/repo)", got, want)
	}
}

func TestIntegrationKeepWritesNothing(t *testing.T) {
	bin := buildBinary(t)
	claudeDir := fakeClaude(t, "KEEP")
	dataDir, _ := runSlug(t, bin, claudeDir, "")

	if _, err := os.Stat(filepath.Join(dataDir, "slug-proposed-testtag")); !os.IsNotExist(err) {
		t.Errorf("KEEP must not write a proposal (err=%v)", err)
	}
}

func TestIntegrationInvalidWritesNothing(t *testing.T) {
	bin := buildBinary(t)
	claudeDir := fakeClaude(t, "Sandbox restriction. Let me ask directly: where is the transcript?")
	dataDir, _ := runSlug(t, bin, claudeDir, "")

	if _, err := os.Stat(filepath.Join(dataDir, "slug-proposed-testtag")); !os.IsNotExist(err) {
		t.Errorf("invalid model output must not write a proposal (err=%v)", err)
	}
}

// TestIntegrationNestedGuard pins I1/I-B: with PAIR_SLUG_NESTED set, the
// binary must no-op — never invoke the model, never write a proposal — so a
// model child that re-fires the Stop hook can't recurse.
func TestIntegrationNestedGuard(t *testing.T) {
	bin := buildBinary(t)
	dataDir := t.TempDir()
	cwd := t.TempDir()
	marker := filepath.Join(t.TempDir(), "invoked")

	// fake claude that records the fact it ran — it must NOT run here.
	claudeDir := t.TempDir()
	script := "#!/bin/sh\ntouch " + shellQuote(marker) + "\nprintf '=== x | y ===\\n'\n"
	if err := os.WriteFile(filepath.Join(claudeDir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "t.jsonl")
	if err := os.WriteFile(transcript,
		[]byte(`{"type":"user","message":{"role":"user","content":"hi"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"PATH="+claudeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"PAIR_TAG=testtag",
		"PAIR_DATA_DIR="+dataDir,
		"PAIR_AGENT=claude",
		"PAIR_SLUG_TRANSCRIPT="+transcript,
		"PAIR_SLUG_NESTED=1",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("nested guard failed: model was invoked")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "slug-proposed-testtag")); !os.IsNotExist(err) {
		t.Error("nested guard failed: a proposal was written")
	}
}

func TestDescendantPIDsIncludesNestedChildren(t *testing.T) {
	children := map[string][]string{
		"10": {"11", "12"},
		"11": {"13"},
		"13": {"14"},
	}
	got := descendantPIDs("10", children)
	want := []string{"10", "11", "12", "13", "14"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("descendantPIDs = %v, want %v", got, want)
	}
}

func TestCodexRolloutPattern(t *testing.T) {
	path := "/Users/x/.codex/sessions/2026/05/31/rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl"
	if !codexRolloutRE.MatchString(path) {
		t.Fatalf("codexRolloutRE did not match %q", path)
	}
}

func TestResolveLiveCodexTranscriptUsesDescendantLsof(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-testtag"), []byte("10\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
		"rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl")
	binDir := t.TempDir()
	ps := "#!/bin/sh\nprintf ' 10 1\\n 11 10\\n'\n"
	if err := os.WriteFile(filepath.Join(binDir, "ps"), []byte(ps), 0o755); err != nil {
		t.Fatal(err)
	}
	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + path + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
	if err := os.WriteFile(filepath.Join(binDir, "lsof"), []byte(lsof), 0o755); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	got := resolveLiveCodexTranscript(dataDir, "testtag", home)
	if got != path {
		t.Fatalf("resolveLiveCodexTranscript = %q, want %q", got, path)
	}
}

func TestDefaultModelByAgent(t *testing.T) {
	if got := defaultModel("codex"); got != defaultOpenAIModel {
		t.Fatalf("codex default model = %q, want %q", got, defaultOpenAIModel)
	}
	if got := defaultModel("claude"); got != defaultClaudeModel {
		t.Fatalf("claude default model = %q, want %q", got, defaultClaudeModel)
	}
}

func TestResponseTextParsesOutputTextConvenience(t *testing.T) {
	got, err := responseText([]byte(`{"output_text":"=== pair | openai slug ==="}`))
	if err != nil {
		t.Fatal(err)
	}
	if got != "=== pair | openai slug ===" {
		t.Fatalf("responseText = %q", got)
	}
}

func TestResponseTextParsesOutputMessageContent(t *testing.T) {
	raw := []byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"KEEP"}]}]}`)
	got, err := responseText(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != "KEEP" {
		t.Fatalf("responseText = %q", got)
	}
}

func TestRunOpenAIModelPostsResponsesRequest(t *testing.T) {
	var reqBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"=== pair | openai slug ==="}]}]}`))
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("PAIR_SLUG_OPENAI_BASE_URL", srv.URL)
	got, err := runOpenAIModel("gpt-test-mini", "prompt", "input")
	if err != nil {
		t.Fatal(err)
	}
	if got != "=== pair | openai slug ===" {
		t.Fatalf("runOpenAIModel = %q", got)
	}
	if reqBody["model"] != "gpt-test-mini" || reqBody["instructions"] != "prompt" || reqBody["input"] != "input" {
		t.Fatalf("request body = %#v", reqBody)
	}
}
