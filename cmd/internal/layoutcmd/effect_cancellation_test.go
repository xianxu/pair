package layoutcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

type cancelSidecarRuntime struct {
	*fakeRuntime
	cancel context.CancelFunc
}

// cancellingLayout owns the real stateful layout world, including its return
// record. Cancellation happens after an effect actually changes that world.
type cancellingLayout struct {
	*fullscreenWorld
	cancel  context.CancelFunc
	after   int
	effects []string
}

func (r *cancellingLayout) completed(name string, err error) error {
	r.effects = append(r.effects, name)
	if len(r.effects) == r.after {
		r.cancel()
	}
	return err
}
func (r *cancellingLayout) FullscreenStore() workbenchshortcut.FullscreenStore { return r }
func (r *cancellingLayout) Write(record string) error {
	return r.completed("save", r.fullscreenWorld.Write(record))
}
func (r *cancellingLayout) Clear() error { return r.completed("clear", r.fullscreenWorld.Clear()) }
func (r *cancellingLayout) RunZellijAction(args ...string) error {
	return r.completed(strings.Join(args, " "), r.fullscreenWorld.RunZellijAction(args...))
}
func (r *cancellingLayout) NudgeWrap() error {
	return r.completed("nudge", r.fullscreenWorld.NudgeWrap())
}

func TestLayoutCancellationRetainsEveryCompletedEffectPrefix(t *testing.T) {
	for _, mode := range []string{"normal", "focus", "fullscreen"} {
		newWorld := func() *cancellingLayout {
			w := newFullscreenWorld()
			if mode == "focus" {
				w.pane("4").IsFloating = true
				w.visible = true
				w.record = "2 swap=small-split order=3,4"
			} else if mode == "fullscreen" {
				on := true
				w.pane("4").IsFullscreen = &on
			}
			return &cancellingLayout{fullscreenWorld: w}
		}
		healthy := newWorld()
		if code := RunToggleFocusedContext(t.Context(), nil, healthy, io.Discard); code != 0 {
			t.Fatalf("%s healthy code=%d logs=%v", mode, code, healthy.logs)
		}
		if len(healthy.effects) == 0 {
			t.Fatalf("%s: no healthy effects", mode)
		}
		for n := 1; n <= len(healthy.effects); n++ {
			t.Run(fmt.Sprintf("%s/after-%d", mode, n), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				r := newWorld()
				r.cancel, r.after = cancel, n
				_ = RunToggleFocusedContext(ctx, nil, r, io.Discard)
				if !slices.Equal(r.effects, healthy.effects[:n]) {
					t.Fatalf("effects=%v want completed prefix=%v", r.effects, healthy.effects[:n])
				}
				if r.busy {
					t.Fatal("cancellation retained the layout lock")
				}
			})
		}
	}
}

func (r cancelSidecarRuntime) TerminalPaneIDs() ([]string, error) {
	r.cancel()
	return []string{"3"}, nil
}

func TestSidecarCancellationRefusesEveryTerminalEffect(t *testing.T) {
	for _, action := range []struct {
		name string
		run  func(context.Context, Runtime) error
	}{
		{"focus", FocusRightTerminalContext},
		{"tab-write", func(ctx context.Context, rt Runtime) error {
			return SwitchRightTerminalTabContext(ctx, rt, workbenchshortcut.ChordAltShiftRight)
		}},
	} {
		t.Run(action.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			rt := cancelSidecarRuntime{fakeRuntime: &fakeRuntime{}, cancel: cancel}
			err := action.run(ctx, rt)
			if !errors.Is(err, context.Canceled) || len(rt.ops) != 0 || rt.listCalls != 0 {
				t.Fatalf("canceled positive sidecar: error=%v effects=%v native reads=%d", err, rt.ops, rt.listCalls)
			}
		})
	}
}
