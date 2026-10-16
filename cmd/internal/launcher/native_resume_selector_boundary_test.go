package launcher

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNativeResumeUnsupportedContextSelectorsRefuse(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
	}{
		{"claude", []string{"--teleport", "session"}},
		{"claude", []string{"--teleport=session"}},
		{"claude", []string{"--from-pr", "42"}},
		{"claude", []string{"--from-pr=42"}},
		{"claude", []string{"--cloud", "task"}},
		{"claude", []string{"--resume", "id", "--teleport", "session"}},
		{"claude", []string{"--from-pr", "42", "--resume", "id"}},
		{"claude", []string{"--session-id", "new", "--teleport", "session"}},
		{"claude", []string{"--fork-session", "--from-pr", "42"}},
		{"claude", []string{"--cloud", "task", "--fork-session"}},
		{"claude", []string{"attach", "session"}},
		{"claude", []string{"respawn", "session"}},
		{"qoder", []string{"--remote", "session"}},
		{"qoder", []string{"--remote-session=session"}},
		{"qoder", []string{"--remote-control", "session"}},
		{"qoder", []string{"--session-id", "new", "--teleport", "session"}},
	} {
		t.Run(tc.agent+"/"+strings.Join(tc.args, " "), func(t *testing.T) {
			if got, err := ParseNativeResumeSelector(tc.agent, tc.args); err == nil {
				t.Fatalf("unsupported context escaped as %+v", got)
			}
		})
	}
}
func TestNativeResumeFreshSelectorAuthorityHasNoUnclassifiedContextFlag(t *testing.T) {
	for agent, spec := range freshAgentSpecs {
		for _, flag := range strings.Fields(spec.forbiddenFlags) {
			got, err := ParseNativeResumeSelector(agent, []string{flag})
			if err == nil && got.Kind == NativeResumeNone {
				t.Errorf("Fresh authority %s %s escaped native admission", agent, flag)
			}
		}
	}
}
func TestNativeResumeUnsupportedSelectorWordsRemainLiteralValuesAndPrompts(t *testing.T) {
	for _, args := range [][]string{
		{"--model", "--teleport"}, {"--system-prompt", "--from-pr"},
		{"--system-prompt=--teleport"}, {"--model", "attach"},
		{"--", "--teleport", "session"}, {"--", "--from-pr", "42"},
		{"--", "attach", "session"}, {"explain --teleport and --from-pr"},
	} {
		got, err := ParseNativeResumeSelector("claude", args)
		if err != nil || got.Kind != NativeResumeNone {
			t.Errorf("literal %q became context selection: %+v %v", args, got, err)
		}
	}
}
func TestNativeCreateUnsupportedContextSelectorsPublishNothing(t *testing.T) {
	for _, args := range [][]string{{"--teleport", "session"}, {"--from-pr=42"}, {"--cloud", "task"}, {"attach", "session"}, {"respawn", "session"}, {"--resume", "id", "--teleport", "session"}, {"--session-id", "new", "--from-pr", "42"}, {"--fork-session", "--cloud", "task"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rt := newFakeRuntime()
			rt.nativeResolve = func(context.Context, NativeResumeRequest) (NativeResumeResolution, error) {
				t.Fatal("unsupported mapping reached resolution")
				return NativeResumeResolution{}, nil
			}
			opts := baseOpts(LaunchArgs{Agent: "claude", AgentExplicit: true, ForcedTag: "work", AgentArgs: args, AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if code == 0 && err == nil {
				t.Fatalf("unsupported selector launched: %s", &stderr)
			}
			if rt.launchCount != 0 || len(rt.preparedLaunches) != 0 || len(rt.ledger["work"]) != 0 {
				t.Fatal("unsupported selector published a native writer")
			}
		})
	}
}
