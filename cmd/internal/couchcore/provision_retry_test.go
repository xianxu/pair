//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreationLeaseRetryBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name     string
		elapsed  time.Duration
		attempt  int
		released bool
		want     time.Duration
	}{
		{"first active", 0, 0, false, 2 * time.Millisecond},
		{"active clamps", 99 * time.Millisecond, 8, false, time.Millisecond},
		{"active expires", 100 * time.Millisecond, 0, false, 0},
		{"released continues", 100 * time.Millisecond, 6, true, 20 * time.Millisecond},
		{"released clamps", 1999 * time.Millisecond, 6, true, time.Millisecond},
		{"released expires", 2 * time.Second, 0, true, 0},
		{"turnover held", 200 * time.Millisecond, 6, false, 0},
		{"repeated release cannot renew", 3 * time.Second, 1000, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := creationLeaseRetry(tt.elapsed, tt.attempt, tt.released); got != tt.want {
				t.Fatalf("retry delay = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvisionLeaseRecordsReleaseHint(t *testing.T) {
	root := t.TempDir()
	lease, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	path := filepath.Join(root, CouchWorkspacesDir, "creation.lock")
	assert := func(want string) {
		t.Helper()
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("hint = %q, %v; want %q", got, err, want)
		}
	}
	assert("H")
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	assert("R")
	next, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	assert("H")
}

func TestProvisionLeaseReleaseErrorStillClosesDescriptor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	if err := os.WriteFile(path, []byte("H"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path) // Read-only: marking release must fail.
	if err != nil {
		t.Fatal(err)
	}
	lease := &HostCreationLease{file: file}
	if err = lease.Close(); err == nil {
		t.Fatal("release write failure lost")
	}
	if _, err = file.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("descriptor still open: %v", err)
	}
}

func TestProvisionLeaseUnknownHintsDoNotGrantAdmission(t *testing.T) {
	for _, hint := range []string{"", "?", "H"} {
		t.Run("hint="+hint, func(t *testing.T) {
			root := t.TempDir()
			owner, err := AcquireHostCreationLease(root)
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			if err = owner.File().Truncate(0); err != nil {
				t.Fatal(err)
			}
			if _, err = owner.File().WriteAt([]byte(hint), 0); err != nil {
				t.Fatal(err)
			}
			next, err := AcquireHostCreationLease(root)
			if next != nil {
				next.Close()
			}
			if !errors.Is(err, ErrHostCreationBusy) {
				t.Fatalf("live owner admitted: %v", err)
			}
		})
	}
}
