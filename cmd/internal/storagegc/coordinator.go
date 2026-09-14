package storagegc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"golang.org/x/sys/unix"
)

// Coordinator serializes retention metadata. It never owns the payload files.
// BeforePersist is a deterministic failure seam for durable-order testing.
type Coordinator struct {
	Root          string
	Now           func() time.Time
	Probe         ProcessProbe
	BeforePersist func() error
	SyncDirectory func(string) error
}

type UseIntent struct {
	ID      string          `json:"id"`
	Process ProcessIdentity `json:"process"`
	Target  string          `json:"target"`
	At      time.Time       `json:"at"`
}

type ProcessRegistration struct {
	ID      string          `json:"id"`
	Process ProcessIdentity `json:"process"`
	Role    string          `json:"role"`
	Target  string          `json:"target,omitempty"`
}

type OwnerState struct {
	Starts    []StartReservation    `json:"starts,omitempty"`
	Activity  ActivityRecord        `json:"activity"`
	Intents   []UseIntent           `json:"intents,omitempty"`
	Processes []ProcessRegistration `json:"processes,omitempty"`
}

func NewCoordinator(root string) (*Coordinator, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("retention root must be absolute")
	}
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	return &Coordinator{Root: physical, Now: time.Now, Probe: OSProcessProbe{}}, nil
}

func (c *Coordinator) validateOwner(o artifactpath.StorageOwner) error {
	valid, err := artifactpath.NewStorageOwner(o.DataDir, o.RepoScope, o.Tag)
	if err != nil || valid != o || o.DataDir != c.Root {
		return errors.New("owner does not belong to retention root")
	}
	return nil
}

func (c *Coordinator) statePath(o artifactpath.StorageOwner) string {
	hash := sha256.Sum256([]byte(o.Key()))
	return filepath.Join(c.Root, ".retention", "owners", hex.EncodeToString(hash[:])+".json")
}

