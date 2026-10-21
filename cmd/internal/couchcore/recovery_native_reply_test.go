package couchcore

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// Only process identity is synthetic. Ownership commands, listings and client
// counts traverse the production query collector and strict reply admission.
type recoveryNativeProcesses struct{ socket string }

func (p recoveryNativeProcesses) Table() ([]procutil.Process, error) {
	return []procutil.Process{{PID: 636, Identity: "server-636", Name: "zellij"}}, nil
}
func (p recoveryNativeProcesses) Command(pid int) (procutil.CommandInfo, error) {
	if pid != 636 {
		return procutil.CommandInfo{}, fmt.Errorf("unexpected process %d", pid)
	}
	return procutil.CommandInfo{Executable: "/fixture/zellij", Args: []string{"zellij", "--server", p.socket}}, nil
}
func (p recoveryNativeProcesses) Identity(pid int) (string, error) {
	if pid != 636 {
		return "", fmt.Errorf("unexpected process %d", pid)
	}
	return "server-636", nil
}

type recoveryNativeFixture struct {
	env    *relaunchEnv
	native *zellijtest.Fixture
	source ThreadRecord
}

func newRecoveryNativeFixture(t *testing.T) recoveryNativeFixture {
	t.Helper()
	native := zellijtest.New(t)
	for _, entry := range native.Env() {
		key, value, _ := strings.Cut(entry, "=")
		t.Setenv(key, value)
	}
	if path, err := exec.LookPath("zellij"); err != nil || path != native.Path() {
		t.Fatalf("query isolation: path=%q err=%v", path, err)
	}
	env, source := switchEnvWithLiveThread(t)
	env.Proc.Kill(source.Incarnations[0].PID)
	data := t.TempDir()
	indexSession(t, data, source.Address, "pair-exact")
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: source.Address.RepoScope, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	command := fmt.Sprintf("nvim %q", paths.Draft())
	pane := zellijtest.DefaultPane(1, command)
	// Actual pane_command, not the layout's terminal command, proves ownership.
	pane["pane_command"] = command
	native.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-exact": {Panes: []zellijtest.Pane{pane}, Tab: zellijtest.DefaultTab()}}})
	// Unix socket paths have a small native limit; keep this private path short.
	//lint:allow tmp-literal — the ownership probe requires a real Unix socket with its session leaf; macOS TMPDIR can exceed the AF_UNIX address limit
	socketDir, err := os.MkdirTemp("/tmp", "pair636-socket-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })
	socket := filepath.Join(socketDir, "pair-exact")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	processes := procutil.NewSource(recoveryNativeProcesses{socket: socket})
	checker := NewScopedThreadArtifactCollisionChecker(data)
	deleter := &fakeSessionDeleter{}
	checker.Sessions = deleter
	checker.Zellij = launcher.ZellijSource{Path: native.Path()}
	checker.Servers = launcher.OSServerStates{Processes: processes}
	checker.OwnerProbe = launcher.SessionOwnerProbe{Processes: processes}
	env.Couch.Artifacts = checker
	t.Cleanup(func() {
		if len(deleter.deleted) != 0 {
			t.Errorf("recovery deleted sessions: %v", deleter.deleted)
		}
		if effects := native.Effects(); len(effects) != 0 {
			t.Errorf("recovery mutated native session: %v", effects)
		}
	})
	return recoveryNativeFixture{env: env, native: native, source: source}
}

func recoveryNativeInitialQueries() [][]string {
	return [][]string{
		{"--session", "pair-exact", "action", "list-panes", "--json", "--command"},
		{"--session", "pair-exact", "action", "list-panes", "--json", "--command"},
		{"list-sessions", "--no-formatting"},
		{"--session", "pair-exact", "action", "list-clients"},
	}
}

func TestRecoverThreadNativeReplyUnknownStopsBeforeEffects(t *testing.T) {
	for _, stage := range []struct {
		name    string
		ordinal int
	}{{"first-owner", 1}, {"second-owner", 2}, {"listing", 3}, {"clients", 4}} {
		for _, reply := range []struct{ name, raw string }{{"empty", ""}, {"whitespace", " \t\n"}, {"wrong-kind", `{"active_swap_layout_name":null,"is_swap_layout_dirty":false}`}, {"malformed", "[{"}} {
			t.Run(stage.name+"/"+reply.name, func(t *testing.T) {
				f := newRecoveryNativeFixture(t)
				// Exhaust only the known upstream signature's bounded read recovery;
				// every later explicit observation would still be healthy.
				attempts := 1
				if reply.name != "malformed" {
					attempts = 3
				}
				faults := make([]zellijtest.Fault, attempts)
				for i := range faults {
					faults[i] = zellijtest.Fault{Ordinal: stage.ordinal + i, Reply: &reply.raw}
				}
				f.native.SetFaults(faults...)
				before, err := f.env.Couch.Threads.GetThread(f.source.Address)
				if err != nil {
					t.Fatal(err)
				}
				registry := f.env.Couch.actorRegistry()
				ops := append([]string(nil), f.env.Runner.Ops...)
				_, err = f.env.Couch.RecoverThread(t.Context(), f.source.Address, "")
				if err == nil {
					t.Fatal("unknown native reply admitted recovery")
				}
				if reply.name == "empty" || reply.name == "whitespace" {
					if !errors.Is(err, launcher.ErrZellijNoReply) {
						t.Fatalf("no-reply classification lost: %v", err)
					}
				}
				if !strings.Contains(err.Error(), "could not be checked") {
					t.Fatalf("unknown reply became an absence/ownership decision: %v", err)
				}
				after, readErr := f.env.Couch.Threads.GetThread(f.source.Address)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("unknown reply changed durable record: before=%+v after=%+v", before, after)
				}
				if !reflect.DeepEqual(registry, f.env.Couch.actorRegistry()) {
					t.Fatal("unknown reply changed actor registry")
				}
				if !reflect.DeepEqual(ops, f.env.Runner.Ops) {
					t.Fatalf("unknown reply caused launcher effects: %v", f.env.Runner.Ops)
				}
				if len(f.env.Artifacts.quiesced) != 0 || len(f.env.Proc.Signals) != 0 {
					t.Fatalf("unknown reply quiesced source: sessions=%v signals=%v", f.env.Artifacts.quiesced, f.env.Proc.Signals)
				}
				queries := f.native.Queries()
				if len(queries) != stage.ordinal+attempts-1 {
					t.Fatalf("unknown reply exceeded bounded reads: queries=%+v", queries)
				}
				for i, query := range queries {
					want := min(i, stage.ordinal-1)
					if query.Ordinal != i+1 || !reflect.DeepEqual(query.Args, recoveryNativeInitialQueries()[want]) {
						t.Fatalf("query %d: %+v", i, query)
					}
				}
			})
		}
	}
}

