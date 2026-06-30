package sessionwatch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentSpecExtractsCodexSessionID(t *testing.T) {
	home := "/tmp/home"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"

	spec, ok := SpecForAgent("codex", home)
	if !ok {
		t.Fatalf("codex spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
		t.Fatalf("codex match = %+v, want id %q", got, sid)
	}
}

func TestAgentSpecExtractsAgySessionID(t *testing.T) {
	home := "/tmp/home"
	sid := "123e4567-e89b-12d3-a456-426614174000"
	path := home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"

	spec, ok := SpecForAgent("agy", home)
	if !ok {
		t.Fatalf("agy spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || got.NearMiss || got.ID != sid || got.Path != path {
		t.Fatalf("agy match = %+v, want id %q", got, sid)
	}
}

func TestAgentSpecReportsNearMissForPatternWithBadID(t *testing.T) {
	home := "/tmp/home"
	path := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"

	spec, ok := SpecForAgent("codex", home)
	if !ok {
		t.Fatalf("codex spec not found")
	}
	got := spec.Match(path)
	if !got.Matched || !got.NearMiss || got.ID != "" || got.Path != path {
		t.Fatalf("codex near miss = %+v", got)
	}
}

func TestAgentSpecRejectsUnsupportedAgent(t *testing.T) {
	if _, ok := SpecForAgent("claude", "/tmp/home"); ok {
		t.Fatalf("claude should not use async session watch")
	}
}

func TestStripResumeArgsRemovesCanonicalResumeBindings(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		args  []string
		want  []string
	}{
		{
			name:  "codex leading resume",
			agent: "codex",
			args:  []string{"resume", "abc", "--no-alt-screen"},
			want:  []string{"--no-alt-screen"},
		},
		{
			name:  "flag resume",
			agent: "agy",
			args:  []string{"--model", "x", "--resume", "abc", "--flag"},
			want:  []string{"--model", "x", "--flag"},
		},
		{
			name:  "unrelated args keep order",
			agent: "codex",
			args:  []string{"--foo", "bar", "resume"},
			want:  []string{"--foo", "bar", "resume"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripResumeArgs(tt.agent, tt.args)
			if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
				t.Fatalf("StripResumeArgs(%q, %#v) = %#v, want %#v", tt.agent, tt.args, got, tt.want)
			}
		})
	}
}

func TestConfigJSONUsesStructuredEncoding(t *testing.T) {
	got, err := ConfigJSON(ConfigPayload{
		Agent:     "codex",
		Args:      []string{`say "hi"`, "--flag"},
		SessionID: "019eff64-6ceb-7e72-9d41-a735a97029ac",
	})
	if err != nil {
		t.Fatalf("ConfigJSON error: %v", err)
	}
	var decoded ConfigPayload
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("ConfigJSON produced invalid JSON %q: %v", got, err)
	}
	if decoded.Agent != "codex" || decoded.SessionID == "" || len(decoded.Args) != 2 || decoded.Args[0] != `say "hi"` {
		t.Fatalf("decoded payload = %+v", decoded)
	}
}
