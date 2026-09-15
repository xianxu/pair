package diagnosticlog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestAppendInterruptionReconcilesObservedBytes(t *testing.T) {
	for _, phase := range []string{"append-intent", "append-written", "append-verified", "append-committed"} {
		t.Run(phase, func(t *testing.T) {
			path, now, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("base")); err != nil {
				t.Fatal(err)
			}
			crash := errors.New("interrupted")
			w.options.Fault = func(step string) error {
				if step == phase {
					return crash
				}
				return nil
			}
			n, err := w.Write([]byte("tail"))
			if !errors.Is(err, crash) {
				t.Fatalf("fault missing n%d %v", n, err)
			}
			expected := "basetail"
			if phase == "append-intent" {
				expected = "base"
				if n != 0 {
					t.Fatal(n)
				}
			} else if n != 4 {
				t.Fatal(n)
			}
			w.Close()
			w, err = Open(path, opts)
			if err != nil {
				t.Fatal("reopen could not recover", err)
			}
			if _, err := w.Write([]byte("next")); err != nil {
				t.Fatal(err)
			}
			w.Close()
			raw, err := os.ReadFile(path)
			if err != nil || string(raw) != expected+"next" {
				t.Fatalf("recovery duplicated/invented bytes %q %v", raw, err)
			}
			*now = now.Add(8 * 24 * time.Hour)
			if _, err := Collect(path, opts, 100); err != nil {
				t.Fatal("collection remains blocked", err)
			}
		})
	}
}

