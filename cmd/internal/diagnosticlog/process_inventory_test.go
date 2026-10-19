package diagnosticlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	env        map[string]string
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
	if w.env != nil {
		return w.env, w.envErr
	}
	return map[string]string{"PAIR_RETENTION_PROTOCOL": "1"}, w.envErr
}

func TestRuntimeInventoryRemovedRootDoesNotBlockProtocolWriters(t *testing.T) {
	root := filepath.Join(t.TempDir(), "removed")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	for _, protocol := range []bool{true, false} {
		t.Run(fmt.Sprintf("protocol=%t", protocol), func(t *testing.T) {
			env := map[string]string{"PAIR_DATA_DIR": root}
			if protocol {
				env["PAIR_RETENTION_PROTOCOL"] = "1"
			}
			w := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "zellij", Identity: "old"}, CommandInfo: procutil.CommandInfo{Executable: "/bin/zellij"}}}, generation: "old", strict: "birth", env: env}
			probe := OSInspection{processes: w}
			err := VerifyWriters(context.Background(), filepath.Join(t.TempDir(), "absent-trace"), []Registration{{PID: 456, Birth: "live", Root: "/live-root"}}, probe)
			if protocol {
				if err != nil {
					t.Fatalf("protocol runtime with removed root blocks unrelated retention: %v", err)
				}
			} else if !errors.Is(err, ErrUnknownWriters) {
				t.Fatalf("unresolved legacy runtime root accepted: %v", err)
			}
			// Exemption must still be based on one stable process generation.
			w.recycle = true
			if _, err := probe.Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) {
				t.Fatalf("generation changed while reading protocol: %v", err)
			}
			w.recycle = false
			w.generation = "new"
			before := w.envCalls
			if _, err := probe.Runtimes(context.Background()); !errors.Is(err, ErrUnknownWriters) || w.envCalls != before {
				t.Fatalf("stale snapshot read protocol: err=%v envCalls=%d, before=%d", err, w.envCalls, before)
			}
		})
	}
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

func TestCouchClientWithoutPairDataDirBlocksOnlyItsOwner(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := "e40c2099f5fe78d6"
	own := filepath.Join(root, "repos", scope, "wrap-events-1-brain-41.jsonl")
	other := filepath.Join(root, "repos", scope, "wrap-events-other.jsonl")
	world := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: 123, Name: "pair", Identity: "old"}, CommandInfo: procutil.CommandInfo{Executable: "/bin/pair", Args: []string{"pair", "--couch-session-v1", "resume", "1-brain-41"}}}}, generation: "old", strict: "birth", env: map[string]string{"COUCH_PAIR_DATA_DIR": root, "COUCH_THREAD_SCOPE": scope, "COUCH_THREAD_TAG": "1-brain-41", "COUCH_TREE": "/work/brain"}}
	inspector := OSInspection{processes: world, openFileCommand: func(context.Context, ...string) ([]byte, []byte, error) { return nil, nil, nil }}
	proof := InspectionProof(inspector)
	ctx := SweepContext(context.Background(), []string{own, other})
	// Canonicalization requires a real parent, not actual payload files.
	if err := os.MkdirAll(filepath.Dir(own), 0700); err != nil {
		t.Fatal(err)
	}
	if err := proof(ctx, other, nil); err != nil {
		t.Fatalf("Couch client blocked unrelated diagnostics: %v", err)
	}
	if err := proof(ctx, own, nil); !errors.Is(err, ErrUnknownWriters) || !strings.Contains(err.Error(), "brain slot") || !strings.Contains(err.Error(), "relaunch to register") {
		t.Fatalf("own diagnostic lacks actionable blocker: %v", err)
	}
	// Conflicting ownership does not turn into an empty/irrelevant inventory.
	world.env["PAIR_TAG"] = "different"
	if err := proof(SweepContext(context.Background(), []string{own, other}), other, nil); !errors.Is(err, ErrUnknownWriters) {
		t.Fatalf("conflicting owner accepted: %v", err)
	}
	delete(world.env, "PAIR_TAG")
	delete(world.env, "COUCH_THREAD_TAG")
	if err := proof(SweepContext(context.Background(), []string{own, other}), other, nil); !errors.Is(err, ErrUnknownWriters) {
		t.Fatalf("incomplete owner accepted: %v", err)
	}
}
