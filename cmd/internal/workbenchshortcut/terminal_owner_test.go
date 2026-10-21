package workbenchshortcut

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

type terminalProcess struct {
	identity                string
	args                    []string
	commandErr, identityErr error
	afterCommand            func(*terminalProcess)
}

type terminalProcessWorld struct {
	processes                    map[int]*terminalProcess
	commands, identities, tables int
}

func (w *terminalProcessWorld) Table() ([]procutil.Process, error) {
	w.tables++
	return nil, errors.New("whole-host enumeration forbidden")
}
func (w *terminalProcessWorld) Command(pid int) (procutil.CommandInfo, error) {
	w.commands++
	p := w.processes[pid]
	if p == nil {
		return procutil.CommandInfo{}, os.ErrNotExist
	}
	args := append([]string(nil), p.args...)
	err := p.commandErr
	if p.afterCommand != nil {
		f := p.afterCommand
		p.afterCommand = nil
		f(p)
	}
	return procutil.CommandInfo{Args: args}, err
}
func (w *terminalProcessWorld) Identity(pid int) (string, error) {
	w.identities++
	p := w.processes[pid]
	if p == nil {
		return "", os.ErrNotExist
	}
	if p.identityErr != nil {
		return "", p.identityErr
	}
	return p.identity, p.identityErr
}

func TestTerminalOwnerArgs(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"/app/bin/pair", "term"}, true},
		{[]string{"/path with spaces/pair", "term"}, true},
		{[]string{"pair-go", "term"}, true},
		{[]string{"sh", "-i"}, false},
		{[]string{"sh", "-c", "pair term"}, false},
		{[]string{"not-pair", "term"}, false},
		{[]string{"pair", "terminal"}, false},
		{[]string{"pair", "term", "--test-shortcut", "Alt+k"}, false},
		{[]string{"pair", "term", "--help"}, false},
		{nil, false},
	} {
		if got := terminalOwnerArgs(tc.args); got != tc.want {
			t.Errorf("args %q: %v, want %v", tc.args, got, tc.want)
		}
	}
}

func TestTerminalOwnerEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		p      terminalProcess
		want   terminalOwnerState
		reason string
	}{
		{"wrapper", terminalProcess{identity: "one", args: []string{"pair", "term"}}, terminalOwnerPair, ""},
		{"shell same PID and birth", terminalProcess{identity: "one", args: []string{"sh", "-i"}}, terminalOwnerOther, ""},
		{"argv unavailable", terminalProcess{identity: "one", commandErr: errors.New("PRIVATE ARGUMENT")}, terminalOwnerUnavailable, "argv-unavailable"},
		{"identity unavailable", terminalProcess{identityErr: errors.New("PRIVATE IDENTITY")}, terminalOwnerUnavailable, "identity-before-unavailable"},
		{"identity lost after argv", terminalProcess{identity: "one", args: []string{"pair", "term"}, afterCommand: func(p *terminalProcess) { p.identityErr = os.ErrPermission }}, terminalOwnerUnavailable, "identity-after-unavailable"},
		{"replaced during argv", terminalProcess{identity: "one", args: []string{"pair", "term"}, afterCommand: func(p *terminalProcess) { p.identity = "replacement" }}, terminalOwnerOther, "identity-changed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &terminalProcessWorld{processes: map[int]*terminalProcess{42: &tc.p}}
			got := inspectTerminalOwner(procutil.NewSource(w), 42)
			if got.state != tc.want || got.reason != tc.reason {
				t.Fatalf("evidence %+v, want %v/%s", got, tc.want, tc.reason)
			}
			if w.tables != 0 || w.commands > 1 || w.identities > 2 {
				t.Fatalf("unbounded inspection: %+v", w)
			}
		})
	}
}

func TestLiveTerminalOwnersExecAndUnavailable(t *testing.T) {
	reg := TerminalPaneRegistry{DataDir: t.TempDir(), Tag: "test"}
	for _, entry := range []struct {
		id  string
		pid int
	}{{"4", 42}, {"7", 77}, {"4", 42}} {
		if err := reg.Register(entry.id, entry.pid); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.OpenFile(reg.Path(), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("future record with unknown fields\n")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(reg.Path())
	if err != nil {
		t.Fatal(err)
	}
	w := &terminalProcessWorld{processes: map[int]*terminalProcess{42: {identity: "same", args: []string{"pair", "term"}}, 77: {identity: "other", args: []string{"pair", "term"}}}}
	var reasons []string
	read := func(want []string) {
		t.Helper()
		ids, err := reg.liveOwnerIDs(func(int) bool { return true }, procutil.NewSource(w), func(reason string) { reasons = append(reasons, reason) })
		if err != nil || !reflect.DeepEqual(ids, want) {
			t.Fatalf("ids=%v err=%v want=%v", ids, err, want)
		}
	}
	read([]string{"4", "7"})
	w.processes[42].args = []string{"sh", "-i"}
	read([]string{"7"})
	w.processes[42].commandErr = os.ErrPermission
	read([]string{"4", "7"})
	if len(reasons) != 1 || reasons[0] != "argv-unavailable=1" {
		t.Fatalf("unavailable not recorded once: %q", reasons)
	}
	after, err := os.ReadFile(reg.Path())
	if err != nil || string(after) != string(before) {
		t.Fatalf("read changed registry/unknown lines: %v", err)
	}
}

func TestLiveTerminalOwnerInspectionCount(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			reg := TerminalPaneRegistry{DataDir: t.TempDir(), Tag: "count"}
			w := &terminalProcessWorld{processes: map[int]*terminalProcess{}}
			for i := 1; i <= n; i++ {
				w.processes[i] = &terminalProcess{identity: "birth", commandErr: os.ErrPermission}
				for j := 0; j < 3; j++ {
					if err := reg.Register(fmt.Sprint(i), i); err != nil {
						t.Fatal(err)
					}
				}
			}
			logs := 0
			ids, err := reg.liveOwnerIDs(func(int) bool { return true }, procutil.NewSource(w), func(string) { logs++ })
			if err != nil || len(ids) != n || w.commands != n || w.identities != 2*n || w.tables != 0 || logs != 1 {
				t.Fatalf("n=%d ids=%v err=%v command=%d identity=%d table=%d logs=%d", n, ids, err, w.commands, w.identities, w.tables, logs)
			}
			for _, p := range w.processes {
				p.commandErr = nil
				p.args = []string{"sh", "-i"}
			}
			w.commands, w.identities, logs = 0, 0, 0
			ids, err = reg.liveOwnerIDs(func(int) bool { return true }, procutil.NewSource(w), func(string) { logs++ })
			if err != nil || len(ids) != 0 || w.commands != n || w.identities != 2*n || logs != 0 {
				t.Fatalf("mismatch duplicate scan: ids=%v err=%v command=%d identity=%d logs=%d", ids, err, w.commands, w.identities, logs)
			}
		})
	}
}
