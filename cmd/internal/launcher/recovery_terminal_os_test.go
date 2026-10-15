package launcher

import (
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

func recoveryNativeRows() ([]procutil.Observation, []procutil.Process) {
	server := procutil.Process{PID: 10, PPID: 1, Identity: "server-start", Name: "zellij"}
	editor := procutil.Process{PID: 11, PPID: 10, Identity: "editor-start", Name: "nvim"}
	return []procutil.Observation{
		{Process: server, CommandInfo: procutil.CommandInfo{Args: []string{"zellij", "--server", "/tmp/sockets/session"}}},
		{Process: editor, CommandInfo: procutil.CommandInfo{Args: []string{"nvim", "/data/repos/0123456789abcdef/draft-work.md"}}},
	}, []procutil.Process{server, editor}
}

func TestRecoveryNativeProjectionPreservesStableSnapshotEvidence(t *testing.T) {
	rows, after := recoveryNativeRows()
	got, err := projectRecoveryTerminalSnapshot("session", rows, after)
	if err != nil || len(got.Servers) != 1 || got.Servers[0].Identity != "server-start" || got.Commands[11] != "nvim /data/repos/0123456789abcdef/draft-work.md" || len(got.Processes) != 2 {
		t.Fatalf("snapshot=%+v err=%v", got, err)
	}
}

func TestRecoveryNativeProjectionRefusesMovedNamedServer(t *testing.T) {
	for _, change := range []string{"identity", "parent", "gone"} {
		t.Run(change, func(t *testing.T) {
			rows, after := recoveryNativeRows()
			switch change {
			case "identity":
				after[0].Identity = "replacement"
			case "parent":
				after[0].PPID = 99
			default:
				after = after[1:]
			}
			if _, err := projectRecoveryTerminalSnapshot("session", rows, after); err == nil {
				t.Fatal("changed named server retained attribution")
			}
		})
	}
}

func TestRecoveryNativeProjectionWithholdsChangedOrUnreadableDescendantCommand(t *testing.T) {
	for _, change := range []string{"identity", "parent", "unavailable", "empty"} {
		t.Run(change, func(t *testing.T) {
			rows, after := recoveryNativeRows()
			switch change {
			case "identity":
				after[1].Identity = "replacement"
			case "parent":
				after[1].PPID = 12
			case "unavailable":
				rows[1].Err = errors.New("denied")
			default:
				rows[1].Args = nil
			}
			got, err := projectRecoveryTerminalSnapshot("session", rows, after)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := got.Commands[11]; ok {
				t.Fatal("uncertain descendant command retained")
			}
			if len(got.Processes) != 2 {
				t.Fatal("unknown process ancestry evidence omitted")
			}
			if change != "parent" {
				owner, err := artifactpath.NewStorageOwner("/data", "0123456789abcdef", "work")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := attributeRecoveryServers(owner, "session", got); err == nil {
					t.Fatal("missing descendant command became attribution authority")
				}
			}
		})
	}
}

func TestRecoveryNativeProjectionOmitsGoneServersButRefusesUnknownEvidence(t *testing.T) {
	rows, after := recoveryNativeRows()
	rows[0].Err = syscall.ESRCH
	got, err := projectRecoveryTerminalSnapshot("session", rows, after)
	if err != nil || len(got.Servers) != 0 {
		t.Fatalf("gone snapshot=%+v err=%v", got, err)
	}
	rows[0].Err = syscall.EPERM
	if _, err := projectRecoveryTerminalSnapshot("session", rows, after); err == nil {
		t.Fatal("permission failure became server absence")
	}
}

func TestRecoveryNativeProjectionKeepsTotalCommandBound(t *testing.T) {
	rows, after := recoveryNativeRows()
	rows[1].Args = []string{strings.Repeat("x", 8<<20)}
	if _, err := projectRecoveryTerminalSnapshot("session", rows, after); err == nil {
		t.Fatal("aggregate command bound bypassed")
	}
}

func TestRecoveryNativeProjectionRejectsDuplicateProcessRows(t *testing.T) {
	rows, after := recoveryNativeRows()
	if _, err := projectRecoveryTerminalSnapshot("session", append(rows, rows[0]), after); err == nil {
		t.Fatal("duplicate observed generation accepted")
	}
	if _, err := projectRecoveryTerminalSnapshot("session", rows, append(after, after[0])); err == nil {
		t.Fatal("duplicate final generation accepted")
	}
}

func TestRecoveryNativeObservationHonorsCanceledAndMissingContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newRecoveryTerminalOS().Observe(ctx, "session"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := newRecoveryTerminalOS().Observe(nil, "session"); err == nil {
		t.Fatal("nil context accepted")
	}
}

func TestRecoveryNativeProjectionPreservesArgvBoundariesWithoutAliases(t *testing.T) {
	rows, after := recoveryNativeRows()
	rows[1].Args = []string{"nvim", "/data with spaces/draft-work.md", "a'b"}
	got, err := projectRecoveryTerminalSnapshot("session", rows, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Args[11]) != 3 || got.Args[11][1] != "/data with spaces/draft-work.md" {
		t.Fatalf("argv=%q", got.Args[11])
	}
	rows[1].Args[1] = "mutated"
	if got.Args[11][1] == "mutated" {
		t.Fatal("observation retains mutable source argv")
	}
	parsed, err := splitObservedCommand(recoveryArgCommand(got.Args[11]))
	if err != nil || len(parsed) != 3 || parsed[2] != "a'b" {
		t.Fatalf("argv roundtrip=%q err=%v", parsed, err)
	}
}

// Native adapter conformance uses the same process source and signal seam as
// production, with a stateful generation reader instead of host processes.
type recoverySignalReader struct {
	row  procutil.Process
	args []string
	err  error
}

func (r *recoverySignalReader) Table() ([]procutil.Process, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.row.PID == 0 {
		return nil, nil
	}
	return []procutil.Process{r.row}, nil
}
func (r *recoverySignalReader) Command(int) (procutil.CommandInfo, error) {
	return procutil.CommandInfo{Args: r.args}, r.err
}
func (r *recoverySignalReader) Identity(int) (string, error) { return r.row.Identity, r.err }
func TestRecoveryNativeExactOwnerSignalBoundary(t *testing.T) {
	for _, kind := range []string{"exact", "gone", "reused", "reparented", "argv", "unknown", "last-identity", "canceled", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			witness := recoveryProcessWitness{PID: 13, PPID: 12, Identity: "native-start", Args: []string{"codex", "resume", "native"}}
			r := &recoverySignalReader{row: procutil.Process{PID: 13, PPID: 12, Identity: witness.Identity}, args: append([]string(nil), witness.Args...)}
			kills := 0
			o := recoveryTerminalOS{osSessionQuiescenceOps{processes: procutil.NewSource(r), processIdentity: func(string) string {
				if kind == "last-identity" {
					return "replacement"
				}
				return r.row.Identity
			}, killProcess: func(pid int) error {
				if pid != 13 {
					t.Fatalf("foreign PID %d", pid)
				}
				kills++
				return nil
			}}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "gone":
				r.row = procutil.Process{}
			case "reused":
				r.row.Identity = "replacement"
			case "reparented":
				r.row.PPID = 1
			case "argv":
				r.args = []string{"codex", "resume", "foreign"}
			case "unknown":
				r.err = errors.New("unreadable")
			case "canceled":
				cancel()
			case "invalid":
				witness.PID = 0
			}
			err := o.KillProcess(ctx, witness)
			if kind == "exact" {
				if err != nil || kills != 1 {
					t.Fatalf("exact signal kills=%d err=%v", kills, err)
				}
			} else {
				if kills != 0 {
					t.Fatalf("unsafe signal %s", kind)
				}
				if kind != "gone" && err == nil {
					t.Fatal("changed/unknown evidence accepted")
				}
			}
		})
	}
}
