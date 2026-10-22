package procutil

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Reader supplies native process observations. Each Table call enumerates the
// whole host; Command and Identity inspect only the supplied PID.
type Reader interface {
	Table() ([]Process, error)
	Command(int) (CommandInfo, error)
	Identity(int) (string, error)
}

// Source owns process IO without caching observations. Its zero value reads the
// host. Copies share the injected Reader, including its synchronization/lifetime.
type Source struct{ reader processReader }

func NewSource(reader Reader) Source {
	if reader == nil {
		return Source{}
	}
	return Source{reader: readerAdapter{reader}}
}

type readerAdapter struct{ Reader }

func (r readerAdapter) table() ([]Process, error)            { return r.Table() }
func (r readerAdapter) command(pid int) (CommandInfo, error) { return r.Command(pid) }
func (r readerAdapter) identity(pid int) (string, error)     { return r.Identity(pid) }

func (s Source) runtime() processReader {
	if s.reader != nil {
		return s.reader
	}
	return nativeReader{}
}
func (s Source) Table() ([]Process, error) { return s.runtime().table() }

// ReadIdentity returns the native start stamp and error for one positive PID.
// It preserves both results so callers can distinguish absence from uncertainty.
func (s Source) ReadIdentity(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("process identity: PID must be positive: %d", pid)
	}
	return s.runtime().identity(pid)
}

// Identity returns the kernel start stamp for one pid, or "" when it cannot be
// read. Callers fence pid reuse by comparing it against a stamp they already
// hold, and "" never matches: an unreadable identity is uncertainty, not
// absence.
func (s Source) Identity(pid int) string {
	identity, _ := s.runtime().identity(pid)
	return identity
}

// Command returns one pid's argv and whether the read succeeded. ok == false is
// not an empty argv: a caller that attributes a process by its argv must treat
// an unreadable one as unattributed, never as a match.
func (s Source) Command(pid int) (args []string, ok bool) {
	info, err := s.runtime().command(pid)
	if err != nil {
		return nil, false
	}
	return info.Args, true
}

func (s Source) Snapshot(ctx context.Context) ([]Observation, error) {
	return snapshotCapture(ctx, s.runtime(), nil, os.Getenv("PAIR_STARTUP_TIMING"), os.Stderr, time.Now)
}
func (s Source) SnapshotNamed(ctx context.Context, name string) ([]Observation, error) {
	return snapshotCapture(ctx, s.runtime(), func(p Process) bool { return processNameCandidate(p, name) }, os.Getenv("PAIR_STARTUP_TIMING"), os.Stderr, time.Now)
}
