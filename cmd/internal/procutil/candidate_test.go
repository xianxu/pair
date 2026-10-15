package procutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"testing"
)

func TestSnapshotNamedReadsOnlyCandidates(t *testing.T) {
	w := &snapshotWorld{
		rows: []Process{
			{PID: 1, Identity: "one", Name: "other"},
			{PID: 2, Identity: "two", Name: "zellij"},
			{PID: 3, Identity: "three"},
			{PID: 4, Identity: "four", Name: "zellij", Zombie: true},
		},
		commands: map[int]CommandInfo{
			1: {Executable: "/zellij", Args: []string{"zellij"}},
			2: {Executable: "/other", Args: []string{"other"}},
			3: {Executable: "/unknown", Args: []string{"unknown"}},
		},
		identities: map[int]string{1: "one", 2: "two", 3: "three", 4: "four"},
	}
	got, err := snapshotSelected(context.Background(), w, func(p Process) bool { return processNameCandidate(p, "zellij") })
	if err != nil {
		t.Fatal(err)
	}
	if w.tableCalls != 1 || !reflect.DeepEqual(w.calls, []int{2, 3}) {
		t.Fatalf("argv reads=%v tables=%d, want only candidate PIDs [2 3] and one table", w.calls, w.tableCalls)
	}
	if len(got) != 2 || got[0].PID != 2 || got[1].PID != 3 || got[0].Executable != "/other" {
		t.Fatalf("metadata must select candidates independently of executable/argv: %+v", got)
	}
	w.calls = nil
	w.tableCalls = 0
	full, err := snapshot(context.Background(), w)
	if err != nil || w.tableCalls != 1 || !reflect.DeepEqual(w.calls, []int{1, 2, 3}) || len(full) != 3 || full[0].Executable != "/zellij" {
		t.Fatalf("full snapshot changed: got=%+v err=%v calls=%v tables=%d", full, err, w.calls, w.tableCalls)
	}
}

func TestProcessNameCandidate(t *testing.T) {
	for _, name := range []string{"", "zellij", "Zellij", "zellij-server", "other", "zellij\x00"} {
		for _, requested := range []string{"zellij", "other", ""} {
			p := Process{Name: name}
			got := processNameCandidate(p, requested)
			if name == "" || name == requested {
				if !got {
					t.Errorf("excluded possible candidate: name=%q requested=%q", name, requested)
				}
			} else if got {
				t.Errorf("selected known unrelated name=%q requested=%q", name, requested)
			}
		}
	}
}

func FuzzProcessNameCandidate(f *testing.F) {
	f.Add("zellij", "zellij")
	f.Add("", "zellij")
	f.Add("other", "zellij")
	f.Fuzz(func(t *testing.T, knownName, requested string) {
		selected := processNameCandidate(Process{Name: knownName}, requested)
		if knownName == "" || knownName == requested {
			if !selected {
				t.Fatalf("possible candidate excluded: name=%q requested=%q", knownName, requested)
			}
		} else if selected {
			t.Fatalf("known unrelated process selected: name=%q requested=%q", knownName, requested)
		}
	})
}

func TestSnapshotNamedPreservesReadFailuresAndGenerationFences(t *testing.T) {
	for _, readErr := range []error{syscall.EINVAL, syscall.ESRCH, os.ErrNotExist, syscall.EPERM, syscall.EACCES, syscall.EIO, ErrUnsupported, errors.New("malformed argv"), nil} {
		for _, finalIdentity := range []string{"old", "new", ""} {
			t.Run(fmt.Sprintf("%v/%q", readErr, finalIdentity), func(t *testing.T) {
				w := &snapshotWorld{
					rows:          []Process{{PID: 1, Identity: "old", Name: "zellij"}, {PID: 2, Identity: "other", Name: "other"}},
					commands:      map[int]CommandInfo{1: {Executable: "/zellij", Args: []string{"zellij"}}},
					commandErrors: map[int]error{1: readErr},
					identities:    map[int]string{1: "old"},
				}
				w.after = func(pid int) { w.identities[pid] = finalIdentity }
				got, err := snapshotSelected(context.Background(), w, func(p Process) bool { return processNameCandidate(p, "zellij") })
				if err != nil || w.tableCalls != 1 || !reflect.DeepEqual(w.calls, []int{1}) {
					t.Fatalf("got=%+v err=%v reads=%v tables=%d", got, err, w.calls, w.tableCalls)
				}
				gone := errors.Is(readErr, syscall.EINVAL) || errors.Is(readErr, syscall.ESRCH) || errors.Is(readErr, os.ErrNotExist)
				if gone || finalIdentity == "new" {
					if len(got) != 0 {
						t.Fatalf("retained gone generation: %+v", got)
					}
					return
				}
				if len(got) != 1 || got[0].PID != 1 {
					t.Fatalf("lost selected evidence: %+v", got)
				}
				if readErr != nil || finalIdentity == "" {
					if got[0].Err == nil || got[0].Executable != "" || len(got[0].Args) != 0 {
						t.Fatalf("unavailable evidence became command authority: %+v", got)
					}
					if readErr != nil && !errors.Is(got[0].Err, readErr) {
						t.Fatalf("lost error: %v", got[0].Err)
					}
				} else if got[0].Err != nil || got[0].Executable != "/zellij" {
					t.Fatalf("lost successful evidence: %+v", got)
				}
			})
		}
	}
}

