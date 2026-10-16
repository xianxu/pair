package diagnosticlog

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendUsesCompactReservationAndOneCheckpoint(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	payload := bytes.Repeat([]byte("payload-must-not-be-duplicated:"), 1000)
	initial, err := os.Stat(filepath.Join(directory(path), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	lastInode := fileIdentity(initial)
	publications := 0
	w.options.Fault = func(step string) error {
		if step != "append-intent" && step != "append-committed" {
			return nil
		}
		state, err := load(path)
		if err != nil {
			return err
		}
		if state.Appending != nil {
			t.Fatal("hot append still publishes a payload-bearing state intent")
		}
		body, err := os.ReadFile(filepath.Join(directory(path), "state.json"))
		if err != nil {
			return err
		}
		if bytes.Contains(body, []byte(base64.StdEncoding.EncodeToString(payload))) {
			t.Fatal("payload duplicated in metadata")
		}
		st, err := os.Stat(filepath.Join(directory(path), "state.json"))
		if err != nil {
			return err
		}
		if id := fileIdentity(st); id != lastInode {
			publications++
			lastInode = id
		}
		return nil
	}
	if n, err := w.Write(payload); err != nil || n != len(payload) {
		t.Fatalf("write=%d %v", n, err)
	}
	if publications != 1 {
		t.Fatalf("state publications=%d, want one checkpoint", publications)
	}
	reservation, err := os.ReadFile(filepath.Join(directory(path), ".append-reservation"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reservation) > 4096 || bytes.Contains(reservation, []byte(base64.StdEncoding.EncodeToString(payload))) {
		t.Fatalf("reservation is not compact: %d bytes", len(reservation))
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %v", err)
	}
}

func TestCompactReservationQuarantinesKilledPartialAndContinues(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("base")); err != nil {
		t.Fatal(err)
	}
	interrupted := errors.New("killed after partial syscall")
	w.options.appendWrite = func(f *os.File, b []byte) (int, error) { return f.Write(b[:2]) }
	w.options.Fault = func(step string) error {
		if step == "append-written" {
			return interrupted
		}
		return nil
	}
	if n, err := w.Write([]byte("tail")); n != 2 || !errors.Is(err, interrupted) {
		t.Fatalf("partial=%d %v", n, err)
	}
	w.Close()
	reopened, err := Open(path, opts)
	if err != nil {
		t.Fatalf("optional diagnostics did not continue after preserving partial generation: %v", err)
	}
	defer reopened.Close()
	if n, err := reopened.Write([]byte("next")); n != 4 || err != nil {
		t.Fatalf("fresh write=%d %v", n, err)
	}
	fresh, _ := os.ReadFile(path)
	if string(fresh) != "next" {
		t.Fatalf("unverified bytes adopted as current: %q", fresh)
	}
	names, _, done, err := generationPage(path, "", 100, false, opts)
	if err != nil || !done || len(names) != 1 {
		t.Fatalf("quarantine inventory=%v complete=%v err=%v", names, done, err)
	}
	quarantined := segmentPath(path, names[0])
	raw, err := os.ReadFile(quarantined)
	if err != nil || string(raw) != "baseta" {
		t.Fatalf("uncertain generation modified: %q %v", raw, err)
	}
	metadata, err := os.ReadFile(quarantined + ".json")
	if err != nil || !bytes.Contains(metadata, []byte(`"unverified_tail"`)) {
		t.Fatalf("quarantine lost uncertainty: %q %v", metadata, err)
	}
}

func TestCompactReservationStrictDecodingAndRecovery(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	stop := errors.New("reserved")
	w.options.Fault = func(step string) error {
		if step == "append-intent" {
			return stop
		}
		return nil
	}
	if _, err = w.Write([]byte("valid batch")); !errors.Is(err, stop) {
		t.Fatal(err)
	}
	body, err := os.ReadFile(reservationPath(path))
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeAppendReservation(body)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{append(body, body...), []byte(`{"version":1,"version":1}`), bytes.Repeat([]byte("x"), maxReservationBytes+1)} {
		if _, err := DecodeAppendReservation(bad); err == nil {
			t.Fatal("ambiguous reservation accepted")
		}
	}
	observed := r.Before
	observed.Size += int64(r.Length)
	observed.ModTime = observed.ModTime.Add(time.Second)
	next, err := ClassifyAppendRecovery(r.Before, observed, []byte("valid batch"), r)
	if err != nil || next.Size != observed.Size {
		t.Fatal(next, err)
	}
	if _, err := ClassifyAppendRecovery(r.Before, observed, []byte("wrong batch"), r); err == nil {
		t.Fatal("foreign tail certified")
	}
	observed.Identity.Inode++
	if _, err := ClassifyAppendRecovery(r.Before, observed, []byte("valid batch"), r); err == nil {
		t.Fatal("replacement certified")
	}
}

func FuzzDecodeAppendReservation(f *testing.F) {
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		r, err := DecodeAppendReservation(b)
		if err == nil {
			if len(b) > maxReservationBytes || ValidateAppendReservation(r) != nil {
				t.Fatal("decoder admitted invalid reservation")
			}
		}
	})
}
