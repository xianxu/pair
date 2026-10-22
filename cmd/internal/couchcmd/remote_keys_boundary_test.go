package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/remotesession"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/wrapcmd"
	"golang.org/x/term"
)

const remoteKeysOutputMarker = "REMOTE-KEYS-CHILD-OUTPUT"

// This is the actual wrapper entry, in an owned subprocess. fd3 prevents its
// real registry hello racing the parent's exact-PID authority installation.
func TestRemoteKeysWrapperHelper(t *testing.T) {
	if os.Getenv("PAIR_TEST_REMOTE_KEYS") != "wrapper" {
		t.Skip("subprocess fixture")
	}
	for _, key := range []string{"PAIR_DATA_DIR", "PAIR_TAG", "PAIR_SCOPE_KEY", "PAIR_LAUNCH_ORDINAL", "PAIR_RETENTION_START_ID", "PAIR_SCROLLBACK_EVENTS_PATH", "PAIR_WRAP_LOG", "COUCH_LIFETIME_LEASE"} {
		if value := os.Getenv(key); value != "" {
			t.Fatalf("inherited %s crossed wrapper boundary", key)
		}
	}
	gate := os.NewFile(3, "wrapper-start")
	var b [1]byte
	if _, err := io.ReadFull(gate, b[:]); err != nil {
		t.Fatal(err)
	}
	_ = gate.Close()
	os.Exit(wrapcmd.Run([]string{os.Getenv("PAIR_TEST_REMOTE_AGENT")}, os.Stdin, os.Stdout, os.Stderr))
}

