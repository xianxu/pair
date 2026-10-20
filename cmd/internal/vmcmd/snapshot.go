package vmcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SnapshotPlan struct{ Files []string }
type SnapshotBudget struct {
	MaxFiles int
	MaxBytes int64
}
type SnapshotReport struct {
	Head                     string
	Files, Enumerations      int
	ClonedBytes, CopiedBytes int64
	MetadataBytes            int64
}

// PlanSnapshot admits relative tracked paths in one pass. The IO adapter must
// canonicalize roots first and use rooted file access when copying the plan.
func PlanSnapshot(source, destination string, paths []string) (SnapshotPlan, error) {
	for _, root := range []string{source, destination} {
		if err := validateRunPath(root); err != nil || root == string(filepath.Separator) {
			return SnapshotPlan{}, fmt.Errorf("invalid snapshot root")
		}
	}
	if pathOverlap(source, destination) {
		return SnapshotPlan{}, fmt.Errorf("snapshot source and destination overlap")
	}
	seen := make(map[string]bool, len(paths))
	plan := SnapshotPlan{}
	for _, path := range paths {
		first := strings.SplitN(filepath.ToSlash(path), "/", 2)[0]
		if path == "" || path == "." || filepath.IsAbs(path) || filepath.Clean(path) != path || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) || strings.EqualFold(first, ".git") || strings.EqualFold(first, ".couch") || strings.ContainsRune(path, 0) {
			return SnapshotPlan{}, fmt.Errorf("unsafe tracked snapshot path %q", path)
		}
		if !seen[path] {
			seen[path] = true
			plan.Files = append(plan.Files, path)
		}
	}
	return plan, nil
}

func pathOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator))
}

func SnapshotCheckout(ctx context.Context, source, destination string) (SnapshotReport, error) {
	return snapshotCheckout(ctx, source, destination, nativeCloneFile, SnapshotBudget{MaxFiles: 100000, MaxBytes: 2 << 30})
}

type cloneFileFunc func(source, destinationParent *os.File, name string) error

