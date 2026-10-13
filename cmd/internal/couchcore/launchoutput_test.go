package couchcore

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func launchOutputForTest(t *testing.T, h Handle) (string, []byte) {
	t.Helper()
	output, ok := h.(interface{ LaunchOutput() (string, []byte) })
	if !ok {
		t.Fatal("handle has no launch output capability")
	}
	return output.LaunchOutput()
}

func TestLaunchOutputExecForwardedAndDrained(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	original := os.Stderr
	os.Stderr = f
	h, err := (ExecRunner{}).Start(t.TempDir(), []string{"sh", "-c", "printf 'launch failed\\377' >&2; test ! -t 2 || exit 99; exit 23"}, nil)
	os.Stderr = original
	if err != nil {
		t.Fatal(err)
	}
	if code := h.Wait(); code != 23 {
		t.Fatalf("exit=%d", code)
	}
	stream, got := launchOutputForTest(t, h)
	want := append([]byte("launch failed"), 255)
	if stream != "stderr" || !bytes.Equal(got, want) {
		t.Fatalf("output=%q %q", stream, got)
	}
	forwarded, err := os.ReadFile(f.Name())
	if err != nil || !bytes.Equal(forwarded, want) {
		t.Fatalf("forwarded=%q err=%v", forwarded, err)
	}
	got[0] = 0
	_, again := launchOutputForTest(t, h)
	if !bytes.Equal(again, want) {
		t.Fatal("snapshot aliases retained output")
	}
}

func TestLaunchOutputPTYAndBlockedWrapper(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "blocked"}[blocked], func(t *testing.T) {
			r := &PtyRunner{LaunchHelper: os.Args[0]}
			argv := []string{"sh", "-c", "printf out; printf err >&2; exit 23"}
			var h Handle
			var err error
			if blocked {
				b, e := r.StartBlocked(context.Background(), t.TempDir(), argv, []string{"PAIR_TEST_RUNNER_HELPER=1"}, time.Second)
				err = e
				if e == nil {
					h = b
					err = b.Acknowledge()
				}
			} else {
				h, err = r.Start(t.TempDir(), argv, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if code := h.Wait(); code != 23 {
				t.Fatalf("exit=%d", code)
			}
			stream, got := launchOutputForTest(t, h)
			if stream != "pty stdout/stderr" || string(got) != "outerr" {
				t.Fatalf("output=%q %q", stream, got)
			}
		})
	}
}

func TestLaunchOutputFakeBoundedClone(t *testing.T) {
	r := NewFakeRunner()
	h, err := r.Start(t.TempDir(), []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := bytes.Repeat([]byte{255, 'x'}, 12000)
	r.Terminal(h.ID()).Feed(input)
	stream, got := launchOutputForTest(t, h)
	want := input[len(input)-16*1024:]
	if stream != "pty stdout/stderr" || !bytes.Equal(got, want) {
		t.Fatalf("stream=%q bytes=%d", stream, len(got))
	}
	got[0] = 0
	_, again := launchOutputForTest(t, h)
	if !bytes.Equal(again, want) {
		t.Fatal("snapshot aliases retained output")
	}
	r.SetExited(h.ID(), 23)
}

func TestLaunchOutputExecBlockedTail(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	original := os.Stderr
	os.Stderr = f
	h, err := (ExecRunner{LaunchHelper: os.Args[0]}).StartBlocked(context.Background(), t.TempDir(), []string{"sh", "-c", "printf '" + string(bytes.Repeat([]byte{'x'}, 20000)) + "\\377' >&2; exit 23"}, []string{"PAIR_TEST_RUNNER_HELPER=1"}, time.Second)
	os.Stderr = original
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	if code := h.Wait(); code != 23 {
		t.Fatalf("exit=%d", code)
	}
	stream, got := launchOutputForTest(t, h)
	want := append(bytes.Repeat([]byte{'x'}, 16*1024-1), 255)
	if stream != "stderr" || !bytes.Equal(got, want) {
		t.Fatalf("stream=%q bytes=%d", stream, len(got))
	}
}

func TestLaunchOutputExecDescendantCannotHoldExit(t *testing.T) {
	started := time.Now()
	h, err := (ExecRunner{}).Start(t.TempDir(), []string{"sh", "-c", "sleep 10 & exit 0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Signal(os.Kill)
	select {
	case <-h.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("inherited stderr held direct exit indefinitely")
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("direct child exit=%d, want0 despite drain timeout", code)
	}
	t.Logf("exit observed after %s with descendant stderr still open", time.Since(started))
}

func TestLaunchOutputStdinStdoutDescriptorsPreserved(t *testing.T) {
	cmd, err := buildExecCommand(t.TempDir(), []string{"sh"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Stdin != os.Stdin || cmd.Stdout != os.Stdout {
		t.Fatal("stdin/stdout no longer use original descriptors")
	}
}

func TestLaunchOutputWrapperUnsupportedIsUnavailable(t *testing.T) {
	r := NewFakeRunner()
	h, err := r.Start(t.TempDir(), []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Expose only Handle, representing a runner with no output capability.
	wrapped := newAcknowledgedHandle(struct{ Handle }{h}, nil)
	stream, output := launchOutputForTest(t, wrapped)
	if stream != "" || output != nil {
		t.Fatalf("unsupported output fabricated: %q %q", stream, output)
	}
	r.SetExited(h.ID(), 0)
}

func TestLaunchOutputConcurrentSnapshots(t *testing.T) {
	tee := newLaunchOutputTee(io.Discard)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_, _ = tee.Write(bytes.Repeat([]byte{255}, 1024))
		}
	}()
	for {
		got := tee.snapshot()
		if len(got) > 16*1024 {
			t.Fatalf("unbounded snapshot %d", len(got))
		}
		if len(got) > 0 {
			got[0] = 0
		}
		select {
		case <-done:
			if got := tee.snapshot(); !bytes.Equal(got, bytes.Repeat([]byte{255}, 16*1024)) {
				t.Fatal("snapshot mutation changed stream")
			}
			return
		default:
		}
	}
}

func TestLaunchOutputFakeWaitObservesDirectExitBeforeGroup(t *testing.T) {
	r := NewFakeRunner()
	h, err := r.Start(t.TempDir(), []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Signal(os.Kill)
	r.SetChildExited(h.ID())
	select {
	case <-h.Exited():
	default:
		t.Fatal("direct exit not published")
	}
	result := make(chan int, 1)
	go func() { result <- h.Wait() }()
	select {
	case code := <-result:
		if code != 0 {
			t.Fatalf("direct exit=%d", code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Wait blocked on live group after direct exit")
	}
	if !h.Alive() {
		t.Fatal("direct exit killed surviving group")
	}
	if err := h.Signal(os.Kill); err != nil {
		t.Fatal(err)
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("later group kill changed direct exit to %d", code)
	}
}
