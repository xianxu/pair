package procutil

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func darwinArgs(argc uint32, body string) []byte {
	b := make([]byte, 4)
	binary.NativeEndian.PutUint32(b, argc)
	return append(b, body...)
}
func TestParseDarwinProcArgs(t *testing.T) {
	raw := darwinArgs(3, "/real/app\x00\x00spoof\x00two words\x00\x00SECRET=value\x00")
	got, env, err := ParseDarwinProcArgs(raw)
	if err != nil || got.Executable != "/real/app" || !reflect.DeepEqual(got.Args, []string{"spoof", "two words", ""}) || string(env) != "SECRET=value\x00" {
		t.Fatalf("got %+v env=%q err=%v", got, env, err)
	}
	for _, raw := range [][]byte{nil, {1, 2, 3}, darwinArgs(0, "/app\x00"), darwinArgs(^uint32(0), "/app\x00a\x00"), darwinArgs(2, "/app\x00a\x00"), darwinArgs(1, "/app"), darwinArgs(1, "/app\x00a"), make([]byte, (2<<20)+1)} {
		if _, _, err := ParseDarwinProcArgs(raw); err == nil {
			t.Errorf("accepted malformed %d bytes", len(raw))
		}
	}
}
func TestDarwinArgCountBound(t *testing.T) {
	raw := darwinArgs(100001, "/app\x00first\x00"+strings.Repeat("\x00", 100000))
	if _, _, err := ParseDarwinProcArgs(raw); err == nil {
		t.Fatal("accepted excessive argv allocation")
	}
}
func FuzzParseDarwinProcArgs(f *testing.F) {
	f.Add(darwinArgs(1, "/app\x00app\x00A=B\x00"))
	f.Fuzz(func(t *testing.T, b []byte) {
		info, _, err := ParseDarwinProcArgs(b)
		if err == nil && (len(info.Args) == 0 || info.Executable == "") {
			t.Fatal("incomplete successful parse")
		}
	})
}
func TestLinuxParsersAndReader(t *testing.T) {
	fields := strings.Fields("S 3 0 0 0 0 0 0 0 0 0 0 0 0 0 0 7 0 0 123")
	raw := []byte("41 (a ) name) " + strings.Join(fields, " "))
	p, err := parseLinuxStat(41, raw)
	if err != nil || p.PPID != 3 || p.Name != "a ) name" || p.Nice != 7 || p.Identity != "123" {
		t.Fatalf("%+v %v", p, err)
	}
	for _, b := range [][]byte{nil, []byte("42 (x) S 3"), []byte(strings.Replace(string(raw), "123", "bad", 1)), []byte(strings.Replace(string(raw), "41 (", "42 (", 1))} {
		if _, err := parseLinuxStat(41, b); err == nil {
			t.Errorf("accepted %q", b)
		}
	}
	args, err := parseLinuxCmdline([]byte("spoof\x00two words\x00\x00"))
	if err != nil || !reflect.DeepEqual(args, []string{"spoof", "two words", ""}) {
		t.Fatalf("%q %v", args, err)
	}
	for _, b := range [][]byte{nil, []byte("missing terminator"), make([]byte, (2<<20)+1)} {
		if _, err := parseLinuxCmdline(b); err == nil {
			t.Error("accepted malformed cmdline")
		}
	}
	root := t.TempDir()
	dir := filepath.Join(root, "41")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/actual/executable", filepath.Join(dir, "exe")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "cmdline")
	if err := os.WriteFile(file, []byte("spoof\x00two words\x00"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readLinuxCommand(root, 41)
	if err != nil || got.Executable != "/actual/executable" || !reflect.DeepEqual(got.Args, []string{"spoof", "two words"}) {
		t.Fatalf("%+v %v", got, err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if got, err := readLinuxCommand(root, 41); err == nil || len(got.Args) > 0 {
		t.Fatalf("gone command retained: %+v %v", got, err)
	}
}
func FuzzLinuxParsers(f *testing.F) {
	f.Add([]byte("1 (x) S 0"))
	f.Fuzz(func(t *testing.T, b []byte) { parseLinuxStat(1, b); parseLinuxCmdline(b) })
}

type snapshotWorld struct {
	rows           []Process
	commands       map[int]CommandInfo
	commandErrors  map[int]error
	identities     map[int]string
	identityErrors map[int]error
	calls          []int
	tableCalls     int
	after          func(int)
	err            error
}

func (w *snapshotWorld) table() ([]Process, error) { w.tableCalls++; return w.rows, w.err }
func (w *snapshotWorld) command(pid int) (CommandInfo, error) {
	w.calls = append(w.calls, pid)
	c, ok := w.commands[pid]
	if w.after != nil {
		w.after(pid)
	}
	if !ok {
		return c, os.ErrNotExist
	}
	return c, w.commandErrors[pid]
}
func (w *snapshotWorld) identity(pid int) (string, error) {
	return w.identities[pid], w.identityErrors[pid]
}

func TestSnapshotIdentityEvidenceAfterCommand(t *testing.T) {
	for _, identityErr := range []error{os.ErrNotExist, syscall.ESRCH, syscall.EPERM, syscall.EACCES, syscall.EIO, syscall.EINVAL, ErrUnsupported, nil} {
		t.Run(fmt.Sprint(identityErr), func(t *testing.T) {
			w := &snapshotWorld{
				rows:           []Process{{PID: 1, Identity: "old", Name: "zellij"}},
				commands:       map[int]CommandInfo{1: {Executable: "/zellij", Args: []string{"zellij"}}},
				identityErrors: map[int]error{1: identityErr},
			}
			got, err := snapshot(context.Background(), w)
			if err != nil {
				t.Fatal(err)
			}
			if errors.Is(identityErr, os.ErrNotExist) || errors.Is(identityErr, syscall.ESRCH) {
				if len(got) != 0 {
					t.Fatalf("retained confirmed gone: %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Err == nil || got[0].Executable != "" || len(got[0].Args) != 0 {
				t.Fatalf("identity uncertainty became authority or absence: %+v", got)
			}
			if identityErr != nil && !errors.Is(got[0].Err, identityErr) {
				t.Fatalf("lost identity error: %v", got[0].Err)
			}
		})
	}
}
func TestSnapshotSkipsProcessExitDuringCommandRead(t *testing.T) {
	for _, gone := range []error{syscall.EINVAL, syscall.ESRCH, os.ErrNotExist} {
		w := &snapshotWorld{rows: []Process{{PID: 1, Identity: "old", Name: "zellij"}, {PID: 2, Identity: "live"}}, commands: map[int]CommandInfo{1: {}, 2: {Executable: "/app", Args: []string{"ok"}}}, commandErrors: map[int]error{1: fmt.Errorf("procargs: %w", gone)}, identities: map[int]string{1: "old", 2: "live"}}
		w.after = func(pid int) {
			if pid == 1 {
				delete(w.identities, pid)
			}
		}
		rows, err := snapshot(context.Background(), w)
		if err != nil || len(rows) != 1 || rows[0].PID != 2 || rows[0].Err != nil {
			t.Fatalf("gone %v retained: %+v %v", gone, rows, err)
		}
	}
}
func TestIsProcessGonePreservesInspectionUncertainty(t *testing.T) {
	for _, err := range []error{syscall.EINVAL, syscall.ESRCH, os.ErrNotExist} {
		if !IsProcessGone(fmt.Errorf("process read: %w", err)) {
			t.Fatalf("lost gone classification: %v", err)
		}
	}
	for _, err := range []error{nil, syscall.EPERM, syscall.EACCES, syscall.EIO, ErrUnsupported, errors.New("malformed argv")} {
		if IsProcessGone(err) {
			t.Fatalf("inspection failure hidden: %v", err)
		}
	}
}
func TestSnapshotFencesReusedAndMissingProcesses(t *testing.T) {
	w := &snapshotWorld{rows: []Process{{PID: 1, Identity: "old", Name: "unrelated"}, {PID: 2, Identity: "same"}, {PID: 3, Identity: "same"}}, commands: map[int]CommandInfo{1: {Executable: "/app", Args: []string{"old"}}, 2: {Executable: "/app", Args: []string{"ok"}}}, identities: map[int]string{1: "old", 2: "same", 3: "same"}}
	w.after = func(pid int) {
		if pid == 1 {
			w.identities[1] = "new"
		}
	}
	got, err := snapshot(context.Background(), w)
	if err != nil || len(got) != 1 || w.tableCalls != 1 || !reflect.DeepEqual(w.calls, []int{1, 2, 3}) {
		t.Fatalf("%+v %v calls=%v", got, err, w.calls)
	}
	if got[0].PID != 2 || got[0].Err != nil || got[0].Args[0] != "ok" {
		t.Fatalf("bad fences: %+v", got)
	}
}
func TestSnapshotCancellationAndTableError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &snapshotWorld{rows: []Process{{PID: 1, Identity: "one"}, {PID: 2, Identity: "two"}}, commands: map[int]CommandInfo{1: {Args: []string{"one"}}}, after: func(int) { cancel() }}
	if _, err := snapshot(ctx, w); !errors.Is(err, context.Canceled) || len(w.calls) != 1 {
		t.Fatalf("err=%v calls=%v", err, w.calls)
	}
	w = &snapshotWorld{}
	if _, err := snapshot(ctx, w); !errors.Is(err, context.Canceled) || w.tableCalls != 0 {
		t.Fatalf("pre-canceled read: %+v %v", w, err)
	}
	w = &snapshotWorld{err: os.ErrPermission}
	if _, err := snapshot(context.Background(), w); !errors.Is(err, os.ErrPermission) {
		t.Fatal(err)
	}
}

func TestSnapshotDiscardsPartialCommandAndExitedIdentity(t *testing.T) {
	w := &snapshotWorld{
		rows: []Process{{PID: 1, Identity: "one"}, {PID: 2, Identity: "two"}},
		commands: map[int]CommandInfo{
			1: {Executable: "/partial", Args: []string{"partial"}},
			2: {Executable: "/exited", Args: []string{"exited"}},
		},
		commandErrors: map[int]error{1: os.ErrPermission},
		identities:    map[int]string{1: "one", 2: "two"},
	}
	w.after = func(pid int) { delete(w.identities, pid) }
	rows, err := snapshot(context.Background(), w)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Err == nil || row.Executable != "" || len(row.Args) != 0 {
			t.Fatalf("retained unavailable command: %+v", row)
		}
	}
	if !errors.Is(rows[0].Err, os.ErrPermission) {
		t.Fatal("lost command error")
	}
}
func TestNativeProcessInspectionWithoutPS(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native inspection unsupported")
	}
	t.Setenv("PATH", t.TempDir())
	for _, pid := range []string{"", "0", "-1", "self", "2147483646"} {
		if Command(pid) != "" {
			t.Fatalf("command accepted %q", pid)
		}
	}
	self := strconv.Itoa(os.Getpid())
	if Command(self) == "" {
		t.Fatal("native self command unavailable")
	}
	rows, err := Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if r.PID == os.Getpid() {
			found = true
			if r.Err != nil || r.Identity != Identity(self) || r.Name == "" || r.Executable == "" || !reflect.DeepEqual(r.Args, os.Args) {
				t.Fatalf("self observation: %+v", r)
			}
		}
	}
	if !found {
		t.Fatal("self missing")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestNativeProcessHelper$", "--", "argument with spaces", "")
	cmd.Env = append(os.Environ(), "PROCUTIL_HELPER=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); cmd.Process.Kill(); cmd.Wait() }()
	pid := strconv.Itoa(cmd.Process.Pid)
	if got := Command(pid); !strings.Contains(got, "argument with spaces") {
		t.Fatalf("child command %q", got)
	}
	children := ProcessChildren()[self]
	found = false
	for _, child := range children {
		if child == pid {
			found = true
		}
	}
	if !found {
		t.Fatalf("child %s missing from %v", pid, children)
	}
}
func TestPIDRejectsTruncationToLiveProcess(t *testing.T) {
	pid := strconv.FormatInt(int64(os.Getpid())+(1<<32), 10)
	if _, ok := positivePID(pid); ok {
		t.Fatalf("out-of-range PID accepted: %s", pid)
	}
	if Identity(pid) != "" || Command(pid) != "" {
		t.Fatal("out-of-range PID inspected another process")
	}
}

func TestNativeProcessHelper(t *testing.T) {
	if os.Getenv("PROCUTIL_HELPER") == "1" {
		var b [1]byte
		os.Stdin.Read(b[:])
		os.Exit(0)
	}
}
