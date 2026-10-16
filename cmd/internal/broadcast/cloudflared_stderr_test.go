package broadcast

import (
	"os/exec"
	"testing"
)

// A small failure fits in the real pipe even when the reader has not run.
// Waiting first forces the ordering that used to close StderrPipe before the
// scanner could preserve the child's diagnostic. No scheduler race is needed.
func TestCloudflaredStderrSurvivesWaitBeforeDrain(t *testing.T) {
	const failure = "ERR failed to request quick Tunnel: Post https://api.trycloudflare.com/tunnel"
	bin, _ := fakeCloudflared(t, "echo '"+failure+"' >&2; exit 1")
	cmd := exec.Command(bin)
	stderr, writer, err := cloudflaredStderrPipe(cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close() })
	if writer != nil {
		t.Cleanup(func() { _ = writer.Close() })
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("fake failing process succeeded")
	}
	h := &cloudflaredHandle{}
	h.watchStderr(stderr, false, func(string) {})
	if got := h.lastLine(); got != failure {
		t.Fatalf("lost diagnostic after process wait: got %q, want %q", got, failure)
	}
}

func TestCloudflaredExitPublishesAfterStderrDrain(t *testing.T) {
	const failure = "ERR final child failure"
	bin, _ := fakeCloudflared(t, "echo '"+failure+"' >&2; exit 1")
	cmd := exec.Command(bin)
	stderr, writer, err := cloudflaredStderrPipe(cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close(); _ = writer.Close() })
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	h := &cloudflaredHandle{cmd: cmd, exited: make(chan struct{})}
	// The lifecycle owner's real wait runs first. Its drain effect performs the
	// real scan synchronously here, forcing the delayed-reader ordering without
	// a scheduling delay or an assertion on elapsed time.
	drained := false
	h.waitForExit(func() {
		select {
		case <-h.exited:
			t.Fatal("published exit before stderr drain")
		default:
		}
		h.watchStderr(stderr, false, func(string) {})
		drained = true
	})
	if !drained || h.lastLine() != failure {
		t.Fatalf("exit lost pending stderr: drained=%t last=%q", drained, h.lastLine())
	}
	select {
	case <-h.exited:
	default:
		t.Fatal("complete process and drain did not publish exit")
	}
}
