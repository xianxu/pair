package couchcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestScheduleServiceRoutesManualOwnerOperations(t *testing.T) {
	s := newServiceRig(t).s
	request := couchmessage.Request{Op: "schedule-run", Manual: &couchschedule.ManualRequest{Intent: couchschedule.ManualIntent{ID: "manual-id", Hub: "/hub", Task: "task"}}}
	response := s.handle(t.Context(), request)
	if (response.Code != "unavailable" && response.Code != "refused") || !strings.Contains(response.Error, "scheduler") {
		t.Fatalf("manual service operation not routed: %+v", response)
	}
}

func TestScheduleOwnerRemoteManualRunsThroughOneWriter(t *testing.T) {
	runner, fake, root := manualRunnerFixture(t)
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("remote output")})
	c := newScheduleCoordinator(runner.namespace, []string{root}, time.UTC, time.Now)
	o := newScheduleOwner(c)
	o.configure(scheduleExecutionWorker{namespace: runner.namespace, permitsDir: runner.permitsDir, executor: fake, now: time.Now}, offlineScheduleNotices{})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); o.run(ctx, nativeScheduleClock{}) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("owner did not join")
		}
	}()
	testwait.Until(t, "owner ready", func() bool { return c.view().Error == "" })
	s := &messageService{}
	s.scheduler.Store(o)
	s.schedule.Store(c)
	call := func(ctx context.Context, _ string, request, response any) error {
		*response.(*couchmessage.Response) = s.handle(ctx, request.(couchmessage.Request))
		return nil
	}
	var output, errorsOut bytes.Buffer
	if exit := runScheduleRemote(ctx, &scheduleArgs{verb: "run", task: "report", args: []string{"--help", ""}}, root, "private-test-socket", &output, &errorsOut, call); exit != 10 {
		t.Fatalf("manual exit=%d output=%q errors=%q", exit, output.String(), errorsOut.String())
	}
	if output.String() != "remote output" {
		t.Fatalf("output=%q", output.String())
	}
	runs := fake.Runs()
	if len(runs) != 1 || !runs[0].Authorized || !runs[0].Finished {
		t.Fatalf("native effects=%+v", runs)
	}
	ledger, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
	if state.Active != nil || state.Last == nil || state.Last.ManualIntentDigest == "" || state.Last.Source != "manual" {
		t.Fatalf("owner ledger=%+v", state)
	}
}
