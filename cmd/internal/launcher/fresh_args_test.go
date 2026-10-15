package launcher

import (
	"fmt"
	"testing"
)

// These value-taking options are documented by installed `muse --help`,
// `codex exec --help`, and `codex exec resume --help`. A value before resume
// must not turn a native existing-conversation command into an ordinary prompt.
func TestNativeResumeAfterNativeOptionValuesRetainsExactID(t *testing.T) {
	const id = "019f0102-0304-7506-8708-090a0b0c0d0e"
	for _, tc := range []struct {
		agent         string
		prefix        []string
		option, value string
		afterResume   bool
	}{
		{agent: "muse", option: "-c", value: `reasoning_effort="high"`},
		{agent: "muse", option: "--config", value: `reasoning_effort="high"`},
		{agent: "muse", option: "--effort", value: "high"},
		{agent: "muse", option: "--permission-profile", value: "test"},
		{agent: "codex", prefix: []string{"exec"}, option: "--color", value: "never"},
		{agent: "codex", prefix: []string{"exec"}, option: "-o", value: "/tmp/output"},
		{agent: "codex", prefix: []string{"exec"}, option: "--output-last-message", value: "/tmp/output"},
		{agent: "codex", prefix: []string{"exec"}, option: "--output-schema", value: "/tmp/schema"},
		{agent: "codex", prefix: []string{"exec"}, option: "--thread-source", value: "cli", afterResume: true},
		{agent: "codex", prefix: []string{"exec"}, option: "--cyber-access-program", value: "standard", afterResume: true},
	} {
		positions := []bool{false}
		if tc.agent == "muse" || tc.afterResume || tc.option == "--output-schema" {
			positions = append(positions, true)
		}
		for _, after := range positions {
			t.Run(tc.agent+"/"+tc.option+"/after="+fmt.Sprint(after), func(t *testing.T) {
				args := append([]string(nil), tc.prefix...)
				if after {
					args = append(args, "resume", tc.option, tc.value, id)
				} else {
					args = append(args, tc.option, tc.value, "resume", id)
				}
				s, err := ParseNativeResumeSelector(tc.agent, args)
				if err != nil || s.Kind != NativeResumeExplicit || s.NativeID != id || CodexIndirectResumeCompatibility(s) {
					t.Fatalf("native option value hid exact resume: args=%q kind=%s id=%q err=%v", args, s.Kind, s.NativeID, err)
				}
				if err := ValidateFreshAgentArgs(tc.agent, args); err == nil {
					t.Fatalf("Fresh accepted existing-conversation command after option value: %q", args)
				}
			})
		}
	}
}

func TestNativeOptionValuesRemainLiteralFreshData(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
	}{
		{"muse", []string{"--permission-profile", "resume", "ordinary prompt"}},
		{"codex", []string{"exec", "--output-schema", "resume", "ordinary prompt"}},
		{"codex", []string{"exec", "-o", "resume", "ordinary prompt"}},
	} {
		s, err := ParseNativeResumeSelector(tc.agent, tc.args)
		if err != nil || s.Kind != NativeResumeNone {
			t.Fatalf("option value became resume: %q kind=%s err=%v", tc.args, s.Kind, err)
		}
		if err := ValidateFreshAgentArgs(tc.agent, tc.args); err != nil {
			t.Fatalf("option value refused Fresh: %q err=%v", tc.args, err)
		}
	}
}
