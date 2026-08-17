package runtimebundlegen

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCommandBootstrapsFromCleanTrackedSource(t *testing.T) {
	repoRoot := gitOutput(t, "", "rev-parse", "--show-toplevel")
	cleanRoot := t.TempDir()
	for _, logical := range strings.Split(gitOutput(t, repoRoot, "ls-files", "--cached", "--others", "--exclude-standard"), "\n") {
		if logical == "" {
			continue
		}
		src := filepath.Join(repoRoot, filepath.FromSlash(logical))
		dst := filepath.Join(cleanRoot, filepath.FromSlash(logical))
		info, err := os.Lstat(src)
		if err != nil {
			t.Fatalf("Lstat(%s) error = %v", src, err)
		}
		if info.IsDir() {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(src)
			if err != nil {
				t.Fatalf("Readlink(%s) error = %v", src, err)
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(dst), err)
			}
			if err := os.Symlink(target, dst); err != nil {
				t.Fatalf("Symlink(%s -> %s) error = %v", dst, target, err)
			}
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", src, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", dst, err)
		}
	}

	cmd := exec.Command("go", "run", "./cmd/internal/runtimebundle/generatecmd",
		"--repo", ".", "--out", "cmd/internal/runtimebundle/assets/runtime")
	cmd.Dir = cleanRoot
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run generatecmd from clean tracked source failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(cleanRoot, "cmd/internal/runtimebundle/assets/runtime/manifest.json")); err != nil {
		t.Fatalf("generated manifest missing: %v", err)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			scanner := bufio.NewScanner(strings.NewReader(string(exit.Stderr)))
			if scanner.Scan() {
				t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, scanner.Text())
			}
		}
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}
