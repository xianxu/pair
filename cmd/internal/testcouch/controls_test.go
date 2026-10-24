package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
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

func TestFleetConfigureWorkloadSharesStateOwnerAndRetiresObservation(t *testing.T) {
	f, control := startActorCLIFixture(t)
	before, err := f.Agent(0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()
	start, err := f.ConfigureWorkload(ctx, 0, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStart, ScrollbackLines: 2000})
	if err != nil || start.Status != agentfixture.WorkloadRunning || start.Started.IsZero() || start.ScrollbackLines != 2000 || start.IntervalMS != 100 {
		t.Fatalf("start=%+v err=%v", start, err)
	}
	if _, err := f.ConfigureWorkload(ctx, 0, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStart}); err == nil {
		t.Fatal("accepted duplicate start")
	}
	if err := f.SetState(ctx, 0, agentfixture.Working); err != nil {
		t.Fatal(err)
	}
	stop, err := f.ConfigureWorkload(ctx, 0, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStop})
	if err != nil || stop.Status != agentfixture.WorkloadStopped || stop.Started != start.Started || stop.Finished.Before(stop.Started) || stop.Bytes < start.Bytes || stop.Bytes == 0 {
		t.Fatalf("stop=%+v err=%v", stop, err)
	}
	if start.Status != agentfixture.WorkloadRunning || !start.Finished.IsZero() {
		t.Fatal("returned start observation changed")
	}
	after, err := f.Agent(0)
	if err != nil || before.PID != after.PID || before.Generation != after.Generation || f.active != 0 {
		t.Fatalf("ownership before=%+v after=%+v err=%v active=%d", before, after, err, f.active)
	}
	for _, sub := range []string{"commands", "results", "cancellations"} {
		entries, err := os.ReadDir(filepath.Join(control, sub))
		if err != nil || len(entries) != 0 {
			t.Fatalf("retained %s: %v %v", sub, entries, err)
		}
	}
	for _, r := range []agentfixture.WorkloadRequest{{Action: "unknown"}, {Action: agentfixture.WorkloadStart, ScrollbackLines: 1}} {
		if _, err := f.ConfigureWorkload(ctx, 0, r); err == nil {
			t.Fatal("invalid workload admitted")
		}
	}
	cancelled, end := context.WithCancel(t.Context())
	end()
	if _, err := f.ConfigureWorkload(cancelled, 0, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStart}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled control: %v", err)
	}
	f.closing = true
	if _, err := f.ConfigureWorkload(ctx, 0, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStart}); err == nil {
		t.Fatal("closing owner admitted workload")
	}
}

// This mailbox peer intentionally withholds a terminal acknowledgment until
// cancellation is observed. It exercises both public clients through the same
// correlated owner without depending on scheduler races inside a live fixture.
func TestFleetStateClientsJoinCancellationAndRetainUncertain(t *testing.T) {
	for _, workload := range []bool{false, true} {
		for _, mode := range []string{"cancel", "deadline", "uncertain"} {
			t.Run(fmt.Sprintf("workload=%v/%s", workload, mode), func(t *testing.T) {
				root := t.TempDir()
				worktree := filepath.Join(root, "repo")
				base := filepath.Join(root, "control")
				control := agentfixture.ActorDirectory(base, worktree)
				for _, sub := range []string{"commands", "results", "cancellations"} {
					if err := os.MkdirAll(filepath.Join(control, sub), 0700); err != nil {
						t.Fatal(err)
					}
				}
				ready := agentfixture.Ready{PID: os.Getpid(), RunID: "client-test", Generation: procutil.Identity(fmt.Sprint(os.Getpid()))}
				raw, err := json.Marshal(ready)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(control, "ready.json"), raw, 0600); err != nil {
					t.Fatal(err)
				}
				f := &Fleet{ctx: t.Context(), RunID: ready.RunID, Roots: RootsPlan{Root: root, Mutable: map[string]string{"control": base}}, Workspaces: Workspaces{Primary: worktree}}
				caller, cancel := context.WithCancel(t.Context())
				defer cancel()
				if mode == "deadline" {
					var end context.CancelFunc
					caller, end = context.WithTimeout(t.Context(), 150*time.Millisecond)
					defer end()
				}
				done := make(chan error, 1)
				go func() {
					if workload {
						_, err := f.ConfigureWorkload(caller, 0, agentfixture.WorkloadRequest{Action: agentfixture.WorkloadStart})
						done <- err
					} else {
						done <- f.SetState(caller, 0, agentfixture.Working)
					}
				}()
				watch, end := context.WithTimeout(t.Context(), 3*time.Second)
				defer end()
				var command agentfixture.Command
				if err := waitUntil(watch, func() (bool, error) {
					entries, err := os.ReadDir(filepath.Join(control, "commands"))
					if err != nil {
						return false, err
					}
					if len(entries) == 0 {
						return false, nil
					}
					raw, err := os.ReadFile(filepath.Join(control, "commands", entries[0].Name()))
					if err != nil {
						return false, err
					}
					command, err = agentfixture.DecodeCommand(raw)
					return err == nil, err
				}); err != nil {
					t.Fatal(err)
				}
				if command.Kind != agentfixture.SetState || command.State != agentfixture.Working || command.PID != ready.PID || command.RunID != ready.RunID || command.Generation != ready.Generation || (command.Workload != nil) != workload {
					t.Fatalf("request=%+v", command)
				}
				if mode != "uncertain" {
					if mode == "cancel" {
						cancel()
					}
					if err := waitUntil(watch, func() (bool, error) {
						_, err := os.Stat(filepath.Join(control, "cancellations", command.ID+".json"))
						return err == nil, nil
					}); err != nil {
						t.Fatal(err)
					}
					select {
					case err := <-done:
						t.Fatalf("returned before terminal acknowledgment: %v", err)
					default:
					}
				}
				result := agentfixture.Result{ID: command.ID, Workload: &agentfixture.WorkloadObservation{Status: agentfixture.WorkloadStopped}}
				if mode == "uncertain" {
					result.CleanupUncertain = true
					result.Error = "retained producer"
				}
				raw, err = json.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(control, "results", command.ID+".json"), raw, 0600); err != nil {
					t.Fatal(err)
				}
				select {
				case err := <-done:
					want := context.Canceled
					if mode == "deadline" {
						want = context.DeadlineExceeded
					}
					if mode == "uncertain" {
						want = agentfixture.ErrCleanupUncertain
					}
					if !errors.Is(err, want) {
						t.Fatalf("outcome=%v want=%v", err, want)
					}
				case <-watch.Done():
					t.Fatal("client did not join terminal acknowledgment")
				}
				if f.active != 0 {
					t.Fatalf("retained active calls=%d", f.active)
				}
				for _, sub := range []string{"commands", "results", "cancellations"} {
					entries, err := os.ReadDir(filepath.Join(control, sub))
					if err != nil {
						t.Fatal(err)
					}
					if mode == "uncertain" {
						if sub != "cancellations" && len(entries) != 1 {
							t.Fatalf("lost uncertain %s", sub)
						}
					} else if len(entries) != 0 {
						t.Fatalf("retained %s: %v", sub, entries)
					}
				}
				if mode == "uncertain" {
					if err := f.SetState(t.Context(), 0, agentfixture.Idle); !errors.Is(err, agentfixture.ErrCleanupUncertain) {
						t.Fatalf("uncertain owner reopened: %v", err)
					}
				}
			})
		}
	}
}
