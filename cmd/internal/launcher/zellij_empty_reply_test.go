package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijsession"
)

// These frames model the CLI boundary of the v0.45.1 early-unblock race:
// UnblockInput before Log can terminate one process successfully with no stdout.
// They do not emulate Zellij's Rust IPC wire. A later Log belongs to a fresh,
// explicitly requested observation, never an automatic retry of the first.
type emptyReplyFrame struct {
	Args           []string
	Events         []string
	Stdout, Stderr string
	Exit           int
}
type emptyReplyFake struct {
	source ZellijSource
	state  string
}

func newEmptyReplyFake(t *testing.T, frames ...emptyReplyFrame) emptyReplyFake {
	t.Helper()
	dir := t.TempDir()
	framesPath := filepath.Join(dir, "frames.json")
	body, err := json.Marshal(frames)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(framesPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(dir, "invocations")
	if err = os.WriteFile(state, []byte("0"), 0600); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	script := "#!/bin/sh\nexport PAIR_EMPTY_REPLY_FRAMES=" + quote(framesPath) + "\nexport PAIR_EMPTY_REPLY_STATE=" + quote(state) + "\nexec " + quote(binary) + " -test.run=^TestZellijEmptyReplyHelperProcess$ -- \"$@\"\n"
	path := filepath.Join(dir, "zellij")
	if err = os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return emptyReplyFake{ZellijSource{Path: path}, state}
}
func (f emptyReplyFake) calls(t *testing.T) int {
	t.Helper()
	body, err := os.ReadFile(f.state)
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(string(body))
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func TestZellijEmptyReplyHelperProcess(t *testing.T) {
	path := os.Getenv("PAIR_EMPTY_REPLY_FRAMES")
	if path == "" {
		return
	}
	fail := func(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(97) }
	body, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	var frames []emptyReplyFrame
	if err = json.Unmarshal(body, &frames); err != nil {
		fail(err)
	}
	state := os.Getenv("PAIR_EMPTY_REPLY_STATE")
	body, err = os.ReadFile(state)
	if err != nil {
		fail(err)
	}
	n, err := strconv.Atoi(string(body))
	if err != nil {
		fail(err)
	}
	if n >= len(frames) {
		fail(fmt.Errorf("unexpected automatic invocation %d", n+1))
	}
	frame := frames[n]
	var args []string
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if !reflect.DeepEqual(args, frame.Args) {
		fail(fmt.Errorf("invocation %d args %q, want %q", n, args, frame.Args))
	}
	if err = os.WriteFile(state, []byte(strconv.Itoa(n+1)), 0600); err != nil {
		fail(err)
	}
	if len(frame.Events) == 0 {
		fmt.Fprint(os.Stdout, frame.Stdout)
		fmt.Fprint(os.Stderr, frame.Stderr)
	}
	for _, event := range frame.Events {
		switch event {
		case "Log":
			fmt.Fprint(os.Stdout, frame.Stdout)
			fmt.Fprint(os.Stderr, frame.Stderr)
		case "UnblockInput":
			os.Exit(frame.Exit)
		default:
			fail(fmt.Errorf("unknown fake event %q", event))
		}
	}
	os.Exit(frame.Exit)
}
func earlyUnblockFrame(args []string, stdout string) emptyReplyFrame {
	if stdout == "" {
		// The queued Log is not consumed once UnblockInput exits this process.
		return emptyReplyFrame{Args: args, Events: []string{"UnblockInput", "Log"}, Stdout: "unconsumed late reply"}
	}
	return emptyReplyFrame{Args: args, Events: []string{"Log", "UnblockInput"}, Stdout: stdout}
}
func assertEmptyReply(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrZellijNoReply) {
		t.Errorf("error = %v, want uncertain no reply", err)
	}
}
func TestZellijEmptyReplySnapshotIsNotAbsenceOrDetached(t *testing.T) {
	short := []string{"list-sessions", "--short"}
	full := []string{"list-sessions", "--no-formatting"}
	clients := []string{"--session", "pair-a", "action", "list-clients"}
	for _, stage := range []string{"short", "full", "clients"} {
		for _, empty := range []string{"", " \n\t"} {
			t.Run(fmt.Sprintf("%s/%q", stage, empty), func(t *testing.T) {
				var frames []emptyReplyFrame
				if stage != "short" {
					frames = append(frames, emptyReplyFrame{Args: short, Stdout: "pair-a\n"})
				}
				if stage == "clients" {
					frames = append(frames, emptyReplyFrame{Args: full, Stdout: "pair-a [Created]\n"})
				}
				args := short
				if stage == "full" {
					args = full
				}
				if stage == "clients" {
					args = clients
				}
				frames = append(frames, earlyUnblockFrame(args, empty))
				firstCalls := len(frames)
				frames = append(frames, emptyReplyFrame{Args: short, Events: []string{"Log", "UnblockInput"}, Stdout: "pair-a\n"}, emptyReplyFrame{Args: full, Stdout: "pair-a [Created]\n"}, emptyReplyFrame{Args: clients, Stdout: "CLIENT_ID\n1\n"})
				fake := newEmptyReplyFake(t, frames...)
				got, err := fake.source.Snapshot()
				assertEmptyReply(t, err)
				if got != nil {
					t.Errorf("empty reply granted inventory authority: %+v", got)
				}
				if n := fake.calls(t); n != firstCalls {
					t.Fatalf("automatic retry: calls %d want %d", n, firstCalls)
				}
				got, err = fake.source.Snapshot()
				if err != nil || len(got) != 1 || got[0].State != SessionAttached {
					t.Fatalf("fresh observation = %+v, %v", got, err)
				}
				if n := fake.calls(t); n != firstCalls+3 {
					t.Fatalf("fresh query calls %d", n)
				}
			})
		}
	}
}
func TestZellijEmptyReplyLivenessIsNotAbsence(t *testing.T) {
	short := []string{"list-sessions", "--short"}
	full := []string{"list-sessions", "--no-formatting"}
	for _, stage := range []int{0, 1} {
		t.Run(strconv.Itoa(stage), func(t *testing.T) {
			var frames []emptyReplyFrame
			if stage == 1 {
				frames = append(frames, emptyReplyFrame{Args: short, Stdout: "pair-a\n"})
			}
			args := short
			if stage == 1 {
				args = full
			}
			frames = append(frames, earlyUnblockFrame(args, ""))
			frames = append(frames, emptyReplyFrame{Args: short, Stdout: "pair-a\n"}, emptyReplyFrame{Args: full, Stdout: "pair-a [Created]\n"})
			fake := newEmptyReplyFake(t, frames...)
			got, err := fake.source.LivenessContext(context.Background())
			assertEmptyReply(t, err)
			if got != nil || fake.calls(t) != stage+1 {
				t.Fatalf("uncertain liveness = %+v; calls %d", got, fake.calls(t))
			}
			got, err = fake.source.LivenessContext(context.Background())
			if err != nil || len(got) != 1 || got[0].State != SessionLive {
				t.Fatalf("fresh liveness = %+v %v", got, err)
			}
		})
	}
}

type emptyReplyOwnerIO struct {
	source                      ZellijSource
	server                      SessionServerIdentity
	hasServer, changeGeneration bool
}

func (w *emptyReplyOwnerIO) SessionServers(context.Context, string) ([]SessionServerIdentity, error) {
	if !w.hasServer {
		return nil, nil
	}
	return []SessionServerIdentity{w.server}, nil
}
func (w *emptyReplyOwnerIO) Socket(string) SocketState { return SocketPresent }
func (w *emptyReplyOwnerIO) SessionPresent(ctx context.Context, name string) (bool, error) {
	raw, err := w.source.runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		return false, err
	}
	present, exited := zellijsession.RowState(string(raw), name)
	return present && !exited, nil
}
func (w *emptyReplyOwnerIO) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	raw, err := w.source.runContext(ctx, "--session", name, "action", "list-panes", "--json", "--command")
	if w.changeGeneration {
		w.server.Identity = "replacement"
	}
	return raw, err
}
func TestZellijEmptyReplyOwnerProbeGrantsNoAuthority(t *testing.T) {
	const name = "📁1-37"
	panes := []string{"--session", name, "action", "list-panes", "--json", "--command"}
	valid := `[{"id":1,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"}]`
	for _, empty := range []string{"", " \t\n"} {
		t.Run(fmt.Sprintf("%q", empty), func(t *testing.T) {
			fake := newEmptyReplyFake(t, earlyUnblockFrame(panes, empty), emptyReplyFrame{Args: panes, Events: []string{"Log", "UnblockInput"}, Stdout: valid})
			io := &emptyReplyOwnerIO{source: fake.source, hasServer: true, server: SessionServerIdentity{PID: 37, Identity: "generation", Session: name, Socket: "/fake/socket"}}
			probe := SessionOwnerProbe{IO: io}
			got, err := probe.Probe(context.Background(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
			assertEmptyReply(t, err)
			if got.State != SessionOwnerUnknown || fake.calls(t) != 1 {
				t.Fatalf("uncertain ownership = %+v; calls %d", got, fake.calls(t))
			}
			if err := probe.Revalidate(context.Background(), got); err == nil {
				t.Fatal("uncertain ownership revalidated")
			}
			got, err = probe.Probe(context.Background(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
			if err != nil || got.State != SessionOwnerOwned || fake.calls(t) != 2 {
				t.Fatalf("fresh ownership = %+v %v", got, err)
			}
		})
	}
	t.Run("no server still needs presence answer", func(t *testing.T) {
		fake := newEmptyReplyFake(t, earlyUnblockFrame([]string{"list-sessions", "--no-formatting"}, ""))
		got, err := (SessionOwnerProbe{IO: &emptyReplyOwnerIO{source: fake.source}}).Probe(context.Background(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
		assertEmptyReply(t, err)
		if got.State != SessionOwnerUnknown || fake.calls(t) != 1 {
			t.Fatalf("no reply became absence: %+v", got)
		}
	})
	for _, change := range []bool{false, true} {
		t.Run(fmt.Sprintf("nonempty/change=%v", change), func(t *testing.T) {
			raw := "{malformed"
			want := "invalid live pane JSON"
			if change {
				raw = valid
				want = "session server changed during observation"
			}
			fake := newEmptyReplyFake(t, emptyReplyFrame{Args: panes, Stdout: raw})
			got, err := (SessionOwnerProbe{IO: &emptyReplyOwnerIO{source: fake.source, hasServer: true, changeGeneration: change, server: SessionServerIdentity{PID: 37, Identity: "generation", Session: name, Socket: "/fake/socket"}}}).Probe(context.Background(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
			if err != nil || got.State != SessionOwnerUnknown || got.Diagnostic != want || fake.calls(t) != 1 {
				t.Fatalf("nonempty observation = %+v %v", got, err)
			}
		})
	}
}
func TestZellijEmptyReplyExplicitNoSessionsRemainsValid(t *testing.T) {
	frames := []emptyReplyFrame{{Args: []string{"list-sessions", "--short"}, Stderr: "No active zellij sessions found.\n", Exit: 1}, {Args: []string{"list-sessions", "--no-formatting"}, Stderr: "No active zellij sessions found.\n", Exit: 1}}
	fake := newEmptyReplyFake(t, frames...)
	got, err := fake.source.Snapshot()
	if err != nil || len(got) != 0 || fake.calls(t) != 2 {
		t.Fatalf("explicit empty inventory = %+v %v", got, err)
	}
}

func TestZellijEmptyReplyOwnerPassAllowsFreshCLIObservation(t *testing.T) {
	const name = "pair-a"
	args := []string{"--session", name, "action", "list-panes", "--json", "--command"}
	valid := `[{"id":1,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"}]`
	fake := newEmptyReplyFake(t, earlyUnblockFrame(args, ""), emptyReplyFrame{Args: args, Events: []string{"Log", "UnblockInput"}, Stdout: valid})
	world := &emptyReplyOwnerIO{source: fake.source, hasServer: true, server: SessionServerIdentity{PID: 7, Identity: "same", Session: name, Socket: "/fake/socket"}}
	captures := 0
	source := ownerPassSourceFunc(func(context.Context) (map[string]ServerState, error) {
		captures++
		return map[string]ServerState{name: {Verdict: ServerReachable, Server: world.server}}, nil
	})
	pass := NewSessionOwnerPass(t.Context(), source, SessionOwnerProbe{IO: world})
	got, err := pass.Probe(t.Context(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
	assertEmptyReply(t, err)
	if got.State != SessionOwnerUnknown || fake.calls(t) != 1 {
		t.Fatal("pass authorized empty CLI reply")
	}
	got, err = pass.Probe(t.Context(), name, "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err != nil || got.State != SessionOwnerOwned || fake.calls(t) != 2 || captures != 1 {
		t.Fatalf("fresh pane observation=%+v,%v; captures=%d", got, err, captures)
	}
}
