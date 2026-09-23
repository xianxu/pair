//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProvisionLeaseInherited(t *testing.T) {
	root := t.TempDir()
	lease, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command("sh", "-c", "read answer")
	child.ExtraFiles = []*os.File{lease.File()}
	pipe, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	if second, err := AcquireHostCreationLease(root); !errors.Is(err, ErrHostCreationBusy) {
		if second != nil {
			second.Close()
		}
		t.Fatalf("inherited lease lost: %v", err)
	}
	pipe.Close()
	child.Wait()
	next, err := AcquireHostCreationLease(root)
	if err != nil {
		t.Fatal(err)
	}
	next.Close()
	if _, err = os.Stat(filepath.Join(root, "couch-workspaces", "creation.lock")); err != nil {
		t.Fatal(err)
	}
}

func TestProvisionLeaseRefusesAliases(t *testing.T) {
	for _, kind := range []string{"directory", "file", "common"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			other := t.TempDir()
			switch kind {
			case "directory":
				os.Symlink(other, filepath.Join(root, "couch-workspaces"))
			case "file":
				os.Mkdir(filepath.Join(root, "couch-workspaces"), 0700)
				os.WriteFile(filepath.Join(other, "lock"), []byte("keep"), 0600)
				os.Symlink(filepath.Join(other, "lock"), filepath.Join(root, "couch-workspaces", "creation.lock"))
			case "common":
				alias := filepath.Join(root, "alias")
				os.Symlink(other, alias)
				root = alias
			}
			if lease, err := AcquireHostCreationLease(root); err == nil {
				lease.Close()
				t.Fatal("accepted alias")
			}
		})
	}
}

func TestProvisionLeaseParentDeath(t *testing.T) {
	if root := os.Getenv("PAIR_TEST_CREATION_LEASE_PARENT"); root != "" {
		lease, err := AcquireHostCreationLease(root)
		if err != nil {
			panic(err)
		}
		cmd := exec.Command("sleep", "30")
		cmd.ExtraFiles = []*os.File{lease.File()}
		if err = cmd.Start(); err != nil {
			panic(err)
		}
		fmt.Println(cmd.Process.Pid)
		os.Exit(0) // Abrupt parent exit: no deferred Close or explicit unlock.
	}
	root := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestProvisionLeaseParentDeath$")
	cmd.Env = append(os.Environ(), "PAIR_TEST_CREATION_LEASE_PARENT="+root)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Kill(pid, syscall.SIGKILL)
	if lease, err := AcquireHostCreationLease(root); !errors.Is(err, ErrHostCreationBusy) {
		if lease != nil {
			lease.Close()
		}
		t.Fatalf("parent death released inherited lock: %v", err)
	}
	syscall.Kill(pid, syscall.SIGKILL)
	deadline := time.Now().Add(3 * time.Second)
	for {
		lease, err := AcquireHostCreationLease(root)
		if err == nil {
			lease.Close()
			break
		}
		if !errors.Is(err, ErrHostCreationBusy) || time.Now().After(deadline) {
			t.Fatalf("dead producer retained lease: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
