package historyarchive

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const maxStagingOwnerBytes = 16 << 10

// OwnedStaging is a private, stamped work directory. The caller must hold its
// parent/staging lifetime lock until Close returns. Dir must not be closed by
// the caller; Close verifies ownership before removing any contents.
type OwnedStaging struct {
	Dir        *os.File
	parent     *os.File
	name       string
	owner      StagingOwner
	identity   unix.Stat_t
	maxEntries int
	closed     bool
}

// OpenOwnedStaging exclusively creates a stage or recovers an interrupted stage
// carrying a valid owner stamp. A foreign or unrecognized tree is left intact.
func OpenOwnedStaging(parent *os.File, name, namespace string, maxEntries int) (*OwnedStaging, error) {
	if parent == nil || name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00") || maxEntries < 1 {
		return nil, errors.New("invalid owned staging arguments")
	}
	if namespace != "history-label-index" && namespace != "history-restore" {
		return nil, errors.New("unsupported staging namespace")
	}
	var ps unix.Stat_t
	if err := unix.Fstat(int(parent.Fd()), &ps); err != nil {
		return nil, err
	}
	if ps.Mode&unix.S_IFMT != unix.S_IFDIR {
		return nil, errors.New("staging parent is not a directory")
	}
	held, err := childDir(parent, ".", false)
	if err != nil {
		return nil, err
	}
	s := &OwnedStaging{parent: held, name: name, maxEntries: maxEntries, owner: StagingOwner{Version: 1, Namespace: namespace, ParentDevice: uint64(ps.Dev), ParentInode: uint64(ps.Ino)}}
	err = unix.Mkdirat(int(held.Fd()), name, 0700)
	if errors.Is(err, unix.EEXIST) {
		s.Dir, err = childDir(held, name, false)
		if err == nil {
			err = unix.Fstat(int(s.Dir.Fd()), &s.identity)
		}
		if err == nil {
			err = s.recoverOwner()
		}
		if err == nil {
			err = s.cleanup()
		}
		if s.Dir != nil {
			err = errors.Join(err, s.Dir.Close())
			s.Dir = nil
		}
		if err == nil {
			err = unix.Mkdirat(int(held.Fd()), name, 0700)
		}
	}
	if err != nil {
		held.Close()
		return nil, fmt.Errorf("open owned staging: %w", err)
	}
	// From here the directory was created by this invocation, so failures can
	// safely remove its own stamp and the still-empty directory.
	s.Dir, err = childDir(held, name, false)
	if err == nil {
		err = unix.Fstat(int(s.Dir.Fd()), &s.identity)
	}
	if err == nil {
		var nonce [16]byte
		_, err = rand.Read(nonce[:])
		s.owner.Nonce = hex.EncodeToString(nonce[:])
	}
	if err == nil {
		err = s.writeOwner()
	}
	if err == nil {
		err = s.Dir.Sync()
	}
	if err == nil {
		err = held.Sync()
	}
	if err != nil {
		if s.Dir != nil {
			unix.Unlinkat(int(s.Dir.Fd()), ".owner.json", 0)
			s.Dir.Close()
		}
		unix.Unlinkat(int(held.Fd()), name, unix.AT_REMOVEDIR)
		held.Close()
		return nil, err
	}
	// Keep a useful pathname for integrations using external tools. All internal
	// filesystem authority remains descriptor-relative.
	fd, err := unix.Dup(int(s.Dir.Fd()))
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}
	unix.CloseOnExec(fd)
	s.Dir.Close()
	s.Dir = os.NewFile(uintptr(fd), filepath.Join(parent.Name(), name))
	return s, nil
}

