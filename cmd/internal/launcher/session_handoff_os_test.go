package launcher

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestTerminalHandoffPreservesSocketTransportSpelling(t *testing.T) {
	base := t.TempDir()
	actual := filepath.Join(base, "longer-real-directory")
	alias := filepath.Join(base, "short")
	if err := os.MkdirAll(actual, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(actual, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZELLIJ_SOCKET_DIR", alias)
	attempt, err := newTerminalOSAttempt(OSRuntime{}, "session", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.socketRoot != alias {
		t.Fatalf("transport root = %q, want original %q", attempt.socketRoot, alias)
	}
	canonical, err := filepath.EvalSymlinks(actual)
	if err != nil {
		t.Fatal(err)
	}
	wantSocket := filepath.Join(canonical, "contract_version_1", "session")
	if attempt.socket != wantSocket {
		t.Fatalf("lease identity = %q, want canonical %q", attempt.socket, wantSocket)
	}
	t.Setenv("ZELLIJ_SOCKET_DIR", actual)
	direct, err := newTerminalOSAttempt(OSRuntime{}, "session", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if direct.socket != attempt.socket {
		t.Fatalf("same socket aliases have different lease identities: %q and %q", direct.socket, attempt.socket)
	}
}
