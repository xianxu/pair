package checkpoint

import (
	"testing"
	"time"
)

func TestContinuationRequestTransitions(t *testing.T) {
	cp, err := New("/tmp/checkpoint.md", "---\ntype: continuation\nagent: codex\n---\n## NEXT ACTION\ncontinue testing\n")
	if err != nil {
		t.Fatal(err)
	}
	r := Request{Version: Version, ID: RequestID("scope", "tag", 3, cp.Digest), Checkpoint: cp, Source: Source{Agent: "codex", Session: "pair-test", LaunchOrdinal: 3, Helper: Process{PID: 42, Identity: "helper"}}, CreatedAt: time.Now(), Phase: Pending}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	running, err := Advance(r, Event{Kind: Begin, RequestID: r.ID, Attempt: "attempt-1"})
	if err != nil || running.Phase != Running {
		t.Fatalf("begin = %+v %v", running, err)
	}
	if _, err := Advance(running, Event{Kind: Submitted, RequestID: r.ID, Attempt: "wrong"}); err == nil {
		t.Fatal("stale receipt accepted")
	}
	running, err = Advance(running, Event{Kind: SourceParked, RequestID: r.ID, Attempt: "attempt-1", ParkNonce: "park-source"})
	if err != nil {
		t.Fatal(err)
	}
	target := Process{PID: 43, Identity: "target"}
	running, err = Advance(running, Event{Kind: Registered, At: time.Now(), RequestID: r.ID, Attempt: "attempt-1", Target: &target})
	if err != nil {
		t.Fatal(err)
	}
	done, err := Advance(running, Event{Kind: Submitted, RequestID: r.ID, Attempt: "attempt-1"})
	if err != nil || done.Phase != Complete {
		t.Fatalf("submit = %+v %v", done, err)
	}
	if _, err := Advance(done, Event{Kind: Begin, RequestID: r.ID, Attempt: "attempt-2"}); err == nil {
		t.Fatal("completed request relaunched")
	}
	failed, err := Advance(running, Event{Kind: Fail, RequestID: r.ID, Attempt: "attempt-1", Failure: "delivery uncertain"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Advance(failed, Event{Kind: Begin, RequestID: r.ID, Attempt: "attempt-2"}); err == nil {
		t.Fatal("automatic failed retry accepted")
	}
	retry, err := Advance(failed, Event{Kind: RetryAbsent, RequestID: r.ID, Attempt: "attempt-2"})
	if err != nil || retry.Target != nil || retry.Attempt != "attempt-2" {
		t.Fatalf("retry = %+v %v", retry, err)
	}
	clone := running.Clone()
	clone.Target.Identity = "mutated"
	if running.Target.Identity != "target" {
		t.Fatal("clone aliased target")
	}
}
