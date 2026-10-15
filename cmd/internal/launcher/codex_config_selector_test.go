package launcher

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestCodexConfigSelectorRetainsOnlyInvocationOptions(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		profile   string
		overrides []string
		mode      sessioninventory.CodexInvocationMode
	}{
		{[]string{"-p", "base", "-c", "model_provider=one", "resume", "--last", "--profile=runtime", "--config=sqlite_home='store'", "continue --profile literal"}, "runtime", []string{"model_provider=one", "sqlite_home='store'"}, sessioninventory.CodexModeTUI},
		{[]string{"-pcompact", "-cmodel_provider=one", "e", "resume", "--last", "--", "-p", "prompt"}, "compact", []string{"model_provider=one"}, sessioninventory.CodexModeExec},
		{[]string{"resume", "--last", "a prompt -c forced_login_method=api"}, "", nil, sessioninventory.CodexModeTUI},
	} {
		s, err := ParseNativeResumeSelector("codex", tc.args)
		if err != nil {
			t.Fatal(err)
		}
		field := reflect.ValueOf(s).FieldByName("CodexConfig")
		if !field.IsValid() {
			t.Fatal("native selector drops Codex configuration inputs")
		}
		got := field.Interface().(sessioninventory.CodexConfigInvocation)
		if got.Profile != tc.profile || !reflect.DeepEqual(got.Overrides, tc.overrides) || got.Mode != tc.mode {
			t.Fatalf("args=%q config=%+v", tc.args, got)
		}
		if !reflect.DeepEqual(s.Args, tc.args) {
			t.Fatal("parsing changed invocation")
		}
	}
}

func TestCodexLatestCreateAttemptsResolutionBeforeCompatibility(t *testing.T) {
	rt := newFakeRuntime()
	calls := 0
	rt.nativeResolve = func(_ context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
		calls++
		return NativeResumeResolution{Status: sessioninventory.NativeSelectorUnsupported, Reason: "cloud-unknown"}, nil
	}
	rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
		t.Fatal("cloud-unknown reserved")
		return "", nil
	}
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "--last", "continue work"}, AgentArgsExplicit: true})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if err != nil || code != 0 || calls != 1 || rt.launched == "" {
		t.Fatalf("resolution attempts=%d code=%d err=%v launched=%q", calls, code, err, rt.launched)
	}
	if rt.env[NativeAdmissionEnv] != "" || rt.env["PAIR_SESSION_ID"] != "" {
		t.Fatal("compatibility invented identity")
	}
	command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(command.Argv, " "), "--last continue work") {
		t.Fatalf("native compatibility changed args %q", command.Argv)
	}
}

func TestCodexLatestCreateBindsResolvedScopeAndPinsOnce(t *testing.T) {
	rt := newFakeRuntime()
	resolves, reserves := 0, 0
	scope := sessioninventory.CodexSelectionScope{CodexHome: "/private/codex", SQLiteHome: "/private/state", Provider: "openai", CWD: "/private/work", CWDs: []string{"/private/work"}, Mode: sessioninventory.CodexModeTUI}
	id := "019d1111-1111-7111-8111-111111111111"
	rt.nativeResolve = func(_ context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
		resolves++
		args, err := PinNativeResumeSelector(q.Selector, id)
		return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: id, Args: args, CodexScope: &scope}, err
	}
	rt.nativeReserve = func(_ context.Context, q NativeAdmissionRequest) (string, error) {
		reserves++
		want, err := EncodeNativeAdmissionCodexScope(scope)
		if err != nil || q.NativeID != id || q.CodexScope != want {
			t.Fatalf("scope lost at reservation: %+v %v", q, err)
		}
		rt.nativeResolve = func(context.Context, NativeResumeRequest) (NativeResumeResolution, error) {
			t.Fatal("resolved again after grant")
			return NativeResumeResolution{}, nil
		}
		return "scope-bound-grant", nil
	}
	rt.launchHook = func(int) {
		cmd, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
		if err != nil {
			t.Fatal(err)
		}
		s, err := ParseNativeResumeSelector("codex", cmd.Argv)
		if err != nil || s.NativeID != id || s.Kind != NativeResumeExplicit || rt.env[NativeAdmissionEnv] != "scope-bound-grant" {
			t.Fatalf("unpinned handoff: %+v %v", cmd, err)
		}
		if !strings.Contains(strings.Join(cmd.Argv, " "), "keep prompt") {
			t.Fatal("prompt lost")
		}
	}
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"-p", "runtime", "resume", "--last", "keep prompt"}, AgentArgsExplicit: true})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if code != 0 || err != nil || resolves != 1 || reserves != 1 {
		t.Fatalf("code=%d err=%v resolves=%d reserves=%d", code, err, resolves, reserves)
	}
}

