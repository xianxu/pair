package main

import (
	"bytes"
	"testing"
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
