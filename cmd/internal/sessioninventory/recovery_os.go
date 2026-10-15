package sessioninventory

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

var _ RecoveryRuntime = OSRuntime{}

// Native recovery uses local files. Cancellation is checked between bounded
// syscalls; it cannot interrupt a kernel-stalled regular-file/stat syscall.
// Each relative component is opened under the authorized root with openat and
// NOFOLLOW, rather than checking a pathname and following it later.
func (r OSRuntime) openRecoveryFile(ctx context.Context, root StorageRoot, relative string, directory bool) (*os.File, error) {
	if ctx == nil {
		return nil, errors.New("recovery IO requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner, ok := r.authorizedRoot(root.Name)
	if !ok || owner.Agent != root.Agent {
		return nil, ErrRootUnknown
	}
	if relative != "" && !validArtifact(Artifact{StorageRoot: root.Name, RelativePath: relative}) {
		return nil, ErrPathEscape
	}
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NONBLOCK | unix.O_NOFOLLOW
	fd, err := unix.Open(owner.Path, flags|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	if relative == "" {
		return os.NewFile(uintptr(fd), owner.Path), nil
	}
	parts := strings.Split(filepath.FromSlash(relative), string(filepath.Separator))
	for i, part := range parts {
		if err := ctx.Err(); err != nil {
			unix.Close(fd)
			return nil, err
		}
		mode := flags
		if i < len(parts)-1 || directory {
			mode |= unix.O_DIRECTORY
		}
		next, err := unix.Openat(fd, part, mode, 0)
		unix.Close(fd)
		if err != nil {
			return nil, err
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), filepath.Join(owner.Path, filepath.FromSlash(relative))), nil
}
func (r OSRuntime) recoveryRoot(a Artifact) (StorageRoot, error) {
	root, ok := r.authorizedRoot(a.StorageRoot)
	if !ok {
		return StorageRoot{}, ErrRootUnknown
	}
	if !validArtifact(a) {
		return StorageRoot{}, ErrPathEscape
	}
	return root, nil
}
func (r OSRuntime) ResolveRecoveryCWD(ctx context.Context, cwd string) (string, error) {
	if ctx == nil {
		return "", errors.New("recovery IO requires context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !filepath.IsAbs(cwd) {
		return "", ErrPathEscape
	}
	physical, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(physical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", ErrNotRegularArtifact
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return filepath.Clean(physical), nil
}
func (r OSRuntime) StatRecoveryFile(ctx context.Context, a Artifact) (FileEntry, error) {
	root, err := r.recoveryRoot(a)
	if err != nil {
		return FileEntry{}, err
	}
	f, err := r.openRecoveryFile(ctx, root, a.RelativePath, false)
	if err != nil {
		return FileEntry{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return FileEntry{}, err
	}
	if !info.Mode().IsRegular() {
		return FileEntry{}, ErrNotRegularArtifact
	}
	entry, err := fileEntryFor(a, f.Name(), info)
	if err != nil {
		return FileEntry{}, err
	}
	current, err := os.Lstat(f.Name())
	if err != nil {
		return FileEntry{}, err
	}
	if !sameRecoveryInfo(info, current) {
		return FileEntry{}, errors.New("native metadata changed")
	}
	if err := ctx.Err(); err != nil {
		return FileEntry{}, err
	}
	return entry, nil
}
func sameRecoveryInfo(a, b os.FileInfo) bool {
	return a.Mode().IsRegular() && b.Mode().IsRegular() && os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}
func (r OSRuntime) ReadRecoveryAt(ctx context.Context, expected FileEntry, offset, limit int64) ([]byte, bool, error) {
	a := expected.Artifact
	if offset < 0 || limit <= 0 || limit > readChunkSize {
		return nil, false, ErrReadLimit
	}
	root, err := r.recoveryRoot(a)
	if err != nil {
		return nil, false, err
	}
	f, err := r.openRecoveryFile(ctx, root, a.RelativePath, false)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	if !before.Mode().IsRegular() {
		return nil, false, ErrNotRegularArtifact
	}
	// Pin identity to the opened descriptor, not a pathname that could be
	// replaced between the collector's stat and this chunk open.
	descriptor, err := recoveryDescriptorMetadata(f)
	if err != nil {
		return nil, false, err
	}
	opened := FileEntry{Artifact: a, StableFileID: descriptor.StableFileID, GenerationToken: descriptor.GenerationToken, MutationToken: descriptor.MutationToken, Size: before.Size()}
	modified := before.ModTime().UTC()
	opened.ModTime = &modified
	if !sameRecoveryEntry(expected, opened) {
		return nil, false, errors.New("native opened generation differs from snapshot")
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	raw := make([]byte, int(limit))
	n, err := f.ReadAt(raw, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	eof := errors.Is(err, io.EOF)
	after, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	current, err := os.Lstat(f.Name())
	if err != nil {
		return nil, false, err
	}
	if !sameRecoveryInfo(before, after) || !sameRecoveryInfo(after, current) {
		return nil, false, errors.New("native range changed during read")
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	return raw[:n], eof, nil
}
func (r OSRuntime) VisitRecoveryFiles(ctx context.Context, root StorageRoot, limit int, visit func(FileEntry) error) (int, error) {
	if limit <= 0 || limit > RecoveryMetadataLimit {
		return 0, ErrReadLimit
	}
	// Reading one directory entry at a time bounds even a single huge directory;
	// filepath.WalkDir and os.ReadDir eagerly allocate that entire directory.
	queue := []string{""}
	count := 0
	for len(queue) > 0 {
		relative := queue[0]
		queue = queue[1:]
		f, err := r.openRecoveryFile(ctx, root, relative, true)
		if err != nil {
			return count, err
		}
		for {
			if err := ctx.Err(); err != nil {
				f.Close()
				return count, err
			}
			if count >= limit {
				f.Close()
				return count, ErrReadLimit
			}
			entries, err := f.ReadDir(1)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				f.Close()
				return count, err
			}
			for _, de := range entries {
				count++
				rel := filepath.ToSlash(filepath.Join(relative, de.Name()))
				if de.Type()&os.ModeSymlink != 0 {
					f.Close()
					return count, ErrPathEscape
				}
				if de.IsDir() {
					queue = append(queue, rel)
					continue
				}
				a := Artifact{StorageRoot: root.Name, RelativePath: rel}
				entry, err := r.StatRecoveryFile(ctx, a)
				if err != nil {
					f.Close()
					return count, err
				}
				if err := visit(entry); err != nil {
					f.Close()
					return count, err
				}
			}
		}
		if err := f.Close(); err != nil {
			return count, err
		}
	}
	return count, nil
}
