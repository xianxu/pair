package artifactpath

import "testing"

func TestAgentReadyCandidateIsDiagnosticFilenameOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"agent-ready-thread-claude.json", true},
		{"agent-ready-other-codex.json", true},
		{"agent-ready-.json", true}, // malformed readiness is still diagnostic evidence
		{"agent-ready-thread-claude.json.tmp", false},
		{"ready-thread.json", false},
		{"/private/agent-ready-thread-claude.json", false},
		{"../agent-ready-thread-claude.json", false},
		{"", false},
	} {
		if got := IsAgentReadyCandidate(tc.name); got != tc.want {
			t.Errorf("%q: got%t want%t", tc.name, got, tc.want)
		}
	}
}
