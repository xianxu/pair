package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestRetireTerminalPolicyPreservesCoordinationAndUnknownEvidence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	t.Setenv("ZELLIJ_SOCKET_DIR", root)
	if err := os.MkdirAll(filepath.Join(root, "contract_version_1"), 0700); err != nil {
		t.Fatal(err)
	}
	name := "p528-retirement-" + filepath.Base(home)
	socket, err := canonicalSocket(filepath.Join(root, "contract_version_1", name))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := artifactpath.ResolvePairCache(home)
	if err != nil {
		t.Fatal(err)
	}
	policyPath, err := cache.SessionPolicy(socket)
	if err != nil {
		t.Fatal(err)
	}
	lockPath, err := cache.TerminalLease(socket)
	if err != nil {
		t.Fatal(err)
	}
	p := artifactpath.SessionLaunchPolicy{Version: 1, PolicyVersion: 1, Name: name, Socket: socket, SocketInode: 1, ServerPID: 1, ServerIdentity: "retired", Tag: "tag", Agent: "codex", Nonce: "nonce", ConfigDigest: strings.Repeat("a", 64)}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(policyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := (OSRuntime{}).retireSessionPolicy(context.Background(), name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(policyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired policy still present: %v", err)
	}
	first, err := os.Stat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	// A malformed/future record is not ours to delete; the stable lock survives
	// a second retirement attempt as the same inode.
	if err := os.WriteFile(policyPath, []byte(`{"version":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (OSRuntime{}).retireSessionPolicy(context.Background(), name); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(policyPath); err != nil || string(got) != `{"version":999}` {
		t.Fatalf("unknown policy changed: %s, %v", got, err)
	}
	last, err := os.Stat(lockPath)
	if err != nil || !os.SameFile(first, last) {
		t.Fatalf("coordination inode changed: %v", err)
	}
}
