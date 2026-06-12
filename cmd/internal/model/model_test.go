package model

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultModelByAgent(t *testing.T) {
	if got := DefaultModel("codex"); got != DefaultOpenAIModel {
		t.Fatalf("codex default = %q, want %q", got, DefaultOpenAIModel)
	}
	if got := DefaultModel("claude"); got != DefaultClaudeModel {
		t.Fatalf("claude default = %q, want %q", got, DefaultClaudeModel)
	}
	if got := DefaultModel("agy"); got != DefaultClaudeModel {
		t.Fatalf("agy default = %q, want %q", got, DefaultClaudeModel)
	}
}

func TestResponseTextParsesOutputTextConvenience(t *testing.T) {
	got, err := ResponseText([]byte(`{"output_text":"=== pair | openai ==="}`))
	if err != nil {
		t.Fatal(err)
	}
	if got != "=== pair | openai ===" {
		t.Fatalf("ResponseText = %q", got)
	}
}

func TestResponseTextParsesOutputMessageContent(t *testing.T) {
	raw := []byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"KEEP"}]}]}`)
	got, err := ResponseText(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != "KEEP" {
		t.Fatalf("ResponseText = %q", got)
	}
}

// TestRunForwardsMaxTokensAndVerbosity is the new coverage the #53 extraction
// adds: the OpenAI path must forward the caller's MaxOutputTokens / Verbosity
// (not the old hardcoded 64 / "low"), or a multi-entry change log truncates.
func TestRunForwardsMaxTokensAndVerbosity(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("PAIR_SLUG_OPENAI_BASE_URL", srv.URL)
	got, err := Run(Request{
		Agent: "codex", Model: "gpt-test-mini", Prompt: "prompt", Input: "input",
		MaxOutputTokens: 2000, Verbosity: "medium",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("Run = %q", got)
	}
	if reqBody["model"] != "gpt-test-mini" || reqBody["instructions"] != "prompt" || reqBody["input"] != "input" {
		t.Fatalf("request body = %#v", reqBody)
	}
	// JSON numbers decode to float64.
	if reqBody["max_output_tokens"] != float64(2000) {
		t.Fatalf("max_output_tokens = %v, want 2000", reqBody["max_output_tokens"])
	}
	text, _ := reqBody["text"].(map[string]any)
	if text == nil || text["verbosity"] != "medium" {
		t.Fatalf("text = %#v, want verbosity=medium", reqBody["text"])
	}
}

func TestRunCodexCLIWithoutAPIKey(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "stdin")
	// Fake codex: capture stdin, honor --output-last-message, print the body.
	script := strings.Join([]string{
		"#!/bin/sh",
		"out=''",
		"while [ \"$#\" -gt 0 ]; do",
		"  if [ \"$1\" = '--output-last-message' ]; then shift; out=$1; fi",
		"  shift",
		"done",
		"cat > '" + stdinPath + "'",
		"if [ -z \"$out\" ]; then exit 2; fi",
		"printf '%s\\n' 'codex cli output' > \"$out\"",
		"printf 'progress output ignored\\n'",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := Run(Request{Agent: "codex", Model: "gpt-test-mini", Prompt: "prompt text", Input: "input text"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "codex cli output\n" {
		t.Fatalf("Run = %q", got)
	}
	stdin, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stdin), "prompt text") || !strings.Contains(string(stdin), "input text") {
		t.Fatalf("codex stdin = %q", stdin)
	}
}

func TestRunCodexCLIReportsErrorOutput(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf 'auth failed\\n' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := Run(Request{Agent: "codex", Model: "gpt-test-mini", Prompt: "p", Input: "i"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("error = %v", err)
	}
}
