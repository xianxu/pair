package clipcmd

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type cancelFlashRuntime struct {
	*fakeRuntime
	cancel             context.CancelFunc
	duringReset        bool
	applyErr, resetErr error
}

func (r *cancelFlashRuntime) SetPaneColor(id, bg string) error {
	r.fakeRuntime.SetPaneColor(id, bg)
	if !r.duringReset && r.cancel != nil {
		r.cancel()
	}
	return r.applyErr
}
func (r *cancelFlashRuntime) ResetPaneColorAfter(id string, d time.Duration) error {
	r.fakeRuntime.ResetPaneColorAfter(id, d)
	if r.duringReset && r.cancel != nil {
		r.cancel()
	}
	return r.resetErr
}
func TestFlashCancellationStillOwesReset(t *testing.T) {
	for _, duringReset := range []bool{false, true} {
		t.Run(map[bool]string{false: "apply", true: "reset"}[duringReset], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt := &cancelFlashRuntime{fakeRuntime: newFake(), cancel: cancel, duringReset: duringReset}
			err := flashAdmitted(ctx, FlashPaneOptions{}, "7", rt)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("error=%v, want cancellation", err)
			}
			if len(rt.setColors) != 1 || len(rt.scheduledReset) != 1 {
				t.Fatalf("apply=%v reset=%v; issued color owes one reset", rt.setColors, rt.scheduledReset)
			}
			if err := pasteAdmitted(ctx, ClipboardToPaneOptions{DataDir: t.TempDir(), Tag: "t"}, pastePlan{}, "private selection", rt); !errors.Is(err, context.Canceled) {
				t.Errorf("paste error=%v", err)
			}
			if len(rt.wrote) != 0 || len(rt.focused) != 0 || len(rt.keys) != 0 {
				t.Fatalf("effects after canceled flash: writes=%v focus=%v keys=%v", rt.wrote, rt.focused, rt.keys)
			}
		})
	}
}

func TestFlashFailuresStillAttemptReset(t *testing.T) {
	applyFailure := errors.New("apply failed")
	resetFailure := errors.New("reset start failed")
	for _, tc := range []struct {
		name         string
		apply, reset error
	}{{"apply", applyFailure, nil}, {"reset", nil, resetFailure}, {"both", applyFailure, resetFailure}} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &cancelFlashRuntime{fakeRuntime: newFake(), applyErr: tc.apply, resetErr: tc.reset}
			err := flashAdmitted(context.Background(), FlashPaneOptions{}, "7", rt)
			for _, want := range []error{tc.apply, tc.reset} {
				if want != nil && !errors.Is(err, want) {
					t.Errorf("error=%v missing %v", err, want)
				}
			}
			if len(rt.setColors) != 1 || len(rt.scheduledReset) != 1 {
				t.Fatalf("apply=%v reset=%v", rt.setColors, rt.scheduledReset)
			}
		})
	}
}

func TestFlashCanceledBeforeApplyHasNoCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt := &cancelFlashRuntime{fakeRuntime: newFake()}
	if err := flashAdmitted(ctx, FlashPaneOptions{}, "7", rt); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if len(rt.setColors) != 0 || len(rt.scheduledReset) != 0 {
		t.Fatalf("apply=%v reset=%v", rt.setColors, rt.scheduledReset)
	}
}

func TestAutoquoteFlashFailurePreventsPaste(t *testing.T) {
	for _, stage := range []string{"apply", "reset"} {
		t.Run(stage, func(t *testing.T) {
			base := newFake()
			base.panes = agentFocusedPanes
			base.pasteText = "private selection"
			base.pasteOK = true
			rt := &cancelFlashRuntime{fakeRuntime: base}
			failure := errors.New("fixture " + stage + " failure")
			if stage == "apply" {
				rt.applyErr = failure
			} else {
				rt.resetErr = failure
			}
			var stderr bytes.Buffer
			status := RunCopyOnSelectOrchestrate(CopyOnSelectOptions{Clipboard: ClipboardToPaneOptions{DataDir: t.TempDir(), Tag: "t"}}, rt, &stderr)
			if status != 1 || len(base.notices) != 1 || !strings.Contains(stderr.String(), failure.Error()) {
				t.Fatalf("status=%d notices=%v diagnostic=%q", status, base.notices, stderr.String())
			}
			if len(base.scheduledReset) != 1 {
				t.Fatalf("reset=%v", base.scheduledReset)
			}
			if len(base.wrote) != 0 || len(base.focused) != 0 || len(base.keys) != 0 {
				t.Fatalf("paste after failure: writes=%v focus=%v keys=%v", base.wrote, base.focused, base.keys)
			}
		})
	}
}

func TestColorRuntimeReportsCommandAndResetStartFailures(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	rt := OSRuntime{}
	if err := rt.SetPaneColor("7", "#ffffff"); err == nil {
		t.Fatal("missing color command silently accepted")
	}
	if err := rt.ResetPaneColorAfter("7", time.Millisecond); err == nil {
		t.Fatal("reset start failure silently accepted")
	}
	if err := startDetached(exec.Command("/nonexistent-pair-fixture-command")); err == nil {
		t.Fatal("detached start failure silently accepted")
	}
}

type cancelPasteRuntime struct {
	*fakeRuntime
	cancel context.CancelFunc
	after  string
}

func (r *cancelPasteRuntime) WriteFile(path, text string) error {
	err := r.fakeRuntime.WriteFile(path, text)
	if r.after == "stage" {
		r.cancel()
	}
	return err
}
func (r *cancelPasteRuntime) FocusPaneContext(ctx context.Context, id string) error {
	err := r.fakeRuntime.FocusPaneContext(ctx, id)
	if r.after == "focus" {
		r.cancel()
	}
	return err
}
func TestPasteCancellationStopsNextEffect(t *testing.T) {
	for _, after := range []string{"before", "stage", "focus"} {
		t.Run(after, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rt := &cancelPasteRuntime{fakeRuntime: newFake(), cancel: cancel, after: after}
			if after == "before" {
				cancel()
			}
			err := pasteAdmitted(ctx, ClipboardToPaneOptions{DataDir: t.TempDir(), Tag: "t"}, pastePlan{}, "private selection", rt)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error=%v", err)
			}
			wantWrites, wantFocus := 1, 0
			if after == "before" {
				wantWrites = 0
			}
			if after == "focus" {
				wantFocus = 1
			}
			if len(rt.wrote) != wantWrites || len(rt.focused) != wantFocus || len(rt.keys) != 0 {
				t.Fatalf("writes=%v focus=%v keys=%v", rt.wrote, rt.focused, rt.keys)
			}
		})
	}
}
