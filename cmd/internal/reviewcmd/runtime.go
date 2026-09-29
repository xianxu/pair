package reviewcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// OSRuntime implements Runtime with real git/nvim/zellij/fs calls; the fs
// primitives come from the embedded osfs.FS (#93 M3).
type OSRuntime struct{ osfs.FS }

func NewOSRuntime() OSRuntime { return OSRuntime{} }

func (OSRuntime) ProcessAlive(pid string) bool { return procutil.Alive(pid) }
func (OSRuntime) Kill(pid string)              { _ = exec.Command("kill", pid).Run() }

// AbsFile makes file absolute (logical) when its directory exists, else leaves
// it (the target seam's `if [ -d dirname ]; then cd dirname && pwd/basename`).
func (OSRuntime) AbsFile(file string) string {
	dir := filepath.Dir(file)
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		if abs, err := filepath.Abs(dir); err == nil {
			return filepath.Join(abs, filepath.Base(file))
		}
	}
	return file
}

// LogicalDir returns file's directory as a logical absolute path (open's `pwd`).
func (OSRuntime) LogicalDir(file string) string {
	if abs, err := filepath.Abs(filepath.Dir(file)); err == nil {
		return abs
	}
	return filepath.Dir(file)
}

// PhysicalDir returns file's directory symlink-resolved (readiness's `pwd -P`),
// or "" when the directory doesn't exist (the shell's `|| dir=""`).
func (OSRuntime) PhysicalDir(file string) string {
	abs, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return ""
	}
	return resolved
}

// Git runs `git -C dir <args…>` and returns stdout + error (non-nil on non-zero
// exit — callers read the exit as the fact, e.g. ls-files --error-unmatch).
func (OSRuntime) Git(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).Output()
	return string(out), err
}

// Classify runs the single-source nvim/review/readiness.lua classifier via
// `nvim --headless`, exactly as the shell did. Returns the 4-case string.
func (OSRuntime) Classify(readinessLua string, f ReadinessFacts) (string, error) {
	expr := fmt.Sprintf(
		"lua io.write(dofile(%q).classify({is_git=%s,is_tracked=%s,on_review_branch=%s,file_matches=%s,is_clean=%s}))",
		readinessLua, luaBool(f.IsGit), luaBool(f.IsTracked), luaBool(f.OnReviewBranch), luaBool(f.FileMatches), luaBool(f.IsClean))
	out, err := exec.Command("nvim", "--headless", "-u", "NONE", "--cmd", expr, "--cmd", "qa!").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func luaBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// SpawnReviewPane opens the full-screen floating nvim review pane. The shell
// EXEC'd `zellij run`; Go spawn+wait is equivalent (zellij run returns after the
// server opens the pane). PAIR_NVIM_PID_FILE is exported so review.lua's VimEnter
// can record the embed pid for cleanup.
func (OSRuntime) SpawnReviewPane(cwd, lua, absFile, nvimPidFile string) error {
	cmd := exec.Command("zellij", "run", "--floating", "--close-on-exit", "--name", "review",
		"--width", "100%", "--height", "100%", "--x", "0", "--y", "0", "--cwd", cwd,
		"--", "nvim", "-u", lua, absFile)
	cmd.Env = append(os.Environ(), "PAIR_NVIM_PID_FILE="+nvimPidFile)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func (OSRuntime) EstablishedSessionID(dataDir, scopeKey, tag, agent string) (string, sessioninventory.BindingStatus) {
	home, _ := os.UserHomeDir()
	runtime := sessioninventory.NewOSRuntime(home, dataDir)
	query, err := sessioninventory.QuerySession(runtime, scopeKey, tag, sessioninventory.Agent(agent))
	if err != nil || query.Root == nil {
		return "", query.Status
	}
	return query.Root.NativeID, query.Status
}

func (OSRuntime) CanonicalDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
func (OSRuntime) RegularFileWithin(root, rel string) error {
	path := filepath.Join(root, rel)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if resolved != path {
		return fmt.Errorf("review document traverses a symlink")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("review document is not a regular file")
	}
	return nil
}

type limitedIdentityBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *limitedIdentityBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		return 0, errors.New("review history exceeds 8 MiB")
	}
	return b.buffer.Write(p)
}
func (OSRuntime) GitContext(ctx context.Context, limit int, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.WaitDelay = 50 * time.Millisecond
	out := &limitedIdentityBuffer{limit: limit}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return out.buffer.String(), err
}

// ReadIdentityFile captures bounded working-tree text while preserving regular
// file/path identity. The caller rechecks pinned Git HEAD and branch afterwards.
func (rt OSRuntime) ReadIdentityFile(ctx context.Context, root, rel string, limit int) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := rt.RegularFileWithin(root, rel); err != nil {
		return "", err
	}
	path := filepath.Join(root, rel)
	before, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return "", fmt.Errorf("review file changed while opening")
	}
	if opened.Size() > int64(limit) {
		return "", fmt.Errorf("review snapshot exceeds 8 MiB")
	}
	body, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return "", err
	}
	if len(body) > limit {
		return "", fmt.Errorf("review snapshot exceeds 8 MiB")
	}
	after, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !os.SameFile(opened, after) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return "", fmt.Errorf("review file changed during capture")
	}
	if err := rt.RegularFileWithin(root, rel); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return "", fmt.Errorf("review snapshot is not UTF-8 text")
	}
	return string(body), nil
}
