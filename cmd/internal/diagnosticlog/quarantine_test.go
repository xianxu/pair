package diagnosticlog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func quarantineReservation() (generation, appendReservation) {
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	current := generation{Identity: identity{Device: 1, Inode: 2}, Start: at, LastWrite: at, ModTime: at, Size: 4}
	sum := sha256.Sum256([]byte("tail"))
	r := appendReservation{Version: 1, Before: current, Length: 4, Digest: hex.EncodeToString(sum[:]), At: at.Add(time.Second)}
	r.Checksum = reservationChecksum(r)
	return current, r
}
func TestPartialQuarantineClassification(t *testing.T) {
	current, r := quarantineReservation()
	for _, tc := range []struct {
		name   string
		mutate func(*generation, *generation, *appendReservation)
		want   bool
	}{
		{"partial", func(_, o *generation, _ *appendReservation) { o.Size += 2 }, true},
		{"same size", func(_, o *generation, _ *appendReservation) {}, false},
		{"shrink", func(_, o *generation, _ *appendReservation) { o.Size-- }, false},
		{"full", func(_, o *generation, _ *appendReservation) { o.Size += 4 }, false},
		{"extra", func(_, o *generation, _ *appendReservation) { o.Size += 5 }, false},
		{"replacement", func(_, o *generation, _ *appendReservation) { o.Size += 2; o.Identity.Inode++ }, false},
		{"checkpoint changed", func(c, o *generation, _ *appendReservation) { o.Size += 2; c.LastWrite = c.LastWrite.Add(time.Second) }, false},
		{"missing mtime", func(_, o *generation, _ *appendReservation) { o.Size += 2; o.ModTime = time.Time{} }, false},
		{"bad checksum", func(_, o *generation, r *appendReservation) { o.Size += 2; r.Checksum = "bad" }, false},
		{"bad digest", func(_, o *generation, r *appendReservation) {
			o.Size += 2
			r.Digest = "bad"
			r.Checksum = reservationChecksum(*r)
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, o, rr := current, current, r
			o.ModTime = o.ModTime.Add(time.Second)
			tc.mutate(&c, &o, &rr)
			g, err := ClassifyPartialQuarantine(c, o, rr)
			if (err == nil) != tc.want {
				t.Fatal(g, err)
			}
			if err == nil && (g.Identity != current.Identity || g.Size != o.Size || g.UnverifiedTail == nil || g.UnverifiedTail.Offset != current.Size || g.UnverifiedTail.Length != r.Length || g.UnverifiedTail.Digest != r.Digest || !validUnverifiedTail(g)) {
				t.Fatalf("uncertainty lost: %+v", g)
			}
		})
	}
}
func FuzzPartialQuarantineClassification(f *testing.F) {
	for _, size := range []int64{-1, 0, 3, 4, 5, 7, 8, 9, 1 << 62} {
		f.Add(size, false)
		f.Add(size, true)
	}
	f.Fuzz(func(t *testing.T, size int64, replaced bool) {
		current, r := quarantineReservation()
		observed := current
		observed.Size = size
		observed.ModTime = observed.ModTime.Add(time.Second)
		if replaced {
			observed.Identity.Inode++
		}
		g, err := ClassifyPartialQuarantine(current, observed, r)
		want := !replaced && size > current.Size && size < current.Size+int64(r.Length)
		if (err == nil) != want {
			t.Fatalf("size=%d replaced=%t result=%+v err=%v", size, replaced, g, err)
		}
		if err == nil && !validUnverifiedTail(g) {
			t.Fatal("invalid marker")
		}
	})
}

