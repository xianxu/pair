package launcher

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

const delimiterNativeID = "019d1111-1111-7111-8111-111111111111"
const delimiterReplacementID = "019d2222-2222-7222-8222-222222222222"

func TestNativeResumeDelimiterKeepsCommandPositionalIdentity(t *testing.T) {
	for _, prefix := range [][]string{{"resume"}, {"exec", "resume"}, {"e", "resume"}, {"--model", "m", "resume"}, {"resume", "--model", "m"}} {
		args := append(append([]string(nil), prefix...), "--", delimiterNativeID, "prompt")
		t.Run("codex/"+strings.Join(prefix, " "), func(t *testing.T) {
			got, err := ParseNativeResumeSelector("codex", args)
			if err != nil || got.Kind != NativeResumeExplicit || got.NativeID != delimiterNativeID || CodexIndirectResumeCompatibility(got) {
				t.Fatalf("canonical positional ID escaped: %+v %v", got, err)
			}
			exact, err := PinNativeResumeSelector(got, delimiterNativeID)
			if err != nil || !reflect.DeepEqual(exact, args) {
				t.Fatalf("exact ID moved/duplicated: got=%q want=%q err=%v", exact, args, err)
			}
			want := append([]string(nil), args...)
			want[len(prefix)+1] = delimiterReplacementID
			pinned, err := PinNativeResumeSelector(got, delimiterReplacementID)
			if err != nil || !reflect.DeepEqual(pinned, want) {
				t.Fatalf("pin=%q want=%q err=%v", pinned, want, err)
			}
		})
	}
	args := []string{"--provider", "echo", "resume", "--", delimiterNativeID}
	got, err := ParseNativeResumeSelector("muse", args)
	if err != nil || got.Kind != NativeResumeExplicit || got.NativeID != delimiterNativeID {
		t.Fatalf("Muse positional identity: %+v %v", got, err)
	}
	pinned, err := PinNativeResumeSelector(got, delimiterNativeID)
	if err != nil || !reflect.DeepEqual(pinned, args) {
		t.Fatalf("Muse pin=%q err=%v", pinned, err)
	}
}
func TestNativeResumeDelimiterPreservesPromptRoles(t *testing.T) {
	for _, agent := range AgentInventory() {
		got, err := ParseNativeResumeSelector(agent, []string{"--", "resume", delimiterNativeID, "--resume"})
		if err != nil || got.Kind != NativeResumeNone {
			t.Errorf("%s promoted literal prompt into selector: %+v %v", agent, got, err)
		}
	}
	for _, tc := range []struct {
		agent string
		args  []string
		kind  NativeResumeKind
		id    string
		want  []string
	}{
		{"codex", []string{"exec", "--", "resume", delimiterNativeID}, NativeResumeNone, "", nil},
		{"codex", []string{"resume", "--last", "--", delimiterNativeID}, NativeResumeLatest, "", []string{"resume", delimiterReplacementID, "--", delimiterNativeID}},
		{"codex", []string{"exec", "resume", "--last", "--", delimiterNativeID}, NativeResumeLatest, "", []string{"exec", "resume", delimiterReplacementID, "--", delimiterNativeID}},
		{"muse", []string{"resume", "--last", "--", "literal --resume"}, NativeResumeLatest, "", []string{"resume", delimiterReplacementID, "--", "literal --resume"}},
		{"codex", []string{"resume", delimiterNativeID, "--", delimiterReplacementID}, NativeResumeExplicit, delimiterNativeID, []string{"resume", delimiterReplacementID, "--", delimiterReplacementID}},
		{"claude", []string{"--resume", "--", delimiterNativeID}, NativeResumePicker, "", []string{"--resume", delimiterReplacementID, "--", delimiterNativeID}},
		{"qoder", []string{"--resume", "--", delimiterNativeID}, NativeResumePicker, "", []string{"--resume", delimiterReplacementID, "--", delimiterNativeID}},
		{"grok", []string{"--resume", "--", delimiterNativeID}, NativeResumeLatest, "", []string{"--resume", delimiterReplacementID, "--", delimiterNativeID}},
		{"agy", []string{"--conversation", "--", delimiterNativeID}, NativeResumePicker, "", []string{"--conversation", delimiterReplacementID, "--", delimiterNativeID}},
		{"claude", []string{"--session-id", "new", "--", "--resume", delimiterNativeID}, NativeResumeNew, "", nil},
		{"codex", []string{"fork", "--", delimiterNativeID}, NativeResumeFork, delimiterNativeID, nil},
		{"grok", []string{"--resume", delimiterNativeID, "--fork-session", "--", "prompt"}, NativeResumeFork, delimiterNativeID, nil},
	} {
		got, err := ParseNativeResumeSelector(tc.agent, tc.args)
		if err != nil || got.Kind != tc.kind || got.NativeID != tc.id {
			t.Errorf("%s %q => %+v %v", tc.agent, tc.args, got, err)
			continue
		}
		if tc.want != nil {
			pinned, err := PinNativeResumeSelector(got, delimiterReplacementID)
			if err != nil || !reflect.DeepEqual(pinned, tc.want) {
				t.Errorf("%s pin=%q want=%q err=%v", tc.agent, pinned, tc.want, err)
			}
		}
	}
}
func TestNativeResumeLiteralDelimiterOptionValueDoesNotHideSelector(t *testing.T) {
	for _, agent := range AgentInventory() {
		option := "--model"
		if agent == "muse" {
			option = "--provider"
		}
		args := append([]string{option, "--"}, resumeToken(agent, delimiterNativeID)...)
		got, err := ParseNativeResumeSelector(agent, args)
		if err != nil || got.Kind != NativeResumeExplicit || got.NativeID != delimiterNativeID {
			t.Errorf("%s option value hid ID: %+v %v", agent, got, err)
			continue
		}
		want := append([]string{option, "--"}, resumeToken(agent, delimiterReplacementID)...)
		pinned, err := PinNativeResumeSelector(got, delimiterReplacementID)
		if err != nil || !reflect.DeepEqual(pinned, want) {
			t.Errorf("%s literal delimiter corrupted: got=%q want=%q err=%v", agent, pinned, want, err)
		}
	}
}
func TestNativeCreateDelimiterCanonicalIDChecksCapabilityAndKeepsInvocation(t *testing.T) {
	for _, args := range [][]string{{"resume", "--", delimiterNativeID}, {"exec", "resume", "--", delimiterNativeID}, {"--model", "--", "resume", "--", delimiterNativeID}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rt := newFakeRuntime()
			checked := 0
			rt.nativeResolve = func(_ context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
				if q.Selector.Kind != NativeResumeExplicit || q.Selector.NativeID != delimiterNativeID {
					t.Fatalf("wrong resolution %+v", q.Selector)
				}
				pinned, err := PinNativeResumeSelector(q.Selector, delimiterNativeID)
				return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: delimiterNativeID, Args: pinned}, err
			}
			rt.nativeCapability = func(_ context.Context, _ NativeLockExemption) error {
				checked++
				return nil
			}
			rt.launchHook = func(int) {
				command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(command.Argv, inlineModeArgs("codex", args, nil)) {
					t.Fatalf("invocation changed: %q want=%q", command.Argv, args)
				}
			}
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: args, AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if err != nil || code != 0 || checked != 1 {
				t.Fatalf("canonical command unchecked: count=%d code=%d err=%v stderr=%s", checked, code, err, &stderr)
			}
		})
	}
}