func TestAppendPartialFailurePreservesWriterRetrySemantics(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	fail := errors.New("short append")
	w.options.appendWrite = func(f *os.File, b []byte) (int, error) {
		n, err := f.Write(b[:2])
		if err != nil {
			return n, err
		}
		return n, fail
	}
	n, err := w.Write([]byte("record"))
	if n != 2 || !errors.Is(err, fail) {
		t.Fatalf("partial return %d %v", n, err)
	}
	w.options.appendWrite = nil
	if n, err := w.Write([]byte("cord")); n != 4 || err != nil {
		t.Fatal(n, err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "record" {
		t.Fatalf("recovery replayed unacknowledged suffix %q", raw)
	}
}

func TestAppendLargeWriteUsesBoundedIntents(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	chunks := 0
	w.options.Fault = func(step string) error {
		if step == "append-intent" {
			chunks++
			s, err := load(path)
			if err != nil {
				return err
			}
			if s.Appending == nil || len(s.Appending.Data) > maxAppendBytes {
				t.Fatal("unbounded intent")
			}
		}
		return nil
	}
	data := bytes.Repeat([]byte("x"), 2*maxAppendBytes+3)
	n, err := w.Write(data)
	if err != nil || n != len(data) || chunks != 3 {
		t.Fatalf("chunking n%d chunks%d %v", n, chunks, err)
	}
	raw, _ := os.ReadFile(path)
	if !bytes.Equal(raw, data) {
		t.Fatal("chunked payload changed")
	}
}

func TestAppendCanceledAfterPayloadRetainsRecoverableIntent(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.options.Context = ctx
	w.options.Fault = func(step string) error {
		if step == "append-written" {
			cancel()
		}
		return nil
	}
	n, err := w.Write([]byte("written"))
	if n != 7 || !errors.Is(err, context.Canceled) {
		t.Fatal(n, err)
	}
	w.Close()
	if _, err := Open(path, opts); err != nil {
		t.Fatal("cancelled append not recoverable", err)
	}
}

func TestAppendRecoveryRejectsChangedEvidence(t *testing.T) {
	for _, kind := range []string{"replacement", "truncated", "foreign-tail", "oversized-tail", "timestamp-only", "oversized-intent", "zero-clock", "conflicting-intent", "before-mismatch", "invalid-current-name"} {
		t.Run(kind, func(t *testing.T) {
			path, _, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("base")); err != nil {
				t.Fatal(err)
			}
			stop := errors.New("intent staged")
			w.options.Fault = func(step string) error {
				if step == "append-intent" {
					return stop
				}
				return nil
			}
			if _, err := w.Write([]byte("tail")); !errors.Is(err, stop) {
				t.Fatal(err)
			}
			w.Close()
			switch kind {
			case "replacement":
				if err := os.Rename(path, path+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("base"), 0600); err != nil {
					t.Fatal(err)
				}
			case "truncated":
				if err := os.Truncate(path, 2); err != nil {
					t.Fatal(err)
				}
			case "foreign-tail", "oversized-tail":
				data := []byte("X")
				if kind == "oversized-tail" {
					data = []byte("tailX")
				}
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.Write(data); err != nil {
					t.Fatal(err)
				}
				f.Close()
			case "timestamp-only":
				st, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, st.ModTime().Add(time.Second), st.ModTime().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
			default:
				s, err := load(path)
				if err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "oversized-intent":
					s.Appending.Data = make([]byte, maxAppendBytes+1)
				case "before-mismatch":
					s.Appending.Before.Size++
				case "invalid-current-name":
					s.Current.Name = "segment-0123456789abcdef0123456789abcdef.log"
					s.Appending.Before = s.Current
				case "zero-clock":
					s.Appending.At = time.Time{}
				case "conflicting-intent":
					g := s.Current
					g.Name = "segment-0123456789abcdef0123456789abcdef.log"
					s.Pending = &g
				}
				if err := save(path, s, true, opts); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Open(path, opts); err == nil {
				t.Fatal("changed evidence adopted")
			}
			if _, err := Collect(path, opts, 100); err == nil {
				t.Fatal("changed evidence collected")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("refusal modified payload", err)
			}
		})
	}
}

func TestAppendKilledPublisherReconcilesActualPayload(t *testing.T) {
	for _, phase := range []string{"append-intent", "append-written", "append-verified", "append-committed"} {
		t.Run(phase, func(t *testing.T) {
			path, _, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("base")); err != nil {
				t.Fatal(err)
			}
			w.Close()
			cmd := exec.Command(os.Args[0], "-test.run=^TestAppendPublisherChild$")
			cmd.Env = append(os.Environ(), "PAIR_APPEND_CHILD="+path, "PAIR_APPEND_PHASE="+phase)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := false
			defer func() {
				if !done {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
			select {
			case line := <-ready:
				if line != "interrupted\n" {
					t.Fatalf("append barrier %q", line)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("append barrier timeout")
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			done = true
			w, err = Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("next")); err != nil {
				t.Fatal(err)
			}
			w.Close()
			want := "basetailnext"
			if phase == "append-intent" {
				want = "basenext"
			}
			raw, _ := os.ReadFile(path)
			if string(raw) != want {
				t.Fatalf("recovery duplicated/invented bytes %q", raw)
			}
		})
	}
}

func TestAppendPublisherChild(t *testing.T) {
	path := os.Getenv("PAIR_APPEND_CHILD")
	if path == "" {
		return
	}
	opts := Options{Now: func() time.Time { return time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC) }, Proof: func(context.Context, string, []Registration) error { return nil }}
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.options.Fault = func(step string) error {
		if step == os.Getenv("PAIR_APPEND_PHASE") {
			fmt.Println("interrupted")
			for {
				time.Sleep(time.Hour)
			}
		}
		return nil
	}
	_, err = w.Write([]byte("tail"))
	t.Fatal("append barrier returned", err)
}

func TestAppendHotPathAvoidsSyncAndLifecycleRecoverySyncs(t *testing.T) {
	path, _, opts := fixture(t)
	syncs := 0
	opts.appendSync = func(f *os.File) error { syncs++; return f.Sync() }
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("first")); err != nil {
		t.Fatal(err)
	}
	if syncs != 0 {
		t.Fatalf("foreground payload syncs%d", syncs)
	}
	stop := errors.New("after payload")
	w.options.Fault = func(step string) error {
		if step == "append-written" {
			return stop
		}
		return nil
	}
	if _, err := w.Write([]byte("pending")); !errors.Is(err, stop) {
		t.Fatal(err)
	}
	w.Close()
	if _, err := Collect(path, opts, 100); err != nil {
		t.Fatal(err)
	}
	if syncs != 1 {
		t.Fatalf("durable append recovery syncs%d", syncs)
	}
}

func TestRotationPayloadSyncFailurePrecedesDurableIntent(t *testing.T) {
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("kept")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	*now = now.Add(25 * time.Hour)
	failed := errors.New("payload sync failed")
	opts.appendSync = func(*os.File) error { return failed }
	if err := Maintain(path, opts); !errors.Is(err, failed) {
		t.Fatalf("failed sync ignored %v", err)
	}
	state, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Pending != nil {
		t.Fatal("rotation intent published before payload sync")
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "kept" {
		t.Fatal("sync failure changed current payload", err)
	}
	opts.appendSync = nil
	if err := Maintain(path, opts); err != nil {
		t.Fatal("durable retry failed", err)
	}
}
