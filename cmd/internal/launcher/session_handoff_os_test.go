package launcher

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
	ready, _ := readiness.Encode(readiness.ReadyRecord{Tag: p.Tag, Agent: p.Agent, Nonce: p.Nonce, Session: p.Name, PID: 30})
	if err = os.WriteFile(AgentReadyPath(root, p.Tag, p.Agent), []byte(ready), 0600); err != nil {
		t.Fatal(err)
	}
	table := []procutil.Process{{PID: 10, PPID: 1, Identity: "server"}, {PID: 20, PPID: 10, Identity: "wrapper"}, {PID: 30, PPID: 20, Identity: "agent"}}
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); !got.Usable {
		t.Fatalf("healthy: %+v", got)
	}
	table[1].PPID = 99
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable {
		t.Fatal("old agent made replacement usable")
	}
	table[1].PPID = 10
	server.Identity = "reused"
	if got := ObserveTerminalUsability(server, root, "", p.Tag, table); got.Usable {
		t.Fatal("stale policy certified reused pid")
	}
}
