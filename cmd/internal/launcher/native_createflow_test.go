package launcher

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Native inventory is consulted once. A newer session arriving after admission
// cannot change which conversation the native CLI opens.
func TestNativeCreatePinsLatestBeforePublication(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
	}{

		{"claude", []string{"--continue", "--model", "test-model"}},
		{"qoder", []string{"--continue"}},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			rt := newFakeRuntime()
			resolves := 0
			rt.nativeResolve = func(_ context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
				resolves++
				if q.Selector.Kind != NativeResumeLatest || q.CWD != baseOpts(LaunchArgs{}).Env.Cwd {
					t.Fatalf("wrong native scope: %+v", q)
				}
				args, err := PinNativeResumeSelector(q.Selector, "33333333-3333-4333-8333-333333333333")
				return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: "33333333-3333-4333-8333-333333333333", Args: args}, err
			}
			rt.launchHook = func(int) {
				command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				parsed, err := ParseNativeResumeSelector(tc.agent, command.Argv)
				if err != nil || parsed.Kind != NativeResumeExplicit || parsed.NativeID != "33333333-3333-4333-8333-333333333333" {
					t.Fatalf("not pinned %+v err=%v", command, err)
				}
				if tc.agent == "codex" && !strings.Contains(strings.Join(command.Argv, " "), "continue work") {
					t.Fatal("native prompt lost")
				}
			}
			opts := baseOpts(LaunchArgs{Agent: tc.agent, AgentExplicit: true, ForcedTag: "work", AgentArgs: tc.args, AgentArgsExplicit: true})
			// A stale saved config must not replace the user's native selector.
			rt.files[filepath.Join(opts.Env.DataDir, "config-work-"+tc.agent+".json")] = `{"agent":"` + tc.agent + `","args":[],"session_id":"stale"}`
			code, err := run(t, opts, rt)
			if code != 0 || err != nil || resolves != 1 {
				t.Fatalf("code=%d err=%v resolves=%d", code, err, resolves)
			}
		})
	}
}

func TestNativeCreateUnresolvedSelectorHasNoWriterEffects(t *testing.T) {
	for _, status := range []sessioninventory.NativeSelectorStatus{sessioninventory.NativeSelectorNone, sessioninventory.NativeSelectorAmbiguous, sessioninventory.NativeSelectorIncomplete, sessioninventory.NativeSelectorUnsupported} {
		t.Run(string(status), func(t *testing.T) {
			rt := newFakeRuntime()
			rt.nativeResolve = func(context.Context, NativeResumeRequest) (NativeResumeResolution, error) {
				return NativeResumeResolution{Status: status, Reason: "bounded native inventory did not resolve"}, nil
			}
			opts := baseOpts(LaunchArgs{Agent: "muse", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "--last"}, AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			code, err := run(t, opts, rt)
			if code == 0 && err == nil {
				t.Fatal("unresolved native selector launched")
			}
			if rt.launched != "" || len(rt.preparedLaunches) != 0 || len(rt.ledger["work"]) != 0 {
				t.Fatal("unresolved selector published a native writer")
			}
		})
	}
}

func TestNativeCreateForkSkipsCapabilityAndSourceWriterIdentity(t *testing.T) {
	rt := newFakeRuntime()
	opts := baseOpts(LaunchArgs{Agent: "claude", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"--resume", "fork-source", "--fork-session"}, AgentArgsExplicit: true})
	rt.files[filepath.Join(opts.Env.DataDir, "config-work-claude.json")] = `{"agent":"claude","args":[],"session_id":"stale"}`
	code, err := run(t, opts, rt)
	if code != 0 || err != nil || rt.launchCount != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_SESSION_ID"] != "" {
		t.Fatal("fork inherited source writer identity")
	}
}

func TestNativeCreateCodexIndirectCompatibilityPreservesInvocation(t *testing.T) {
	for _, args := range [][]string{
		{"resume", "--last"},
		{"--profile", "operator-profile", "resume", "--last", "continue this task"},
		{"resume"},
		{"resume", "native-human-name", "--all"},
		{"resume", "11111111"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rt := newFakeRuntime()
			rt.nativeResolve = func(context.Context, NativeResumeRequest) (NativeResumeResolution, error) {
				t.Error("compatibility invoked unresolved adapter")
				return NativeResumeResolution{}, errors.New("not implemented")
			}
			rt.launchHook = func(int) {
				cmd, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(cmd.Argv, inlineModeArgs("codex", args, nil)) {
					t.Fatalf("native invocation changed: got %q original %q", cmd.Argv, args)
				}
				if rt.env["PAIR_SESSION_ID"] != "" {
					t.Fatal("indirect launch invented native identity or authority")
				}
			}
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: args, AgentArgsExplicit: true})
			rt.files[filepath.Join(opts.Env.DataDir, "config-work-codex.json")] = `{"agent":"codex","args":[],"session_id":"11111111-1111-4111-8111-111111111111"}`
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if strings.Contains(stderr.String(), "starting fresh") {
				t.Errorf("native selection falsely reported Fresh: %s", stderr.String())
			}
			if code != 0 || err != nil || rt.launched == "" {
				t.Fatalf("Codex CLI regression: code=%d err=%v", code, err)
			}
			if got := rt.ledger["work"]; len(got) != 1 || got[0].SessionID != "" {
				t.Fatalf("indirect selector recorded as canonical identity: %+v", got)
			}
		})
	}
}

func TestNativeCreateChosenNewIDBypassesSavedPicker(t *testing.T) {
	for _, agent := range []string{"claude", "qoder"} {
		t.Run(agent, func(t *testing.T) {
			rt := newFakeRuntime()
			args := []string{"--session-id", "44444444-4444-4444-8444-444444444444", "--model", "chosen-model"}
			opts := baseOpts(LaunchArgs{Agent: agent, AgentExplicit: true, ForcedTag: "work", AgentArgs: args, AgentArgsExplicit: true})
			rt.files[filepath.Join(opts.Env.DataDir, "config-work-"+agent+".json")] = `{"agent":"` + agent + `","args":[],"session_id":"stale"}`
			rt.launchHook = func(int) {
				cmd, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(cmd.Argv, inlineModeArgs(agent, args, nil)) {
					t.Fatalf("new-conversation invocation replaced: %q", cmd.Argv)
				}
			}
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if code != 0 || err != nil || rt.launchCount != 1 || strings.Contains(stderr.String(), "starting fresh") {
				t.Fatalf("new choice intercepted: code=%d err=%v launches=%d stderr=%s", code, err, rt.launchCount, &stderr)
			}
		})
	}
}
