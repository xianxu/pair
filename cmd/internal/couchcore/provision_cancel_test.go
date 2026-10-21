package couchcore

import (
	"errors"
	"os"
	"slices"
	"syscall"
	"testing"
	"time"
)

// A group may remain after its leader exits. Unknown probes preserve it.
type provisionCancelWorld struct {
	present                    bool
	probeErr, termErr, killErr error
	signals                    []syscall.Signal
	probes, disappearOnProbe   int
}

func (w *provisionCancelWorld) signal(pid int, sig syscall.Signal) error {
	if pid != -41 {
		panic("wrong process group")
	}
	switch sig {
	case 0:
		w.probes++
		if w.disappearOnProbe > 0 && w.probes >= w.disappearOnProbe {
			w.present = false
		}
		if w.probeErr != nil {
			return w.probeErr
		}
		if !w.present {
			return syscall.ESRCH
		}
		return nil
	case syscall.SIGTERM:
		w.signals = append(w.signals, sig)
		return w.termErr
	case syscall.SIGKILL:
		w.signals = append(w.signals, sig)
		w.present = false
		return w.killErr
	default:
		panic("unexpected signal")
	}
}

func TestProvisionGroupCancellationConfirmsGroupExit(t *testing.T) {
	term := []syscall.Signal{syscall.SIGTERM}
	escalate := []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL}
	tests := []struct {
		name    string
		world   provisionCancelWorld
		want    []syscall.Signal
		wantErr error
	}{
		{name: "whole group gone", want: term},
		{name: "leader gone descendant remains", world: provisionCancelWorld{present: true}, want: escalate},
		{name: "permission unavailable", world: provisionCancelWorld{probeErr: syscall.EPERM}, want: escalate},
		{name: "inspection unavailable", world: provisionCancelWorld{probeErr: syscall.EIO}, want: escalate},
		{name: "initial group absent", world: provisionCancelWorld{termErr: syscall.ESRCH}, want: term, wantErr: os.ErrProcessDone},
		{name: "escalation group absent", world: provisionCancelWorld{present: true, killErr: syscall.ESRCH}, want: escalate},
		{name: "term error retained", world: provisionCancelWorld{present: true, termErr: syscall.EPERM}, want: escalate, wantErr: syscall.EPERM},
		{name: "kill error retained", world: provisionCancelWorld{present: true, killErr: syscall.EPERM}, want: escalate, wantErr: syscall.EPERM},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A ready grace cannot justify signaling a confirmed absent group.
			grace := make(chan time.Time, 1)
			grace <- time.Time{}
			starts, stops := 0, 0
			err := cancelProvisionGroup(41, tt.world.signal, func() provisionGroupClock {
				starts++
				if !slices.Equal(tt.world.signals, term) {
					t.Fatalf("grace started before TERM: signals=%v", tt.world.signals)
				}
				return provisionGroupClock{grace: grace, stop: func() { stops++ }}
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error=%v want %v", err, tt.wantErr)
			}
			if !slices.Equal(tt.world.signals, tt.want) {
				t.Fatalf("signals=%v want %v", tt.world.signals, tt.want)
			}
			wantStarts := 1
			if errors.Is(tt.world.termErr, syscall.ESRCH) {
				wantStarts = 0
			}
			if starts != wantStarts || stops != wantStarts {
				t.Fatalf("clock starts=%d stops=%d want %d", starts, stops, wantStarts)
			}
		})
	}
}

func TestProvisionGroupCancellationCompletesBeforeGrace(t *testing.T) {
	// The descendant survives the first observation and exits on the next poll.
	// No grace event is ever delivered: completion must come from group absence.
	w := &provisionCancelWorld{present: true, disappearOnProbe: 2}
	poll := make(chan time.Time, 1)
	poll <- time.Time{}
	if err := cancelProvisionGroup(41, w.signal, func() provisionGroupClock {
		return provisionGroupClock{poll: poll, stop: func() {}}
	}); err != nil {
		t.Fatal(err)
	}
	if w.probes != 2 || !slices.Equal(w.signals, []syscall.Signal{syscall.SIGTERM}) {
		t.Fatalf("probes=%d signals=%v", w.probes, w.signals)
	}
}
