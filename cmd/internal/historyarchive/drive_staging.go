package historyarchive

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

func namedStageChild(parent *os.File, name string) (*os.File, error) {
	child, err := childDir(parent, name, true)
	if err != nil {
		return nil, err
	}
	defer child.Close()
	fd, err := unix.Dup(int(child.Fd()))
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	return os.NewFile(uintptr(fd), filepath.Join(parent.Name(), name)), nil
}
func openUploadStaging(store string) (*os.File, error) {
	root, err := openSafeDir(store, true)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	// openSafeDir's descriptor name is '.', so retain the validated absolute root.
	fd, err := unix.Dup(int(root.Fd()))
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	named := os.NewFile(uintptr(fd), store)
	defer named.Close()
	return namedStageChild(named, ".drive-staging")
}
func openBatchObjects(stage *os.File) (*os.File, error) { return namedStageChild(stage, "objects") }

func validStagedObject(name string) bool {
	for _, suffix := range []string{".tar.zst.gpg", ".json"} {
		if strings.HasSuffix(name, suffix) {
			id := strings.TrimSuffix(name, suffix)
			return validHex(id, 64) && strings.ToLower(id) == id
		}
	}
	return false
}

// SweepUploadStaging runs only under the continuous pass owner. It removes known
// private link/copy names, never follows a symlink or touches a source pathname.
func SweepUploadStaging(ctx context.Context, dir *os.File) error {
	scan, err := childDir(dir, ".", false)
	if err != nil {
		return err
	}
	defer scan.Close()
	count := 0
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		entries, e := scan.ReadDir(128)
		for _, entry := range entries {
			count++
			if count > MaxFiles {
				return errors.New("Drive staging cleanup exceeds root entry bound")
			}
			name := entry.Name()
			if name == "objects" {
				objects, err := childDir(dir, name, false)
				if err != nil {
					return err
				}
				err = sweepBatchObjects(ctx, objects)
				objects.Close()
				if err != nil {
					return err
				}
				if err = unix.Unlinkat(int(dir.Fd()), name, unix.AT_REMOVEDIR); err != nil {
					return err
				}
			} else {
				if name != ".hashes" && !strings.HasPrefix(name, ".object-") {
					return errors.New("unrecognized Drive staging entry")
				}
				f, err := openRegularAt(dir, name, unix.O_RDONLY)
				if err != nil {
					return err
				}
				f.Close()
				if err = unix.Unlinkat(int(dir.Fd()), name, 0); err != nil {
					return err
				}
			}
		}
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return e
		}
	}
}
func sweepBatchObjects(ctx context.Context, objects *os.File) error {
	count := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, e := objects.ReadDir(128)
		for _, entry := range entries {
			count++
			if count > maxFlatInventoryEntries {
				return errors.New("Drive staging object count exceeds bound")
			}
			if !validStagedObject(entry.Name()) {
				return errors.New("unrecognized Drive batch object")
			}
			f, err := openRegularAt(objects, entry.Name(), unix.O_RDONLY)
			if err != nil {
				return err
			}
			f.Close()
			if err = unix.Unlinkat(int(objects.Fd()), entry.Name(), 0); err != nil {
				return err
			}
		}
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return e
		}
	}
}

// PrepareBatchObject builds a flat view of immutable local bytes. A hard link
// is checked against the opened source descriptor before hashing; EXDEV alone
// permits a bounded verified copy. Both paths preserve the original source.
func PrepareBatchObject(ctx context.Context, source LocalSink, key string, want UploadObject, dir *os.File) (md5sum string, linked bool, err error) {
	if !validStagedObject(want.Name) || want.Size <= 0 || want.Size > MaxCiphertextBytes || !validHex(want.SHA256, 64) {
		return "", false, errors.New("invalid batch object")
	}
	root, err := openSafeDir(source.Root, false)
	if err != nil {
		return "", false, err
	}
	defer root.Close()
	parent, name, err := safeParent(root, key, false)
	if err != nil {
		return "", false, err
	}
	defer parent.Close()
	original, err := openRegularAt(parent, name, unix.O_RDONLY)
	if err != nil {
		return "", false, err
	}
	defer original.Close()
	before, err := original.Stat()
	if err != nil {
		return "", false, err
	}
	if before.Size() != want.Size {
		return "", false, errors.New("local object size mismatch")
	}
	linkErr := source.fault("link-upload")
	if linkErr == nil {
		linkErr = unix.Linkat(int(parent.Fd()), name, int(dir.Fd()), want.Name, 0)
	}
	if linkErr != nil && !errors.Is(linkErr, unix.EXDEV) {
		return "", false, linkErr
	}
	created := linkErr == nil
	defer func() {
		if err != nil && created {
			unix.Unlinkat(int(dir.Fd()), want.Name, 0)
		}
	}()
	hash := md5.New()
	var n int64
	var sha string
	if linkErr == nil {
		linked = true
		staged, e := openRegularAt(dir, want.Name, unix.O_RDONLY)
		if e != nil {
			return "", false, e
		}
		defer staged.Close()
		linkedInfo, e := staged.Stat()
		if e != nil {
			return "", false, e
		}
		if !os.SameFile(before, linkedInfo) || !sameSource(before, linkedInfo) {
			return "", false, errors.New("local object changed during link")
		}
		n, sha, err = fileDigest(ctx, io.TeeReader(staged, hash), nil, want.Size)
		after, e := staged.Stat()
		if err == nil {
			err = e
		}
		if err == nil && !sameSource(before, after) {
			err = errors.New("local object changed during verification")
		}
	} else {
		staged, e := openRegularAt(dir, want.Name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
		if e != nil {
			return "", false, e
		}
		created = true
		n, sha, err = fileDigest(ctx, io.TeeReader(original, hash), staged, want.Size)
		after, e := original.Stat()
		if err == nil {
			err = e
		}
		if err == nil && !sameSource(before, after) {
			err = errors.New("local object changed during copy")
		}
		if err == nil {
			err = staged.Sync()
		}
		err = errors.Join(err, staged.Close())
	}
	if err != nil {
		return "", false, err
	}
	if n != want.Size || sha != want.SHA256 {
		return "", false, errors.New("local encrypted object checksum mismatch")
	}
	return hex.EncodeToString(hash.Sum(nil)), linked, nil
}

func writeDownloadList(ctx context.Context, stage *os.File, names map[string]string) (string, error) {
	if len(names) > maxFlatInventoryEntries {
		return "", errors.New("checksum list exceeds entry bound")
	}
	list := make([]string, 0, len(names))
	for name := range names {
		if !validStagedObject(name) {
			return "", errors.New("invalid checksum list name")
		}
		list = append(list, name)
	}
	sort.Strings(list)
	f, err := openRegularAt(stage, ".hashes", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return "", err
	}
	for _, name := range list {
		if err = ctx.Err(); err != nil {
			break
		}
		if _, err = io.WriteString(f, name+"\n"); err != nil {
			break
		}
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return "", err
	}
	return filepath.Join(stage.Name(), ".hashes"), nil
}
