package diagnosticlog

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCurrentCreationCancellationAtEveryBoundary(t *testing.T) {
	for _, step := range []string{"creation-staged", "creation-intent", "creation-published", "creation-finalized"} {
		t.Run(step, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.log")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			o := Options{Context: ctx, Fault: func(at string) error {
				if at == step {
					cancel()
				}
				return nil
			}}
			if _, err := Open(path, o); !errors.Is(err, context.Canceled) {
				t.Fatalf("interruption=%v", err)
			}
			w, err := Open(path, Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			state, err := load(path)
			if err != nil {
				t.Fatal(err)
			}
			st, err := regular(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = matches(st, state.Current); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Lstat(creationStagePath(path)); !os.IsNotExist(err) {
				t.Fatalf("stage retained: %v", err)
			}
		})
	}
}

func TestCurrentCreationRefusesReplacement(t *testing.T) {
	for _, step := range []string{"creation-intent", "creation-published"} {
		t.Run(step, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.log")
			stop := errors.New("stop")
			if _, err := Open(path, Options{Fault: func(at string) error {
				if at == step {
					return stop
				}
				return nil
			}}); !errors.Is(err, stop) {
				t.Fatalf("fault=%v", err)
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("unrelated"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(path, Options{}); err == nil {
				t.Fatal("accepted replacement")
			}
			if b, err := os.ReadFile(path); err != nil || string(b) != "unrelated" {
				t.Fatalf("replacement modified: %q %v", b, err)
			}
		})
	}
}

func TestCurrentCreationKilledPublisher(t *testing.T) {
	for _, step := range []string{"creation-staged", "creation-intent", "creation-published", "creation-finalized"} {
		t.Run(step, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.log")
			killCurrentCreationPublisher(t, path, step)
			w, err := Open(path, Options{})
			if err != nil {
				t.Fatalf("reopen: %v", err)
			}
			defer w.Close()
			state, err := load(path)
			if err != nil {
				t.Fatal(err)
			}
			st, err := regular(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = matches(st, state.Current); err != nil {
				t.Fatal(err)
			}
			if st.Size() != 0 {
				t.Fatalf("creation invented contents: %d", st.Size())
			}
		})
	}
}
func TestCurrentCreationChild(t *testing.T) {
	path := os.Getenv("PAIR_CREATION_CHILD")
	if path == "" {
		t.Skip("subprocess")
	}
	_, err := Open(path, Options{Fault: func(at string) error {
		if at == os.Getenv("PAIR_CREATION_STEP") {
			fmt.Println("ready")
			for {
				time.Sleep(time.Hour)
			}
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal("boundary not reached")
}

func TestCurrentCreationAfterRotationAndCollection(t *testing.T) {
	for _, transition := range []string{"rotation", "collection"} {
		t.Run(transition, func(t *testing.T) {
			path, now, o := fixture(t)
			w, err := Open(path, o)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write([]byte("old")); err != nil {
				t.Fatal(err)
			}
			if err = w.Close(); err != nil {
				t.Fatal(err)
			}
			if transition == "rotation" {
				*now = now.Add(25 * time.Hour)
				if err = Maintain(path, o); err != nil {
					t.Fatal(err)
				}
			} else {
				*now = now.Add(8 * 24 * time.Hour)
				if _, err = Collect(path, o, 100); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("transition did not retire current path: %v", err)
			}
			stop := errors.New("stop creation")
			o.Fault = func(step string) error {
				if step == "creation-published" {
					return stop
				}
				return nil
			}
			if _, err = Open(path, o); !errors.Is(err, stop) {
				t.Fatalf("creation boundary=%v", err)
			}
			o.Fault = nil
			next, err := Open(path, o)
			if err != nil {
				t.Fatal(err)
			}
			defer next.Close()
			if _, err = next.Write([]byte("new")); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil || string(b) != "new" {
				t.Fatalf("recreated payload=%q %v", b, err)
			}
		})
	}
}

func TestCurrentCreationStageRejectsUnexpectedEntries(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "nonempty"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.log")
			if err := os.Mkdir(directory(path), 0700); err != nil {
				t.Fatal(err)
			}
			stage := creationStagePath(path)
			var err error
			switch kind {
			case "symlink":
				err = os.Symlink(filepath.Join(filepath.Dir(path), "unrelated"), stage)
			case "directory":
				err = os.Mkdir(stage, 0700)
			default:
				err = os.WriteFile(stage, []byte("unrelated"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Open(path, Options{}); err == nil {
				t.Fatal("unexpected stage accepted")
			}
			if _, err = os.Lstat(stage); err != nil {
				t.Fatalf("unexpected stage removed: %v", err)
			}
			if _, err = os.Lstat(path); !os.IsNotExist(err) {
				t.Fatalf("published without authority: %v", err)
			}
		})
	}
}

func killCurrentCreationPublisher(t *testing.T, path, step string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCurrentCreationChild$")
	cmd.Env = append(os.Environ(), "PAIR_CREATION_CHILD="+path, "PAIR_CREATION_STEP="+step)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	ready := make(chan bool, 1)
	go func() { scanner := bufio.NewScanner(out); ready <- scanner.Scan() && scanner.Text() == "ready" }()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("publisher did not reach boundary")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("publisher blocked")
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}

func TestCurrentCreationKilledPublisherCollectedWithoutReopen(t *testing.T) {
	for _, step := range []string{"creation-staged", "creation-intent", "creation-published", "creation-finalized"} {
		t.Run(step, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.log")
			killCurrentCreationPublisher(t, path, step)
			options := Options{Now: func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }, Proof: func(context.Context, string, []Registration) error { return nil }}
			for range 2 {
				if _, err := Collect(path, options, 100); err != nil {
					t.Fatalf("collect without reopen: %v", err)
				}
				for _, p := range []string{path, directory(path)} {
					if _, err := os.Lstat(p); !os.IsNotExist(err) {
						t.Fatalf("collected path remains %s: %v", p, err)
					}
				}
			}
		})
	}
}
