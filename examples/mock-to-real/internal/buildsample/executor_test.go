package buildsample

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// compilerFake owns candidate files and the cancellation worker. It validates
// actual stage state, so pipeline tests do not merely prescribe call returns.
type compilerFake struct {
	final                                 string
	failCompile, failValidate             bool
	admitted, cancelSeen, release, joined chan struct{}
	stages                                []string
	phases                                []string
	candidate                             string
	workerErr                             error
}

func (f *compilerFake) Run(ctx context.Context, dir, name string, args ...string) error {
	if name == "go" {
		f.stages = append(f.stages, dir)
		f.phases = append(f.phases, "compile")
		var candidate string
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-o" {
				candidate = args[i+1]
				break
			}
		}
		if candidate == "" || filepath.Dir(candidate) != dir {
			return errors.New("missing stage candidate")
		}
		if _, e := os.Stat(filepath.Join(dir, "cmd/production/main.go")); e != nil {
			return e
		}
		f.candidate = candidate
		artifact := "candidate-artifact"
		if f.failCompile {
			artifact = "partial-candidate"
		}
		if e := os.WriteFile(candidate, []byte(artifact), 0755); e != nil {
			return e
		}
		f.phases = append(f.phases, "artifact")
		if f.cancelSeen != nil {
			close(f.admitted)
			// Run cannot acknowledge cancellation until its admitted worker has joined.
			workerDone := make(chan struct{})
			go func() {
				defer close(workerDone)
				<-ctx.Done()
				close(f.cancelSeen)
				<-f.release
				f.workerErr = os.WriteFile(candidate, []byte("late-worker-artifact"), 0755)
			}()
			<-workerDone
			close(f.joined)
			return errors.Join(ctx.Err(), f.workerErr)
		}
		if f.failCompile {
			return errors.New("compile failed after partial candidate")
		}
		return nil
	}
	if name != f.candidate || !reflect.DeepEqual(args, []string{"--check"}) {
		return errors.New("unexpected validation command")
	}
	candidate, e := os.ReadFile(name)
	if e != nil || string(candidate) != "candidate-artifact" {
		return errors.New("validation did not observe compiled artifact")
	}
	if _, e := os.Stat(f.final); !os.IsNotExist(e) {
		return errors.New("published before validation")
	}
	f.phases = append(f.phases, "validate")
	if f.failValidate {
		return errors.New("validate candidate configuration failed")
	}
	return nil
}
func assertRetired(t *testing.T, f *compilerFake, output string) {
	t.Helper()
	for _, stage := range f.stages {
		if _, e := os.Stat(stage); !os.IsNotExist(e) {
			t.Fatalf("stage remains: %s (%v)", stage, e)
		}
	}
	if _, e := os.Stat(output); !os.IsNotExist(e) {
		t.Fatalf("final artifact remains: %v", e)
	}
}
func TestExecutorFailureRetiresPartialCandidate(t *testing.T) {
	for _, phase := range []string{"compile", "validate"} {
		t.Run(phase, func(t *testing.T) {
			root := copySample(t)
			out := filepath.Join(t.TempDir(), "output")
			f := &compilerFake{final: out, failCompile: phase == "compile", failValidate: phase == "validate"}
			var report bytes.Buffer
			e := buildWithExecutor(context.Background(), root, out, &report, f)
			if e == nil || !strings.Contains(e.Error(), phase) {
				t.Fatalf("wrong error %v", e)
			}
			assertRetired(t, f, out)
			if report.Len() != 0 {
				t.Fatalf("failure reported success: %s", &report)
			}
		})
	}
}
func TestExecutorSuccessValidatesThenPublishesOnce(t *testing.T) {
	root := copySample(t)
	out := filepath.Join(t.TempDir(), "output")
	f := &compilerFake{final: out}
	var report bytes.Buffer
	if e := buildWithExecutor(context.Background(), root, out, &report, f); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(f.phases, []string{"compile", "artifact", "validate"}) {
		t.Fatal(f.phases)
	}
	b, e := os.ReadFile(out)
	if e != nil || string(b) != "candidate-artifact" {
		t.Fatalf("artifact %q: %v", b, e)
	}
	for _, stage := range f.stages {
		if _, e := os.Stat(stage); !os.IsNotExist(e) {
			t.Fatal("stage not retired")
		}
	}
	if strings.Count(report.String(), "Built ") != 1 {
		t.Fatal(report.String())
	}
	if e := buildWithExecutor(context.Background(), root, out, io.Discard, f); e == nil {
		t.Fatal("replaced published artifact")
	}
	if len(f.phases) != 3 {
		t.Fatal("recompiled existing output")
	}
}
func TestExecutorCancellationJoinsBeforeRetirement(t *testing.T) {
	root := copySample(t)
	out := filepath.Join(t.TempDir(), "output")
	f := &compilerFake{final: out, admitted: make(chan struct{}), cancelSeen: make(chan struct{}), release: make(chan struct{}), joined: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var report bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- buildWithExecutor(ctx, root, out, &report, f) }()
	// Cancel only after a compiler artifact exists and its worker is admitted.
	// The compile phase notification is protected independently of the context.
	select {
	case <-f.admitted:
	case e := <-done:
		t.Fatalf("compile admission failed: %v", e)
	}
	cancel()
	<-f.cancelSeen
	select {
	case e := <-done:
		t.Fatalf("returned before worker join: %v", e)
	default:
	}
	if _, e := os.Stat(f.candidate); e != nil {
		t.Fatal("retired active worker stage", e)
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("published canceled build")
	}
	close(f.release)
	if e := <-done; !errors.Is(e, context.Canceled) {
		t.Fatalf("cancellation: %v", e)
	}
	select {
	case <-f.joined:
	default:
		t.Fatal("fake worker not joined")
	}
	if f.workerErr != nil {
		t.Fatalf("worker stage retired before join: %v", f.workerErr)
	}
	assertRetired(t, f, out)
	if report.Len() != 0 {
		t.Fatal("reported canceled build")
	}
}
