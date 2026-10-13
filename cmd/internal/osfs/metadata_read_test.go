package osfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestBoundedRegularMetadata(t *testing.T) {
	for _, kind := range []string{"regular", "at bound", "oversize", "fifo", "directory", "symlink", "missing"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metadata")
			var err error
			switch kind {
			case "regular":
				err = os.WriteFile(path, []byte("proof"), 0600)
			case "at bound":
				err = os.WriteFile(path, []byte(strings.Repeat("x", 32)), 0600)
			case "oversize":
				err = os.WriteFile(path, []byte(strings.Repeat("x", 33)), 0600)
			case "fifo":
				err = syscall.Mkfifo(path, 0600)
			case "directory":
				err = os.Mkdir(path, 0700)
			case "symlink":
				target := filepath.Join(filepath.Dir(path), "target")
				if err = os.WriteFile(target, []byte("proof"), 0600); err == nil {
					err = os.Symlink(target, path)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			done := make(chan struct{})
			var raw []byte
			var readErr error
			go func() { raw, readErr = ReadRegularMetadata(ctx, path, 32); close(done) }()
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("metadata read blocked past cancellation")
			}
			if kind == "regular" || kind == "at bound" {
				if readErr != nil || len(raw) == 0 || len(raw) > 32 {
					t.Fatalf("raw=%q err=%v", raw, readErr)
				}
			} else if readErr == nil || len(raw) != 0 {
				t.Fatalf("unsafe metadata accepted: %q %v", raw, readErr)
			}
			if kind == "oversize" || kind == "fifo" || kind == "directory" || kind == "symlink" {
				var invalid *InvalidMetadataError
				if !errors.As(readErr, &invalid) {
					t.Fatalf("static defect lacks typed error: %v", readErr)
				}
			}
			if kind == "missing" && !errors.Is(readErr, os.ErrNotExist) {
				t.Fatalf("lost missing classification: %v", readErr)
			}
		})
	}
}

func TestBoundedRegularMetadataCanceledBeforeObservation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ReadRegularMetadata(ctx, filepath.Join(t.TempDir(), "absent"), 32)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read: %v", err)
	}
}

func TestBoundedRegularMetadataRejectsInvalidBound(t *testing.T) {
	for _, bound := range []int64{0, -1, 1<<63 - 1} {
		if _, err := ReadRegularMetadata(t.Context(), "unused", bound); err == nil {
			t.Fatalf("accepted bound %d", bound)
		}
	}
}

func TestMetadataSnapshotRejectsReplacementInode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata")
	replacement := filepath.Join(dir, "replacement")
	if err := os.WriteFile(path, []byte("proof"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("proof"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(replacement, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if sameMetadataFile(before, after) {
		t.Fatal("same bytes, size and mtime hid a replacement inode")
	}
}
