package launcher

import (
	"strings"
	"testing"
)

func TestParseLaunchArgsDefaultsToClaude(t *testing.T) {
	args, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", args.Agent)
	}
	if args.ForcedTag != "" {
		t.Fatalf("ForcedTag = %q, want empty", args.ForcedTag)
	}
	if len(args.AgentArgs) != 0 {
		t.Fatalf("AgentArgs = %#v, want empty", args.AgentArgs)
	}
}

func TestParseLaunchArgsAgentAndForwardedArgs(t *testing.T) {
	args, err := ParseArgs([]string{"codex", "--", "-p", "say hi"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "codex" {
		t.Fatalf("Agent = %q, want codex", args.Agent)
	}
	if got := strings.Join(args.AgentArgs, " "); got != "-p say hi" {
		t.Fatalf("AgentArgs = %q, want forwarded args", got)
	}
}

func TestParseLaunchArgsDefaultAgentWithForwardedArgs(t *testing.T) {
	args, err := ParseArgs([]string{"--", "--dangerously-skip-permissions"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", args.Agent)
	}
	if got := strings.Join(args.AgentArgs, " "); got != "--dangerously-skip-permissions" {
		t.Fatalf("AgentArgs = %q, want forwarded args", got)
	}
}

func TestParseLaunchArgsResumeNormalizesForcedTag(t *testing.T) {
	args, err := ParseArgs([]string{"resume", "pair-demo"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "" {
		t.Fatalf("Agent = %q, want empty for resume inference", args.Agent)
	}
	if args.ForcedTag != "demo" {
		t.Fatalf("ForcedTag = %q, want demo", args.ForcedTag)
	}
}

func TestParseLaunchArgsUnexpectedPositionalGuidesAgentArgs(t *testing.T) {
	_, err := ParseArgs([]string{"codex", "extra"})
	if err == nil {
		t.Fatal("ParseArgs returned nil error")
	}
	msg := err.Error()
	for _, want := range []string{"unexpected positional arg", "use '--' to forward args to the agent"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error missing %q: %s", want, msg)
		}
	}
}

func TestParseLaunchArgsUnsupportedLaunchSubcommandsAreExplicit(t *testing.T) {
	for _, verb := range []string{"continue", "rename", "list"} {
		t.Run(verb, func(t *testing.T) {
			_, err := ParseArgs([]string{verb})
			if err == nil {
				t.Fatal("ParseArgs returned nil error")
			}
			if !strings.Contains(err.Error(), "not implemented by pair-go launch") {
				t.Fatalf("error = %q, want explicit unsupported message", err)
			}
		})
	}
}
