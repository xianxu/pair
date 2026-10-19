package sessioninventory

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// directoryFingerprint is separate from regular-file generation evidence. The
// retained descriptor pins its inode; timestamps also detect entry mutations.
type directoryFingerprint struct {
	dev, ino                 uint64
	mode                     uint32
	size                     int64
	msec, mnsec, csec, cnsec int64
}

func fingerprintDirectory(s unix.Stat_t) (directoryFingerprint, error) {
	if s.Mode&unix.S_IFMT != unix.S_IFDIR {
		return directoryFingerprint{}, ErrNotRegularArtifact
	}
	return directoryFingerprint{uint64(s.Dev), uint64(s.Ino), uint32(s.Mode), int64(s.Size), int64(s.Mtim.Sec), int64(s.Mtim.Nsec), int64(s.Ctim.Sec), int64(s.Ctim.Nsec)}, nil
}
func snapshotDirectory(f *os.File) (directoryFingerprint, error) {
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return directoryFingerprint{}, err
	}
	return fingerprintDirectory(st)
}
func unchangedDirectory(f *os.File, before directoryFingerprint) error {
	now, err := snapshotDirectory(f)
	if err != nil {
		return err
	}
	var st unix.Stat_t
	if err := unix.Fstatat(unix.AT_FDCWD, f.Name(), &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	path, err := fingerprintDirectory(st)
	if err != nil {
		return err
	}
	if before != now || before != path {
		return ErrArtifactChanged
	}
	return nil
}

type claudeExactRuntime interface {
	openRecoveryFile(context.Context, StorageRoot, string, bool) (*os.File, error)
	StatRecoveryFile(context.Context, Artifact) (FileEntry, error)
}

// VisitNativeSelectorFiles inspects exact canonical Claude leaves under an
// authorized root with bounded immediate-entry work and no recursive traversal.
func (r OSRuntime) VisitNativeSelectorFiles(ctx context.Context, root StorageRoot, id string, limit int, visit func(FileEntry) error) (int, error) {
	return visitClaudeExactFiles(ctx, r, root, id, limit, visit)
}

// visitClaudeExactFiles never descends into project history. Parent descriptors
// stay alive across each probe, including negative probes, and are checked both
// by descriptor and by the current no-follow pathname before accepting evidence.
func visitClaudeExactFiles(ctx context.Context, r claudeExactRuntime, root StorageRoot, id string, limit int, visit func(FileEntry) error) (int, error) {
	if ctx == nil {
		return 0, errors.New("native selector requires context")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if root.Agent != AgentClaude || !uuidPattern.MatchString(id) {
		return 0, ErrPathEscape
	}
	if limit <= 0 || limit > RecoveryMetadataLimit {
		return 0, ErrReadLimit
	}
	dir, err := r.openRecoveryFile(ctx, root, "", true)
	if err != nil {
		return 0, err
	}
	defer dir.Close()
	before, err := snapshotDirectory(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		entries, err := dir.ReadDir(1)
		if errors.Is(err, io.EOF) {
			return count, unchangedDirectory(dir, before)
		}
		if err != nil {
			return count, err
		}
		if count >= limit {
			return count, ErrReadLimit
		}
		count++
		de := entries[0]
		if de.Type()&os.ModeSymlink != 0 {
			return count, ErrPathEscape
		}
		if !de.IsDir() {
			continue
		}
		err = probeClaudeExactProject(ctx, r, root, de.Name(), id, visit)
		if err != nil {
			return count, err
		}
		if err := unchangedDirectory(dir, before); err != nil {
			return count, err
		}
	}
}
func probeClaudeExactProject(ctx context.Context, r claudeExactRuntime, root StorageRoot, project, id string, visit func(FileEntry) error) error {
	dir, err := r.openRecoveryFile(ctx, root, project, true)
	if err != nil {
		return err
	}
	defer dir.Close()
	before, err := snapshotDirectory(dir)
	if err != nil {
		return err
	}
	entry, probeErr := r.StatRecoveryFile(ctx, Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(filepath.Join(project, id+".jsonl"))})
	if err := unchangedDirectory(dir, before); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(probeErr, os.ErrNotExist) {
		return nil
	}
	if probeErr != nil {
		return probeErr
	}
	return visit(entry)
}
