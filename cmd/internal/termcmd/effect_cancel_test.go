package termcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

type effectCancelRuntime struct {
	*fakeRuntime
	cancel context.CancelFunc
	after  int
	events []string
}

func (r *effectCancelRuntime) committed(event string) {
	r.events = append(r.events, event)
	if len(r.events) == r.after {
		r.cancel()
	}
}
func (r *effectCancelRuntime) RecordLastLeftPaneID(id string) error {
	r.committed("left")
	return r.fakeRuntime.RecordLastLeftPaneID(id)
}
func (r *effectCancelRuntime) RecordLastTerminalPaneID(id string) error {
	r.committed("terminal")
	return r.fakeRuntime.RecordLastTerminalPaneID(id)
}
func (r *effectCancelRuntime) RunZellijAction(args ...string) error {
	r.committed("focus")
	return r.fakeRuntime.RunZellijAction(args...)
}
func (r *effectCancelRuntime) RunZellijActionQuiet(args ...string) error {
	r.committed("split")
	return r.fakeRuntime.RunZellijActionQuiet(args...)
}

func TestTerminalEffectSequenceStopsAfterCancellation(t *testing.T) {
	for _, after := range []int{0, 1, 2, 3, -1} {
		t.Run(fmt.Sprint(after), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt := &effectCancelRuntime{fakeRuntime: &fakeRuntime{currentPaneID: "4"}, cancel: cancel, after: after}
			if after == 0 {
				cancel()
			}
			decision := workbenchshortcut.ShortcutDecision{Disposition: workbenchshortcut.DispositionHandle, Action: workbenchshortcut.ActionFocusPane, TargetPaneID: "2", RecordLastLeftPaneID: "1", RecordLastTerminalPaneID: "4"}
			err := runDecisionContext(ctx, decision, workbenchPanes{}, rt, strings.NewReader(""), io.Discard)
			want := after
			if want < 0 {
				want = 3
			}
			if len(rt.events) != want {
				t.Fatalf("committed=%v, want %d", rt.events, want)
			}
			if after >= 0 && after < 3 {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation=%v", err)
				}
			} else if err != nil {
				t.Fatalf("issued final result changed: %v", err)
			}
		})
	}
}

func TestTerminalSplitEffectUsesExistingCancellationToken(t *testing.T) {
	for _, before := range []bool{false, true} {
		t.Run(fmt.Sprint(before), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt := &effectCancelRuntime{fakeRuntime: &fakeRuntime{currentPaneID: "4"}, cancel: cancel, after: 1}
			if before {
				cancel()
			}
			err := splitTerminalDownAdmitted(ctx, rt)
			if before {
				if !errors.Is(err, context.Canceled) || len(rt.events) != 0 {
					t.Fatalf("canceled split issued: events=%v error=%v", rt.events, err)
				}
			} else if err != nil || len(rt.events) != 1 {
				t.Fatalf("issued split result: events=%v error=%v", rt.events, err)
			}
		})
	}
}
