package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type runtimeRegistry struct {
	Version   int                   `json:"version"`
	Processes []ProcessRegistration `json:"processes"`
}

func (c *Coordinator) runtimesPath() string {
	return filepath.Join(c.Root, ".retention", "runtimes.json")
}

// ReadRuntimes is a metadata-only snapshot of upgraded root runtimes. Missing
// metadata is reported as such; reading never creates a registry or reaps it.
func (c *Coordinator) ReadRuntimes() ([]ProcessRegistration, error) {
	if err := checkDirectory(filepath.Join(c.Root, ".retention"), false); err != nil {
		return nil, err
	}
	var registry runtimeRegistry
	if err := readStateJSON(c.runtimesPath(), &registry); err != nil {
		return nil, err
	}
	if registry.Version != 1 || registry.Processes == nil || len(registry.Processes) > 256 {
		return nil, errors.New("invalid runtime registry schema")
	}
	ids := map[string]bool{}
	identities := map[string]bool{}
	for _, entry := range registry.Processes {
		if entry.ID == "" || ids[entry.ID] || entry.Role == "" || len(entry.Role) > 128 || entry.Target != "" || entry.Process.PID <= 0 || !validRuntimeBirth(entry.Process.Birth) {
			return nil, errors.New("invalid runtime registration")
		}
		key := strconv.Itoa(entry.Process.PID) + "/" + entry.Process.Birth + "/" + entry.Role
		if identities[key] {
			return nil, errors.New("duplicate runtime registration")
		}
		identities[key] = true
		ids[entry.ID] = true
	}
	return registry.Processes, nil
}

// RegisterRuntime publishes the actual process incarnation without touching
// owner activity. Repeated registration reuses the record; only positively dead
// incarnations are reaped. Unknown entries count toward the fixed bound.
func (c *Coordinator) RegisterRuntime(ctx context.Context, process ProcessIdentity, role string) error {
	if process.PID <= 0 || !validRuntimeBirth(process.Birth) || role == "" || len(role) > 128 {
		return errors.New("runtime registration needs a strict process identity and role")
	}
	return c.WithLock(ctx, func(l *Locked) error {
		if c.Probe == nil || c.Probe.Inspect(process) != ProcessAlive {
			return errors.New("cannot verify registering runtime")
		}
		entries, err := c.ReadRuntimes()
		if errors.Is(err, os.ErrNotExist) {
			entries = []ProcessRegistration{}
		} else if err != nil {
			return err
		}
		kept := make([]ProcessRegistration, 0, len(entries)+1)
		found, changed := false, false
		for _, entry := range entries {
			if c.Probe.Inspect(entry.Process) == ProcessDead {
				changed = true
				continue
			}
			if entry.Process == process && entry.Role == role {
				found = true
			}
			kept = append(kept, entry)
		}
		if !found {
			if len(kept) >= 256 {
				return errors.New("runtime registration limit reached")
			}
			id, err := randomID()
			if err != nil {
				return err
			}
			kept = append(kept, ProcessRegistration{ID: id, Process: process, Role: role})
			changed = true
		}
		if !changed {
			return nil
		}
		return l.atomicJSON(c.runtimesPath(), runtimeRegistry{Version: 1, Processes: kept})
	})
}

var bootIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validRuntimeBirth(birth string) bool {
	if len(birth) > 128 {
		return false
	}
	parts := strings.Split(birth, ":")
	if len(parts) == 2 && parts[0] == "darwin" {
		timeParts := strings.Split(parts[1], ".")
		if len(timeParts) != 2 {
			return false
		}
		seconds, e1 := strconv.ParseInt(timeParts[0], 10, 64)
		micros, e2 := strconv.ParseInt(timeParts[1], 10, 64)
		return e1 == nil && e2 == nil && seconds > 0 && micros >= 0 && micros < 1000000 && strconv.FormatInt(seconds, 10) == timeParts[0] && strconv.FormatInt(micros, 10) == timeParts[1]
	}
	if len(parts) == 3 && parts[0] == "linux" && bootIDPattern.MatchString(parts[1]) {
		ticks, err := strconv.ParseUint(parts[2], 10, 64)
		return err == nil && ticks > 0 && strconv.FormatUint(ticks, 10) == parts[2]
	}
	return false
}
