//go:build darwin

package procutil

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
)

func TestDarwinIdentityReadClassifiesOnlyPositiveAbsence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		raw       []byte
		err, want error
	}{
		{"absent", nil, nil, unix.ESRCH},
		{"denied", nil, unix.EPERM, unix.EPERM},
		{"io", nil, unix.EIO, unix.EIO},
		{"invalid", nil, unix.EINVAL, unix.EINVAL},
		{"short", []byte{1}, nil, unix.EIO},
		{"wrong pid", make([]byte, unix.SizeofKinfoProc), nil, unix.EIO},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got, err := darwinIdentity(123, func(name string, args ...int) ([]byte, error) {
				calls++
				if name != "kern.proc.pid" || !reflect.DeepEqual(args, []int{123}) {
					t.Fatalf("wrong lookup %s %v", name, args)
				}
				return tc.raw, tc.err
			})
			if got != "" || !errors.Is(err, tc.want) || calls != 1 {
				t.Fatalf("identity=%q err=%v calls=%d", got, err, calls)
			}
		})
	}
}

type exitAfterCommandReader struct {
	nativeReader
	row  Process
	exit func()
	t    *testing.T
}

func (r exitAfterCommandReader) table() ([]Process, error) { return []Process{r.row}, nil }
func (r exitAfterCommandReader) command(pid int) (CommandInfo, error) {
	c, err := r.nativeReader.command(pid)
	if err != nil || c.Executable == "" || len(c.Args) == 0 {
		r.t.Fatalf("regression requires successful command before exit: %+v %v", c, err)
	}
	r.exit()
	return c, err
}

func TestSnapshotDarwinExitAfterSuccessfulCommand(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(exe, "-test.run=^TestNativeProcessHelper$")
	child.Env = append(os.Environ(), "PROCUTIL_HELPER=1")
	stdin, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close(); child.Process.Kill(); child.Wait() })
	rows, err := Table()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.PID != child.Process.Pid {
			continue
		}
		r := exitAfterCommandReader{t: t, row: row, exit: func() {
			stdin.Close()
			if err := child.Wait(); err != nil {
				t.Fatal(err)
			}
		}}
		got, err := snapshot(context.Background(), r)
		if err != nil || len(got) != 0 {
			t.Fatalf("confirmed reaped generation retained after successful argv: %+v err=%v", got, err)
		}
		return
	}
	t.Fatal("owned child missing from table")
}

func TestDarwinReapedChildIdentityEvidence(t *testing.T) {
	child := exec.Command("/usr/bin/true")
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	pid := child.Process.Pid
	raw, err := unix.SysctlRaw("kern.proc.pid", pid)
	_, fixedErr := unix.SysctlKinfoProc("kern.proc.pid", pid)
	t.Logf("reaped PID %d: raw bytes=%d err=%v; fixed-size reader=%v", pid, len(raw), err, fixedErr)
	if err != nil || len(raw) != 0 || !errors.Is(fixedErr, unix.EIO) {
		t.Fatalf("unexpected absent-process contract: bytes=%d err=%v fixed=%v", len(raw), err, fixedErr)
	}
}

func TestDarwinUnreapedChildStatus(t *testing.T) {
	pid := exitedUnreapedChild(t)
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("exited unreaped PID %d: kernel P_stat=%d (sys/proc.h SZOMB=5)", pid, info.Proc.P_stat)
	if info.Proc.P_stat != 5 {
		t.Fatalf("unexpected terminated status %d", info.Proc.P_stat)
	}
	if identity, err := nativeIdentity(pid); identity == "" || !errors.Is(err, unix.ESRCH) {
		t.Fatalf("terminal child became live identity: %q %v", identity, err)
	}
	if got := Identity(strconv.Itoa(pid)); got == "" {
		t.Fatal("public Identity lost unreaped generation token")
	}
}
