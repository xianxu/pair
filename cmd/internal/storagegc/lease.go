package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// ProcessLease protects one top-level command role for the current process.
// Acquiring the same role again reuses its durable registration across exec;
// these handles are not nested, reference-counted leases.
type ProcessLease struct {
	Coordinator *Coordinator
	Owner       artifactpath.StorageOwner
	ID          string
}

// AcquireSelectedProcess is unmanaged only when both identity variables are
// absent. A partially configured managed process must refuse artifact effects.
func AcquireSelectedProcess(ctx context.Context, getenv func(string) string, role string) (*ProcessLease, error) {
	return AcquireSelectedProcessTarget(ctx, getenv, role, "")
}

func AcquireSelectedProcessTarget(ctx context.Context, getenv func(string) string, role, target string) (*ProcessLease, error) {
	if getenv == nil {
		return nil, errors.New("process lease requires an environment")
	}
	dataDir, tag := getenv("PAIR_DATA_DIR"), getenv("PAIR_TAG")
	if dataDir == "" && tag == "" {
		return nil, nil
	}
	if dataDir == "" || tag == "" || role == "" {
		return nil, errors.New("managed process needs PAIR_DATA_DIR, PAIR_TAG and role")
	}
	owner, err := SelectedOwner(dataDir, getenv("PAIR_SCOPE_KEY"), tag)
	if err != nil {
		return nil, err
	}
	c, err := NewCoordinator(owner.DataDir)
	if err != nil {
		return nil, err
	}
	process, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		return nil, err
	}
	id, err := c.AcquireRoleProcessTarget(ctx, owner, process, role, getenv("PAIR_RETENTION_START_ID"), target)
	if err != nil {
		return nil, err
	}
	return &ProcessLease{Coordinator: c, Owner: owner, ID: id}, nil
}

// AcquireRoleProcess registers an actual process once per role and acknowledges
// its launch reservation before releasing coordination. It is shared by Go
// processes and short CLI helpers registering a long-lived editor PID.
func (c *Coordinator) AcquireRoleProcess(ctx context.Context, owner artifactpath.StorageOwner, process ProcessIdentity, role, startID string) (string, error) {
	return c.AcquireRoleProcessTarget(ctx, owner, process, role, startID, "")
}

// AcquireRoleProcessTarget optionally limits a reader registration to one exact
// file. Empty Target preserves the existing owner-wide lifetime contract.
func (c *Coordinator) AcquireRoleProcessTarget(ctx context.Context, owner artifactpath.StorageOwner, process ProcessIdentity, role, startID, target string) (string, error) {
	if err := validateProcessTarget(owner, target); err != nil {
		return "", err
	}

	if role == "" || process.PID <= 0 || process.Birth == "" {
		return "", errors.New("role acquisition needs an actual process identity")
	}
	var id string
	err := c.WithLock(ctx, func(l *Locked) error {
		if c.Probe == nil || c.Probe.Inspect(process) != ProcessAlive {
			return errors.New("cannot verify actual role process")
		}
		state, err := l.load(owner)
		if err != nil {
			return err
		}
		for _, registration := range state.Processes {
			if registration.Process == process && registration.Role == role && registration.Target == target {
				id = registration.ID
				break
			}
		}
		if id == "" {
			if len(state.Processes) >= 256 {
				return errors.New("process registration limit reached")
			}
			id, err = randomID()
			if err != nil {
				return err
			}
			state.Processes = append(state.Processes, ProcessRegistration{ID: id, Process: process, Role: role, Target: target})
			if err := l.save(state); err != nil {
				return err
			}
		}
		if startID != "" {
			return l.AcknowledgeStart(owner, startID, role, process)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// Close releases only this command's registration and never refreshes use.
// A failed durable release retains its ID so the caller can retry.
func (l *ProcessLease) Close() error {
	if l == nil || l.ID == "" {
		return nil
	}
	if l.Coordinator == nil {
		return errors.New("process lease has no coordinator")
	}
	if err := l.Coordinator.ReleaseProcess(context.Background(), l.Owner, l.ID); err != nil {
		return err
	}
	l.ID = ""
	return nil
}

func validateProcessTarget(owner artifactpath.StorageOwner, target string) error {
	if target == "" {
		return nil
	}
	if !filepath.IsAbs(target) || filepath.Clean(target) != target {
		return errors.New("process target must be an exact absolute path")
	}
	relative, err := filepath.Rel(owner.Directory(), target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("process target is outside selected owner directory")
	}
	if owner.RepoScope == "" && (relative == "repos" || strings.HasPrefix(relative, "repos"+string(filepath.Separator))) {
		return errors.New("flat target cannot select scoped storage")
	}
	return nil
}
