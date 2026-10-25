package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestLauncherNativeNamespaceSurvivesLongTemporaryParent(t *testing.T) {
	const child = "PAIR_LAUNCHER_NAMESPACE_CHILD"
	const prefix = "launcher-native-root="
	if os.Getenv(child) == "1" {
		root := os.Getenv("ZELLIJ_SOCKET_DIR")
		physical, err := filepath.EvalSymlinks(root)
		if err != nil || physical != root || len(root) > 36 {
			t.Fatalf("native namespace has no portable socket budget: root=%q physical=%q err=%v", root, physical, err)
		}
		info, err := os.Stat(root)
		if err != nil || info.Mode().Perm() != 0700 {
			t.Fatalf("native namespace is not private: %v %v", info, err)
		}
		fmt.Println(prefix + root)
		return
	}
	longTemp := filepath.Join(t.TempDir(), strings.Repeat("long-native-parent-", 6))
	if err := os.Mkdir(longTemp, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLauncherNativeNamespaceSurvivesLongTemporaryParent$")
	cmd.Env = testpair.Environment(t, t.TempDir(), "TMPDIR="+longTemp, child+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("launcher TestMain with long TMPDIR: %v\n%s", err, out)
	}
	var root string
	for _, line := range strings.Split(string(out), "\n") {
		if value, ok := strings.CutPrefix(line, prefix); ok {
			root = value
		}
	}
	if root == "" {
		t.Fatalf("child did not report its native namespace: %s", out)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("TestMain left its owned namespace behind: %q: %v", root, err)
	}
}
