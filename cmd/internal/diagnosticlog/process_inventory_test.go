package diagnosticlog

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"testing"
)

type processWorld struct {
	rows       []procutil.Observation
	generation string
	strict     string
	recycle    bool
	envCalls   int
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
	return map[string]string{"PAIR_RETENTION_PROTOCOL": "1"}, nil
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
