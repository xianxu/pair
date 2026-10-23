package agentfixture

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestFixtureReadinessPublishesLiteralActualArgv(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			config := fixtureConfig(t, agent)
			config.Args = []string{"--model", "literal '$HOME' $(no expansion) 雪", "--", strings.Repeat("x", 5000)}
			want := append([]string{}, config.Args...)
			input, writer := io.Pipe()
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- Run(ctx, config, input, io.Discard) }()
			t.Cleanup(func() {
				cancel()
				_ = writer.Close()
				if err := <-done; err != nil {
					t.Error(err)
				}
			})
			var raw []byte
			waitFixture(t, func() bool {
				var err error
				raw, err = os.ReadFile(filepath.Join(config.ControlDir, "ready.json"))
				return err == nil
			})
			var ready struct{ Argv []string }
			if err := json.Unmarshal(raw, &ready); err != nil || !reflect.DeepEqual(ready.Argv, want) {
				t.Fatalf("actual argv elements=%d want=%d err=%v", len(ready.Argv), len(want), err)
			}
			config.Args[1] = "changed caller slice"
			if err := WriteCommand(config.ControlDir, Command{ID: "after-argv", Kind: SetState, State: Working}); err != nil {
				t.Fatal(err)
			}
			waitFixture(t, func() bool { _, err := ReadResult(config.ControlDir, "after-argv"); return err == nil })
			raw, err := os.ReadFile(filepath.Join(config.ControlDir, "ready.json"))
			if err != nil || json.Unmarshal(raw, &ready) != nil || !reflect.DeepEqual(ready.Argv, want) {
				t.Fatal("readiness aliased later caller argv", err)
			}
		})
	}
}

func TestFixtureStateCancellationAndGenerationFence(t *testing.T) {
	for _, scenario := range []string{"cancelled", "stale-generation"} {
		t.Run(scenario, func(t *testing.T) {
			config := fixtureConfig(t, "claude")
			generation := procutil.Identity(strconv.Itoa(os.Getpid()))
			if err := atomicJSON(filepath.Join(config.ControlDir, "ready.json"), Ready{PID: os.Getpid(), RunID: config.RunID, Generation: generation}, false); err != nil {
				t.Fatal(err)
			}
			if err := WriteCommand(config.ControlDir, Command{ID: "state-fenced", Kind: SetState, State: Working}); err != nil {
				t.Fatal(err)
			}
			if scenario == "cancelled" {
				if err := CancelCommand(config.ControlDir, "state-fenced"); err != nil {
					t.Fatal(err)
				}
			} else {
				path := filepath.Join(config.ControlDir, "commands", "state-fenced.json")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var command map[string]any
				if err := json.Unmarshal(raw, &command); err != nil {
					t.Fatal(err)
				}
				command["generation"] = generation + "-stale"
				if err := atomicJSON(path, command, false); err != nil {
					t.Fatal(err)
				}
			}
			input, writer := io.Pipe()
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- Run(ctx, config, input, io.Discard) }()
			t.Cleanup(func() {
				cancel()
				_ = writer.Close()
				if err := <-done; err != nil {
					t.Error(err)
				}
			})
			var result Result
			waitFixture(t, func() bool {
				var err error
				result, err = ReadResult(config.ControlDir, "state-fenced")
				return err == nil
			})
			want := "incarnation mismatch"
			if scenario == "cancelled" {
				want = "canceled"
			}
			if result.ExitCode == 0 || !strings.Contains(result.Error, want) {
				t.Fatalf("result=%+v want %s refusal", result, want)
			}
			raw, err := os.ReadFile(filepath.Join(config.ControlDir, "state.json"))
			var state AgentState
			if err != nil || json.Unmarshal(raw, &state) != nil || state.Phase != Idle {
				t.Fatalf("refused command changed state: %+v %v", state, err)
			}
		})
	}
}

func TestSYNTHETICClaudeLimitHasNoAutomaticSubmission(t *testing.T) {
	const phase Phase = "synthetic-claude-limit"
	paint := Paint(AgentState{Agent: "claude", Phase: phase}, 120, 32)
	if !strings.Contains(paint, "You've hit your limit · resets 4pm (Asia/Kuala_Lumpur)") || !strings.Contains(paint, "/upgrade to increase your usage limit.") || strings.Contains(paint, "❯") || strings.Contains(paint, "Working") {
		t.Fatal("SYNTHETIC limit screen must contain only sourced limit text and no composer/working hint")
	}
	before := AgentState{Agent: "claude", Phase: phase}
	after, accepted, err := ReduceAgent(before, []byte("unsubmitted input\r\x03\r"))
	if err != nil || len(accepted) != 0 || !reflect.DeepEqual(before, after) {
		t.Fatalf("synthetic limit input changed state=%+v accepted=%v err=%v", after, accepted, err)
	}
}
