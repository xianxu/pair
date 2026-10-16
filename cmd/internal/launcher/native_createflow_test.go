package launcher

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestNativeCreateAdmissionPrecedesLedgerAndReachesOnlyAgentHandoff(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "agy", "muse", "qoder", "grok"} {
		t.Run(agent, func(t *testing.T) {
			rt := newFakeRuntime()
			calls := 0
			rt.nativeReserve = func(_ context.Context, request NativeAdmissionRequest) (string, error) {
				calls++
				if request.Agent != agent || request.NativeID != "11111111-1111-4111-8111-111111111111" || request.Attempt == "" || request.CommandDigest == "" {
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
			opts := baseOpts(LaunchArgs{Agent: agent, AgentExplicit: true, ForcedTag: "work", AgentArgs: resumeToken(agent, "11111111-1111-4111-8111-111111111111"), AgentArgsExplicit: true})
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
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "11111111-1111-4111-8111-111111111111"}, AgentArgsExplicit: true})
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
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"resume", "11111111-1111-4111-8111-111111111111"}, AgentArgsExplicit: true})
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
			rt.nativeReserve = func(_ context.Context, q NativeAdmissionRequest) (string, error) {
				if q.NativeID != "33333333-3333-4333-8333-333333333333" {
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
			rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
				t.Fatal("unresolved selector reserved")
				return "", nil
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

func TestNativeCreateForkDoesNotReserveOrRecordSourceAsWriter(t *testing.T) {
	rt := newFakeRuntime()
	rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
		t.Fatal("fork reserved source conversation")
		return "", nil
	}
	opts := baseOpts(LaunchArgs{Agent: "claude", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"--resume", "fork-source", "--fork-session"}, AgentArgsExplicit: true})
	rt.files[filepath.Join(opts.Env.DataDir, "config-work-claude.json")] = `{"agent":"claude","args":[],"session_id":"stale"}`
	code, err := run(t, opts, rt)
	if code != 0 || err != nil || rt.launchCount != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_SESSION_ID"] != "" || rt.env[NativeAdmissionEnv] != "" {
		t.Fatal("fork inherited source writer identity")
	}
}

// Keep the first terminal handoff open while a second slot tries admission.
// The filesystem authority serializes same-ID writers without holding the lock
// for the long-lived terminal, so independent IDs still reach their handoff.
func TestNativeCreateTwoSlotAdmissionBarrier(t *testing.T) {
	for _, same := range []bool{true, false} {
		t.Run(map[bool]string{true: "same conversation", false: "independent conversations"}[same], func(t *testing.T) {
			world := newAdmissionWorld()
			store := world.store(t.TempDir())
			first, second := newFakeRuntime(), newFakeRuntime()
			for _, rt := range []*fakeRuntime{first, second} {
				rt.nativeReserve = func(ctx context.Context, q NativeAdmissionRequest) (string, error) {
					return store.Reserve(ctx, q, noLegacy)
				}
				rt.nativeCancel = store.CancelBeforeHandoff
			}
			entered, release := make(chan struct{}), make(chan struct{})
			first.launchHook = func(int) {
				close(entered)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
					t.Error("second slot did not complete admission while first terminal remained open")
				}
			}
			opts := func(tag, id string) LaunchOptions {
				o := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: tag, AgentArgs: []string{"resume", id}, AgentArgsExplicit: true})
				o.SkipConfigPicker = true
				return o
			}
			type result struct {
				code int
				err  error
			}
			done := make(chan result, 1)
			go func() {
				code, err := run(t, opts("slot-one", "11111111-1111-4111-8111-111111111111"), first)
				done <- result{code, err}
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("first slot never reached native handoff")
			}
			id := "11111111-1111-4111-8111-111111111111"
			if !same {
				id = "22222222-2222-4222-8222-222222222222"
			}
			code, err := run(t, opts("slot-two", id), second)
			close(release)
			a := <-done
			if a.code != 0 || a.err != nil {
				t.Fatalf("first slot failed: %+v", a)
			}
			if same {
				if code == 0 && err == nil {
					t.Fatal("two slots launched the same native conversation")
				}
				if second.launched != "" || len(second.preparedLaunches) != 0 || len(second.ledger["slot-two"]) != 0 {
					t.Fatal("losing slot published a writer")
				}
			} else if code != 0 || err != nil || second.launched == "" {
				t.Fatalf("independent writer blocked: code=%d err=%v", code, err)
			}
		})
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
			rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
				t.Error("indirect compatibility invented a reservation")
				return "", errors.New("wrong admission")
			}
			rt.launchHook = func(int) {
				cmd, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(cmd.Argv, inlineModeArgs("codex", args, nil)) {
					t.Fatalf("native invocation changed: got %q original %q", cmd.Argv, args)
				}
				if rt.env[NativeAdmissionEnv] != "" || rt.env["PAIR_SESSION_ID"] != "" {
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
			rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
				t.Fatal("new conversation reserved an existing writer")
				return "", nil
			}
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

// pair#597 close review BR-1: only Couch's resume and recovery profiles
// (registry-checked) skip admission. An ordinary Couch spawn whose saved
// argv names a conversation still reserves and hands the wrapper a grant.
func TestCouchSpawnWithResumeArgvStillReservesAdmission(t *testing.T) {
	rt := newFakeRuntime()
	reserved := 0
	rt.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
		reserved++
		return "exact-grant", nil
	}
	rt.launchHook = func(int) {
		if rt.env[NativeAdmissionEnv] != "exact-grant" {
			t.Fatal("Couch spawn resume handed off without a grant")
		}
	}
	raw, err := BuildCouchLaunchProfile("work", "claude", []string{"--resume", "11111111-1111-4111-8111-111111111111"}, "explicit", "explicit")
	if err != nil {
		t.Fatal(err)
	}
	opts := baseOpts(LaunchArgs{ForcedTag: "work"})
	if opts.Args, _, err = ApplyCouchLaunchProfile(opts.Args, raw); err != nil {
		t.Fatal(err)
	}
	opts.SkipConfigPicker = true
	scope, err := ResolveRepoScope(opts.Env.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, "work"
	code, err := run(t, opts, rt)
	if err != nil || code != 0 || reserved != 1 {
		t.Fatalf("Couch spawn resume: code=%d err=%v reserved=%d", code, err, reserved)
	}
}
