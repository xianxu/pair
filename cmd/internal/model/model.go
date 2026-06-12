// Package model is the per-agent small-model dispatch shared by pair-slug and
// pair-changelog. It was extracted near-verbatim from pair-slug's package main
// (issue #53) so both binaries call one model surface instead of duplicating
// the claude / codex / agy / OpenAI-Responses plumbing. (The runCodexCLI temp
// prefix was renamed pair-slug-codex-* → pair-model-codex-*.)
//
// The one behavioral change from the original is that MaxOutputTokens and
// Verbosity are now per-call parameters (the OpenAI path previously hardcoded
// 64 / "low", sized for a one-line slug — which would truncate a multi-entry
// change log). The CLI paths (claude / agy / codex) take no token cap, matching
// the original.
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// Timeout bounds a single model call so a hung child never leaves the
	// caller (spawned in the hot turn-end path) resident.
	Timeout = 30 * time.Second
	// DefaultClaudeModel / DefaultOpenAIModel mirror pair-slug's historical
	// defaults; callers may override per Request.
	DefaultClaudeModel = "claude-haiku-4-5"
	DefaultOpenAIModel = "gpt-5.4-mini"
)

// Request is one model call.
//
// MaxOutputTokens and Verbosity are honored only by the OpenAI Responses path;
// the CLI paths ignore them (no cap), as in the original pair-slug code.
//
// NOTE (preserved quirk): the agy path passes only Prompt — Input is dropped,
// exactly as the original runAgyModel did. Fixing that is out of scope for the
// #53 extraction; revisit if agy change-log quality suffers.
type Request struct {
	Agent           string        // "claude" | "codex" | "agy"
	Model           string        // "" → DefaultModel(Agent)
	Prompt          string        // instructions / system prompt
	Input           string        // content on stdin
	MaxOutputTokens int           // OpenAI hard cap; must be > 0 for the OpenAI path
	Verbosity       string        // "low" | "medium" | "high"; "" → "low"
	Timeout         time.Duration // per-call timeout; 0 → package Timeout default
}

// timeout returns the per-call timeout, defaulting to the package Timeout.
func (r Request) timeout() time.Duration {
	if r.Timeout > 0 {
		return r.Timeout
	}
	return Timeout
}

// DefaultModel is the small-model id for an agent family.
func DefaultModel(agent string) string {
	if agent == "codex" {
		return DefaultOpenAIModel
	}
	return DefaultClaudeModel
}

// Run invokes the small model for the active agent family.
func Run(r Request) (string, error) {
	if r.Model == "" {
		r.Model = DefaultModel(r.Agent)
	}
	if r.Verbosity == "" {
		r.Verbosity = "low"
	}
	if r.MaxOutputTokens <= 0 {
		r.MaxOutputTokens = 256 // defensive floor; the OpenAI API rejects <= 0
	}
	switch r.Agent {
	case "codex":
		if os.Getenv("OPENAI_API_KEY") != "" {
			return runOpenAI(r)
		}
		return runCodexCLI(r)
	case "agy":
		return runAgy(r)
	default:
		return runClaude(r)
	}
}

// runClaude invokes `claude -p --model <m> <prompt>` with input on stdin,
// returning raw stdout. The child inherits env with PAIR_SLUG_NESTED=1 — a
// breaker against any recursion through the agent.
func runClaude(r Request) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "-p", "--model", r.Model, r.Prompt)
	cmd.Stdin = strings.NewReader(r.Input)
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	// Run in an empty sandbox dir (as runAgy does): otherwise `claude -p` loads
	// the agent repo's CLAUDE.md + MCP servers + tools on every call — a ~25s
	// startup tax that blew the changelog timeout (#58). The distill only needs
	// the prompt + stdin, never the cwd.
	cmd.Dir = os.TempDir()
	out, err := cmd.Output()
	return string(out), err
}

// runAgy invokes `agy -p <prompt>`. Setting Dir to os.TempDir() is crucial: it
// forces the agy agent loop to execute in an empty sandbox directory,
// preventing it from discovering workspace files/projects or triggering slow
// background tool executions.
func runAgy(r Request) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, "agy", "-p", r.Prompt)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "PAIR_SLUG_NESTED=1")
	out, err := cmd.Output()
	return string(out), err
}

// runCodexCLI invokes the authenticated Codex CLI path. This lets
// subscription-authenticated Codex sessions generate output even when no
// OPENAI_API_KEY is exported for direct API calls.
func runCodexCLI(r Request) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()

	f, err := os.CreateTemp("", "pair-model-codex-*.txt")
	if err != nil {
		return "", err
	}
	outPath := f.Name()
	_ = f.Close()
	defer os.Remove(outPath)

	cmd := exec.CommandContext(ctx, "codex", "exec",
		"--model", r.Model,
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--ephemeral",
		"--output-last-message", outPath,
		"-",
	)
	cmd.Stdin = strings.NewReader(r.Prompt + "\n\n" + r.Input)
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

// runOpenAI invokes the Responses API directly. No SDK dependency: this is
// spawned in the hot turn-end path, and a tiny JSON POST is enough.
func runOpenAI(r Request) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()
	body := map[string]any{
		"model":             r.Model,
		"instructions":      r.Prompt,
		"input":             r.Input,
		"max_output_tokens": r.MaxOutputTokens,
		"text": map[string]string{
			"verbosity": r.Verbosity,
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
	return ResponseText(respBody)
}

// ResponseText extracts the text from an OpenAI Responses API payload, handling
// both the output_text convenience field and the structured output array.
func ResponseText(data []byte) (string, error) {
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
