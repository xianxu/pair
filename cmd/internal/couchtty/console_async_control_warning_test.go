package couchtty

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type controlWarningBrowser struct {
	*webcast.FakeDriven
	armed atomic.Bool
	err   error
}

func (b *controlWarningBrowser) WriteControl(ctx context.Context, ctl *recording.Ctl) error {
	if b.armed.Load() {
		return &capability.Failure{Outcome: capability.Retryable, Cause: b.err}
	}
	return b.FakeDriven.WriteControl(ctx, ctl)
}

func TestAsyncDriveFailureAttentionClassification(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		warn bool
	}{
		{"unexpected", errors.New("writer unavailable"), true},
		{"seat left", drive.ErrNoSeat, false},
		{"control changed", drive.ErrControlOp, false},
		{"unavailable", drive.ErrDriveUnavailable, false},
		{"closed", drive.ErrClosed, false},
		{"paused", drive.ErrNotLive, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var browser *controlWarningBrowser
			r := newDriveCast(t, func(d *webcast.FakeDriven) webcast.Browser {
				browser = &controlWarningBrowser{FakeDriven: d, err: fmt.Errorf("control write: %w", test.err)}
				return browser
			})
			who := r.viewer(t, "Ana")
			r.waitDrive(t, "request observed before action", func() bool { return strings.Contains(r.notice(), "Ana asks") })
			before := r.con.attentionCount()
			browser.armed.Store(true)
			r.con.startDriveOperation(r.cast(), recording.CtlDeny, recording.Author{ID: who, Name: "Ana"})
			r.waitDrive(t, "failed control completion", func() bool {
				r.con.mu.Lock()
				defer r.con.mu.Unlock()
				return r.con.drivePending == nil
			})
			want := before
			if test.warn {
				want++
			}
			if got := r.con.attentionCount(); got != want {
				t.Fatalf("attention = %d, want %d; feedback %q", got, want, r.notice())
			}
			if got, want := r.notice(), driveFailure(browser.err); got != want {
				t.Fatalf("failed action feedback = %q, want %q", got, want)
			}
		})
	}
}

type capabilityWarningBrowser struct {
	*webcast.FakeDriven
	err error
}

func (b *capabilityWarningBrowser) ClearAnnotations(context.Context, uint64) error {
	return &capability.Failure{Outcome: capability.Retryable, Cause: b.err}
}

func TestAsyncCapabilityFailureAttentionClassification(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		warn bool
	}{
		{"unrecovered", errors.New("recorder unavailable"), true},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"state moved", drive.ErrControlOp, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := newWebcastRig(t, func(r *webcastRig) {
				r.wrap = func(f *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
					d, err := webcast.NewFakeDriven(f, drive.DrivenPage(100), &recording.MemSink{})
					if err != nil {
						t.Fatal(err)
					}
					return &capabilityWarningBrowser{FakeDriven: d, err: fmt.Errorf("clear annotations: %w", test.err)}
				}
			})
			r.open(t)
			r.con.mu.Lock()
			cast := r.con.wcast.cast
			r.con.mu.Unlock()
			if err := cast.SetCapability(context.Background(), webcast.CapabilityAnnotation, true); err != nil {
				t.Fatal(err)
			}
			before := r.con.attentionCount()
			awaitDrive(t, "failed capability completion", r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, false))
			want := before
			if test.warn {
				want++
			}
			if got := r.con.attentionCount(); got != want {
				t.Fatalf("attention = %d, want %d; feedback %q", got, want, r.notice())
			}
			if !strings.Contains(r.notice(), "clear annotations") {
				t.Fatalf("failed capability lost its feedback: %q", r.notice())
			}
		})
	}
}

func TestPointerVisibilityWatchDoesNotRaiseAttention(t *testing.T) {
	f, _, _ := pointerOnFixture(t, nil)
	before := f.con.attentionCount()
	f.con.pointerOffByWatch()
	if f.pointerPhase() != pointerOff || !strings.Contains(f.notice(), "wasn't visible") {
		t.Fatalf("watch did not finish: phase %v notice %q", f.pointerPhase(), f.notice())
	}
	if got := f.con.attentionCount(); got != before {
		t.Fatalf("routine safety shutdown attention = %d, want %d", got, before)
	}
}
