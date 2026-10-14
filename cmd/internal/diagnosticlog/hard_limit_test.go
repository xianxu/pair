package diagnosticlog

import (
	"context"
	"errors"
	"os"
	"testing"
)

func hardLimitOptions(t *testing.T, o Options) Options {
	t.Helper()
	o.HardMaxBytes = true
	o.MaxBytes = 5
	return o
}

func TestHardLimitRejectsWholeRecord(t *testing.T) {
	for _, initial := range []string{"", "abc", "abcde"} {
		t.Run(initial, func(t *testing.T) {
			path, _, o := fixture(t)
			o = hardLimitOptions(t, o)
			o.Proof = func(context.Context, string, []Registration) error { return ErrUnknownWriters }
			w, err := Open(path, o)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if n, err := w.Write([]byte(initial)); err != nil || n != len(initial) {
				t.Fatalf("initial: %d %v", n, err)
			}
			for range 3 {
				n, err := w.Write([]byte("123456"[len(initial):]))
				if n != 0 || !errors.Is(err, ErrLimit) {
					t.Fatalf("over cap: %d %v", n, err)
				}
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != initial {
				t.Fatalf("payload = %q, %v", got, err)
			}
		})
	}
}

func TestHardLimitRotatesBeforeRecord(t *testing.T) {
	for _, synchronous := range []bool{true, false} {
		t.Run(map[bool]string{true: "synchronous", false: "asynchronous"}[synchronous], func(t *testing.T) {
			path, _, o := fixture(t)
			o = hardLimitOptions(t, o)
			o.SynchronousMaintenance = synchronous
			w, err := Open(path, o)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if _, err := w.Write([]byte("abc")); err != nil {
				t.Fatal(err)
			}
			n, err := w.Write([]byte("def"))
			if !synchronous {
				if n != 0 || err == nil {
					t.Fatalf("blocked append: %d %v", n, err)
				}
				w.maintenance.Wait()
				n, err = w.Write([]byte("def"))
			}
			if n != 3 || err != nil {
				t.Fatalf("rotated append: %d %v", n, err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "def" {
				t.Fatalf("current = %q, %v", got, err)
			}
		})
	}
}

func TestDefaultLimitAllowsGrowthWithoutProof(t *testing.T) {
	path, _, o := fixture(t)
	o.MaxBytes = 5
	o.Proof = func(context.Context, string, []Registration) error { return errors.New("unproven") }
	w, err := Open(path, o)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for range 2 {
		if n, err := w.Write([]byte("abcdef")); n != 6 || err != nil {
			t.Fatalf("default append: %d %v", n, err)
		}
	}
}