// The only modeled external agent behavior is a captured usage panel, one Esc,
// and its captured closed panel. No model, network or invented prompt runs.
func TestRemoteKeysAgentHelper(t *testing.T) {
	if os.Getenv("PAIR_TEST_REMOTE_KEYS") != "agent" {
		t.Skip("subprocess fixture")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	paint := func(name string) {
		raw, err := os.ReadFile(filepath.Join(os.Getenv("PAIR_TEST_REMOTE_SCREENS"), name+".screen"))
		if err != nil {
			t.Fatal(err)
		}
		rows := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
		cursor := "\x1b[1;31H\x1b[?25l"
		for y, row := range rows {
			if before, after, ok := strings.Cut(row, "‹cursor›"); ok {
				rows[y] = before + after
				cursor = fmt.Sprintf("\x1b[%d;%dH\x1b[?25h", y+1, len([]rune(before))+1)
			}
		}
		fmt.Print("\x1b[?2026h\x1b[2J\x1b[H" + strings.Join(rows, "\r\n") + cursor + "\x1b[?2026l")
	}
	paint("usage-panel")
	var b [1]byte
	if _, err := io.ReadFull(os.Stdin, b[:]); err != nil {
		t.Fatal(err)
	}
	if b[0] != 27 {
		t.Fatalf("received %x, want Esc", b)
	}
	if err := os.WriteFile(os.Getenv("PAIR_TEST_REMOTE_ACK"), b[:], 0600); err != nil {
		t.Fatal(err)
	}
	// Deliberately emit the host-reset encoding as CHILD OUTPUT. It must never
	// be decoded by Console's owned host-input branch.
	fmt.Print("\x1b[114;7u\r\n" + remoteKeysOutputMarker + "\r\n")
	paint("usage-closed")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestRemoteKeysActualWrapperCannotResetHostLatch(t *testing.T) {
	root := t.TempDir()
	write := func(path string, raw []byte, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(path, raw, mode); err != nil {
			t.Fatal(err)
		}
	}
	world := newMessageWorld()
	world.hub = "/repo0"
	r := &serviceRig{t: t, world: world, slotGit: map[string]couchcore.SlotGitStatus{}}
	namespace := filepath.Join(root, "messages")
	var err error
	r.brokerSock, err = couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		t.Fatal(err)
	}
	r.regSock, err = couchmessage.SocketPath(namespace, "registry")
	if err != nil {
		t.Fatal(err)
	}
	r.slotOps = newSlotOperations(nil, func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/repo/.git", Dir: "pair"}}, nil
	}, time.Now)
	r.start()
	hub := r.connect(0)
	b := world.add(2)
	b.Agent, b.Version = "claude", "2.1.304"
	primary := filepath.Join(root, "pair")
	layout := couchcore.NewSlotLayout(primary, b.Repository, 2)
	bin := filepath.Join(root, "bin")
	for _, dir := range []string{bin, layout.Host()} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	branch, head, slot := "main-slot2", strings.Repeat("a", 40), 2
	identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "pair", RepoIdentity: b.Repository, PrimaryRoot: primary, FleetRoot: root, EnvironmentRoot: layout.Env(), WorktreeRoot: layout.Host(), Kind: "slot", Address: &b.Slot, Slot: &slot, Branch: &branch, Head: &head, RestingBranch: &branch, EnvironmentHost: &couchcore.EnvironmentHost{Repo: "pair", Slot: 2, RepoIdentity: b.Repository, PrimaryRoot: primary, WorktreeRoot: layout.Host()}}
	raw, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := couchcore.ParseWorkspaceIdentity(raw); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(root, "workspace.json")
	write(workspace, raw, 0600)
	write(filepath.Join(bin, "sdlc"), []byte("#!/bin/sh\n[ \"$1\" = workspace ] && [ \"$2\" = --json ] || exit 2\nexec /bin/cat \"$PAIR_TEST_REMOTE_WORKSPACE\"\n"), 0700)
	agent := filepath.Join(bin, "claude")
	write(agent, []byte("#!/bin/sh\nif [ \"${1-}\" = --version ]; then printf '2.1.304\\n'; exit; fi\nPAIR_TEST_REMOTE_KEYS=agent; export PAIR_TEST_REMOTE_KEYS\nexec \"$PAIR_TEST_REMOTE_EXE\" -test.run=^TestRemoteKeysAgentHelper$\n"), 0700)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	screens, err := filepath.Abs("../wrapcmd/testdata/drive/claude/2.1.304")
	if err != nil {
		t.Fatal(err)
	}
	ack := filepath.Join(root, "escaped")
	env := testpair.Environment(t, root, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "PAIR_TEST_REMOTE_KEYS=wrapper", "PAIR_TEST_REMOTE_EXE="+exe, "PAIR_TEST_REMOTE_AGENT="+agent, "PAIR_TEST_REMOTE_SCREENS="+screens, "PAIR_TEST_REMOTE_ACK="+ack, "PAIR_TEST_REMOTE_WORKSPACE="+workspace, "COUCH_STORE_DIR="+namespace, "COUCH_THREAD_SCOPE="+b.Scope, "COUCH_THREAD_TAG="+b.Tag, "PAIR_SESSION_NAME="+b.Session, "PAIR_LAUNCH_NONCE="+b.Nonce)
	// Poison absent inherited fields, including managed identity and side-effect
	// paths. The helper checks these before invoking any production wrapper code.
	poison := t.TempDir()
	for key, value := range map[string]string{"PAIR_DATA_DIR": poison, "PAIR_TAG": "poison-tag", "PAIR_SCOPE_KEY": "poison-scope", "PAIR_LAUNCH_ORDINAL": "999", "PAIR_RETENTION_START_ID": "poison-start", "PAIR_SCROLLBACK_EVENTS_PATH": filepath.Join(poison, "events"), "PAIR_WRAP_LOG": filepath.Join(poison, "debug"), "COUCH_LIFETIME_LEASE": filepath.Join(poison, "lifetime.lease")} {
		t.Setenv(key, value)
	}
	// Child.Start overlays its Env onto the parent's environment. Install the
	// complete empty-environment owner too, so absent Pair variables stay absent.
	testpair.ApplyEnvironment(t, root, env...)
	protected, err := broadcast.NewProtectedHost(&broadcast.FakeProtectedTunnel{FakeTunnel: &broadcast.FakeTunnel{}})
	if err != nil {
		t.Fatal(err)
	}
	remote, reset, err := remotesession.NewSupervisor(remotesession.SupervisorConfig{Namespace: filepath.Join(root, "remote"), Origin: "https://remote.example.com", Host: protected})
	if err != nil {
		t.Fatal(err)
	}
	remote.Kill()
	if err := remote.Join(t.Context()); err != nil {
		t.Fatal(err)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 43, Cols: 94})
	input, hostKeys := io.Pipe()
	console := couchtty.New(host, input)
	var resetCalls atomic.Int32
	console.SetRemoteSession(remote, func() error { resetCalls.Add(1); return reset() }, "")
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name != "keys" {
			return nil, fmt.Errorf("unexpected fixture operation %s", call.Name)
		}
		return executeAgentDrive(call)
	})
	gate, release, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Close()
	defer release.Close()
	child, err := ptychild.Start(ptychild.Options{EndpointID: "wrapper", Dir: layout.Host(), Env: env, Argv: []string{exe, "-test.run=^TestRemoteKeysWrapperHelper$"}, Size: ptychild.Size{Rows: 42, Cols: 94}, ExtraFiles: []*os.File{gate}, Sink: func(ctx context.Context, batch ptychild.OutputBatch) error {
		return console.Deliver(ctx, "wrapper", batch)
	}})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	t.Cleanup(func() {
		// Let the real wrapper stop and join its own native child group before
		// Console teardown closes its endpoint (Child.Close is an abrupt kill).
		_ = child.Signal(syscall.SIGTERM)
		select {
		case <-child.Exited():
		case <-testwait.After(t):
			t.Error("wrapper did not join its owned fake agent")
		}
		console.Stop()
		_ = hostKeys.Close()
		_ = child.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("console did not join")
		}
	})
	console.Attach("wrapper", "wrapper", child)
	r.s.installAgentDriveService(console)
	go func() { done <- console.Run() }()
	b.PID = child.PID()
	b.Start, err = (couchcore.OSProcOps{}).Identity(b.PID)
	if err != nil {
		t.Fatal(err)
	}
	world.mu.Lock()
	slotState := world.slots[b.Thread()]
	slotState.binding, slotState.root, slotState.workspace, slotState.pidFile = b, layout.Host(), identity, b.PID
	world.endpoints[b] = couchmessage.RemoteEndpoint{Namespace: namespace, Binding: b}
	world.mu.Unlock()
	r.attach(b, "wrapper")
	if _, err := release.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	_ = release.Close()
	testwait.Until(t, "actual wrapper registry hello", func() bool {
		if child.Done() {
			t.Fatalf("wrapper exited: %s", child.Snapshot())
		}
		if !r.s.isConnected(b) && strings.Contains(string(child.Snapshot()), "pair-wrap:") {
			t.Fatalf("wrapper startup: %s", child.Snapshot())
		}
		return r.s.isConnected(b)
	})
	endpoint := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: b}
	testwait.Until(t, "actual wrapper's captured dismissable panel", func() bool { o, e := endpoint.ObserveDrive(t.Context()); return e == nil && o.Dismissable })
	request := driveRequest(hub, "remote-keys-boundary", b.Slot)
	request.Drive = &agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}}
	response := switchCall(t, r, request)
	if response.Code != "accepted" || response.Operation == nil {
		t.Fatalf("drive not accepted: %+v", response)
	}
	testwait.Until(t, "Esc received by actual fake-agent PTY", func() bool { raw, e := os.ReadFile(ack); return e == nil && string(raw) == "\x1b" })
	testwait.Until(t, "child reset-shaped output presented", func() bool { return strings.Contains(host.Written(), remoteKeysOutputMarker) })
	if got := remote.Snapshot(); got.Status != "disabled" || !got.CanReset || resetCalls.Load() != 0 {
		t.Fatalf("child input/output reset host: %+v, calls=%d", got, resetCalls.Load())
	}
	for _, steps := range [][]string{{"ctrl-alt-r"}, {"text:\x1b[114;7u"}} {
		if _, err := agentdrive.ParseKeySequence(steps); err == nil {
			t.Fatalf("accepted reset-shaped step %q", steps)
		}
	}
	invalid := request
	invalid.ID = "raw-reset-text"
	invalid.Drive = &agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "\x1b[114;7u"}}}
	if response := switchCall(t, r, invalid); response.Code != "invalid-request" {
		t.Fatalf("broker admitted raw reset bytes: %+v", response)
	}
	if entries, err := os.ReadDir(poison); err != nil || len(entries) != 0 {
		t.Fatalf("inherited writer namespace touched: %v %v", entries, err)
	}
}
