package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func TestFleetPressUsesNegotiatedKeyboardAndOwnershipGate(t *testing.T) {
	for _, flags := range []string{"0", "3"} {
		t.Run(flags, func(t *testing.T) {
			out := ttyio.NewFake()
			ep, err := terminal.NewEndpoint("fleet-control", terminal.Geometry{Rows: 24, Cols: 80}, out)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { ep.Close() })
			if _, err := ep.Feed([]byte("\x1b[>"+flags+"u"), time.Now()); err != nil {
				t.Fatal(err)
			}
			f := &Fleet{ctx: t.Context(), console: &terminalProcess{endpoint: ep}}
			if err := f.Press("\x00", 32, 5); err != nil {
				t.Fatal(err)
			}
			if err := ep.Flush(t.Context()); err != nil {
				t.Fatal(err)
			}
			want := "\x00"
			if flags == "3" {
				want = "\x1b[32;5:1u"
			}
			if string(out.Bytes()) != want || f.active != 0 {
				t.Fatalf("key=%q want=%q active=%d", out.Bytes(), want, f.active)
			}
			f.closing = true
			if err := f.Press("x", 120, 1); err == nil {
				t.Fatal("closed fleet accepted key")
			}
			if string(out.Bytes()) != want {
				t.Fatal("closed fleet wrote input")
			}
		})
	}
	uncertain := &Fleet{ctx: t.Context(), model: ReduceRun(OwnedRun{}, RunEvent{Kind: "unknown", Error: "uncertain"})}
	if err := uncertain.Press("x", 120, 1); !errors.Is(err, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("uncertain key=%v", err)
	}
	if err := (&Fleet{ctx: t.Context()}).Press("x", 120, 1); err == nil {
		t.Fatal("missing console accepted key")
	}
}

func TestFleetSetStateUsesExistingActorOwnerAndRetiresRequest(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			f, control := startActorCLIFixture(t, agent)
			before, err := f.Agent(0)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
			defer cancel()
			for _, phase := range []agentfixture.Phase{agentfixture.Working, agentfixture.Picker, agentfixture.Idle} {
				if err := f.SetState(ctx, 0, phase); err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(filepath.Join(control, "state.json"))
				var state agentfixture.AgentState
				if err != nil || json.Unmarshal(raw, &state) != nil || state.Agent != agent || state.Phase != phase {
					t.Fatalf("state=%+v err=%v", state, err)
				}
			}
			if err := f.SetState(ctx, 0, agentfixture.SyntheticClaudeLimit); (err == nil) != (agent == "claude") {
				t.Fatalf("synthetic Claude screen owner=%s: %v", agent, err)
			}
			if err := f.SetState(ctx, 0, agentfixture.Idle); err != nil {
				t.Fatal(err)
			}
			after, err := f.Agent(0)
			if err != nil || before.PID != after.PID || before.Generation != after.Generation || before.NativeID != after.NativeID || f.active != 0 {
				t.Fatalf("state changed actor ownership: before=%+v after=%+v err=%v", before, after, err)
			}
			for _, sub := range []string{"commands", "results", "cancellations"} {
				entries, err := os.ReadDir(filepath.Join(control, sub))
				if err != nil || len(entries) != 0 {
					t.Fatalf("state request retained %s: %v %v", sub, entries, err)
				}
			}
			if err := f.SetState(ctx, 0, "unknown"); err == nil {
				t.Fatal("state API invented new phase")
			}
			cancelled, stop := context.WithCancel(t.Context())
			stop()
			if err := f.SetState(cancelled, 0, agentfixture.Working); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled state=%v", err)
			}
			f.closing = true
			if err := f.SetState(t.Context(), 0, agentfixture.Working); err == nil {
				t.Fatal("closing fleet admitted state")
			}
		})
	}
}
