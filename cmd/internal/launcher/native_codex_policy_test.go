package launcher

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestNativeCreateCodexIndirectDoesNotResolveOrReserve(t *testing.T) {
	for _, args := range [][]string{
		{"resume", "--last"},
		{"exec", "resume", "--last", "continue work"},
		{"e", "resume", "--last"},
		{"resume"},
		{"resume", "native-human-name", "--all"},
		{"resume", "11111111"},
		{"--profile", "opaque-profile", "-c", "malformed = [", "resume", "--last"},
	} {
		t.Run(joinArgs(args), func(t *testing.T) {
			rt := newFakeRuntime()
			resolves := 0
			rt.nativeResolve = func(context.Context, NativeResumeRequest) (NativeResumeResolution, error) {
				resolves++
				return NativeResumeResolution{}, errors.New("native probing must be unreachable")
			}
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: append([]string(nil), args...), AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			opts.InlineOptOut = map[string]bool{"codex": true}
			code, err := run(t, opts, rt)
			if code != 0 || err != nil || resolves != 0 || rt.launchCount != 1 {
				t.Fatalf("indirect invocation probed or refused: code=%d err=%v resolves=%d launches=%d", code, err, resolves, rt.launchCount)
			}
			command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(command.Argv, args) || rt.env["PAIR_SESSION_ID"] != "" {
				t.Fatalf("indirect invocation gained identity or changed args: %+v", command)
			}
			if entries := rt.ledger["work"]; len(entries) != 1 || entries[0].SessionID != "" {
				t.Fatalf("indirect invocation published canonical identity: %+v", entries)
			}
		})
	}
}
func TestNativeCreateCodexExactUUIDStillResolves(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	for _, prefix := range [][]string{nil, {"exec"}, {"e"}} {
		t.Run(joinArgs(prefix), func(t *testing.T) {
			args := append(append([]string(nil), prefix...), "resume", id)
			rt := newFakeRuntime()
			resolves := 0
			rt.nativeResolve = func(_ context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
				resolves++
				pinned, err := PinNativeResumeSelector(q.Selector, id)
				return NativeResumeResolution{Status: sessioninventory.NativeSelectorResolved, NativeID: id, Args: pinned}, err
			}
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: args, AgentArgsExplicit: true})
			opts.SkipConfigPicker = true
			opts.InlineOptOut = map[string]bool{"codex": true}
			code, err := run(t, opts, rt)
			if code != 0 || err != nil || resolves != 1 || rt.launchCount != 1 || rt.env["PAIR_SESSION_ID"] != id {
				t.Fatalf("exact UUID lost its pin: code=%d err=%v resolves=%d launches=%d", code, err, resolves, rt.launchCount)
			}
		})
	}
}
func joinArgs(args []string) string {
	if len(args) == 0 {
		return "tui"
	}
	out := args[0]
	for _, arg := range args[1:] {
		out += " " + arg
	}
	return out
}