func snapshotCheckout(ctx context.Context, source, destination string, clone cloneFileFunc, budget SnapshotBudget) (SnapshotReport, error) {
	report := SnapshotReport{}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if budget.MaxFiles <= 0 || budget.MaxBytes <= 0 {
		return report, fmt.Errorf("invalid snapshot budget")
	}
	var err error
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return report, err
	}
	destination, err = canonicalFuturePath(destination)
	if err != nil {
		return report, err
	}
	if _, err := PlanSnapshot(source, destination, nil); err != nil {
		return report, err
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return report, fmt.Errorf("snapshot destination must be fresh")
	}
	top, err := snapshotGit(ctx, source, "rev-parse", "--show-toplevel")
	if err != nil {
		return report, err
	}
	canonicalTop, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil || canonicalTop != source {
		return report, fmt.Errorf("checkout must name the repository worktree root")
	}
	head, err := snapshotGit(ctx, source, "rev-parse", "HEAD")
	if err != nil {
		return report, err
	}
	report.Head = strings.TrimSpace(string(head))
	listed, err := snapshotGit(ctx, source, "ls-files", "--cached", "-z")
	if err != nil {
		return report, err
	}
	report.Enumerations = 1
	var paths []string
	if len(listed) > 0 {
		if listed[len(listed)-1] != 0 {
			return report, fmt.Errorf("incomplete tracked-file listing")
		}
		paths = strings.Split(string(listed[:len(listed)-1]), "\x00")
	}
	plan, err := PlanSnapshot(source, destination, paths)
	if err != nil {
		return report, err
	}
	if len(plan.Files) > budget.MaxFiles {
		return report, fmt.Errorf("snapshot exceeds tracked-file budget")
	}
	src, err := os.OpenRoot(source)
	if err != nil {
		return report, err
	}
	defer src.Close()
	infos := make(map[string]os.FileInfo, len(plan.Files))
	var bytesTotal int64
	for _, path := range plan.Files {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		info, err := src.Lstat(path)
		if os.IsNotExist(err) {
			continue
		} // Preserve tracked worktree deletions.
		if err != nil {
			return report, err
		}
		if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			return report, fmt.Errorf("tracked snapshot path is not a file or symlink: %q", path)
		}
		if info.Size() > budget.MaxBytes-bytesTotal {
			return report, fmt.Errorf("snapshot exceeds byte budget")
		}
		bytesTotal += info.Size()
		infos[path] = info
	}
	// Enumerate only the pinned commit and its reachable trees/blobs, never
	// its history or a subsequently changed source default branch.
	objects, err := snapshotGit(ctx, source, "rev-list", "--objects", "--no-object-names", "--no-walk", report.Head)
	if err != nil {
		return report, err
	}
	sizes, err := snapshotGitInput(ctx, source, bytes.NewReader(objects), nil, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return report, err
	}
	// Reserve pack framing, index entries, checkout index paths and fixed Git
	// scaffolding before creating anything. A streaming pack cap enforces the
	// reservation even if the source changes or Git emits unexpected data.
	metadataReserve := int64(65536 + 2*len(listed))
	count := 0
	for _, row := range strings.Split(strings.TrimSpace(string(sizes)), "\n") {
		fields := strings.Fields(row)
		if len(fields) != 3 {
			return report, fmt.Errorf("invalid Git object size report")
		}
		size, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || size < 0 || size > budget.MaxBytes {
			return report, fmt.Errorf("snapshot exceeds Git metadata budget")
		}
		charge := size + size/100 + 256
		if charge > budget.MaxBytes-metadataReserve {
			return report, fmt.Errorf("snapshot exceeds Git metadata budget")
		}
		metadataReserve += charge
		count++
	}
	if count != bytes.Count(objects, []byte{'\n'}) || metadataReserve > budget.MaxBytes-bytesTotal {
		return report, fmt.Errorf("snapshot exceeds combined byte budget")
	}
	format := "sha1"
	if len(report.Head) == 64 {
		format = "sha256"
	}
	if _, err := snapshotGit(ctx, filepath.Dir(destination), "init", "--quiet", "--template=", "--object-format="+format, "--", destination); err != nil {
		return report, err
	}
	packPath := filepath.Join(destination, ".git", "objects", "pack", "snapshot.pack")
	pack, err := os.OpenFile(packPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return report, err
	}
	_, packErr := snapshotGitInput(ctx, source, bytes.NewReader(objects), &snapshotLimitedWriter{writer: pack, remaining: metadataReserve}, "pack-objects", "--stdout", "--no-reuse-object", "--no-reuse-delta", "--window=0", "--compression=0")
	closeErr := pack.Close()
	if err := errors.Join(packErr, closeErr); err != nil {
		return report, err
	}
	if _, err := snapshotGit(ctx, destination, "index-pack", "--index-version=2", packPath); err != nil {
		return report, err
	}
	if err := os.WriteFile(filepath.Join(destination, ".git", "shallow"), []byte(report.Head+"\n"), 0600); err != nil {
		return report, err
	}
	if _, err := snapshotGit(ctx, destination, "update-ref", "HEAD", report.Head); err != nil {
		return report, err
	}
	if _, err := snapshotGit(ctx, destination, "reset", "--mixed", report.Head); err != nil {
		return report, err
	}
	err = filepath.WalkDir(filepath.Join(destination, ".git"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			report.MetadataBytes += info.Size()
			if report.MetadataBytes > metadataReserve {
				return fmt.Errorf("Git metadata exceeded reserved capacity")
			}
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	dst, err := os.OpenRoot(destination)
	if err != nil {
		return report, err
	}
	defer dst.Close()
	for _, path := range plan.Files {
		info, present := infos[path]
		if !present {
			continue
		}
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if err := dst.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return report, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := src.Readlink(path)
			if err != nil {
				return report, err
			}
			if err := dst.Symlink(target, path); err != nil {
				return report, err
			}
		} else {
			cloned, err := copySnapshotFile(ctx, src, dst, path, info, clone)
			if err != nil {
				return report, err
			}
			if cloned {
				report.ClonedBytes += info.Size()
			} else {
				report.CopiedBytes += info.Size()
			}
		}
		report.Files++
	}
	finalHead, err := snapshotGit(ctx, source, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(finalHead)) != report.Head {
		return report, fmt.Errorf("checkout HEAD changed during snapshot")
	}
	return report, nil
}

