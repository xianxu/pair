package launcher

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestNativeCreateAdmissionPrecedesLedgerAndReachesOnlyAgentHandoff(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "agy", "muse", "qoder", "grok"} {
		t.Run(agent, func(t *testing.T) {
			rt := newFakeRuntime()
			calls := 0
			rt.nativeReserve = func(_ context.Context, request NativeAdmissionRequest) (string, error) {
				calls++
				if request.Agent != agent || request.NativeID != "canonical-native" || request.Attempt == "" || request.CommandDigest == "" {
					t.Fatalf("invalid reservation %+v", request)
				}
				if len(rt.ledger["work"]) != 0 || len(rt.preparedLaunches) != 0 {
					t.Fatal("identity was published before admission")
				}
				if rt.env[NativeAdmissionEnv] != "" {
					t.Fatal("grant exposed before reservation")
				}
				return "exact-grant", nil
			}
			rt.launchHook = func(int) {
				if calls != 1 || rt.env[NativeAdmissionEnv] != "exact-grant" {
					t.Fatal("native handoff has no reserved grant")
				}
			}
			opts := baseOpts(LaunchArgs{Agent: agent, AgentExplicit: true, ForcedTag: "work", AgentArgs: resumeToken(agent, "canonical-native"), AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			code, err := run(t, opts, rt)
			if err != nil || code != 0 || calls != 1 {
				t.Fatalf("admission calls=%d code=%d err=%v", calls, code, err)
			}
			for _, env := range rt.pollerEnvs {
				if env[NativeAdmissionEnv] != "" {
					t.Fatal("title helper inherited native authority")
				}
			}
		})
	}
}

func TestNativeCreateAdmissionRefusalPublishesNoLaunch(t *testing.T) {
	rt := newFakeRuntime()
	rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
		return "", errors.New("another native writer")
	}
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "canonical-native"}, AgentArgsExplicit: true})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if code == 0 && err == nil {
		t.Fatal("unreserved native resume launched")
	}
	if rt.launched != "" || len(rt.ledger["work"]) != 0 || len(rt.preparedLaunches) != 0 || rt.env[NativeAdmissionEnv] != "" {
		t.Fatal("refused admission published or launched native identity")
	}
}

func TestNativeCreateFreshDropsInheritedAuthority(t *testing.T) {
	rt := newFakeRuntime()
	rt.env[NativeAdmissionEnv] = "inherited-old-grant"
	rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
		t.Fatal("Fresh reserved an existing native identity")
		return "", nil
	}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{}, AgentArgsExplicit: true, FreshRequired: true}), rt)
	if err != nil || code != 0 || rt.env[NativeAdmissionEnv] != "" {
		t.Fatalf("Fresh inherited authority: code=%d err=%v env=%v", code, err, rt.env[NativeAdmissionEnv])
	}
}

func TestNativeCreateCancellationStopsAtExposure(t *testing.T) {
	for _, exposed := range []bool{false, true} {
		t.Run(map[bool]string{false: "before handoff", true: "uncertain handoff"}[exposed], func(t *testing.T) {
			rt := newFakeRuntime()
			reserves, cancels := 0, 0
			rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) { reserves++; return "exact-grant", nil }
			rt.nativeCancel = func(_ context.Context, grant string) error {
				cancels++
				if grant != "exact-grant" || rt.env[NativeAdmissionEnv] != "" {
					t.Fatal("cancel after authority exposure")
				}
				return nil
			}
			if exposed {
				rt.launchErr = errors.New("client failed after possible server start")
			} else {
				rt.appendLedgerErr = errors.New("ledger refused")
			}
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "canonical-native"}, AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			code, err := run(t, opts, rt)
			if code == 0 && err == nil {
				t.Fatal("failed launch reported success")
			}
			want := 1
			if exposed {
				want = 0
			}
			if reserves != 1 || cancels != want {
				t.Fatalf("reserves=%d cancels=%d want=%d", reserves, cancels, want)
			}
			if !exposed && rt.launched != "" {
				t.Fatal("pre-handoff failure launched")
			}
			if exposed && !strings.Contains(rt.launched, "work") {
				t.Fatal("fixture never reached possible handoff")
			}
		})
	}
}

// Native inventory is consulted once. A newer session arriving after admission
// cannot change which conversation the native CLI opens.
func TestNativeCreatePinsLatestBeforePublication(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
	}{
		{"codex", []string{"--model", "test-model", "resume", "--last", "continue work"}},
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
				args, err := PinNativeResumeSelector(q.Selector, "chosen-native")
				return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: "chosen-native", Args: args}, err
			}
			rt.nativeReserve = func(_ context.Context, q NativeAdmissionRequest) (string, error) {
				if q.NativeID != "chosen-native" {
					t.Fatalf("reserved %+v", q)
				}
				// Simulate the native latest changing after resolution.
				rt.nativeResolve = func(context.Context, NativeResumeRequest) (NativeResumeResolution, error) {
					t.Fatal("selector resolved twice")
					return NativeResumeResolution{}, nil
				}
				return "pinned-grant", nil
			}
			rt.launchHook = func(int) {
				command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				parsed, err := ParseNativeResumeSelector(tc.agent, command.Argv)
				if err != nil || parsed.Kind != NativeResumeExplicit || parsed.NativeID != "chosen-native" {
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
			rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
				t.Fatal("unresolved selector reserved")
				return "", nil
			}
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "--last"}, AgentArgsExplicit: true})
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

func TestNativeCreateForkDoesNotReserveOrRecordSourceAsWriter(t *testing.T) {
	rt := newFakeRuntime()
	rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
		t.Fatal("fork reserved source conversation")
		return "", nil
	}
	opts := baseOpts(LaunchArgs{Agent: "claude", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"--resume", "fork-source", "--fork-session"}, AgentArgsExplicit: true})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if code != 0 || err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_SESSION_ID"] != "" || rt.env[NativeAdmissionEnv] != "" {
		t.Fatal("fork inherited source writer identity")
	}
}
