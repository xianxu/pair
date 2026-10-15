package launcher

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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
