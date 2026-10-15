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
		probeErr  error
		probes    int
	}{
		{"empty alive", nil, nil, unix.EIO, nil, 1},
		{"empty absent", nil, nil, unix.ESRCH, unix.ESRCH, 1},
		{"empty denied", nil, nil, unix.EIO, unix.EPERM, 1},
		{"empty io", nil, nil, unix.EIO, unix.EIO, 1},
		{"empty enoent is not ESRCH", nil, nil, unix.EIO, unix.ENOENT, 1},
		{"denied", nil, unix.EPERM, unix.EPERM, nil, 0},
		{"io", nil, unix.EIO, unix.EIO, nil, 0},
		{"invalid", nil, unix.EINVAL, unix.EINVAL, nil, 0},
		{"short", []byte{1}, nil, unix.EIO, nil, 0},
		{"wrong pid", make([]byte, unix.SizeofKinfoProc), nil, unix.EIO, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			probes := 0
			got, err := darwinIdentity(123, func(name string, args ...int) ([]byte, error) {
				calls++
				if name != "kern.proc.pid" || !reflect.DeepEqual(args, []int{123}) {
					t.Fatalf("wrong lookup %s %v", name, args)
				}
				return tc.raw, tc.err
			}, func(pid int) error {
				probes++
				if pid != 123 {
					t.Fatalf("wrong existence PID %d", pid)
				}
				return tc.probeErr
			})
			if got != "" || !errors.Is(err, tc.want) || calls != 1 || probes != tc.probes {
				t.Fatalf("identity=%q err=%v calls=%d probes=%d", got, err, calls, probes)
			}
			if tc.want != unix.ESRCH && IsProcessGone(&IdentityUnavailableError{Err: err}) {
				t.Fatalf("uncertainty became absence: %v", err)
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