func copySnapshotFile(ctx context.Context, src, dst *os.Root, path string, expected os.FileInfo, clone cloneFileFunc) (bool, error) {
	in, err := src.OpenFile(path, os.O_RDONLY|snapshotNonblock, 0)
	if err != nil {
		return false, err
	}
	defer in.Close()
	actual, err := in.Stat()
	if err != nil {
		return false, err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(expected, actual) || actual.Size() != expected.Size() || !actual.ModTime().Equal(expected.ModTime()) {
		return false, fmt.Errorf("tracked file changed during snapshot: %q", path)
	}
	parent, err := dst.Open(filepath.Dir(path))
	if err != nil {
		return false, err
	}
	defer parent.Close()
	err = clone(in, parent, filepath.Base(path))
	cloned := err == nil
	if err != nil {
		if !cloneFallbackAllowed(err) {
			return false, err
		}
		out, err := dst.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, actual.Mode().Perm())
		if err != nil {
			return false, err
		}
		n, copyErr := io.Copy(out, io.LimitReader(contextFileReader{ctx: ctx, file: in}, expected.Size()+1))
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			return false, errors.Join(copyErr, closeErr)
		}
		if n != expected.Size() {
			return false, fmt.Errorf("tracked file size changed during copy")
		}
	}
	after, err := src.Lstat(path)
	if err != nil || !os.SameFile(expected, after) || after.Size() != expected.Size() || !after.ModTime().Equal(expected.ModTime()) {
		return false, fmt.Errorf("tracked file changed during snapshot: %q", path)
	}
	return cloned, nil
}

type contextFileReader struct {
	ctx  context.Context
	file *os.File
}

func (r contextFileReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.file.Read(p)
}

// canonicalFuturePath resolves every existing ancestor, including aliases of a
// missing destination leaf, before an overlap decision or filesystem mutation.
func canonicalFuturePath(path string) (string, error) {
	if err := validateRunPath(path); err != nil {
		return "", err
	}
	var missing []string
	for {
		_, err := os.Lstat(path)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return "", err
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		missing = append(missing, filepath.Base(path))
		path = filepath.Dir(path)
	}
}

func snapshotGit(ctx context.Context, root string, args ...string) ([]byte, error) {
	return snapshotGitInput(ctx, root, nil, nil, args...)
}

func snapshotGitInput(ctx context.Context, root string, input io.Reader, output io.Writer, args ...string) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	argv := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-C", root}
	cmd := exec.CommandContext(ctx, "git", append(argv, args...)...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "LANG=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1"}
	collect := configureSnapshotCommand(cmd)
	cmd.WaitDelay = 5 * time.Second
	out := &snapshotOutput{remaining: 16 << 20, cancel: cancel}
	cmd.Stdin = input
	cmd.Stdout, cmd.Stderr = out, out
	if output != nil {
		cmd.Stdout = output
	}
	err := cmd.Run()
	if cleanupErr := collect(); cleanupErr != nil {
		err = errors.Join(err, fmt.Errorf("collect Git process group: %w", cleanupErr))
	}
	if out.overflow {
		return nil, errors.Join(fmt.Errorf("Git snapshot output exceeded 16 MiB"), err)
	}
	if err != nil {
		return out.buffer.Bytes(), fmt.Errorf("Git snapshot operation: %w", err)
	}
	return out.buffer.Bytes(), nil
}

type snapshotOutput struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	remaining int
	overflow  bool
	cancel    context.CancelFunc
}

func (w *snapshotOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(p) > w.remaining {
		w.overflow = true
		w.cancel()
		return 0, fmt.Errorf("snapshot output overflow")
	}
	w.remaining -= len(p)
	return w.buffer.Write(p)
}

// snapshotLimitedWriter bounds binary pack transfer independently of diagnostics.
type snapshotLimitedWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *snapshotLimitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, fmt.Errorf("Git pack exceeded metadata budget")
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}
