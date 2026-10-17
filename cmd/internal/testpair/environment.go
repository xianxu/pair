package testpair

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

// Environment uses the same empty-environment owner as shell and Python tests.
// root is a disposable test directory; callers explicitly name any additional
// fixture authority. No ambient Pair, Couch, native-agent, or credential values
// cross this boundary.
func Environment(t testing.TB, root string, explicit ...string) []string {
	t.Helper()
	env, err := EnvironmentAt(root, explicit...)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

// EnvironmentAt is the error-returning form for self-executed fixture children
// that do not run under testing.M. It has the same explicit authority contract.
func EnvironmentAt(root string, explicit ...string) ([]string, error) {
	repo, err := reporoot.Current()
	if err != nil {
		return nil, err
	}
	return EnvironmentFrom(filepath.Join(repo, "tests/lib/pair-test-env.sh"), root, explicit...)
}

// EnvironmentFrom lets a fixture child in a disposable checkout use the exact
// helper selected by its parent, without discovering authority from its cwd.
func EnvironmentFrom(script, root string, explicit ...string) ([]string, error) {
	if !filepath.IsAbs(script) {
		return nil, fmt.Errorf("test environment helper must be absolute")
	}
	args := append([]string{script, "--print0", root}, explicit...)
	cmd := exec.Command("/bin/sh", args...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("disposable subprocess environment: %w", err)
	}
	return strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00"), nil
}

// ApplyEnvironment installs the same boundary for tests whose production path
// spawns its own children. testing's Setenv guard forbids parallel use; cleanup
// restores the complete prior environment after owned child shutdowns.
func ApplyEnvironment(t testing.TB, root string, explicit ...string) {
	t.Helper()
	env := Environment(t, root, explicit...)
	prior := os.Environ()
	t.Setenv("HOME", os.Getenv("HOME"))
	t.Cleanup(func() {
		os.Clearenv()
		for _, entry := range prior {
			key, value, _ := strings.Cut(entry, "=")
			_ = os.Setenv(key, value)
		}
	})
	os.Clearenv()
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		if err := os.Setenv(key, value); err != nil {
			t.Fatal(err)
		}
	}
}
