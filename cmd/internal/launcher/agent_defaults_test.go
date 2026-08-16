package launcher

import (
	"reflect"
	"strings"
	"testing"
)

func TestAgentDefaultRoundTripNormalizesArgs(t *testing.T) {
	raw, err := BuildAgentDefault("codex", nil)
	if err != nil {
		t.Fatalf("BuildAgentDefault returned error: %v", err)
	}
	got, err := ParseAgentDefault("codex", raw)
	if err != nil {
		t.Fatalf("ParseAgentDefault returned error: %v", err)
	}
	if got.Agent != "codex" {
		t.Fatalf("Agent = %q, want codex", got.Agent)
	}
	if got.Args == nil || len(got.Args) != 0 {
		t.Fatalf("Args = %#v, want empty non-nil slice", got.Args)
	}
}

func TestParseAgentDefaultRejectsMalformedOrMismatchedJSON(t *testing.T) {
	for _, tc := range []struct {
		name  string
		agent string
		raw   string
	}{
		{"malformed", "codex", `{"agent":`},
		{"empty agent", "codex", `{"agent":"","args":[]}`},
		{"wrong agent", "codex", `{"agent":"claude","args":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAgentDefault(tc.agent, tc.raw); err == nil {
				t.Fatalf("ParseAgentDefault(%q, %q) returned nil error", tc.agent, tc.raw)
			}
		})
	}
}

func TestAgentDefaultBuildAndParseDefensiveCopies(t *testing.T) {
	args := []string{"--sandbox", "workspace-write"}
	raw, err := BuildAgentDefault("codex", args)
	if err != nil {
		t.Fatalf("BuildAgentDefault returned error: %v", err)
	}
	args[0] = "--mutated"
	got, err := ParseAgentDefault("codex", raw)
	if err != nil {
		t.Fatalf("ParseAgentDefault returned error: %v", err)
	}
	if !reflect.DeepEqual(got.Args, []string{"--sandbox", "workspace-write"}) {
		t.Fatalf("Args = %#v, want original args", got.Args)
	}
	got.Args[0] = "--changed"
	again, err := ParseAgentDefault("codex", raw)
	if err != nil {
		t.Fatalf("ParseAgentDefault second parse returned error: %v", err)
	}
	if again.Args[0] != "--sandbox" {
		t.Fatalf("ParseAgentDefault leaked args slice mutation: %#v", again.Args)
	}
}

func TestScopedPathsAgentDefaultStaysUnderScopeDir(t *testing.T) {
	scope, err := ResolveRepoScope("/Users/x/workspace/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	paths := NewScopedPaths("/data", scope, "work")
	scopeDir := paths.ScopeDir()

	if got, want := paths.AgentDefault("codex"), scopeDir+"/agent-default-codex.json"; got != want {
		t.Fatalf("AgentDefault(codex) = %q, want %q", got, want)
	}
	unsafe := paths.AgentDefault("../../codex")
	if !strings.HasPrefix(unsafe, scopeDir+"/") {
		t.Fatalf("unsafe agent default path escaped scope: %q", unsafe)
	}
	if strings.Contains(unsafe, "..") || strings.Contains(strings.TrimPrefix(unsafe, scopeDir+"/"), "/") {
		t.Fatalf("unsafe agent default path kept traversal tokens: %q", unsafe)
	}
}
