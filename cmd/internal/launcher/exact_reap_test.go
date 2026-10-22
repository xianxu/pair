package launcher

import (
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"
)

type exactWorld struct {
	snapshots int
	rows      []ProcessRow
	signals   []ReapStep
	sigs      []syscall.Signal
	keep      bool
	err       error
}

func (w *exactWorld) Snapshot(context.Context) ([]ProcessRow, error) {
	w.snapshots++
	return append([]ProcessRow(nil), w.rows...), w.err
}
func (w *exactWorld) Identity(int) string              { panic("exact reap must use batch snapshots") }
func (w *exactWorld) Command(int) ([]string, bool)     { panic("no command discovery") }
func (w *exactWorld) Signal(int, syscall.Signal) error { panic("signal must use guarded callback") }
func (w *exactWorld) signal(_ context.Context, s ReapStep, sig syscall.Signal) error {
	w.signals = append(w.signals, s)
	w.sigs = append(w.sigs, sig)
	if !w.keep || sig == syscall.SIGKILL {
		for i, r := range w.rows {
			if r.PID == s.PID && r.Identity == s.Identity {
				w.rows = append(w.rows[:i], w.rows[i+1:]...)
				break
			}
		}
	}
	return nil
}
func exactRunner(w *exactWorld) Reaper {
	return Reaper{Table: w, TermWait: time.Millisecond, KillWait: time.Millisecond, Poll: time.Millisecond}
}
func TestExactReapNeverExpandsDescendants(t *testing.T) {
	w := &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "old"}, {PID: 11, PPID: 10, Identity: "live-child"}}}
	got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}, {PID: 10, Identity: "old"}}, w.signal)
	if len(got) != 1 || got[0].Status != "reaped" || len(w.signals) != 1 || w.signals[0].PID != 10 || len(w.rows) != 1 || w.rows[0].PID != 11 {
		t.Fatalf("outcomes=%+v world=%+v", got, w)
	}
}
func TestExactReapEscalationAndUncertainty(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{{"refused", ErrExactReapRefused, "refused"}, {"changed", ErrServerChanged, "skipped"}, {"lost", errors.New("reply lost"), "failed"}} {
		t.Run(tc.name, func(t *testing.T) {
			w := &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "old"}}, keep: true}
			calls := 0
			got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}}, func(context.Context, ReapStep, syscall.Signal) error { calls++; return tc.err })
			if len(got) != 1 || got[0].Status != tc.want || calls != 1 {
				t.Fatalf("%+v calls%d", got, calls)
			}
		})
	}
	w := &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "old"}}, keep: true}
	got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}}, w.signal)
	if got[0].Status != "reaped" || len(w.sigs) != 2 || w.sigs[1] != syscall.SIGKILL {
		t.Fatalf("%+v signals%v", got, w.sigs)
	}
}

func TestExactReapUnknownProbeNeverEscalates(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows []ProcessRow
		err  error
	}{
		{name: "unreadable", err: errors.New("census failed")},
		{name: "empty identity", rows: []ProcessRow{{PID: 10}}},
		{name: "duplicate pid", rows: []ProcessRow{{PID: 10, Identity: "old"}, {PID: 10, Identity: "old"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &exactWorld{rows: tc.rows, err: tc.err}
			calls := 0
			got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}}, func(context.Context, ReapStep, syscall.Signal) error { calls++; return nil })
			if got[0].Status != "failed" || calls != 1 {
				t.Fatalf("%+v calls%d", got, calls)
			}
		})
	}
}
func TestExactReapCancellationAndIdentityChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "old"}}}
	calls := 0
	got := exactRunner(w).ReapExact(ctx, []ReapStep{{PID: 10, Identity: "old"}, {PID: 11, Identity: "other"}}, func(context.Context, ReapStep, syscall.Signal) error { calls++; cancel(); return nil })
	if calls != 1 || len(got) != 2 || got[0].Status != "failed" || got[1].Status != "failed" {
		t.Fatalf("%+v calls%d", got, calls)
	}
	w = &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "replacement"}}}
	calls = 0
	got = exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}}, func(context.Context, ReapStep, syscall.Signal) error { calls++; return nil })
	if calls != 1 || got[0].Status != "reaped" || len(w.rows) != 1 {
		t.Fatalf("%+v calls%d", got, calls)
	}
}
func TestExactReapRejectsMissingIdentityAndCallback(t *testing.T) {
	w := &exactWorld{}
	got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10}, {PID: 11, Identity: "valid"}}, nil)
	if len(got) != 2 || got[0].Status != "refused" || got[1].Status != "refused" {
		t.Fatalf("%+v", got)
	}
}
func TestExactReapStatefulWorldEffectConformance(t *testing.T) {
	// The callback is the effect barrier: the new descendant is deliberately
	// created after admission. The survivor must be reproven for KILL, while the
	// foreign current child remains outside the immutable keys.
	w := &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "old"}}, keep: true}
	got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}}, func(ctx context.Context, step ReapStep, sig syscall.Signal) error {
		if step.PID != 10 || step.Identity != "old" {
			t.Fatalf("foreign effect %+v", step)
		}
		if sig == syscall.SIGTERM {
			w.rows = append(w.rows, ProcessRow{PID: 11, PPID: 10, Identity: "current"})
		}
		return w.signal(ctx, step, sig)
	})
	if got[0].Status != "reaped" || len(w.sigs) != 2 || w.sigs[0] != syscall.SIGTERM || w.sigs[1] != syscall.SIGKILL || len(w.rows) != 1 || w.rows[0].PID != 11 {
		t.Fatalf("%+v %+v", got, w)
	}
}

func TestExactReapSurvivingKillAndCancellationRetainProgress(t *testing.T) {
	w := &exactWorld{rows: []ProcessRow{{PID: 10, Identity: "old"}}}
	calls := 0
	got := exactRunner(w).ReapExact(context.Background(), []ReapStep{{PID: 10, Identity: "old"}}, func(context.Context, ReapStep, syscall.Signal) error { calls++; return nil })
	if calls != 2 || got[0].Status != "failed" || !strings.Contains(got[0].Detail, "TERM and KILL acknowledged") {
		t.Fatalf("%+v calls%d", got, calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got = exactRunner(w).ReapExact(ctx, []ReapStep{{PID: 10, Identity: "old"}}, func(context.Context, ReapStep, syscall.Signal) error { cancel(); return nil })
	if got[0].Status != "failed" || !strings.Contains(got[0].Detail, "TERM acknowledged") {
		t.Fatalf("%+v", got)
	}
}

func TestExactReapBatchObservationsDoNotGrowPerTarget(t *testing.T) {
	for _, count := range []int{8, 16} {
		w := &exactWorld{keep: true}
		steps := make([]ReapStep, count)
		for i := range steps {
			steps[i] = ReapStep{PID: i + 1, Identity: "generation"}
			w.rows = append(w.rows, ProcessRow{PID: i + 1, Identity: "generation"})
		}
		r := Reaper{Table: w}
		got := r.ReapExact(context.Background(), steps, w.signal)
		if len(got) != count || len(w.sigs) != 2*count || w.snapshots != 2 {
			t.Fatalf("K=%d outcomes%d signals%d shared snapshots%d", count, len(got), len(w.sigs), w.snapshots)
		}
		for _, row := range got {
			if row.Status != "reaped" {
				t.Fatalf("%+v", row)
			}
		}
	}
}