func (s *OwnedStaging) writeOwner() error {
	b, err := json.Marshal(s.owner)
	if err != nil {
		return err
	}
	f, err := openRegularAt(s.Dir, ".owner.json", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	return errors.Join(err, f.Close())
}

func (s *OwnedStaging) readOwner() (StagingOwner, error) {
	var owner StagingOwner
	f, err := openRegularAt(s.Dir, ".owner.json", unix.O_RDONLY)
	if err != nil {
		return owner, err
	}
	b, err := io.ReadAll(io.LimitReader(f, maxStagingOwnerBytes+1))
	err = errors.Join(err, f.Close())
	if err != nil {
		return owner, err
	}
	if len(b) > maxStagingOwnerBytes {
		return owner, errors.New("staging owner exceeds size bound")
	}
	err = json.Unmarshal(b, &owner)
	return owner, err
}

func (s *OwnedStaging) recoverOwner() error {
	owner, err := s.readOwner()
	if err != nil {
		return err
	}
	if owner.Version != 1 || owner.Namespace != s.owner.Namespace || owner.ParentDevice != s.owner.ParentDevice || owner.ParentInode != s.owner.ParentInode || len(owner.Nonce) != 32 || !validHex(owner.Nonce, 32) || strings.ToLower(owner.Nonce) != owner.Nonce {
		return errors.New("unrecognized staging owner")
	}
	s.owner = owner
	return nil
}

type stagingEntry struct {
	name     string
	identity unix.Stat_t
	children []stagingEntry
}

// scanStaging preflights the entire tree, bounded in entries and bytes. It does
// not follow symlinks, including on filesystems without directory entry types.
func scanStaging(dir *os.File, root bool, remaining *int) ([]stagingEntry, error) {
	scan, err := childDir(dir, ".", false)
	if err != nil {
		return nil, err
	}
	defer scan.Close()
	var nodes []stagingEntry
	for {
		entries, readErr := scan.ReadDir(128)
		for _, entry := range entries {
			*remaining--
			if *remaining < 0 {
				return nil, errors.New("owned staging exceeds entry bound")
			}
			node := stagingEntry{name: entry.Name()}
			if err = unix.Fstatat(int(dir.Fd()), node.name, &node.identity, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return nil, err
			}
			mode := node.identity.Mode & unix.S_IFMT
			if mode != unix.S_IFDIR && mode != unix.S_IFREG {
				return nil, errors.New("staging contains nonregular payload")
			}
			if root {
				switch node.name {
				case ".owner.json", "files-from":
					if mode != unix.S_IFREG {
						return nil, errors.New("staging root file has wrong type")
					}
				case "labels", "ciphertext":
					if mode != unix.S_IFDIR {
						return nil, errors.New("staging payload directory has wrong type")
					}
				case "output":
				default:
					return nil, errors.New("unrecognized staging root child")
				}
			}
			if mode == unix.S_IFDIR {
				child, err := childDir(dir, node.name, false)
				if err != nil {
					return nil, err
				}
				node.children, err = scanStaging(child, false, remaining)
				err = errors.Join(err, child.Close())
				if err != nil {
					return nil, err
				}
			}
			nodes = append(nodes, node)
		}
		if readErr == io.EOF {
			return nodes, nil
		}
		if readErr != nil {
			return nil, readErr
		}
	}
}

func sameStagingIdentity(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode&unix.S_IFMT == b.Mode&unix.S_IFMT
}

func removeStagingEntries(dir *os.File, nodes []stagingEntry) error {
	// Remove the authority stamp last, preserving recovery after partial failure.
	for _, node := range nodes {
		if node.name == ".owner.json" {
			continue
		}
		if err := removeStagingEntry(dir, node); err != nil {
			return err
		}
	}
	for _, node := range nodes {
		if node.name == ".owner.json" {
			if err := removeStagingEntry(dir, node); err != nil {
				return err
			}
		}
	}
	return dir.Sync()
}

func removeStagingEntry(dir *os.File, node stagingEntry) error {
	var current unix.Stat_t
	if err := unix.Fstatat(int(dir.Fd()), node.name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if !sameStagingIdentity(current, node.identity) {
		return errors.New("staging entry changed after preflight")
	}
	flags := 0
	if node.identity.Mode&unix.S_IFMT == unix.S_IFDIR {
		child, err := childDir(dir, node.name, false)
		if err != nil {
			return err
		}
		err = removeStagingEntries(child, node.children)
		err = errors.Join(err, child.Close())
		if err != nil {
			return err
		}
		flags = unix.AT_REMOVEDIR
	}
	return unix.Unlinkat(int(dir.Fd()), node.name, flags)
}

func (s *OwnedStaging) cleanup() error {
	owner, err := s.readOwner()
	if err != nil {
		return err
	}
	if owner.Version != s.owner.Version || owner.Namespace != s.owner.Namespace || owner.Nonce != s.owner.Nonce || owner.ParentDevice != s.owner.ParentDevice || owner.ParentInode != s.owner.ParentInode {
		return errors.New("staging ownership changed")
	}
	var current unix.Stat_t
	if err = unix.Fstatat(int(s.parent.Fd()), s.name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if !sameStagingIdentity(current, s.identity) {
		return errors.New("staging directory changed")
	}
	remaining := s.maxEntries
	nodes, err := scanStaging(s.Dir, true, &remaining)
	if err != nil {
		return err
	}
	if err = removeStagingEntries(s.Dir, nodes); err != nil {
		return err
	}
	if err = unix.Unlinkat(int(s.parent.Fd()), s.name, unix.AT_REMOVEDIR); err != nil {
		return err
	}
	return s.parent.Sync()
}

// Close preflights and removes only a recognized owned tree, then closes all
// descriptors even if cleanup fails. A refused tree remains available to inspect.
func (s *OwnedStaging) Close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	err := s.cleanup()
	return errors.Join(err, s.Dir.Close(), s.parent.Close())
}
