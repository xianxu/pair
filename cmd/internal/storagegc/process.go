package storagegc

import (
	"errors"
	"fmt"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// ProcessIdentity identifies a single process incarnation across PID reuse.
type ProcessIdentity struct {
	PID   int    `json:"pid"`
	Birth string `json:"birth"`
}

type Liveness string

const (
	ProcessAlive   Liveness = "alive"
	ProcessDead    Liveness = "dead"
	ProcessUnknown Liveness = "unknown"
)

// ProcessProbe treats inspection failures as unknown, never as permission to
// remove data. Dead includes a verified different incarnation of the same PID.
type ProcessProbe interface {
	Inspect(ProcessIdentity) Liveness
}

// OSProcessProbe's zero value inspects the local kernel without signaling it.
type OSProcessProbe struct {
	kill     func(int, syscall.Signal) error
	identity func(int) (string, error)
}

func strictBirth(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid process PID %d", pid)
	}
	birth := procutil.StrictIdentity(strconv.Itoa(pid))
	if birth == "" {
		return "", fmt.Errorf("strict process identity unavailable for PID %d", pid)
	}
	return birth, nil
}

func CurrentProcessIdentity(pid int) (ProcessIdentity, error) {
	birth, err := strictBirth(pid)
	if err != nil {
		return ProcessIdentity{}, err
	}
	return ProcessIdentity{PID: pid, Birth: birth}, nil
}

func (p OSProcessProbe) Inspect(owner ProcessIdentity) Liveness {
	// Never pass process-group selectors to a syscall.
	if owner.PID <= 0 {
		return ProcessUnknown
	}
	kill := p.kill
	if kill == nil {
		kill = syscall.Kill
	}
	if err := kill(owner.PID, 0); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return ProcessDead
		}
		return ProcessUnknown
	}
	if owner.Birth == "" {
		return ProcessUnknown
	}
	identity := p.identity
	if identity == nil {
		identity = strictBirth
	}
	current, err := identity(owner.PID)
	if err != nil || current == "" {
		return ProcessUnknown
	}
	if current != owner.Birth {
		return ProcessDead
	}
	return ProcessAlive
}
