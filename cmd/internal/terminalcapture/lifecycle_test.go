package terminalcapture

import (
	"errors"
	"testing"
)

func TestLifecycleSequences(t *testing.T) {
	diskErr := errors.New("disk unavailable")
	type step struct {
		event                  lifecycleEvent
		phase                  Phase
		closeAdmission, notify bool
		wantErrors             []error
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"successful close and duplicates", []step{
			{lifecycleEvent{kind: workerCompleted}, Recording, false, false, nil},
			{lifecycleEvent{kind: closeRequested}, Draining, true, true, nil},
			{lifecycleEvent{kind: closeRequested}, Draining, false, false, nil},
			{lifecycleEvent{kind: workerCompleted}, Closed, false, true, nil},
			{lifecycleEvent{kind: workerCompleted}, Closed, false, false, nil},
			{lifecycleEvent{kind: closeRequested}, Closed, false, false, nil},
			{lifecycleEvent{kind: drainTimedOut}, Closed, false, false, nil},
		}},
		{"failure stops admission once", []step{
			{lifecycleEvent{kind: captureFailed, err: ErrQueueFull}, Failed, true, true, []error{ErrQueueFull}},
			{lifecycleEvent{kind: closeRequested}, Failed, false, false, []error{ErrQueueFull}},
			{lifecycleEvent{kind: captureFailed, err: ErrQueueFull}, Failed, false, false, []error{ErrQueueFull}},
			{lifecycleEvent{kind: captureFailed, err: diskErr}, Failed, false, true, []error{ErrQueueFull, diskErr}},
			{lifecycleEvent{kind: workerCompleted}, Failed, false, false, []error{ErrQueueFull, diskErr}},
		}},
		{"failure while draining", []step{
			{lifecycleEvent{kind: closeRequested}, Draining, true, true, nil},
			{lifecycleEvent{kind: captureFailed, err: diskErr}, Failed, false, true, []error{diskErr}},
			{lifecycleEvent{kind: workerCompleted}, Failed, false, false, []error{diskErr}},
		}},
		{"timeout then late worker completion", []step{
			{lifecycleEvent{kind: closeRequested}, Draining, true, true, nil},
			{lifecycleEvent{kind: drainTimedOut}, Failed, false, true, []error{ErrCloseTimeout}},
			{lifecycleEvent{kind: closeRequested}, Failed, false, false, []error{ErrCloseTimeout}},
			{lifecycleEvent{kind: drainTimedOut}, Failed, false, false, []error{ErrCloseTimeout}},
			{lifecycleEvent{kind: workerCompleted}, Failed, false, false, []error{ErrCloseTimeout}},
		}},
		{"failed drain also times out", []step{
			{lifecycleEvent{kind: captureFailed, err: ErrQueueFull}, Failed, true, true, []error{ErrQueueFull}},
			{lifecycleEvent{kind: drainTimedOut}, Failed, false, true, []error{ErrQueueFull, ErrCloseTimeout}},
			{lifecycleEvent{kind: workerCompleted}, Failed, false, false, []error{ErrQueueFull, ErrCloseTimeout}},
		}},
		{"invalid notifications ignored", []step{
			{lifecycleEvent{kind: captureFailed}, Recording, false, false, nil},
			{lifecycleEvent{kind: drainTimedOut}, Recording, false, false, nil},
			{lifecycleEvent{kind: lifecycleEventKind(255)}, Recording, false, false, nil},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := newLifecycle()
			if got := model.snapshot(); got.phase != Recording || got.err != nil {
				t.Fatalf("initial=%+v", got)
			}
			for i, step := range test.steps {
				before := model.snapshot()
				effects := model.transition(step.event)
				after := model.snapshot()
				if after.phase != step.phase || effects.closeAdmission != step.closeAdmission || effects.notify != step.notify {
					t.Fatalf("step%d snapshot=%+v effects=%+v", i, after, effects)
				}
				if len(step.wantErrors) == 0 && after.err != nil {
					t.Fatalf("step%d unexpected error %v", i, after.err)
				}
				for _, want := range step.wantErrors {
					if !errors.Is(after.err, want) {
						t.Fatalf("step%d missing error %v in %v", i, want, after.err)
					}
				}
				// Snapshot mutation cannot change the model's authoritative phase.
				before.phase = Disabled
				if model.snapshot().phase != step.phase {
					t.Fatal("snapshot aliases model")
				}
			}
		})
	}
}
