package testpair

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/internal/reporoot"
)

func TestShortRootConfinesRealMessageListenerAndCleansUp(t *testing.T) {
	longTemp := t.TempDir()
	t.Setenv("TMPDIR", longTemp)
	var root string
	t.Run("owned lifetime", func(t *testing.T) {
		root = ShortRoot(t)
		physical, err := filepath.EvalSymlinks(root)
		if err != nil || physical != root {
			t.Fatalf("canonical root=%q physical=%q: %v", root, physical, err)
		}
		info, err := os.Stat(root)
		if err != nil || info.Mode().Perm() != 0700 {
			t.Fatalf("private root: %v %v", info, err)
		}
		t.Setenv("COUCH_ISOLATED_ROOT", root)
		socket, err := couchmessage.SocketPath(filepath.Join(root, "store"), "broker")
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Dir(socket) != filepath.Join(root, "m") {
			t.Fatalf("socket escaped root: %q", socket)
		}
		listener, err := couchmessage.StartServer(t.Context(), socket, func(context.Context, []byte) ([]byte, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Error(err)
			}
		})
		binding := couchmessage.Binding{Slot: "fixture:1", Repository: root, Scope: "scope", Tag: "fixture", Session: "session", Nonce: "nonce", Agent: "claude", Version: "1", PID: 2147483647, Start: "generation"}
		wrapper, err := couchmessage.EndpointSocket(filepath.Join(root, "store"), binding)
		if err != nil {
			t.Fatal(err)
		}
		endpoint, err := couchmessage.StartServer(t.Context(), wrapper, func(context.Context, []byte) ([]byte, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := endpoint.Close(); err != nil {
				t.Error(err)
			}
		})
	})
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("owned root remains after test cleanup: %v", err)
	}
}

func TestIsolatedShellUsesShortOwnedRootDespiteLongTemporaryParent(t *testing.T) {
	repo, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	longTemp := t.TempDir()
	cmd := exec.Command("/bin/sh", filepath.Join(repo, "tests", "with-isolated-pair.sh"), "/bin/sh", "-c", `printf '%s' "$COUCH_ISOLATED_ROOT"`)
	cmd.Env = Environment(t, t.TempDir(), "TMPDIR="+longTemp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated shell: %v %s", err, out)
	}
	root := string(out)
	if !filepath.IsAbs(root) || len(root) > 36 || strings.HasPrefix(root, longTemp+string(filepath.Separator)) {
		t.Fatalf("unusable socket root: %q", root)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("shell root not cleaned: %v", err)
	}
}
