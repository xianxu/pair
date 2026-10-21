package sessionsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// operatorLockFiles is the filesystem boundary shared by the portable backend
// and stateful uncertainty tests. All calls run inside the registry transaction.
type operatorLockFiles interface {
	Read(string) ([]byte, error)
	Publish(string, []byte) error
	Remove(string) error
	SyncDir(string) error
}

type osOperatorLockFiles struct{}

func (osOperatorLockFiles) Read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular operator lock file", path)
	}
	return readRegular(path, 4096)
}

func operatorStagePath(lockPath string) string {
	return filepath.Join(filepath.Dir(lockPath), "."+filepath.Base(lockPath)+".operator-stage")
}

func (osOperatorLockFiles) Publish(path string, raw []byte) error {
	return durablefile.WriteAtomicStaged(path, raw, operatorStagePath(path))
}

func (osOperatorLockFiles) Remove(path string) error  { return os.Remove(path) }
func (osOperatorLockFiles) SyncDir(path string) error { return durablefile.SyncDirectory(path) }

func operatorPathError(err error) error {
	return registryRefusal("operator-path", err.Error())
}

func operatorUncertain(err error) error {
	return registryRefusal("operator-lock-uncertain", fmt.Sprintf("operator protection durability is unconfirmed: %v; retain the turn and retry the identical command", err))
}

// AcquireOperator acknowledges protection only after earlier registered writers
// have completed and the exact turn's sidecar has directory-confirmed durability.
func (s *Sync) AcquireOperator(req OperatorLockRequest) (OperatorLock, error) {
	return s.operatorLock(req, operatorAcquire)
}

// ReleaseOperator removes only this registration's exact turn, including copied
// move protection at the new canonical path. An absent lock grants no ownership.
func (s *Sync) ReleaseOperator(req OperatorLockRequest) (OperatorLock, error) {
	return s.operatorLock(req, operatorRelease)
}

func (s *Sync) operatorLock(req OperatorLockRequest, action operatorLockAction) (OperatorLock, error) {
	turn := operatorTurn{ID: req.ID, Token: req.Turn}
	if err := validateOperatorTurn(turn); err != nil {
		return OperatorLock{}, err
	}
	result, err := withRegistry(s, func(t *registryTxn) (OperatorLock, error) {
		canonical, err := canonicalDocument(req.File)
		if err != nil {
			return OperatorLock{}, operatorPathError(err)
		}
		d := t.find(canonical)
		if d == nil {
			return OperatorLock{}, registryRefusal("operator-path", "operator protection requires a currently registered document")
		}
		if d.ID != req.ID {
			return OperatorLock{}, registryRefusal("operator-identity", "the document registration changed; inspect its current identity before retrying")
		}
		info, err := os.Lstat(canonical)
		if err != nil {
			return OperatorLock{}, operatorPathError(err)
		}
		if !info.Mode().IsRegular() {
			return OperatorLock{}, registryRefusal("operator-path", "registered document is not a regular file")
		}
		paths, err := documentLockPaths(*d, req.File)
		if err != nil {
			return OperatorLock{}, operatorPathError(err)
		}
		files := t.store.OperatorLocks
		if files == nil {
			files = osOperatorLockFiles{}
		}
		observed := make([]operatorLockObservation, 0, len(paths))
		for _, path := range paths {
			raw, err := files.Read(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return OperatorLock{}, operatorPathError(err)
			}
			observed = append(observed, operatorLockObservation{Path: path, Text: string(raw), Exists: err == nil})
		}
		lockPath := canonical + ".lock"
		decision, err := decideOperatorLock(turn, action, lockPath, observed)
		if err != nil {
			return OperatorLock{}, err
		}
		// Only the single reserved private stage belongs to this transaction.
		// Reclaim it after a crash, including on a no-publication retry/release.
		stage := operatorStagePath(lockPath)
		if _, err := files.Read(stage); err == nil {
			if err := files.Remove(stage); err != nil && !errors.Is(err, os.ErrNotExist) {
				return OperatorLock{}, operatorUncertain(err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return OperatorLock{}, operatorPathError(err)
		}
		res := OperatorLock{ID: d.ID, File: canonical, Turn: req.Turn, LockPath: lockPath, State: decision.State}
		switch decision.State {
		case OperatorAcquired:
			res.Holder = HolderOperator
			res.LockText = operatorLockText(turn, s.now())
			if err := files.Publish(lockPath, []byte(res.LockText)); err != nil {
				return OperatorLock{}, operatorUncertain(err)
			}
		case OperatorAlreadyHeld:
			res.Holder = HolderOperator
			res.LockText = decision.LockText
			if decision.LockPath != lockPath {
				// Alias-only ownership is the same turn, but the editor needs one
				// canonical on-disk witness. Keep exact proof bytes and leave
				// alias cleanup to the existing lock/move lifetime owner.
				if err := files.Publish(lockPath, []byte(res.LockText)); err != nil {
					return OperatorLock{}, operatorUncertain(err)
				}
			}
		case OperatorReleased:
			for _, path := range decision.Remove {
				if err := files.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					return OperatorLock{}, operatorUncertain(err)
				}
			}
		}
		// Absence can follow an earlier unlink with failed fsync. Confirm every
		// admitted directory even on already-released and already-held retries.
		confirmed := map[string]bool{}
		for _, path := range paths {
			dir := filepath.Dir(path)
			if !confirmed[dir] {
				if err := files.SyncDir(dir); err != nil {
					return OperatorLock{}, operatorUncertain(err)
				}
				confirmed[dir] = true
			}
		}
		return res, nil
	})
	var unavailable *registryAccessError
	if errors.As(err, &unavailable) {
		return OperatorLock{}, &Refusal{Exit: ExitUnavailable, Code: "session-sync-unavailable", Reason: unavailable.Error()}
	}
	var refusal *Refusal
	if err != nil && !errors.As(err, &refusal) {
		return OperatorLock{}, registryInvalid(err.Error())
	}
	return result, err
}

// registryAccessError marks store access IO, not validation, document paths,
// lock ownership or callback failures. Existing verbs keep the same message.
type registryAccessError struct{ err error }

func (e *registryAccessError) Error() string { return e.err.Error() }
func (e *registryAccessError) Unwrap() error { return e.err }

func registryAccessFailure(err error) error {
	// Invalid authority shapes are not unavailable transport/storage. They
	// must never select the editor's degraded-but-editable availability policy.
	for _, invalid := range []error{syscall.ENOTDIR, syscall.EISDIR, syscall.ELOOP, syscall.ENAMETOOLONG, syscall.EINVAL} {
		if errors.Is(err, invalid) {
			return err
		}
	}
	var pathErr *os.PathError
	var errno syscall.Errno
	if errors.As(err, &pathErr) || errors.As(err, &errno) {
		return &registryAccessError{err: err}
	}
	return err
}
