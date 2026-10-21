package operationalrun

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/sys/unix"
)

// Callers hold the permanent task lock. A released lock proves that no
// cooperating publisher/collector still owns one of these private stages.
// Open directory handles and captured identities fence cleanup from replacement.
func (s Store) cleanupStages(ctx context.Context, task string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeDir(task, false); err != nil {
		return err
	}
	fd, err := unix.Open(task, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	root := os.NewFile(uintptr(fd), task)
	defer root.Close()
	rootID, err := stageOpenedIdentity(root)
	if err != nil {
		return err
	}
	ns, err := stageNames(root)
	if err != nil {
		return err
	}
	for _, name := range ns {
		if err = ctx.Err(); err != nil {
			return err
		}
		if name == ".collection-stage" {
			id, e := stageChildIdentity(root, name, false)
			if e != nil {
				return e
			}
			if err = s.fail("cleanup-collection-captured"); err != nil {
				return err
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			if err = verifyStageRoot(root, rootID); err != nil {
				return err
			}
			current, e := stageChildIdentity(root, name, false)
			if e != nil {
				return e
			}
			if !id.matches(current) {
				return errors.New("operational collection stage replaced")
			}
			if err = unix.Unlinkat(fd, name, 0); err != nil {
				return err
			}
			if err = s.fail("cleanup-collection-removed"); err != nil {
				return err
			}
			continue
		}
		if !stageName.MatchString(name) {
			continue
		}
		if err = s.cleanupRunStage(ctx, root, rootID, name); err != nil {
			return err
		}
	}
	return root.Sync()
}

func stageOpenedIdentity(f *os.File) (fileIdentity, error) {
	st, err := f.Stat()
	if err != nil {
		return fileIdentity{}, err
	}
	return identityInfo(st)
}
func stageOpenChild(dir *os.File, name string, directory bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat(int(dir.Fd()), name, flags, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), filepath.Join(dir.Name(), name))
	id, err := stageOpenedIdentity(f)
	if err != nil || directory && !os.FileMode(id.Mode).IsDir() || !directory && !os.FileMode(id.Mode).IsRegular() {
		f.Close()
		return nil, errors.New("unsafe operational staging member")
	}
	return f, nil
}
func stageChildIdentity(dir *os.File, name string, directory bool) (fileIdentity, error) {
	f, err := stageOpenChild(dir, name, directory)
	if err != nil {
		return fileIdentity{}, err
	}
	defer f.Close()
	return stageOpenedIdentity(f)
}
func stageNames(dir *os.File) ([]string, error) {
	f, err := stageOpenChild(dir, ".", true)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ns, err := f.Readdirnames(4097)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(ns) > 4096 {
		return nil, errors.New("operational stage inventory exceeds 4096 entries")
	}
	sort.Strings(ns)
	return ns, nil
}
func verifyStageRoot(root *os.File, want fileIdentity) error {
	current, err := identity(root.Name())
	if err != nil {
		return err
	}
	if !want.matches(current) {
		return errors.New("operational task directory replaced")
	}
	return nil
}
func verifyRunStage(root *os.File, rootID fileIdentity, name string, stageID fileIdentity) error {
	if err := verifyStageRoot(root, rootID); err != nil {
		return err
	}
	current, err := stageChildIdentity(root, name, true)
	if err != nil {
		return err
	}
	if !stageID.matches(current) {
		return errors.New("operational publication stage replaced")
	}
	return nil
}

func (s Store) cleanupRunStage(ctx context.Context, root *os.File, rootID fileIdentity, name string) error {
	stage, err := stageOpenChild(root, name, true)
	if err != nil {
		return err
	}
	defer stage.Close()
	id, err := stageOpenedIdentity(stage)
	if err != nil {
		return err
	}
	children, err := stageNames(stage)
	if err != nil {
		return err
	}
	files := map[string]fileIdentity{}
	for _, child := range children {
		if child != "record.json" && child != "output" && !stageFileName.MatchString(child) {
			return errors.New("unknown operational publication staging member")
		}
		files[child], err = stageChildIdentity(stage, child, false)
		if err != nil {
			return err
		}
	}
	return s.cleanupCapturedRunStage(ctx, root, rootID, name, stage, id, children, files)
}

// The expected directory and member identities come from creation for a live
// publisher, or discovery under the task lock for abandoned-stage recovery.
func (s Store) cleanupCapturedRunStage(ctx context.Context, root *os.File, rootID fileIdentity, name string, stage *os.File, id fileIdentity, children []string, files map[string]fileIdentity) error {
	var err error
	if err = s.fail("cleanup-run-captured"); err != nil {
		return err
	}
	if err = verifyCapturedRunStage(root, rootID, name, stage, id, files); err != nil {
		return err
	}
	for _, child := range children {
		if err = s.fail("cleanup-run-before-unlink"); err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = verifyRunStage(root, rootID, name, id); err != nil {
			return err
		}
		got, e := stageChildIdentity(stage, child, false)
		if e != nil {
			return e
		}
		if !files[child].matches(got) {
			return errors.New("operational staging payload replaced")
		}
		if err = unix.Unlinkat(int(stage.Fd()), child, 0); err != nil {
			return err
		}
		if err = s.fail("cleanup-run-member-removed"); err != nil {
			return err
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = verifyRunStage(root, rootID, name, id); err != nil {
		return err
	}
	if current, err := stageNames(stage); err != nil {
		return err
	} else if len(current) != 0 {
		return errors.New("operational staging members changed")
	}
	return unix.Unlinkat(int(root.Fd()), name, unix.AT_REMOVEDIR)
}

// Verify the whole captured bundle before either publishing or deleting it.
func verifyCapturedRunStage(root *os.File, rootID fileIdentity, name string, stage *os.File, id fileIdentity, files map[string]fileIdentity) error {
	if err := verifyRunStage(root, rootID, name, id); err != nil {
		return err
	}
	current, err := stageNames(stage)
	if err != nil {
		return err
	}
	if len(current) != len(files) {
		return errors.New("operational staging members changed")
	}
	for _, child := range current {
		want, ok := files[child]
		if !ok {
			return errors.New("operational staging members changed")
		}
		got, err := stageChildIdentity(stage, child, false)
		if err != nil {
			return err
		}
		if !want.matches(got) {
			return errors.New("operational staging payload replaced")
		}
	}
	return nil
}
