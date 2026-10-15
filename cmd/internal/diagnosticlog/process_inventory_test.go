package diagnosticlog

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

type processWorld struct {
	rows       []procutil.Observation
	generation string
	strict     string
	recycle    bool
	envCalls   int
	envErr     error
}

func (w *processWorld) Snapshot(ctx context.Context) ([]procutil.Observation, error) {
	return w.rows, ctx.Err()
}
func (w *processWorld) Identity(string) string       { return w.generation }
func (w *processWorld) StrictIdentity(string) string { return w.strict }
func (w *processWorld) Environment(int) (map[string]string, error) {
	w.envCalls++
	if w.recycle {
		w.strict = "new-generation"
	}
	return map[string]string{"PAIR_RETENTION_PROTOCOL": "1"}, w.envErr
}

func TestRuntimeInventoryUsesFullExecutableAndFencesGeneration(t *testing.T) {
	for _, name := range []string{"pair-session-watch", "pair-title-poller"} {
		w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: name[:15], Identity: "old"}, CommandInfo: procutil.CommandInfo{Executable: "/bin/" + name, Args: []string{"unrelated-argv-zero"}}}}, generation: "old", strict: "birth"}
		probe := OSInspection{processes: w}
		got, err := probe.Runtimes(context.Background())
		if err != nil || len(got) != 1 || !got[0].Protocol {
			t.Fatalf("%s runtimes=%v err=%v", name, got, err)
		}
		w.recycle = true
		if _, err := probe.Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) {
			t.Fatalf("recycled process accepted: %v", err)
		}
		w.recycle = false
		w.generation = "new"
		before := w.envCalls
		if _, err := probe.Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) || w.envCalls != before {
			t.Fatalf("stale snapshot read environment: %v", err)
		}
	}
}
func TestRuntimeInventoryRefusesUnreadableTruncatedCandidate(t *testing.T) {
	w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair-session-wa"}, Err: errors.New("denied")}}}
	if _, err := (OSInspection{processes: w}).Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) {
		t.Fatalf("unknown candidate disappeared: %v", err)
	}
	w.rows[0].Name = "unrelated"
	if got, err := (OSInspection{processes: w}).Runtimes(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("unrelated inaccessible process: %v %v", got, err)
	}
}
func TestRuntimeInventoryHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (OSInspection{}).Runtimes(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation=%v", err)
	}
}

func TestRuntimeInventoryRecognizesUnlinkedExecutable(t *testing.T) {
	w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair", Identity: "old"}, CommandInfo: procutil.CommandInfo{Executable: "/bin/pair (deleted)", Args: []string{"pair"}}}}, generation: "old", strict: "birth"}
	got, err := (OSInspection{processes: w}).Runtimes(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("unlinked running executable disappeared: %+v %v", got, err)
	}
}

func TestRuntimeInventorySkipsGoneAndZombieRows(t *testing.T) {
	for _, gone := range []error{syscall.EINVAL, syscall.ESRCH, syscall.ENOENT} {
		t.Run(gone.Error(), func(t *testing.T) {
			w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair-session-wa", Identity: "old"}, Err: fmt.Errorf("process command: %w", gone)}}, generation: "old", strict: "birth"}
			got, err := (OSInspection{processes: w}).Runtimes(context.Background())
			if err != nil || len(got) != 0 || w.envCalls != 0 {
				t.Fatalf("gone runtimes=%v err=%v environment=%d", got, err, w.envCalls)
			}
		})
	}
	w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair", Identity: "old", Zombie: true}, CommandInfo: procutil.CommandInfo{Executable: "/bin/pair", Args: []string{"pair"}}}}, generation: "old", strict: "birth"}
	got, err := (OSInspection{processes: w}).Runtimes(context.Background())
	if err != nil || len(got) != 0 || w.envCalls != 0 {
		t.Fatalf("zombie runtimes=%v err=%v environment=%d", got, err, w.envCalls)
	}
}
func TestRuntimeInventoryEnvironmentGoneAfterSnapshot(t *testing.T) {
	for _, gone := range []error{syscall.EINVAL, syscall.ESRCH, syscall.ENOENT} {
		t.Run(gone.Error(), func(t *testing.T) {
			w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair", Identity: "old"}, CommandInfo: procutil.CommandInfo{Executable: "/bin/pair", Args: []string{"pair"}}}}, generation: "old", strict: "birth", envErr: fmt.Errorf("environment: %w", gone)}
			got, err := (OSInspection{processes: w}).Runtimes(context.Background())
			if err != nil || len(got) != 0 || w.envCalls != 1 {
				t.Fatalf("gone after snapshot runtimes=%v err=%v environment=%d", got, err, w.envCalls)
			}
		})
	}
}

func TestRuntimeInventoryIdentityEINVALStaysUnknown(t *testing.T) {
	for _, name := range []string{"pair-session-wa", ""} {
		failure := fmt.Errorf("process 123: %w", &procutil.IdentityUnavailableError{Err: syscall.EINVAL})
		w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: name, Identity: "old"}, Err: failure}}, generation: "old", strict: "birth"}
		if _, err := (OSInspection{processes: w}).Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) || w.envCalls != 0 {
			t.Fatalf("identity failure became absence: name=%q err=%v environment=%d", name, err, w.envCalls)
		}
		// The environment seam must preserve a wrapped identity uncertainty too;
		// raw environment EINVAL remains covered by the disappearance test above.
		w.rows[0].Name = "pair"
		w.rows[0].Err = nil
		w.rows[0].CommandInfo = procutil.CommandInfo{Executable: "/bin/pair", Args: []string{"pair"}}
		w.envErr = failure
		if _, err := (OSInspection{processes: w}).Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) || w.envCalls != 1 {
			t.Fatalf("wrapped environment identity failure became absence: err=%v environment=%d", err, w.envCalls)
		}
	}
}
func TestRuntimeInventoryPreservesUnknownAndPermissionFailures(t *testing.T) {
	for _, failure := range []error{syscall.EPERM, syscall.EACCES, errors.New("unrecognized process read")} {
		w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair", Identity: "old"}, Err: fmt.Errorf("command: %w", failure)}}, generation: "old", strict: "birth"}
		if _, err := (OSInspection{processes: w}).Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) {
			t.Fatalf("command failure became absence: %v", err)
		}
		w.rows[0].Err = nil
		w.rows[0].CommandInfo = procutil.CommandInfo{Executable: "/bin/pair", Args: []string{"pair"}}
		w.envErr = fmt.Errorf("environment: %w", failure)
		if _, err := (OSInspection{processes: w}).Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) {
			t.Fatalf("environment failure became absence: %v", err)
		}
		w.generation = "reused"
		before := w.envCalls
		if _, err := (OSInspection{processes: w}).Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) || w.envCalls != before {
			t.Fatalf("stale generation read environment: %v", err)
		}
	}
}
