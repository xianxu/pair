package launcher

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Each agent's measured first-message grammar (#517's audit, #522's --help
// read): agy takes `-i PROMPT` before any `--`; the others a trailing
// positional.
func TestPromptArgsPerAgent(t *testing.T) {
	const p = "Continue from the saved continuation at \"/s/c.md\"."
	for _, c := range []struct {
		agent string
		argv  []string
		want  []string
	}{
		{"claude", []string{"--model", "opus"}, []string{"--model", "opus", p}},
		{"codex", []string{}, []string{p}},
		{"grok", []string{"--x"}, []string{"--x", p}},
		{"qoder", nil, []string{p}},
		{"muse", []string{"--y"}, []string{"--y", p}},
		{"/opt/bin/agy", []string{"--mode", "plan"}, []string{"--mode", "plan", "-i", p}},
		{"agy", []string{"--a", "--", "x"}, []string{"--a", "-i", p, "--", "x"}},
	} {
		argv := slices.Clone(c.argv)
		got := PromptArgs(c.agent, argv, p)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s %q: got %q, want %q", c.agent, c.argv, got, c.want)
		}
		if !slices.Equal(argv, c.argv) {
			t.Errorf("%s: PromptArgs changed its input", c.agent)
		}
		if empty := PromptArgs(c.agent, c.argv, ""); !slices.Equal(empty, append([]string{}, c.argv...)) {
			t.Errorf("%s: an empty prompt changed argv to %q", c.agent, empty)
		}
	}
	for _, agent := range supportedAgents {
		if got := PromptArgs(agent, nil, p); !slices.Contains(got, p) {
			t.Errorf("supported agent %s has no prompt form: %q", agent, got)
		}
	}
}

func TestValidatePrompt(t *testing.T) {
	for _, bad := range []string{"", "two\nlines", "-flag-like", strings.Repeat("x", MaxPromptBytes+1), "nul\x00", "\xff"} {
		if ValidatePrompt(bad) == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	if err := ValidatePrompt("Continue from the saved continuation at \"/a b/c.md\"."); err != nil {
		t.Fatal(err)
	}
}

// The prompt is admitted only on a fresh profile, and it reaches the agent
// command beside the argv -- never the argv, the ledger or the saved config,
// so a later launch can't replay it (#522).
func TestLaunchPromptRidesBesideTheArgvOnly(t *testing.T) {
	const prompt = "Continue from the saved continuation at \"/store/continuation.md\": read it."
	fresh := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "codex", Argv: []string{"--model", "m"}, AgentSource: "explicit", ArgvSource: "explicit", FreshRequired: true, FreshNonce: "attempt-1", Prompt: prompt}
	ordinary := fresh
	ordinary.FreshRequired, ordinary.FreshNonce = false, ""
	if ValidateTrustedLaunchProfile(ordinary) == nil {
		t.Fatal("a prompt was admitted without a fresh launch")
	}
	resume := ordinary
	resume.ResumeRequired, resume.RequiredSessionID, resume.AgentSource, resume.ArgvSource = true, "sid", "saved", "saved"
	if ValidateTrustedLaunchProfile(resume) == nil {
		t.Fatal("a prompt was admitted on a resume")
	}
	bad := fresh
	bad.Prompt = "-x"
	if ValidateTrustedLaunchProfile(bad) == nil {
		t.Fatal("a flag-like prompt was admitted")
	}
	raw, _ := json.Marshal(fresh)
	args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	rt := newFakeRuntime()
	opts := baseOpts(args)
	opts.Env.DataDir = t.TempDir()
	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("launch %d %v", code, err)
	}
	command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
	if err != nil {
		t.Fatal(err)
	}
	if command.Prompt != prompt || slices.Contains(command.Argv, prompt) {
		t.Fatalf("agent command %+v: the prompt must ride beside argv", command)
	}
	if rt.env["PAIR_LAUNCH_NONCE"] != "attempt-1" {
		t.Fatalf("nonce %q", rt.env["PAIR_LAUNCH_NONCE"])
	}
	for path, body := range rt.files {
		if strings.Contains(body, "saved continuation") {
			t.Errorf("%s persisted the prompt", path)
		}
	}
	for key, entries := range rt.ledger {
		if strings.Contains(fmt.Sprint(entries), "saved continuation") {
			t.Errorf("ledger %s carries the prompt", key)
		}
	}
}