// A corrupt native read is recovered inside the production read boundary;
// the recovery operation and its single launcher effect are never replayed.
func TestRecoverThreadNativeKnownBugReadReattaches(t *testing.T) {
	for _, stage := range []int{1, 2, 3, 4} {
		for _, raw := range []string{"", `{"active_swap_layout_name":null,"is_swap_layout_dirty":false}`} {
			t.Run(fmt.Sprintf("stage-%d/%q", stage, raw), func(t *testing.T) {
				f := newRecoveryNativeFixture(t)
				f.native.SetFaults(zellijtest.Fault{Ordinal: stage, Reply: &raw})
				result, err := f.env.Couch.RecoverThread(t.Context(), f.source.Address, "")
				if err != nil {
					t.Fatal(err)
				}
				if result.Handle == nil || result.Record.Shape != StartWarmReattach {
					t.Fatalf("recovery=%+v", result)
				}
				t.Cleanup(func() { f.env.Runner.SetExited(result.Handle.ID(), 0) })
				child := f.env.Runner.Child(result.Handle.ID())
				if !reflect.DeepEqual(child.Argv, []string{"pair", launcher.CouchSessionFlag, "resume", string(f.source.Address.Tag)}) {
					t.Fatalf("fresh-agent launch=%v", child.Argv)
				}
				queries := f.native.Queries()
				if len(queries) <= stage || !reflect.DeepEqual(queries[stage-1].Args, queries[stage].Args) {
					t.Fatalf("read was not reissued at its boundary: %+v", queries)
				}
			})
		}
	}
}

func TestRecoverThreadNativeHeaderOnlyClientsReattaches(t *testing.T) {
	f := newRecoveryNativeFixture(t)
	result, err := f.env.Couch.RecoverThread(t.Context(), f.source.Address, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Handle == nil || result.Record.Shape != StartWarmReattach {
		t.Fatalf("header-only detached response did not permit warm recovery: %+v", result)
	}
	t.Cleanup(func() { f.env.Runner.SetExited(result.Handle.ID(), 0) })
	queries := f.native.Queries()
	if len(queries) < len(recoveryNativeInitialQueries()) {
		t.Fatalf("warm recovery skipped initial observations: %+v", queries)
	}
	for i, args := range recoveryNativeInitialQueries() {
		if queries[i].Ordinal != i+1 || !reflect.DeepEqual(queries[i].Args, args) {
			t.Fatalf("query %d: %+v", i, queries[i])
		}
	}
	child := f.env.Runner.Child(result.Handle.ID())
	if !reflect.DeepEqual(child.Argv, []string{"pair", launcher.CouchSessionFlag, "resume", string(f.source.Address.Tag)}) {
		t.Fatalf("recovery launched a fresh agent: %v", child.Argv)
	}
}

// The capture must see the failing query in the actual recovery call chain,
// rather than a later diagnostic query against a potentially changed session.
func TestRecoveryQueryCaptureRetainsFailedRecoveryRequest(t *testing.T) {
	capture := newRecoveryQueryCapture(t)
	f := newRecoveryNativeFixture(t)
	empty := ""
	f.native.SetFaults(zellijtest.Fault{Ordinal: 4, Reply: &empty}, zellijtest.Fault{Ordinal: 5, Reply: &empty}, zellijtest.Fault{Ordinal: 6, Reply: &empty})
	ctx := capture.Context(t.Context())
	capture.Mark("recover-enter")
	_, err := f.env.Couch.RecoverThread(ctx, f.source.Address, "")
	capture.Mark("recover-return")
	if !errors.Is(err, launcher.ErrZellijNoReply) {
		t.Fatalf("recovery classification: %v", err)
	}
	snapshot := capture.snapshot()
	if len(snapshot.Records) != 8 || snapshot.DroppedRecords != 0 || len(f.native.Queries()) != 6 {
		t.Fatalf("capture lost or added observations: records=%d dropped=%d queries=%v", len(snapshot.Records), snapshot.DroppedRecords, f.native.Queries())
	}
	failed := snapshot.Records[4]
	if !failed.NoReply || failed.ExitCode != 0 || failed.StdoutBytes != 0 || len(failed.Stdout) != 0 || !reflect.DeepEqual(failed.Args, recoveryNativeInitialQueries()[3]) {
		t.Fatalf("failed request evidence: %+v", failed)
	}
	if snapshot.Records[0].Name != "recover-enter" || snapshot.Records[7].Name != "recover-return" || failed.Started.Before(snapshot.Records[0].Started) || failed.Finished.After(snapshot.Records[7].Started) {
		t.Fatal("capture did not correlate the failing query with the recovery call")
	}
}
