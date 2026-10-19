package diagnosticlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDeletionRecoveryEveryEntryPointProtectsExactSegment(t *testing.T) {
	for _, entry := range []string{"Open", "Write", "maintain", "direct"} {
		t.Run(entry, func(t *testing.T) {
			path, clock, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if _, err := w.Write([]byte("expired")); err != nil {
				t.Fatal(err)
			}
			*clock = clock.Add(25 * time.Hour)
			if _, err := w.Write([]byte("current")); err != nil {
				t.Fatal(err)
			}
			path = w.path
			names, _, _, err := generationPage(path, "", 10, false, opts)
			if err != nil || len(names) != 1 {
				t.Fatal(names, err)
			}
			target := segmentPath(path, names[0])
			var g generation
			if err := readJSON(target+".json", &g); err != nil {
				t.Fatal(err)
			}
			state, err := load(path)
			if err != nil {
				t.Fatal(err)
			}
			state.Deleting = &g
			if err := save(path, state, true, opts); err != nil {
				t.Fatal(err)
			}
			held := true
			targetChecks := 0
			// No ReaderProof is supplied: ordinary writer/recovery callers must target
			// their legacy Proof at the actual payload, not the current pathname.
			proof := func(_ context.Context, p string, _ []Registration) error {
				if p == target {
					targetChecks++
					if held {
						return ErrUnknownWriters
					}
				}
				return nil
			}
			opts.Proof = proof
			w.options.Proof = proof
			invoke := func() error {
				switch entry {
				case "Open":
					other, e := Open(path, opts)
					if other != nil {
						other.Close()
					}
					return e
				case "Write":
					_, e := w.Write([]byte("!"))
					return e
				case "maintain":
					return maintain(path, opts, 0)
				default:
					s, e := load(path)
					if e != nil {
						return e
					}
					return recoverDeletion(path, &s, opts)
				}
			}
			if err := invoke(); !errors.Is(err, ErrUnknownWriters) {
				t.Fatalf("%s replay ignored held target: %v checks=%d", entry, err, targetChecks)
			}
			if targetChecks == 0 {
				t.Fatal("exact target was never inspected")
			}
			if b, err := os.ReadFile(target); err != nil || string(b) != "expired" {
				t.Fatal("held segment changed", err)
			}
			held = false
			if err := invoke(); err != nil {
				t.Fatal("released target did not recover", err)
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatal("released segment remains", err)
			}
			if b, err := os.ReadFile(path); err != nil || string(b) != "current" && string(b) != "current!" {
				t.Fatal(fmt.Sprintf("unrelated current changed %q", b), err)
			}
		})
	}
}
