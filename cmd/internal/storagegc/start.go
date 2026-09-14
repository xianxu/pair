package storagegc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// StartReservation survives its launching parent. Spawned marks the boundary
// after which failure cannot prove that detached children made no effects.
type StartReservation struct {
	ID            string                     `json:"id"`
	Parent        ProcessIdentity            `json:"parent"`
	CreatedAt     time.Time                  `json:"created_at"`
	ExpectedRoles []string                   `json:"expected_roles"`
	Acknowledged  map[string]ProcessIdentity `json:"acknowledged,omitempty"`
	Spawned       bool                       `json:"spawned"`
}

func (s StartReservation) validate() error {
	if s.ID == "" || s.Parent.PID <= 0 || s.Parent.Birth == "" || s.CreatedAt.IsZero() || len(s.ExpectedRoles) == 0 || len(s.ExpectedRoles) > 16 {
		return errors.New("invalid start reservation")
	}
	seen := map[string]bool{}
	for _, r := range s.ExpectedRoles {
		if r == "" || len(r) > 128 || strings.ContainsAny(r, " /\\\t\n") || seen[r] {
			return errors.New("invalid expected start role")
		}
		seen[r] = true
	}
	for role, p := range s.Acknowledged {
		if !seen[role] || p.PID <= 0 || p.Birth == "" || !s.Spawned {
			return errors.New("invalid start acknowledgment")
		}
	}
	return nil
}
func (c *Coordinator) ReserveStart(ctx context.Context, o artifactpath.StorageOwner, parent ProcessIdentity, roles []string) (id string, err error) {
	if c.Probe == nil || c.Probe.Inspect(parent) != ProcessAlive {
		return "", errors.New("cannot verify launch parent")
	}
	err = c.WithLock(ctx, func(l *Locked) error {
		state, err := l.load(o)
		if err != nil {
			return err
		}
		if len(state.Starts) >= 32 {
			return errors.New("pending start limit reached")
		}
		id, err = randomID()
		if err != nil {
			return err
		}
		start := StartReservation{ID: id, Parent: parent, CreatedAt: c.Now(), ExpectedRoles: append([]string(nil), roles...)}
		if err := start.validate(); err != nil {
			return err
		}
		state.Starts = append(state.Starts, start)
		return l.save(state)
	})
	return id, err
}
func (c *Coordinator) MarkStartSpawned(ctx context.Context, o artifactpath.StorageOwner, id string, parent ProcessIdentity) error {
	return c.WithLock(ctx, func(l *Locked) error {
		state, err := c.ReadOwner(o)
		if err != nil {
			return err
		}
		for i, start := range state.Starts {
			if start.ID == id && start.Parent == parent {
				state.Starts[i].Spawned = true
				return l.save(state)
			}
		}
		return errors.New("unknown start reservation")
	})
}

// CancelStartBeforeSpawn is only for the launcher path that has not invoked any
// child creation. A failed/unknown spawn must keep its reservation for recovery.
func (c *Coordinator) CancelStartBeforeSpawn(ctx context.Context, o artifactpath.StorageOwner, id string, parent ProcessIdentity) error {
	return c.WithLock(ctx, func(l *Locked) error {
		state, err := c.ReadOwner(o)
		if err != nil {
			return err
		}
		for i, start := range state.Starts {
			if start.ID != id || start.Parent != parent {
				continue
			}
			if start.Spawned || len(start.Acknowledged) != 0 {
				return errors.New("start may have child effects")
			}
			state.Starts = append(state.Starts[:i], state.Starts[i+1:]...)
			return l.save(state)
		}
		return errors.New("unknown start reservation")
	})
}

// AcknowledgeStart is called after durable process registration while holding
// the SAME root lock. Missing IDs are settled/late helpers and need no mutation.
func (l *Locked) AcknowledgeStart(o artifactpath.StorageOwner, id, role string, process ProcessIdentity) error {
	if id == "" {
		return nil
	}
	state, err := l.load(o)
	if err != nil {
		return err
	}
	for i, start := range state.Starts {
		if start.ID != id {
			continue
		}
		required := false
		for _, r := range start.ExpectedRoles {
			required = required || r == role
		}
		if !required {
			return nil
		}
		if !start.Spawned {
			return errors.New("start has not crossed spawn boundary")
		}
		registered := false
		for _, p := range state.Processes {
			registered = registered || p.Process == process && p.Role == role
		}
		if !registered {
			return errors.New("start acknowledgment has no process registration")
		}
		if start.Acknowledged == nil {
			start.Acknowledged = map[string]ProcessIdentity{}
		}
		start.Acknowledged[role] = process
		state.Starts[i] = start
		if len(start.Acknowledged) == len(start.ExpectedRoles) {
			state.Starts = append(state.Starts[:i], state.Starts[i+1:]...)
			now := l.coordinator.Now()
			if now.After(state.Activity.LastUse) {
				state.Activity.LastUse = now
			}
		}
		return l.save(state)
	}
	return nil
}
