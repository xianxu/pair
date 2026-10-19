package sessionwatch

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"testing"
	"time"
)

func TestReduceBoundFollowCommitsOnlyAcknowledgedWholeBatch(t *testing.T) {
	first := LifecycleRecord{TurnID: "first", TranscriptOffset: 10}
	last := LifecycleRecord{TurnID: "last", TranscriptOffset: 20}
	original := boundFollowState{phase: followReading, watermark: 5, committed: boundCheckpoint{scanner: sessioninventory.ScannerState{NativeID: "old"}}}
	batch := boundPrepared{checkpoint: boundCheckpoint{scanner: sessioninventory.ScannerState{NativeID: "next"}}, publications: []lifecyclePublication{{first, 11 << 8}, {last, 21 << 8}}, watermark: 30 << 8}
	staged := reduceBoundFollow(original, boundFollowEvent{prepared: &batch})
	if staged.phase != followPublishing || staged.committed.scanner.NativeID != "old" || staged.watermark != 5 {
		t.Fatalf("staging advanced authority: %+v", staged)
	}
	if got := reduceBoundFollow(staged, boundFollowEvent{ack: &last}); got.index != 0 || got.watermark != 5 {
		t.Fatal("out-of-order acknowledgment advanced")
	}
	prefix := reduceBoundFollow(staged, boundFollowEvent{ack: &first})
	if prefix.watermark != 11<<8 || prefix.committed.scanner.NativeID != "old" || prefix.index != 1 {
		t.Fatalf("prefix=%+v", prefix)
	}
	if got := reduceBoundFollow(prefix, boundFollowEvent{ack: &first}); got.index != 1 {
		t.Fatal("duplicate advanced")
	}
	for _, err := range []error{errors.New("failed"), errors.New("uncertain")} {
		stopped := reduceBoundFollow(prefix, boundFollowEvent{failure: err})
		if stopped.phase != followStopped || stopped.pending != nil || stopped.watermark != prefix.watermark || stopped.committed.scanner.NativeID != "old" {
			t.Fatalf("stopped=%+v", stopped)
		}
		if got := reduceBoundFollow(stopped, boundFollowEvent{ack: &last}); got.phase != followStopped || got.watermark != prefix.watermark {
			t.Fatal("late result resurrected watcher")
		}
	}
	done := reduceBoundFollow(prefix, boundFollowEvent{ack: &last})
	if done.phase != followReading || done.pending != nil || done.watermark != batch.watermark || done.committed.scanner.NativeID != "next" {
		t.Fatalf("completed=%+v", done)
	}
	empty := batch
	empty.publications = nil
	if got := reduceBoundFollow(original, boundFollowEvent{prepared: &empty}); got.phase != followReading || got.committed.scanner.NativeID != "next" {
		t.Fatal("validated batch without publications did not commit")
	}
}

func TestNextLifecycleDelayIsBoundedAndResetsOnProgress(t *testing.T) {
	for _, base := range []time.Duration{time.Nanosecond, 100 * time.Millisecond, 10 * time.Second} {
		previous := time.Duration(0)
		for i := 0; i < 100; i++ {
			next := nextLifecycleDelay(base, previous, false)
			if next <= 0 || next > 5*time.Second || next < previous {
				t.Fatalf("delay base=%v previous=%v next=%v", base, previous, next)
			}
			previous = next
		}
		if previous != 5*time.Second {
			t.Fatalf("not capped: %v", previous)
		}
		want := base
		if want > 5*time.Second {
			want = 5 * time.Second
		}
		if got := nextLifecycleDelay(base, previous, true); got != want {
			t.Fatalf("progress delay=%v want%v", got, want)
		}
	}
}

func TestPublishLifecycleBatchPreservesPrefixAndNeverRetriesUncertain(t *testing.T) {
	for _, tc := range []struct {
		name      string
		err       error
		wantPhase boundFollowPhase
		wantCalls int
		wantMark  uint64
	}{
		{"failed", errors.New("write failed"), followStopped, 2, 11 << 8},
		{"uncertain", commitoutcome.Wrap(commitoutcome.Indeterminate, errors.New("sync uncertain")), followStopped, 2, 11 << 8},
		{"reconciled committed", commitoutcome.Wrap(commitoutcome.Committed, errors.New("close failed")), followReading, 3, 40 << 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pubs := []lifecyclePublication{{LifecycleRecord{TurnID: "one", TranscriptOffset: 10}, 11 << 8}, {LifecycleRecord{TurnID: "two", TranscriptOffset: 20}, 21 << 8}, {LifecycleRecord{TurnID: "three", TranscriptOffset: 30}, 31 << 8}}
			s := reduceBoundFollow(boundFollowState{}, boundFollowEvent{prepared: &boundPrepared{publications: pubs, watermark: 40 << 8}})
			calls := 0
			s, err := publishLifecycleBatch(Options{AppendLifecycle: func(string, LifecycleRecord) error {
				calls++
				if calls == 2 {
					return tc.err
				}
				return nil
			}}, quietRuntime{}, "journal", s)
			if s.phase != tc.wantPhase || s.watermark != tc.wantMark || calls != tc.wantCalls {
				t.Fatalf("phase=%v watermark=%d calls=%d error=%v", s.phase, s.watermark, calls, err)
			}
			if tc.wantPhase == followStopped && err == nil {
				t.Fatal("uncertain/failure lost")
			}
			if tc.wantPhase == followReading && err != nil {
				t.Fatal("committed cleanup warning stopped publication")
			}
			if tc.wantPhase == followStopped {
				s, _ = publishLifecycleBatch(Options{AppendLifecycle: func(string, LifecycleRecord) error { t.Fatal("retried stopped publication"); return nil }}, quietRuntime{}, "journal", s)
			}
		})
	}
}
