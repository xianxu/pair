package launcher

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTerminalLaunchArgsEnforceNonSerializingPolicy(t *testing.T) {
	got := terminalLaunchArgs("name", "/config", "/layout", true)
	want := []string{"--config-dir", "/config", "--new-session-with-layout", "/layout", "--session", "name", "options", "--session-serialization", "false"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %q", got)
	}
	got = terminalLaunchArgs("name", "/config", "", false)
	if !reflect.DeepEqual(got, []string{"--config-dir", "/config", "attach", "name"}) {
		t.Fatalf("attach argv = %q", got)
	}
}

func TestSavedLayoutBlocksNameReuseWithoutCleanup(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf 'pair-old [Created 9days ago] (EXITED - attach to resurrect)\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if !(OSRuntime{}).SessionBlocksReuse("pair-old") {
		t.Fatal("resurrectable legacy name was implicitly made reusable")
	}
}

func TestTerminalUsabilityRejectsOldAgentBehindReplacementServer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	dir, err := os.MkdirTemp("/tmp", "p528-s-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "session")
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	socket, err = canonicalSocket(socket)
	if err != nil {
		t.Fatal(err)
	}
	dev, ino, err := socketFileID(socket)
	if err != nil {
		t.Fatal(err)
	}
	server := SessionServerIdentity{PID: 10, Identity: "server", Session: "session", Socket: socket}
	p := artifactpath.SessionLaunchPolicy{Version: 1, PolicyVersion: 1, Socket: socket, SocketDevice: dev, SocketInode: ino, Name: server.Session, ServerPID: server.PID, ServerIdentity: server.Identity, Tag: "tag", Agent: "codex", Nonce: "nonce", ConfigDigest: strings.Repeat("a", 64)}
	path, err := sessionPolicyPath(socket)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ready, _ := readiness.Encode(readiness.ReadyRecord{Tag: p.Tag, Agent: p.Agent, Nonce: p.Nonce, Session: p.Name, PID: 30, AgentIdentity: "agent"})
	if err = os.WriteFile(AgentReadyPath(root, p.Tag, p.Agent), []byte(ready), 0600); err != nil {
		t.Fatal(err)
	}
	table := []procutil.Process{{PID: 10, PPID: 1, Identity: "server"}, {PID: 20, PPID: 10, Identity: "wrapper"}, {PID: 30, PPID: 20, Identity: "agent"}}
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); !got.Usable() {
		t.Fatalf("healthy: %+v", got)
	}
	table[2].Identity = "agent-reused"
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable() {
		t.Fatal("stale ready pid certified replacement agent generation")
	}
	table[2].Identity = "agent"
	legacy, _ := readiness.Encode(readiness.ReadyRecord{Tag: p.Tag, Agent: p.Agent, Nonce: p.Nonce, Session: p.Name, PID: 30})
	os.WriteFile(AgentReadyPath(root, p.Tag, p.Agent), []byte(legacy), 0600)
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable() {
		t.Fatal("legacy readiness silently gained process identity")
	}
	// A native fresh reexec may update PID and generation under the same server
	// and launch nonce; the policy must not pin the old child process.
	table[2] = procutil.Process{PID: 40, PPID: 20, Identity: "fresh-agent"}
	fresh, _ := readiness.Encode(readiness.ReadyRecord{Tag: p.Tag, Agent: p.Agent, Nonce: p.Nonce, Session: p.Name, PID: 40, AgentIdentity: "fresh-agent"})
	os.WriteFile(AgentReadyPath(root, p.Tag, p.Agent), []byte(fresh), 0600)
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); !got.Usable() {
		t.Fatalf("native reexec did not replace readiness: %+v", got)
	}
	table[2] = procutil.Process{PID: 30, PPID: 20, Identity: "agent"}
	os.WriteFile(AgentReadyPath(root, p.Tag, p.Agent), []byte(ready), 0600)
	otherDir := filepath.Join(dir, "other")
	if err = os.Mkdir(otherDir, 0700); err != nil {
		t.Fatal(err)
	}
	otherSocket := filepath.Join(otherDir, filepath.Base(socket))
	otherListener, err := net.Listen("unix", otherSocket)
	if err != nil {
		t.Fatal(err)
	}
	defer otherListener.Close()
	p.Socket = otherSocket
	p.SocketDevice, p.SocketInode, err = socketFileID(otherSocket)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(p)
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable() {
		t.Fatal("policy for another socket certified observed server")
	}
	p.Socket = socket
	p.SocketDevice = dev
	p.SocketInode = ino
	raw, _ = json.Marshal(p)
	os.WriteFile(path, raw, 0600)
	table[1].PPID = 99
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable() {
		t.Fatal("old agent made replacement usable")
	}
	table[1].PPID = 10
	server.Identity = "reused"
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable() {
		t.Fatal("stale policy certified reused pid")
	}
}

