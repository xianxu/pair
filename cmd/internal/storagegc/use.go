package storagegc

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// RegisterProcess publishes the actual writer/reader identity. It protects
// lifetime independently from meaningful use; background work never touches
// last-use merely by registering or exiting.
func (c *Coordinator) RegisterProcess(ctx context.Context, o artifactpath.StorageOwner, p ProcessIdentity, role string) (id string, err error) {
	if role == "" || p.PID <= 0 || p.Birth == "" {
		return "", errors.New("registration needs role and process identity")
	}
	if c.Probe == nil || c.Probe.Inspect(p) != ProcessAlive {
		return "", errors.New("cannot verify registering process")
	}
	err = c.WithLock(ctx, func(l *Locked) error {
		s, e := l.load(o)
		if e != nil {
			return e
		}
		if len(s.Processes) >= 256 {
			return errors.New("process registration limit reached")
		}
		id, e = randomID()
		if e != nil {
			return e
		}
		s.Processes = append(s.Processes, ProcessRegistration{ID: id, Process: p, Role: role})
		return l.save(s)
	})
	return id, err
}

func (c *Coordinator) ReleaseProcess(ctx context.Context, o artifactpath.StorageOwner, id string) error {
	return c.WithLock(ctx, func(l *Locked) error {
		s, err := c.ReadOwner(o)
		if err != nil {
			return err
		}
		for i, p := range s.Processes {
			if p.ID == id {
				s.Processes = append(s.Processes[:i], s.Processes[i+1:]...)
				return l.save(s)
			}
		}
		return errors.New("unknown process registration")
	})
}

// RecoverUse keeps unresolved live/unknown effects blocking collection. A
// verified dead writer receives conservative recovery-time grace because its
// final content effect might have happened long after intent creation.
func (c *Coordinator) RecoverUse(ctx context.Context, o artifactpath.StorageOwner) error {
	return c.WithLock(ctx, func(l *Locked) error { return l.recoverUse(o) })
}
func (l *Locked) recoverUse(o artifactpath.StorageOwner) error {
	c := l.coordinator
	s, err := c.ReadOwner(o)
	if err != nil {
		return err
	}
	if c.Probe == nil {
		return errors.New("process probe unavailable")
	}
	beforeIntents, beforeProcesses, beforeStarts := len(s.Intents), len(s.Processes), len(s.Starts)
	kept := s.Intents[:0]
	recovered := false
	for _, v := range s.Intents {
		if c.Probe.Inspect(v.Process) != ProcessDead {
			kept = append(kept, v)
		} else {
			recovered = true
		}
	}
	s.Intents = kept
	if recovered {
		now := c.Now()
		if now.After(s.Activity.LastUse) {
			s.Activity.LastUse = now
		}
	}
	processes := s.Processes[:0]
	for _, p := range s.Processes {
		if c.Probe.Inspect(p.Process) != ProcessDead {
			processes = append(processes, p)
		}
	}
	s.Processes = processes
	s.Starts = recoverUnspawnedStarts(s.Starts, c.Probe)
	if len(s.Intents) == beforeIntents && len(s.Processes) == beforeProcesses && len(s.Starts) == beforeStarts {
		return nil
	}
	return l.save(s)
}

// CancelUnchangedUse retires an intent only when its caller has verified that
// the operation performed no content effect. Indeterminate writes must retain it.
func (c *Coordinator) CancelUnchangedUse(ctx context.Context, o artifactpath.StorageOwner, id string) error {
	return c.finishUse(ctx, o, id, false)
}