func TestCodexLatestCreateIncompleteAdapterPreservesCompatibility(t *testing.T) {
	for _, failure := range []string{"adapter-error", "deadline", "canceled", "oversized-scope", "missing-scope"} {
		for _, mode := range []string{"ordinary", "required", "recovery", "explicit"} {
			if failure == "missing-scope" && mode != "ordinary" {
				continue
			}
			t.Run(failure+"/"+mode, func(t *testing.T) {
				original := []string{"resume", "--last", "continue work", "--no-alt-screen"}
				rt := newFakeRuntime()
				opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: append([]string(nil), original...), AgentArgsExplicit: true})
				opts.SkipConfigPicker = true
				var runtime Runtime = rt
				const id = "019d1111-1111-7111-8111-111111111111"
				switch mode {
				case "required":
					opts.Args.ResumeRequired = true
					opts.Args.RequiredSessionID = id
					opts.Args.AgentArgs = []string{"--no-alt-screen"}
					rt.bindingStatuses["work|codex"] = sessioninventory.BindingEstablished
					rt.establishedSessions["work|codex"] = id
				case "explicit":
					opts.Args.AgentArgs = []string{"--no-alt-screen", "resume", id}
				case "recovery":
					a := testRecoveryAuthority()
					scope, err := ResolveRepoScope(opts.Env.Cwd)
					if err != nil {
						t.Fatal(err)
					}
					a.Address.RepoScope = scope.Key
					a.Intent.Context.Address = a.Address
					a.Intent.Context.ExpiresAt = a.Intent.Context.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
					raw, err := BuildCouchRecoveryLaunchProfile("work", "codex", []string{"--no-alt-screen"}, a)
					if err != nil {
						t.Fatal(err)
					}
					opts.Args, _, err = ApplyCouchLaunchProfile(opts.Args, raw)
					if err != nil {
						t.Fatal(err)
					}
					opts.Args.CouchSessionV1 = true
					opts.Args.CouchSession = &CouchSessionIntent{Scope: scope.Key, Tag: "work", Name: "📁1-3", Nonce: a.Attempt, Disposition: "create"}
					opts.Env.CouchThreadScope = scope.Key
					opts.Env.CouchThreadTag = "work"
					runtime = &recoveryGateRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: rt}, record: recoveryAuthorityRecord(a, opts.Env.Cwd, "codex", 42, "helper")}
				}
				resolves, reserves := 0, 0
				rt.nativeResolve = func(ctx context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
					resolves++
					if ctx.Err() != nil {
						t.Fatal("adapter inherited cancellation before observation")
					}
					switch failure {
					case "adapter-error":
						return NativeResumeResolution{}, errors.New("observation unavailable")
					case "deadline":
						return NativeResumeResolution{}, context.DeadlineExceeded
					case "canceled":
						return NativeResumeResolution{}, context.Canceled
					}
					if failure == "missing-scope" {
						pinned, err := PinNativeResumeSelector(q.Selector, id)
						return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: id, Args: pinned}, err
					}
					scope := sessioninventory.CodexSelectionScope{CodexHome: "/private/codex", SQLiteHome: "/private/sqlite", Provider: "openai", CWD: "/private/work", Mode: sessioninventory.CodexModeTUI}
					for i := 0; i < 8; i++ {
						scope.CWDs = append(scope.CWDs, "/private/"+string(rune('a'+i))+"/"+strings.Repeat("segment/", 140)+"work")
					}
					if _, err := EncodeNativeAdmissionCodexScope(scope); err == nil || !strings.Contains(err.Error(), "too large") {
						t.Fatal("fixture did not exceed scope envelope", err)
					}
					pinned, err := PinNativeResumeSelector(q.Selector, id)
					return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: id, Args: pinned, CodexScope: &scope}, err
				}
				rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
					reserves++
					return "unexpected-grant", nil
				}
				var stderr bytes.Buffer
				code, err := RunLaunch(opts, runtime, &stderr)
				if err != nil || resolves != 1 || reserves != 0 {
					t.Fatalf("unexpected adapter effects code=%d err=%v resolves=%d reserves=%d: %s", code, err, resolves, reserves, &stderr)
				}
				if mode != "ordinary" {
					if code != 1 || rt.launchCount != 0 {
						t.Fatalf("strict/explicit launch downgraded code=%d launches=%d: %s", code, rt.launchCount, &stderr)
					}
					return
				}
				if code != 0 || rt.launchCount != 1 {
					t.Fatalf("ordinary native fallback refused code=%d launches=%d: %s", code, rt.launchCount, &stderr)
				}
				command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(command.Argv, original) || rt.env[NativeAdmissionEnv] != "" || rt.env["PAIR_SESSION_ID"] != "" {
					t.Fatalf("fallback changed original invocation: %+v grant=%t session=%q", command, rt.env[NativeAdmissionEnv] != "", rt.env["PAIR_SESSION_ID"])
				}
				if !reflect.DeepEqual(opts.Args.AgentArgs, original) {
					t.Fatal("fallback mutated caller args")
				}
			})
		}
	}
}
