//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
)

// SupervisorObservation keeps kernel ownership separate from diagnostic identity.
type SupervisorObservation struct {
	Held  bool
	Owner *SupervisorOwner
}

// ObserveSupervisor never creates a lock or writes metadata. Only a live lease
// handle acquired by this process may exempt its own lock from contention.
func ObserveSupervisor(ns CouchNamespace, proc ProcOps, owned *SupervisorLease) (SupervisorObservation, error) {
	var out SupervisorObservation
	if owned != nil && owned.file != nil && owned.namespace == ns {
		current, e := os.Lstat(filepath.Join(ns.Dir(), supervisorLockName))
		if e != nil {
			return out, e
		}
		held, e := owned.file.Stat()
		if e != nil {
			return out, e
		}
		if !os.SameFile(current, held) {
			return out, errors.New("owned supervisor lock changed")
		}
		return out, nil
	}
	f, e := openSupervisorObservation(filepath.Join(ns.Dir(), supervisorLockName))
	if errors.Is(e, os.ErrNotExist) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	defer f.Close()
	e = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if e == nil {
		return out, unix.Flock(int(f.Fd()), unix.LOCK_UN)
	}
	if !errors.Is(e, unix.EAGAIN) && !errors.Is(e, unix.EWOULDBLOCK) {
		return out, e
	}
	out.Held = true
	metadata, e := openSupervisorObservation(filepath.Join(ns.Dir(), supervisorOwnerName))
	if e != nil {
		return out, e
	}
	defer metadata.Close()
	raw, e := io.ReadAll(io.LimitReader(metadata, 4097))
	if e != nil {
		return out, e
	}
	if len(raw) > 4096 {
		return out, errors.New("supervisor metadata too large")
	}
	var owner SupervisorOwner
	if e = strictjson.Decode(raw, &owner); e != nil {
		return out, e
	}
	if owner.PID <= 0 || owner.Identity == "" {
		return out, errors.New("invalid supervisor owner")
	}
	if proc == nil {
		proc = OSProcOps{}
	}
	if proc.Exists(owner.PID) != Live {
		return out, fmt.Errorf("cannot verify supervisor pid %d", owner.PID)
	}
	identity, e := proc.Identity(owner.PID)
	if e != nil {
		return out, e
	}
	if identity != owner.Identity {
		return out, errors.New("supervisor identity changed")
	}
	out.Owner = &owner
	return out, nil
}
func openSupervisorObservation(path string) (*os.File, error) {
	fd, e := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		f.Close()
		if e != nil {
			return nil, e
		}
		return nil, errors.New("supervisor metadata must be regular")
	}
	return f, nil
}

// WithStoreInspectionLocks fences existing thread stores during an adoption
// recheck. It never initializes an absent store or recovers its journal.
func WithStoreInspectionLocks(namespaces []CouchNamespace, fn func() error) (err error) {
	var locks []*threadStoreLock
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			err = errors.Join(err, locks[i].Close())
		}
	}()
	for _, ns := range namespaces {
		root := filepath.Join(ns.Dir(), "threadstore")
		if _, e := os.Lstat(root); errors.Is(e, os.ErrNotExist) {
			continue
		} else if e != nil {
			return e
		}
		l, e := NewThreadStore(ns).retentionReadLock()
		if e != nil {
			return e
		}
		locks = append(locks, l)
	}
	return fn()
}
