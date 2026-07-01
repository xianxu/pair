package wrapcmd

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestStdoutBatcherAccumulatesAndFlushes(t *testing.T) {
	var b stdoutBatcher
	b.append([]byte("ab"))
	b.append([]byte("cd"))

	got, chunks := b.flush()
	if string(got) != "abcd" || chunks != 2 {
		t.Fatalf("flush = %q, %d chunks; want abcd, 2", got, chunks)
	}
	if b.pendingBytes() != 0 {
		t.Fatalf("pendingBytes after flush = %d, want 0", b.pendingBytes())
	}
}

func TestStdoutPumpFlushesOnlyOnTick(t *testing.T) {
	var out bytes.Buffer
	pump := newStdoutPump(&out)

	pump.queue([]byte("a"))
	pump.queue([]byte("b"))
	if out.String() != "" {
		t.Fatalf("stdout written before tick: %q", out.String())
	}

	rec := pump.flush("tick")
	if out.String() != "ab" {
		t.Fatalf("stdout after first tick = %q, want ab", out.String())
	}
	if rec.Chunks != 2 || rec.Bytes != 2 || rec.Reason != "tick" {
		t.Fatalf("flush record = %#v, want 2 chunks/2 bytes/tick", rec)
	}

	pump.queue([]byte("c"))
	if out.String() != "ab" {
		t.Fatalf("stdout wrote before second tick: %q", out.String())
	}
	pump.flush("tick")
	if out.String() != "abc" {
		t.Fatalf("stdout after second tick = %q, want abc", out.String())
	}
}

func TestStdoutPumpFlushesOnEOF(t *testing.T) {
	var out bytes.Buffer
	pump := newStdoutPump(&out)
	pump.queue([]byte("pending"))

	rec := pump.flush("eof")
	if out.String() != "pending" {
		t.Fatalf("stdout after EOF = %q, want pending", out.String())
	}
	if rec.Reason != "eof" {
		t.Fatalf("reason = %q, want eof", rec.Reason)
	}
}

func TestMasterPumpFlushesStdoutOnTick(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	var out bytes.Buffer
	p := &proxy{
		ptmx:             reader,
		agentBasename:    "claude",
		stdoutPump:       newStdoutPump(&out),
		stdoutFlushEvery: 5 * time.Millisecond,
		captureWindow:    defaultCaptureWindow,
		notifyModeActive: notifyModeDefault,
		now:              time.Now,
	}
	done := make(chan struct{})
	go func() {
		p.masterPump()
		close(done)
	}()

	if _, err := writer.Write([]byte("tick")); err != nil {
		t.Fatal(err)
	}
	waitForStdoutBatch(t, 250*time.Millisecond, func() bool {
		return out.String() == "tick"
	})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	waitForStdoutBatch(t, 250*time.Millisecond, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestMasterPumpFlushesStdoutOnEOF(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	var out bytes.Buffer
	p := &proxy{
		ptmx:             reader,
		agentBasename:    "claude",
		stdoutPump:       newStdoutPump(&out),
		stdoutFlushEvery: time.Hour,
		captureWindow:    defaultCaptureWindow,
		notifyModeActive: notifyModeDefault,
		now:              time.Now,
	}
	done := make(chan struct{})
	go func() {
		p.masterPump()
		close(done)
	}()

	if _, err := writer.Write([]byte("eof")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	waitForStdoutBatch(t, 250*time.Millisecond, func() bool {
		select {
		case <-done:
			return out.String() == "eof"
		default:
			return false
		}
	})
}

func waitForStdoutBatch(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