func killedPartialForQuarantine(t *testing.T) (string, Options) {
	t.Helper()
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("base")); err != nil {
		t.Fatal(err)
	}
	stop := errors.New("killed partial")
	w.options.appendWrite = func(f *os.File, b []byte) (int, error) { return f.Write(b[:2]) }
	w.options.Fault = func(step string) error {
		if step == "append-written" {
			return stop
		}
		return nil
	}
	if n, err := w.Write([]byte("tail")); n != 2 || !errors.Is(err, stop) {
		t.Fatal(n, err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	return path, opts
}
func assertPreservedQuarantine(t *testing.T, path string, opts Options) {
	t.Helper()
	names, _, done, err := generationPage(path, "", 100, false, opts)
	if err != nil || !done || len(names) != 1 {
		t.Fatal(names, done, err)
	}
	segment := segmentPath(path, names[0])
	body, err := os.ReadFile(segment)
	if err != nil || string(body) != "baseta" {
		t.Fatalf("preserved bytes=%q err=%v", body, err)
	}
	var g generation
	if err := readJSON(segment+".json", &g); err != nil {
		t.Fatal(err)
	}
	if g.UnverifiedTail == nil || g.UnverifiedTail.Offset != 4 || g.UnverifiedTail.Length != 4 || !validUnverifiedTail(g) {
		t.Fatalf("lost uncertainty: %+v", g)
	}
}
func TestPartialQuarantineRotationReplay(t *testing.T) {
	for _, step := range []string{"intent", "rename", "state"} {
		t.Run(step, func(t *testing.T) {
			path, opts := killedPartialForQuarantine(t)
			stop := errors.New("interrupted " + step)
			faulted := false
			interruptedOpts := opts
			interruptedOpts.Fault = func(s string) error {
				if s == step {
					faulted = true
					return stop
				}
				return nil
			}
			if w, err := Open(path, interruptedOpts); !errors.Is(err, stop) {
				if w != nil {
					w.Close()
				}
				t.Fatalf("fault=%t err=%v", faulted, err)
			}
			if !faulted {
				t.Fatal("missing interruption")
			}
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if n, err := w.Write([]byte("fresh")); n != 5 || err != nil {
				t.Fatal(n, err)
			}
			body, err := os.ReadFile(path)
			if err != nil || string(body) != "fresh" {
				t.Fatal(string(body), err)
			}
			assertPreservedQuarantine(t, path, opts)
		})
	}
}
func TestPartialQuarantineUnknownProofRetainsAndRefuses(t *testing.T) {
	path, opts := killedPartialForQuarantine(t)
	paths := []string{path, reservationPath(path), filepath.Join(directory(path), "state.json")}
	before := make([][]byte, len(paths))
	for i, p := range paths {
		var err error
		before[i], err = os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
	}
	opts.Proof = func(context.Context, string, []Registration) error { return ErrUnknownWriters }
	if w, err := Open(path, opts); !errors.Is(err, ErrUnknownWriters) {
		if w != nil {
			w.Close()
		}
		t.Fatal(err)
	}
	for i, p := range paths {
		after, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(before[i], after) {
			t.Fatalf("evidence changed: %s %v", p, err)
		}
	}
	names, _, _, err := generationPage(path, "", 100, false, opts)
	if err != nil || len(names) != 0 {
		t.Fatal(names, err)
	}
}
func TestQuarantineRejectsForeignEffects(t *testing.T) {
	for _, mode := range []string{"wrong digest", "replacement", "shrink", "same size"} {
		t.Run(mode, func(t *testing.T) {
			path, _, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write([]byte("base")); err != nil {
				t.Fatal(err)
			}
			stop := errors.New("reserved")
			w.options.Fault = func(step string) error {
				if step == "append-intent" {
					return stop
				}
				return nil
			}
			if _, err = w.Write([]byte("tail")); !errors.Is(err, stop) {
				t.Fatal(err)
			}
			w.Close()
			switch mode {
			case "wrong digest":
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, err = f.WriteString("WRNG")
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
			case "replacement":
				if err = os.Rename(path, path+".foreign"); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, []byte("baseta"), 0600); err != nil {
					t.Fatal(err)
				}
			case "shrink":
				if err = os.Truncate(path, 2); err != nil {
					t.Fatal(err)
				}
			case "same size":
				if err = os.WriteFile(path, []byte("BASE"), 0600); err != nil {
					t.Fatal(err)
				}
				future := time.Now().Add(time.Hour)
				if err = os.Chtimes(path, future, future); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if reopened, err := Open(path, opts); err == nil {
				reopened.Close()
				t.Fatal("foreign change quarantined or adopted")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("evidence changed", err)
			}
			names, _, _, err := generationPage(path, "", 100, false, opts)
			if err != nil || len(names) != 0 {
				t.Fatal(names, err)
			}
		})
	}
}
func TestPartialQuarantineRegistersRecovererBeforeProof(t *testing.T) {
	path, opts := killedPartialForQuarantine(t)
	state, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	live := state.Writers[0]
	state.Writers = []Registration{{PID: live.PID, Birth: "other-generation", Root: live.Root}}
	if err = locked(path, false, func() error { return save(path, state, true, opts) }); err != nil {
		t.Fatal(err)
	}
	calls := 0
	probe := &fakeInspection{runtimes: []Runtime{{Process: live, Protocol: true}}}
	opts.Proof = func(ctx context.Context, p string, registered []Registration) error {
		calls++
		persisted, err := load(path)
		if err != nil {
			return err
		}
		found := false
		for _, r := range persisted.Writers {
			if r == live {
				found = true
			}
		}
		if !found {
			t.Error("recovering writer not published on disk before proof")
			return ErrUnknownWriters
		}
		return VerifyWriters(ctx, p, registered, probe)
	}
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if calls != 1 {
		t.Fatal("proof calls", calls)
	}
	if _, err = w.Write([]byte("fresh")); err != nil {
		t.Fatal(err)
	}
	assertPreservedQuarantine(t, path, opts)
}

func TestQuarantinedGenerationExpiresByCertifiedAge(t *testing.T) {
	path, opts := killedPartialForQuarantine(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	names, _, complete, err := generationPage(path, "", 10, false, opts)
	if err != nil || !complete || len(names) != 1 {
		t.Fatal(names, complete, err)
	}
	preserved := segmentPath(path, names[0])
	later := opts.Now().Add(RetentionPeriod + time.Hour)
	opts.Now = func() time.Time { return later }
	if _, err := Collect(path, opts, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(preserved); !os.IsNotExist(err) {
		t.Fatal("expired quarantine remains", err)
	}
}
