package agentfixture

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type heldWorkloadClock struct {
	mu     sync.Mutex
	now    time.Time
	ticker *heldWorkloadTicker
}
type heldWorkloadTicker struct {
	ch      chan time.Time
	stopped chan struct{}
	once    sync.Once
}

func (c *heldWorkloadClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *heldWorkloadClock) NewTicker(d time.Duration) WorkloadTicker {
	if d != WorkloadInterval {
		panic("interval")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ticker = &heldWorkloadTicker{ch: make(chan time.Time, 4), stopped: make(chan struct{})}
	return c.ticker
}
func (t *heldWorkloadTicker) C() <-chan time.Time { return t.ch }
func (t *heldWorkloadTicker) Stop()               { t.once.Do(func() { close(t.stopped) }) }
func (c *heldWorkloadClock) advance(t time.Time) {
	c.mu.Lock()
	c.now = t
	tick := c.ticker
	c.mu.Unlock()
	tick.ch <- t
}
func TestRunWorkloadControlledClock(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			c := fixtureConfig(t, agent)
			clock := &heldWorkloadClock{now: time.Unix(1000, 0)}
			c.WorkloadClock = clock
			input, writer := io.Pipe()
			defer writer.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := &lockedOutput{}
			done := make(chan error, 1)
			go func() { done <- Run(ctx, c, input, out) }()
			waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(c.ControlDir, "ready.json")); return e == nil })
			request := func(id string, r *WorkloadRequest, phase Phase) Result {
				t.Helper()
				if err := WriteCommand(c.ControlDir, Command{ID: id, Kind: SetState, State: phase, Workload: r}); err != nil {
					t.Fatal(err)
				}
				var result Result
				waitFixture(t, func() bool { var e error; result, e = ReadResult(c.ControlDir, id); return e == nil })
				return result
			}
			start := request("start", &WorkloadRequest{Action: WorkloadStart, ScrollbackLines: 2000}, Working)
			if start.Workload == nil || start.Workload.Status != WorkloadRunning || start.Workload.Bytes == 0 {
				t.Fatal(start)
			}
			if strings.Count(out.String(), "fixture scrollback ") != 2000 || start.Workload.Bytes > 256<<10 {
				t.Fatal("actual prefill bound")
			}

			before := out.String()
			clock.advance(clock.Now().Add(WorkloadInterval / 2))
			// An early/duplicate signal cannot disarm the live timer.
			if err := WriteCommand(c.ControlDir, Command{ID: "early-barrier", Kind: SetState, State: Working}); err != nil {
				t.Fatal(err)
			}
			waitFixture(t, func() bool { _, e := ReadResult(c.ControlDir, "early-barrier"); return e == nil })
			select {
			case <-clock.ticker.stopped:
				t.Fatal("early signal disarmed live timer")
			default:
			}
			before = out.String()
			clock.advance(clock.Now().Add(WorkloadInterval / 2))
			waitFixture(t, func() bool { return len(out.String()) > len(before) })
			before = out.String()
			duplicate := request("duplicate", &WorkloadRequest{Action: WorkloadStart}, Working)
			if duplicate.Error == "" || duplicate.Workload == nil || duplicate.Workload.Ticks != 1 || duplicate.Workload.Status != WorkloadRunning || out.String() != before {
				t.Fatal("replacement altered workload", duplicate)
			}
			stop := request("stop", &WorkloadRequest{Action: WorkloadStop}, Working)
			if stop.Workload == nil || stop.Workload.Ticks != 1 || stop.Workload.Status != WorkloadStopped {
				t.Fatal(stop)
			}
			clock.mu.Lock()
			timer := clock.ticker
			clock.mu.Unlock()
			select {
			case <-timer.stopped:
			default:
				t.Fatal("timer not stopped before receipt")
			}
			before = out.String()
			clock.advance(clock.Now().Add(WorkloadInterval))
			request("barrier", nil, Working)
			// Ordinary SetState repaints, but the stopped clock may never add a patch.
			if got := strings.TrimPrefix(out.String(), before); got != Paint(AgentState{Agent: agent, Phase: Working}, 120, 38) {
				t.Fatal("post-stop output", got)
			}
			if stop.Workload.Bytes != start.Workload.Bytes+uint64(len(WorkloadPatch(0))) {
				t.Fatal("unaccounted writes", start, stop)
			}
			cancel()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

