//go:build darwin || linux

package vmcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSnapshotGitCollectsDescendantsAfterParentExit(t *testing.T) {
	for _, exit := range []string{"0", "7"} {
		t.Run(exit, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range []string{"ready", "alive", "child", "parent"} {
				if err := syscall.Mkfifo(filepath.Join(root, name), 0600); err != nil {
					t.Fatal(err)
				}
			}
			// No inherited host configuration: the executable itself is the fixture.
			script := `#!/bin/sh
root=$8
(
 exec 1>/dev/null 2>/dev/null
 exec 3>"$root/alive"
 printf ready >"$root/ready"
 /bin/cat <"$root/child" >&3
) &
read code <"$root/parent"
exit "$code"
`
			writeSnapshotFixture(t, filepath.Join(root, "git"), script)
			if err := os.Chmod(filepath.Join(root, "git"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", root)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := snapshotGit(ctx, root, "fixture"); done <- err }()
			alive := make(chan error, 1)
			go func() {
				f, err := os.Open(filepath.Join(root, "alive"))
				if err == nil {
					_, err = io.Copy(io.Discard, f)
					f.Close()
				}
				alive <- err
			}()
			ready := make(chan error, 1)
			go func() {
				f, err := os.Open(filepath.Join(root, "ready"))
				if err == nil {
					_, err = io.ReadAll(f)
					f.Close()
				}
				ready <- err
			}()
			select {
			case err := <-ready:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("child never ready")
			}
			gate, err := os.OpenFile(filepath.Join(root, "child"), os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer gate.Close()
			parent, err := os.OpenFile(filepath.Join(root, "parent"), os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(parent, exit)
			parent.Close()
			select {
			case err := <-done:
				if (err == nil) != (exit == "0") {
					t.Fatalf("lost parent status: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("Git parent did not return")
			}
			select {
			case err := <-alive:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("owned Git descendant survived parent exit")
			}
		})
	}
}

func TestSnapshotGroupConfirmationWaitsForPositiveAbsence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	observed := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		calls := 0
		done <- awaitSnapshotGroupGone(ctx, func() (bool, error) {
			calls++
			if calls == 1 {
				return true, nil
			}
			close(observed)
			select {
			case <-release:
				return false, nil
			case <-ctx.Done():
				return true, ctx.Err()
			}
		})
	}()
	select {
	case <-observed:
	case err := <-done:
		t.Fatalf("cleanup returned before absence: %v", err)
	case <-ctx.Done():
		t.Fatal("no confirmation probe")
	}
	select {
	case err := <-done:
		t.Fatalf("cleanup escaped held confirmation: %v", err)
	default:
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("cleanup did not join confirmation")
	}
	sentinel := fmt.Errorf("inspection unavailable")
	if err := awaitSnapshotGroupGone(ctx, func() (bool, error) { return false, sentinel }); err != sentinel {
		t.Fatalf("inspection error collapsed to absence: %v", err)
	}
	expired, stop := context.WithCancel(context.Background())
	stop()
	if err := awaitSnapshotGroupGone(expired, func() (bool, error) { return true, nil }); err == nil {
		t.Fatal("occupied group accepted after expiry")
	}
}

func TestOwnedGroupCleanupRequiresProofAfterAmbiguousSignal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	probeError := errors.New("group inspection unavailable")
	for _, tc := range []struct {
		name    string
		present func() (bool, error)
		want    error
	}{
		{"gone", func() (bool, error) { return false, nil }, nil},
		{"unknown", func() (bool, error) { return false, probeError }, probeError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := confirmOwnedGroupCleanup(ctx, func() error { return syscall.EPERM }, tc.present)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("ambiguous signal outcome: %v", err)
			}
		})
	}
	short, stop := context.WithTimeout(ctx, 10*time.Millisecond)
	defer stop()
	err := confirmOwnedGroupCleanup(short, func() error { return syscall.EPERM }, func() (bool, error) { return true, nil })
	if !errors.Is(err, syscall.EPERM) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost occupied-group uncertainty: %v", err)
	}
}
