package testcouch

import (
	"context"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestCrashCohortUsesOwnedClosureAndExcludesProtected(t *testing.T) {
	rows := []procutil.Process{{PID: 10, Identity: "old", TTY: "worker"}, {PID: 11, PPID: 10, Identity: "child", TTY: "worker"}, {PID: 12, PPID: 10, Identity: "retained"}, {PID: 20, Identity: "hub", TTY: "hub"}, {PID: 99, Identity: "foreign"}}
	got, err := selectCrashCohort(rows, map[int]bool{10: true, 20: true}, []Resource{{ID: "12", Generation: "retained"}, {ID: "20", Generation: "hub"}})
	if err != nil || len(got) != 2 || got[0].ID != "10" || got[1].ID != "11" {
		t.Fatalf("%+v %v", got, err)
	}
}
func TestCrashCohortIdentityGuardNeverSignalsReplacement(t *testing.T) {
	io := &crashGuardIO{rows: []procutil.Process{{PID: 10, Identity: "replacement"}}}
	_, err := stopOwned(context.Background(), io, ReduceRun(OwnedRun{}, RunEvent{Kind: "own", Resource: Resource{ID: "10", Generation: "old"}}), syscall.SIGKILL)
	if err != nil || io.signals != 0 {
		t.Fatalf("signals%d error%v", io.signals, err)
	}
	io.rows = []procutil.Process{{PID: 10}}
	_, err = stopOwned(context.Background(), io, ReduceRun(OwnedRun{}, RunEvent{Kind: "own", Resource: Resource{ID: "10", Generation: "old"}}), syscall.SIGKILL)
	if err == nil || io.signals != 0 {
		t.Fatalf("unknown identity signals%d error%v", io.signals, err)
	}
}

type crashGuardIO struct {
	rows    []procutil.Process
	signals int
}

func (f *crashGuardIO) Table(context.Context) ([]procutil.Process, error) { return f.rows, nil }
func (f *crashGuardIO) Signal(int, string, syscall.Signal) error          { f.signals++; return nil }

func TestCrashCohortUnknownOwnershipNeverBecomesAuthority(t *testing.T) {
	rows := []procutil.Process{{PID: 10, Identity: "old"}}
	selected, err := selectCrashCohort(rows, nil, nil)
	if err != nil || len(selected) != 0 {
		t.Fatalf("unmarked process admitted %+v %v", selected, err)
	}
	rows[0].Identity = ""
	if _, err = selectCrashCohort(rows, map[int]bool{10: true}, nil); err == nil {
		t.Fatal("unknown identity admitted")
	}
}

func TestCrashGenerationDistinguishesGoneFromUnknown(t *testing.T) {
	old := Resource{ID: "10", Generation: "old"}
	for _, tc := range []struct {
		name string
		rows []procutil.Process
		live bool
		bad  bool
	}{
		{name: "gone"}, {name: "replaced", rows: []procutil.Process{{PID: 10, Identity: "new"}}}, {name: "live", rows: []procutil.Process{{PID: 10, Identity: "old"}}, live: true}, {name: "unknown", rows: []procutil.Process{{PID: 10}}, bad: true}, {name: "ambiguous", rows: []procutil.Process{{PID: 10, Identity: "old"}, {PID: 10, Identity: "old"}}, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live, err := currentCrashGeneration(tc.rows, old)
			if live != tc.live || (err != nil) != tc.bad {
				t.Fatalf("live%v err%v", live, err)
			}
		})
	}
}
