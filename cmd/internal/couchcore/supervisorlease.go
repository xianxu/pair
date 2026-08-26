package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	supervisorLockName  = "supervisor.lock"
	supervisorOwnerName = "supervisor-owner.json"
)

// SupervisorOwner is diagnostic evidence published only after the namespace
// lease is acquired. Identity is a kernel process-start token, not a PID-only
// assertion.
type SupervisorOwner struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}

// SupervisorLeaseHeldError means another process owns the actor-creating
// authority for this namespace. Owner is nil unless its process identity was
// verified while the lock remained held.
type SupervisorLeaseHeldError struct {
	Namespace CouchNamespace
	Owner     *SupervisorOwner
	Cause     error
}

func (e *SupervisorLeaseHeldError) Error() string {
	if e.Owner != nil {
		return fmt.Sprintf("couch namespace %q is supervised by pid %d (identity %s)", e.Namespace.Dir(), e.Owner.PID, e.Owner.Identity)
	}
	if e.Cause != nil {
		return fmt.Sprintf("couch namespace %q already has a supervisor (owner unavailable: %v)", e.Namespace.Dir(), e.Cause)
	}
	return fmt.Sprintf("couch namespace %q already has a supervisor", e.Namespace.Dir())
}

func (e *SupervisorLeaseHeldError) Unwrap() error { return e.Cause }

// SupervisorLease owns actor creation and terminal attachment for one durable
// namespace. The separate ThreadStore lock remains short-lived.
type SupervisorLease struct {
	namespace CouchNamespace
	file      *os.File
}

// AcquireSupervisorLease takes the lifetime namespace lock and then publishes
// verified owner metadata. A failed publication releases the lock.
func AcquireSupervisorLease(namespace CouchNamespace, proc ProcOps) (*SupervisorLease, error) {
	if namespace.Dir() == "" {
		return nil, fmt.Errorf("acquire supervisor lease: empty namespace")
	}
	file, acquired, err := trySupervisorLock(namespace)
	if err != nil {
		return nil, fmt.Errorf("acquire supervisor lease: %w", err)
	}
	if !acquired {
		owner, held, verifyErr := VerifiedOwner(namespace, proc)
		if held {
			return nil, &SupervisorLeaseHeldError{Namespace: namespace, Owner: &owner}
		}
		return nil, &SupervisorLeaseHeldError{Namespace: namespace, Cause: verifyErr}
	}

	lease := &SupervisorLease{namespace: namespace, file: file}
	pid := os.Getpid()
	identity, err := proc.Identity(pid)
	if err != nil {
		_ = lease.Close()
		return nil, fmt.Errorf("identify supervisor pid %d: %w", pid, err)
	}
	owner := SupervisorOwner{PID: pid, Identity: identity}
	if err := writeSupervisorOwner(namespace, owner); err != nil {
		_ = lease.Close()
		return nil, err
	}
	return lease, nil
}

// VerifiedOwner reports an owner only when the namespace lock is held and the
// published PID still has the exact published process-start identity.
func VerifiedOwner(namespace CouchNamespace, proc ProcOps) (SupervisorOwner, bool, error) {
	file, acquired, err := trySupervisorLock(namespace)
	if err != nil {
		return SupervisorOwner{}, false, err
	}
	if acquired {
		if err := unlockSupervisorFile(file); err != nil {
			return SupervisorOwner{}, false, err
		}
		return SupervisorOwner{}, false, nil
	}

	raw, err := os.ReadFile(filepath.Join(namespace.Dir(), supervisorOwnerName))
	if err != nil {
		return SupervisorOwner{}, false, fmt.Errorf("read supervisor owner: %w", err)
	}
	var owner SupervisorOwner
	if err := json.Unmarshal(raw, &owner); err != nil {
		return SupervisorOwner{}, false, fmt.Errorf("decode supervisor owner: %w", err)
	}
	if owner.PID <= 0 || owner.Identity == "" {
		return SupervisorOwner{}, false, fmt.Errorf("invalid supervisor owner metadata")
	}
	switch proc.Exists(owner.PID) {
	case Dead:
		return SupervisorOwner{}, false, fmt.Errorf("published supervisor pid %d is dead", owner.PID)
	case Unknown:
		return SupervisorOwner{}, false, fmt.Errorf("cannot verify supervisor pid %d", owner.PID)
	}
	identity, err := proc.Identity(owner.PID)
	if err != nil {
		return SupervisorOwner{}, false, fmt.Errorf("verify supervisor pid %d: %w", owner.PID, err)
	}
	if identity != owner.Identity {
		return SupervisorOwner{}, false, fmt.Errorf("supervisor pid %d identity changed", owner.PID)
	}
	return owner, true, nil
}

func writeSupervisorOwner(namespace CouchNamespace, owner SupervisorOwner) error {
	raw, err := json.Marshal(owner)
	if err != nil {
		return fmt.Errorf("encode supervisor owner: %w", err)
	}
	tmp, err := os.CreateTemp(namespace.Dir(), ".supervisor-owner-*")
	if err != nil {
		return fmt.Errorf("create supervisor owner temp file: %w", err)
	}
	tmpName := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod supervisor owner temp file: %w", err)
	}
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write supervisor owner: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync supervisor owner: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close supervisor owner: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(namespace.Dir(), supervisorOwnerName)); err != nil {
		return fmt.Errorf("publish supervisor owner: %w", err)
	}
	removeTemp = false
	return nil
}

func (l *SupervisorLease) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	removeErr := os.Remove(filepath.Join(l.namespace.Dir(), supervisorOwnerName))
	if errors.Is(removeErr, os.ErrNotExist) {
		removeErr = nil
	}
	return errors.Join(removeErr, unlockSupervisorFile(file))
}