func TestSnapshotNamedCancellationAndTableError(t *testing.T) {
	selectRow := func(p Process) bool { return processNameCandidate(p, "zellij") }
	ctx, cancel := context.WithCancel(context.Background())
	w := &snapshotWorld{
		rows:     []Process{{PID: 1, Identity: "other", Name: "other"}, {PID: 2, Identity: "two", Name: "zellij"}, {PID: 3, Identity: "three"}},
		commands: map[int]CommandInfo{2: {Args: []string{"zellij"}}},
		after:    func(int) { cancel() },
	}
	if got, err := snapshotSelected(ctx, w, selectRow); !errors.Is(err, context.Canceled) || got != nil || !reflect.DeepEqual(w.calls, []int{2}) {
		t.Fatalf("canceled capture: got=%+v err=%v calls=%v", got, err, w.calls)
	}
	w = &snapshotWorld{}
	if got, err := snapshotSelected(ctx, w, selectRow); !errors.Is(err, context.Canceled) || got != nil || w.tableCalls != 0 {
		t.Fatalf("pre-canceled capture: got=%+v err=%v tables=%d", got, err, w.tableCalls)
	}
	w = &snapshotWorld{err: os.ErrPermission}
	if got, err := snapshotSelected(context.Background(), w, selectRow); !errors.Is(err, os.ErrPermission) || got != nil || len(w.calls) != 0 {
		t.Fatalf("table failure: got=%+v err=%v calls=%v", got, err, w.calls)
	}
}

func TestSnapshotNamedKeepsUnavailableMetadataWithoutArgvRead(t *testing.T) {
	w := &snapshotWorld{rows: []Process{{PID: 1, Name: "zellij"}, {PID: 0, Identity: "bad"}, {PID: 2, Name: "other"}}}
	got, err := snapshotSelected(context.Background(), w, func(p Process) bool { return processNameCandidate(p, "zellij") })
	if err != nil || len(got) != 2 || len(w.calls) != 0 {
		t.Fatalf("got=%+v err=%v calls=%v", got, err, w.calls)
	}
	for _, row := range got {
		if row.Err == nil || row.Executable != "" || len(row.Args) != 0 {
			t.Fatalf("invalid metadata became authority: %+v", row)
		}
	}
}

func TestSnapshotNamedNativeSelf(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native inspection unsupported")
	}
	rows, err := Table()
	if err != nil {
		t.Fatal(err)
	}
	for _, self := range rows {
		if self.PID != os.Getpid() {
			continue
		}
		got, err := SnapshotNamed(context.Background(), self.Name)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, row := range got {
			if row.Name != "" && row.Name != self.Name {
				t.Fatalf("selected unrelated native row: %+v", row)
			}
			if row.PID == self.PID {
				found = true
				if row.Err != nil || row.Identity != Identity(strconv.Itoa(self.PID)) || !reflect.DeepEqual(row.Args, os.Args) {
					t.Fatalf("self command unavailable: %+v", row)
				}
			}
		}
		if !found {
			t.Fatal("native named snapshot lost self")
		}
		return
	}
	t.Fatal("native table lost self")
}