type heldPatchOutput struct {
	lockedOutput
	entered chan struct{}
	release chan struct{}
	fail    bool
}

func (w *heldPatchOutput) Write(p []byte) (int, error) {
	if string(p) == WorkloadPatch(0) {
		close(w.entered)
		<-w.release
		if w.fail {
			return len(p) / 2, io.ErrUnexpectedEOF
		}
	}
	return w.lockedOutput.Write(p)
}
func TestRunWorkloadJoinsHeldOutput(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			c := fixtureConfig(t, "claude")
			clock := &heldWorkloadClock{now: time.Unix(1000, 0)}
			c.WorkloadClock = clock
			in, writer := io.Pipe()
			defer writer.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := &heldPatchOutput{entered: make(chan struct{}), release: make(chan struct{}), fail: fail}
			done := make(chan error, 1)
			go func() { done <- Run(ctx, c, in, out) }()
			waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(c.ControlDir, "ready.json")); return e == nil })
			if err := WriteCommand(c.ControlDir, Command{ID: "start", Kind: SetState, State: Working, Workload: &WorkloadRequest{Action: WorkloadStart}}); err != nil {
				t.Fatal(err)
			}
			waitFixture(t, func() bool { _, e := ReadResult(c.ControlDir, "start"); return e == nil })
			clock.advance(clock.Now().Add(WorkloadInterval))
			<-out.entered
			cancel()
			select {
			case err := <-done:
				t.Fatal("abandoned in-flight output", err)
			default:
			}
			close(out.release)
			err := <-done
			if fail && !errors.Is(err, io.ErrUnexpectedEOF) || !fail && err != nil {
				t.Fatal(err)
			}
			select {
			case <-clock.ticker.stopped:
			default:
				t.Fatal("timer survived Run")
			}
		})
	}
}
func TestRunWorkloadTerminalPaths(t *testing.T) {
	for _, ending := range []string{"eof", "stop", "cancel", "expiry", "phase"} {
		t.Run(ending, func(t *testing.T) {
			c := fixtureConfig(t, "codex")
			clock := &heldWorkloadClock{now: time.Unix(1000, 0)}
			c.WorkloadClock = clock
			in, writer := io.Pipe()
			defer writer.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := &lockedOutput{}
			done := make(chan error, 1)
			go func() { done <- Run(ctx, c, in, out) }()
			waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(c.ControlDir, "ready.json")); return e == nil })
			if err := WriteCommand(c.ControlDir, Command{ID: "start", Kind: SetState, State: Working, Workload: &WorkloadRequest{Action: WorkloadStart}}); err != nil {
				t.Fatal(err)
			}
			waitFixture(t, func() bool { _, e := ReadResult(c.ControlDir, "start"); return e == nil })
			switch ending {
			case "eof":
				writer.Close()
			case "stop":
				if err := WriteCommand(c.ControlDir, Command{ID: "stop", Kind: Stop}); err != nil {
					t.Fatal(err)
				}
			case "cancel":
				cancel()
			case "phase":
				if err := WriteCommand(c.ControlDir, Command{ID: "idle", Kind: SetState, State: Idle}); err != nil {
					t.Fatal(err)
				}
			case "expiry":
				clock.advance(clock.Now().Add(WorkloadLifetime))
			}
			waitFixture(t, func() bool {
				select {
				case <-clock.ticker.stopped:
					return true
				default:
					return false
				}
			})
			if ending == "phase" || ending == "expiry" {
				before := out.String()
				clock.advance(clock.Now().Add(WorkloadInterval))
				// A synchronous receipt is a Run-loop barrier after the stale timer delivery.
				if err := WriteCommand(c.ControlDir, Command{ID: "barrier", Kind: SetState, State: Working, Workload: &WorkloadRequest{Action: WorkloadStop}}); err != nil {
					t.Fatal(err)
				}
				var r Result
				waitFixture(t, func() bool { var e error; r, e = ReadResult(c.ControlDir, "barrier"); return e == nil })
				want := WorkloadStopped
				if ending == "expiry" {
					want = WorkloadExpired
				}
				if r.Workload == nil || r.Workload.Status != want || r.Workload.Ticks != 0 {
					t.Fatal(r)
				}
				if strings.Contains(out.String()[len(before):], WorkloadPatch(0)) {
					t.Fatal("post-terminal patch")
				}
				cancel()
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}