func checkDirectory(path string, create bool) error {
	if create {
		if err := os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe retention directory %s", path)
	}
	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func (c *Coordinator) syncDirectory(path string) error {
	if c.SyncDirectory != nil {
		return c.SyncDirectory(path)
	}
	return syncDirectory(path)
}

// Locked is valid only in the WithLock callback. Callers acquiring a Couch
// store lock do so inside this scope, never in the opposite order.
type Locked struct {
	coordinator *Coordinator
	active      bool
	readOnly    bool
}

func (c *Coordinator) WithLock(ctx context.Context, fn func(*Locked) error) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Join(c.Root, ".retention")
	if err = checkDirectory(dir, true); err != nil {
		return err
	}
	if err = c.syncDirectory(c.Root); err != nil {
		return err
	}
	if err = checkDirectory(filepath.Join(dir, "owners"), true); err != nil {
		return err
	}
	if err = c.syncDirectory(dir); err != nil {
		return err
	}
	fd, err := unix.Open(filepath.Join(dir, "coordinator.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	for {
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	defer func() { err = errors.Join(err, unix.Flock(fd, unix.LOCK_UN)) }()
	lock := &Locked{coordinator: c, active: true}
	defer func() { lock.active = false }()
	return fn(lock)
}

func (c *Coordinator) ReadOwner(o artifactpath.StorageOwner) (OwnerState, error) {
	var state OwnerState
	if err := c.validateOwner(o); err != nil {
		return state, err
	}
	for _, dir := range []string{filepath.Join(c.Root, ".retention"), filepath.Join(c.Root, ".retention", "owners")} {
		if err := checkDirectory(dir, false); err != nil {
			return state, err
		}
	}
	if err := readStateJSON(c.statePath(o), &state); err != nil {
		return state, err
	}
	if err := state.Activity.validate(o); err != nil {
		return state, err
	}
	if len(state.Starts) > 32 {
		return state, errors.New("too many pending starts")
	}
	seenStarts := map[string]bool{}
	for _, start := range state.Starts {
		if err := start.validate(); err != nil {
			return state, err
		}
		if seenStarts[start.ID] {
			return state, errors.New("duplicate pending start")
		}
		seenStarts[start.ID] = true
	}
	for _, intent := range state.Intents {
		if intent.ID == "" || intent.At.IsZero() || intent.Process.PID <= 0 || intent.Process.Birth == "" {
			return state, errors.New("invalid use intent")
		}
	}
	for _, p := range state.Processes {
		if err := validateProcessTarget(o, p.Target); err != nil {
			return state, err
		}
		if p.ID == "" || p.Process.PID <= 0 || p.Process.Birth == "" {
			return state, errors.New("invalid process registration")
		}
	}
	return state, nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (l *Locked) load(o artifactpath.StorageOwner) (OwnerState, error) {
	if !l.active {
		return OwnerState{}, errors.New("expired retention lock")
	}
	if err := l.pendingOwnerTransaction(o); err != nil {
		return OwnerState{}, err
	}
	state, err := l.coordinator.ReadOwner(o)
	if !errors.Is(err, os.ErrNotExist) {
		return state, err
	}
	if err := l.coordinator.validateOwner(o); err != nil {
		return state, err
	}
	id, err := randomID()
	if err != nil {
		return state, err
	}
	now := l.coordinator.Now()
	return OwnerState{Activity: ActivityRecord{Version: 1, Owner: o, Incarnation: id, InitializedAt: now, LastUse: now}}, nil
}

func (l *Locked) save(state OwnerState) error {
	if !l.active {
		return errors.New("expired retention lock")
	}
	c := l.coordinator
	if err := c.validateOwner(state.Activity.Owner); err != nil {
		return err
	}
	return l.atomicJSON(c.statePath(state.Activity.Owner), state)
}

func (c *Coordinator) Initialize(ctx context.Context, o artifactpath.StorageOwner) error {
	return c.WithLock(ctx, func(l *Locked) error {
		s, err := l.load(o)
		if err != nil {
			return err
		}
		return l.save(s)
	})
}

func (c *Coordinator) BeginUse(ctx context.Context, o artifactpath.StorageOwner, p ProcessIdentity, target string) (id string, err error) {
	if target == "" || p.PID <= 0 || p.Birth == "" {
		return "", errors.New("use needs target and process identity")
	}
	if c.Probe == nil || c.Probe.Inspect(p) != ProcessAlive {
		return "", errors.New("cannot verify actual content writer")
	}
	err = c.WithLock(ctx, func(l *Locked) error {
		s, e := l.load(o)
		if e != nil {
			return e
		}
		if len(s.Intents) >= 256 {
			return errors.New("unrecovered use intent limit reached")
		}
		id, e = randomID()
		if e != nil {
			return e
		}
		s.Intents = append(s.Intents, UseIntent{ID: id, Process: p, Target: target, At: c.Now()})
		return l.save(s)
	})
	return id, err
}

func (c *Coordinator) finishUse(ctx context.Context, o artifactpath.StorageOwner, id string, changed bool) error {
	return c.WithLock(ctx, func(l *Locked) error {
		s, err := c.ReadOwner(o)
		if err != nil {
			return err
		}
		found := -1
		for i, v := range s.Intents {
			if v.ID == id {
				found = i
				break
			}
		}
		if found < 0 {
			return errors.New("unknown use operation")
		}
		if changed {
			now := c.Now()
			if now.After(s.Activity.LastUse) {
				s.Activity.LastUse = now
			}
		}
		s.Intents = append(s.Intents[:found], s.Intents[found+1:]...)
		return l.save(s)
	})
}

func (c *Coordinator) CompleteUse(ctx context.Context, o artifactpath.StorageOwner, id string) error {
	return c.finishUse(ctx, o, id, true)
}

// WriteChanged leaves intent intact on any indeterminate payload failure.
// A caller returning changed=false must have performed no content effect.
func (c *Coordinator) WriteChanged(ctx context.Context, o artifactpath.StorageOwner, p ProcessIdentity, target string, write func() (bool, error)) error {
	id, err := c.BeginUse(ctx, o, p, target)
	if err != nil {
		return err
	}
	changed, err := write()
	if err != nil {
		return err
	}
	return c.finishUse(ctx, o, id, changed)
}

// Holds reports whether this callback-scoped token currently owns root's
// coordinator. Store adapters use it to reject inverted or missing locking.
func (l *Locked) Holds(root string) bool {
	return l != nil && l.active && l.coordinator != nil && l.coordinator.Root == root
}
