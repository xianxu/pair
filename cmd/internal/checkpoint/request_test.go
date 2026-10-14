package checkpoint

import (
	"strings"
	"testing"
	"time"
)

func testRequest(t *testing.T) Request {
	t.Helper()
	cp, err := New("/tmp/checkpoint.md", "---\ntype: continuation\nagent: codex\n---\n## NEXT ACTION\ncontinue testing\n")
	if err != nil {
		t.Fatal(err)
	}
	r := Request{Version: Version, ID: RequestID("scope", "tag", 3, cp.Digest), Checkpoint: cp, Source: Source{Agent: "codex", Session: "pair-test", LaunchOrdinal: 3, Helper: Process{PID: 42, Identity: "helper"}}, CreatedAt: time.Now(), Phase: Pending}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return r
}

// Pending, Running, Complete at the target's registration (#522): the prompt
// is part of the launch, so there is nothing after registration to confirm.
func TestContinuationRequestTransitions(t *testing.T) {
	r := testRequest(t)
	running, err := Advance(r, Event{Kind: Begin, RequestID: r.ID, Attempt: "attempt-1"})
	if err != nil || running.Phase != Running {
		t.Fatalf("begin = %+v %v", running, err)
	}
	target := Process{PID: 43, Identity: "target"}
	if _, err := Advance(running, Event{Kind: Registered, At: time.Now(), RequestID: r.ID, Attempt: "wrong", Target: &target}); err == nil {
		t.Fatal("stale receipt accepted")
	}
	running, err = Advance(running, Event{Kind: SourceParked, RequestID: r.ID, Attempt: "attempt-1", ParkNonce: "park-source"})
	if err != nil || running.Phase != Running {
		t.Fatalf("park = %+v %v", running, err)
	}
	done, err := Advance(running, Event{Kind: Registered, At: time.Now(), RequestID: r.ID, Attempt: "attempt-1", Target: &target})
	if err != nil || done.Phase != Complete || done.Target == nil || done.Target.Process != target {
		t.Fatalf("registered = %+v %v", done, err)
	}
	if _, err := Advance(done, Event{Kind: Begin, RequestID: r.ID, Attempt: "attempt-2"}); err == nil {
		t.Fatal("completed request relaunched")
	}
	clone := done.Clone()
	clone.Target.Identity = "mutated"
	if done.Target.Identity != "target" {
		t.Fatal("clone aliased target")
	}
}

// A request that failed before #522 is read only: no event moves it, and the
// thread reads it as its one visible line.
func TestAFailedRequestTakesNoEvents(t *testing.T) {
	r := testRequest(t)
	r.Phase, r.Attempt, r.Failure = Failed, "attempt-1", "delivery uncertain"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []EventKind{Begin, Registered, SourceParked, RefreshSource} {
		target := Process{PID: 43, Identity: "target"}
		if _, err := Advance(r, Event{Kind: kind, RequestID: r.ID, Attempt: "attempt-1", Target: &target, ParkNonce: "p", Helper: target, At: time.Now()}); err == nil {
			t.Errorf("%s moved a failed request", kind)
		}
	}
	m := MissOf(r, r.Failure, r.CreatedAt, false)
	if err := m.Validate(); err != nil || m.Path != "/tmp/checkpoint.md" || m.RequestID != r.ID {
		t.Fatalf("miss %+v %v", m, err)
	}
}

func TestMissLineNamesTheContinuationAndOneAction(t *testing.T) {
	r := testRequest(t)
	m := MissOf(r, "launch failed:\n  exit 1", time.Now(), false)
	for _, c := range []struct {
		m       Miss
		running bool
		want    string
	}{
		{m, false, "Resume the slot"},
		{m, true, "Compact again"},
		{func() Miss { x := m; x.TargetMayExist = true; return x }(), true, "may be running. Relaunch the slot."},
	} {
		line := c.m.Line("pair:3", c.running)
		if !strings.HasPrefix(line, "pair:3: compaction didn't finish: launch failed: exit 1; continuation at /tmp/checkpoint.md") || !strings.Contains(line, c.want) || strings.Contains(line, "\n") {
			t.Errorf("line %q, want %q", line, c.want)
		}
	}
	long := MissOf(r, strings.Repeat("é", MaxFailureBytes), time.Now(), false)
	if err := long.Validate(); err != nil {
		t.Fatalf("a long reason is bounded, not refused: %v", err)
	}
	if err := (Miss{RequestID: r.ID, At: time.Now()}).Validate(); err == nil {
		t.Fatal("a miss with no reason validated")
	}
}