func TestTerminalObservationPreservesUnknown(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want TerminalUsability
	}{
		{os.ErrPermission, TerminalUnknown},
		{context.DeadlineExceeded, TerminalUnknown},
		{os.ErrNotExist, TerminalUnsafe},
		{unsafeTerminalError("invalid policy"), TerminalUnsafe},
	} {
		if got := terminalObservationError(tc.err); got.State != tc.want {
			t.Fatalf("%v: state %v, want %v", tc.err, got.State, tc.want)
		}
	}
}

func TestTerminalPolicyInspectionPreservesUnknown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	socket := filepath.Join(t.TempDir(), "session")
	server := SessionServerIdentity{Socket: socket}
	path, err := sessionPolicyPath(socket)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	// A looping parent is an inspection failure, not proof of legacy policy.
	parent := filepath.Dir(path)
	if err = os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(parent, parent); err != nil {
		t.Fatal(err)
	}
	if got := ObserveTerminalUsability(server, "", "", "", nil); got.State != TerminalUnknown {
		t.Fatalf("inspection failed: %+v", got)
	}
	if err = os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	if got := ObserveTerminalUsability(server, "", "", "", nil); got.State != TerminalUnsafe {
		t.Fatalf("missing policy: %+v", got)
	}
	if err = os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := ObserveTerminalUsability(server, "", "", "", nil); got.State != TerminalUnsafe {
		t.Fatalf("invalid policy: %+v", got)
	}
}

func TestAdmissionMetadataConsumersRejectUnsafeFiles(t *testing.T) {
	for _, consumer := range []string{"config", "policy", "readiness"} {
		for _, kind := range []string{"fifo", "oversize"} {
			t.Run(consumer+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				t.Setenv("HOME", root)
				path := filepath.Join(root, "config.kdl")
				bound := maxTerminalConfigBytes
				read := func() error { _, err := readTerminalConfig(t.Context(), root); return err }
				if consumer == "policy" {
					socket := filepath.Join(root, "socket")
					var err error
					path, err = sessionPolicyPath(socket)
					if err != nil {
						t.Fatal(err)
					}
					bound = artifactpath.MaxSessionLaunchPolicyBytes
					read = func() error { _, err := readSessionPolicy(socket); return err }
				}
				if consumer == "readiness" {
					path = AgentReadyPath(root, "tag", "codex")
					bound = maxAgentReadyBytes
					read = func() error {
						return observePolicyAgentWithTable(artifactpath.SessionLaunchPolicy{Tag: "tag", Agent: "codex"}, root, nil)
					}
				}
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if kind == "fifo" {
					if err := syscall.Mkfifo(path, 0600); err != nil {
						t.Fatal(err)
					}
				} else {
					f, err := os.Create(path)
					if err != nil {
						t.Fatal(err)
					}
					err = f.Truncate(bound + 1)
					f.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				done := make(chan error, 1)
				go func() { done <- read() }()
				select {
				case err := <-done:
					if err == nil {
						t.Fatal("unsafe metadata accepted")
					}
				case <-time.After(time.Second):
					// Release the deliberately blocked old reader before failing the probe.
					if kind == "fifo" {
						f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
						if err == nil {
							f.Close()
						}
					}
					select {
					case <-done:
					case <-time.After(time.Second):
						t.Fatal("reader did not join")
					}
					t.Fatal("metadata consumer blocked")
				}
			})
		}
	}
}
