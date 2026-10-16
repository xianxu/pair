package archivetest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func TestMemorySinkRetainsFailedPublication(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("sync")
	s := &MemorySink{AfterPutError: boom}
	b := []byte("ciphertext")
	if !errors.Is(s.Put(ctx, "kind/b", bytes.NewReader(b), 100), boom) {
		t.Fatal("missing late error")
	}
	r, err := s.Get(ctx, "kind/b")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if !bytes.Equal(got, b) {
		t.Fatal("published bytes lost")
	}
	s.AfterPutError = nil
	if err := s.Put(ctx, "kind/b", strings.NewReader("replace"), 100); !errors.Is(err, os.ErrExist) {
		t.Fatalf("replacement allowed: %v", err)
	}
	if err := s.Verify(ctx, "kind/b", digest(b), int64(len(b))); err != nil {
		t.Fatal(err)
	}
	s.VerifyError = boom
	if err := s.Verify(ctx, "kind/b", digest(b), int64(len(b))); !errors.Is(err, boom) {
		t.Fatal("missing repeat verify failure")
	}
	s.VerifyError = nil
	if err := s.Corrupt("kind/b", []byte("bad")); err != nil {
		t.Fatal(err)
	}
	if s.Verify(ctx, "kind/b", digest(b), int64(len(b))) == nil {
		t.Fatal("corruption accepted")
	}
	keys, err := s.List(ctx, "kind/")
	if err != nil || len(keys) != 1 || keys[0] != "kind/b" {
		t.Fatalf("list %v %v", keys, err)
	}
}
func TestMemorySinkFailureBoundsAndCancellation(t *testing.T) {
	ctx := context.Background()
	s := &MemorySink{BeforePutError: errors.New("offline")}
	if s.Put(ctx, "a", strings.NewReader("b"), 1) == nil {
		t.Fatal("failure ignored")
	}
	if _, err := s.Get(ctx, "a"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected object %v", err)
	}
	s.BeforePutError = nil
	if s.Put(ctx, "a", strings.NewReader("large"), 1) == nil {
		t.Fatal("limit ignored")
	}
	c, cancel := context.WithCancel(ctx)
	s.Barrier = func(ctx context.Context, stage, key string) error {
		if stage == "before-put" {
			cancel()
		}
		return nil
	}
	if err := s.Put(c, "a", strings.NewReader("x"), 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v", err)
	}
	if len(s.Objects) != 0 {
		t.Fatal("canceled publish")
	}
}
func TestCodecStateAndReversibleMarker(t *testing.T) {
	ctx := context.Background()
	fp := strings.Repeat("A", 40)
	c := &TestCodec{KnownRecipients: map[string]bool{fp: true}}
	if c.CheckRecipients(ctx, []string{strings.Repeat("B", 40)}) == nil {
		t.Fatal("unknown key accepted")
	}
	input := []byte{0, 1, 2, 255}
	var out bytes.Buffer
	if err := c.Encode(ctx, bytes.NewReader(input), &out, []string{fp}); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeTestOutput(out.Bytes())
	if err != nil || !bytes.Equal(decoded, input) {
		t.Fatalf("decode %v", err)
	}
	if c.Attempts != 1 || len(c.Outputs) != 1 || !bytes.Equal(c.Outputs[0], out.Bytes()) {
		t.Fatal("attempt not retained")
	}
	out.Reset()
	c.AfterEncodeError = errors.New("late failure")
	if c.Encode(ctx, bytes.NewReader(input), &out, []string{fp}) == nil {
		t.Fatal("late failure ignored")
	}
	if c.Attempts != 2 || len(c.Outputs) != 2 || out.Len() == 0 {
		t.Fatal("late output not retained")
	}
}
func TestCodecBarrierCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fp := strings.Repeat("A", 40)
	c := &TestCodec{KnownRecipients: map[string]bool{fp: true}, Barrier: func(ctx context.Context, stage string) error { cancel(); return nil }}
	var out bytes.Buffer
	if err := c.Encode(ctx, strings.NewReader("x"), &out, []string{fp}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation %v", err)
	}
	if out.Len() != 0 || len(c.Outputs) != 0 {
		t.Fatal("canceled output")
	}
}

func TestCodecBlockedInputCancellationJoins(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, w := io.Pipe()
	defer w.Close()
	started := make(chan struct{})
	fp := strings.Repeat("A", 40)
	c := &TestCodec{KnownRecipients: map[string]bool{fp: true}, Barrier: func(ctx context.Context, stage string) error {
		if stage == "before-encode" {
			close(started)
		}
		return nil
	}}
	done := make(chan error, 1)
	go func() { done <- c.Encode(ctx, r, io.Discard, []string{fp}) }()
	<-started
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled transform succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocked reader survived cancellation")
	}
}
func TestMemorySinkCopiesCallerBytes(t *testing.T) {
	ctx := context.Background()
	s := &MemorySink{}
	data := []byte("original")
	if err := s.Put(ctx, "b", bytes.NewReader(data), 100); err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	read, err := s.Get(ctx, "b")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Corrupt("b", []byte("changed")); err != nil {
		t.Fatal(err)
	}
	old, _ := io.ReadAll(read)
	read.Close()
	if string(old) != "original" {
		t.Fatal("Get did not return an independent snapshot")
	}
}
