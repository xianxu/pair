package launcher

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Count actual reader calls, including an empty enumeration. Returning complete
// synthetic rows keeps ancestry, candidate selection and generation fences real.
type attachProcessReader struct {
	mu                       sync.Mutex
	rows                     []procutil.Process
	commands                 map[int]procutil.CommandInfo
	tables, argv, identities int
}

func (r *attachProcessReader) Table() ([]procutil.Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tables++
	return append([]procutil.Process(nil), r.rows...), nil
}
func (r *attachProcessReader) Command(pid int) (procutil.CommandInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.argv++
	c, ok := r.commands[pid]
	if !ok {
		return c, os.ErrNotExist
	}
	return c, nil
}
func (r *attachProcessReader) Identity(pid int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.identities++
	for _, row := range r.rows {
		if row.PID == pid {
			return row.Identity, nil
		}
	}
	return "", os.ErrNotExist
}
func (r *attachProcessReader) counts() (int, int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tables, r.argv, r.identities
}

type countedAttachRuntime struct {
	*couchSessionRuntime
	processOS    *OSRuntime
	beforeAttach func()
}

func (r *countedAttachRuntime) InZellijPane() bool            { return r.processOS.InZellijPane() }
func (r *countedAttachRuntime) SweepOrphanNvim(tags []string) { r.processOS.SweepOrphanNvim(tags) }
func (r *countedAttachRuntime) AttachSession(name, config string) (int, error) {
	r.beforeAttach()
	return r.fakeRuntime.AttachSession(name, config)
}

func TestWorkbenchPerformanceAttachTableBudgets(t *testing.T) {
	for _, kind := range []string{"selected-couch", "standalone", "continuation-fallthrough"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			reader := &attachProcessReader{rows: []procutil.Process{{PID: os.Getpid(), PPID: 1, Name: "pair", Identity: "client"}}, commands: map[int]procutil.CommandInfo{os.Getpid(): {Args: []string{"pair"}}}}
			source := procutil.NewSource(reader)
			processOS := NewOSRuntime(root, "/pair")
			processOS.Processes = source
			opts := managedSessionOptions("attach")
			opts.Env.DataDir = root
			expected := opts.Args.CouchSession.Name
			budget := 1
			rt := &countedAttachRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}, processOS: processOS}
			if kind != "selected-couch" {
				opts = baseOpts(LaunchArgs{ForcedTag: "work"})
				opts.Env.DataDir = root
				expected = "📁work"
				budget = 2
				scope := mustScope(t, opts.Env.Cwd)
				rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{SessionName: expected, Tag: "work", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName}}}
				if kind == "continuation-fallthrough" {
					opts.ContinueSlug = "retained-checkpoint"
					budget = 3
				}
			}
			rt.sessions = []Session{{Name: expected, State: SessionDetached}}
			handoffs := 0
			rt.beforeAttach = func() {
				handoffs++
				tables, _, _ := reader.counts()
				if tables > budget {
					t.Fatalf("full tables at handoff=%d, budget=%d", tables, budget)
				}
				if tables == 0 {
					t.Fatal("ancestry enumeration counter was disconnected")
				}
			}
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if err != nil || code != 0 || handoffs != 1 || len(rt.attached) != 1 || rt.attached[0] != expected || rt.launchCount != 0 {
				t.Fatalf("code=%d err=%v handoffs=%d attached=%v creates=%d stderr=%s", code, err, handoffs, rt.attached, rt.launchCount, &stderr)
			}
			tables, commands, identities := reader.counts()
			if tables > budget {
				t.Fatalf("tables=%d budget=%d", tables, budget)
			}
			if kind == "selected-couch" && (commands != 0 || identities != 0) {
				t.Fatalf("selected attach enumerated argv/identity: %d/%d", commands, identities)
			}
			if commands != identities {
				t.Fatalf("successful argv observations lost generation fences: argv=%d identity=%d", commands, identities)
			}
		})
	}
}

type countedOwnerIO struct {
	osSessionOwnerIO
	world *ownerPassWorld
}

func (w countedOwnerIO) SessionPresent(ctx context.Context, name string) (bool, error) {
	return w.world.SessionPresent(ctx, name)
}
func (w countedOwnerIO) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	return w.world.SessionPanes(ctx, name)
}
func (w countedOwnerIO) Socket(string) SocketState { return SocketPresent }

func TestWorkbenchPerformanceOwnerPassCountsEveryNativeFence(t *testing.T) {
	for _, n := range []int{1, 3, 21} {
		t.Run(fmt.Sprintf("callers=%d", n), func(t *testing.T) {
			reader := &attachProcessReader{commands: map[int]procutil.CommandInfo{}}
			world := newOwnerPassWorld(n)
			dir := nativeSocketFixtureRoot(t, fmt.Sprintf("session-%d", n-1))
			for i := 0; i < n; i++ {
				name := fmt.Sprintf("session-%d", i)
				socket := filepath.Join(dir, name)
				listener, err := net.Listen("unix", socket)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := listener.Close(); err != nil {
						t.Error(err)
					}
				})
				reader.rows = append(reader.rows, procutil.Process{PID: i + 10, Name: "zellij", Identity: "generation"})
				reader.commands[i+10] = procutil.CommandInfo{Args: []string{"zellij", "--server", socket}}
			}
			source := procutil.NewSource(reader)
			// Initial native capture and every after-pane/effect fence share the same
			// injected reader. Explicit IO replaces only CLI/socket effects, not scanning.
			probe := SessionOwnerProbe{Processes: source, IO: countedOwnerIO{osSessionOwnerIO: osSessionOwnerIO{processes: source}, world: world}}
			pass := NewSessionOwnerPass(t.Context(), nil, probe)
			for i := 0; i < n; i++ {
				name := fmt.Sprintf("session-%d", i)
				observed, err := pass.Probe(t.Context(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
				if err != nil || observed.State != SessionOwnerOwned {
					t.Fatalf("probe=%+v err=%v", observed, err)
				}
				if err := pass.Revalidate(t.Context(), observed); err != nil {
					t.Fatal(err)
				}
			}
			tables, commands, identities := reader.counts()
			if tables > 1+2*n {
				t.Fatalf("tables=%d budget=%d", tables, 1+2*n)
			}
			if tables == 0 || commands == 0 || identities == 0 || commands != n*tables || identities != commands || world.paneReads != 2*n {
				t.Fatalf("tables=%d budget=%d argv=%d identities=%d panes=%d", tables, 1+2*n, commands, identities, world.paneReads)
			}
			// Explicit IO precedence is behavioral: replacing native IO must not inspect
			// the configured source, even for an ownership probe with live fences.
			before, _, _ := reader.counts()
			explicit := SessionOwnerProbe{Processes: source, IO: newOwnerPassWorld(1)}
			observed, err := explicit.Probe(t.Context(), "session-0", "/tmp/data", "0123456789abcdef", "1-repo-1")
			if err != nil || observed.State != SessionOwnerOwned {
				t.Fatalf("explicit IO=%+v err=%v", observed, err)
			}
			if after, _, _ := reader.counts(); after != before {
				t.Fatalf("explicit IO consulted process source: before=%d after=%d", before, after)
			}
		})
	}
}

func (r *countedAttachRuntime) AttachSessionContext(lifetime, admission context.Context, session, config string) (int, error) {
	if err := admission.Err(); err != nil {
		return 1, err
	}
	return r.AttachSession(session, config)
}
