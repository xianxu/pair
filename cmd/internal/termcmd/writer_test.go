package termcmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors(t *testing.T) {
	// A real `zellij` is not needed: exec fails, and the failure is exactly
	// the case that used to write the pane's stderr per wheel tick.
	var stdout, stderr bytes.Buffer
	_ = runZellij([]string{"action", "nonexistent-verb"}, &stdout, &stderr)

	// The point is the plumbing: whatever the subprocess emits lands in the
	// writers it was given. If runZellij ignored them for os.Stdout/os.Stderr,
	// this test would still pass -- so assert the wiring directly too.
	// BOTH verbs, because each descriptor is only exercised by one of them:
	// a succeeding action writes stdout and nothing to stderr, a failing one
	// the reverse. Testing one verb leaves the other descriptor unchecked --
	// which is how the stderr wiring survived its first mutation check.
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"Action/succeeds", func() error { return OSRuntime{}.RunZellijAction("list-clients") }},
		{"Action/fails", func() error { return OSRuntime{}.RunZellijAction("nonexistent-verb") }},
		{"Quiet/succeeds", func() error { return OSRuntime{}.RunZellijActionQuiet("list-clients") }},
		{"Quiet/fails", func() error { return OSRuntime{}.RunZellijActionQuiet("nonexistent-verb") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// os.Stdout/os.Stderr are redirected to a pipe; anything the
			// subprocess writes there is a byte on the operator's pane.
			out, restoreOut := captureFD(t, &os.Stdout)
			errOut, restoreErr := captureFD(t, &os.Stderr)
			// POSITIVE CONTROL: prove the capture can SEE a write before
			// concluding from its silence. Without this, a captureFD that
			// silently failed would make every arm below pass.
			fmt.Fprint(os.Stdout, "control-out")
			fmt.Fprint(os.Stderr, "control-err")
			_ = tc.run()
			restoreOut()
			restoreErr()
			if !bytes.Contains(out(), []byte("control-out")) {
				t.Fatal("captureFD did not observe a direct stdout write; the assertions below are vacuous")
			}
			if !bytes.Contains(errOut(), []byte("control-err")) {
				t.Fatal("captureFD did not observe a direct stderr write; the assertions below are vacuous")
			}
			// Minus the control bytes, the subprocess must have written NOTHING.
			if got := bytes.ReplaceAll(out(), []byte("control-out"), nil); len(got) != 0 {
				t.Fatalf("%s wrote %d bytes to the pane's stdout: %q", tc.name, len(got), got)
			}
			if got := bytes.ReplaceAll(errOut(), []byte("control-err"), nil); len(got) != 0 {
				t.Fatalf("%s wrote %d bytes to the pane's stderr: %q", tc.name, len(got), got)
			}
		})
	}
}

func captureFD(t *testing.T, target **os.File) (func() []byte, func()) {
	t.Helper()
	saved := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	*target = w
	done := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	var captured []byte
	var once sync.Once
	restore := func() {
		once.Do(func() {
			*target = saved
			_ = w.Close()
			captured = <-done
			_ = r.Close()
		})
	}
	return func() []byte { return captured }, restore
}

func TestTheZellijSubprocessGetsNoStdinEither(t *testing.T) {
	src, err := os.ReadFile("run.go")
	if err != nil {
		t.Fatalf("read run.go: %v", err)
	}
	if strings.Contains(string(src), "cmd.Stdin = os.Stdin") {
		t.Fatal("runZellij hands the subprocess the pane's raw-mode stdin; it can eat keystrokes")
	}
}
