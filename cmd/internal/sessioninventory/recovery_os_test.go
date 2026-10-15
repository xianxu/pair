package sessioninventory

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func recoveryOSFixture(t *testing.T) (OSRuntime, Artifact, string) {
	t.Helper()
	home := t.TempDir()
	r := NewOSRuntime(home, t.TempDir())
	a := recoveryCodexEntry().Artifact
	a.StorageRoot = "codex-sessions"
	p := filepath.Join(home, ".codex", "sessions", filepath.FromSlash(a.RelativePath))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	for _, record := range recoveryCodexRecords(`"cli"`) {
		raw = append(raw, record.Bytes...)
		raw = append(raw, '\n')
	}
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return r, a, p
}
func TestRecoveryOSImplementsBoundedCapability(t *testing.T) {
	r, a, _ := recoveryOSFixture(t)
	capability, ok := any(r).(RecoveryRuntime)
	if !ok {
		t.Fatal("OSRuntime lacks bounded capability")
	}
	ctx := context.Background()
	before, err := capability.StatRecoveryFile(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := capability.ReadRecoveryAt(ctx, before, 0, 64<<10)
	if err != nil || int64(len(data)) != before.Size {
		t.Fatalf("read=%d err=%v", len(data), err)
	}
	count, err := capability.VisitRecoveryFiles(ctx, r.NativeRoots(AgentCodex)[0], 100, func(e FileEntry) error { return nil })
	if err != nil || count < 4 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
func TestRecoveryOSBoundsCancelAndSpecialFiles(t *testing.T) {
	for _, kind := range []string{"limit", "cancel", "fifo", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			r, a, p := recoveryOSFixture(t)
			capability, ok := any(r).(RecoveryRuntime)
			if !ok {
				t.Fatal("missing capability")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "limit":
				_, err := capability.VisitRecoveryFiles(ctx, r.NativeRoots(AgentCodex)[0], 1, func(e FileEntry) error { return nil })
				if !errors.Is(err, ErrReadLimit) {
					t.Fatalf("limit err=%v", err)
				}
				return
			case "cancel":
				cancel()
			case "fifo", "symlink", "directory":
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "fifo":
					if err := unix.Mkfifo(p, 0600); err != nil {
						t.Fatal(err)
					}
				case "symlink":
					if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), p); err != nil {
						t.Fatal(err)
					}
				case "directory":
					if err := os.Mkdir(p, 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
			start := time.Now()
			_, _, err := capability.ReadRecoveryAt(ctx, FileEntry{Artifact: a}, 0, 1024)
			if err == nil || time.Since(start) > time.Second {
				t.Fatalf("special/cancel read err=%v duration=%v", err, time.Since(start))
			}
		})
	}
}

func TestRecoveryOSRejectsChunkGenerationReplacement(t *testing.T) {
	r, a, p := recoveryOSFixture(t)
	expected, err := r.StatRecoveryFile(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("changed bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.ReadRecoveryAt(context.Background(), expected, 0, 1024); err == nil {
		t.Fatal("read replacement under original fingerprint")
	}
}
