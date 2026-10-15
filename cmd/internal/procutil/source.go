package procutil

import (
	"context"
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
func (s Source) Snapshot(ctx context.Context) ([]Observation, error) {
	return snapshotCapture(ctx, s.runtime(), nil, os.Getenv("PAIR_STARTUP_TIMING"), os.Stderr, time.Now)
}
func (s Source) SnapshotNamed(ctx context.Context, name string) ([]Observation, error) {
	return snapshotCapture(ctx, s.runtime(), func(p Process) bool { return processNameCandidate(p, name) }, os.Getenv("PAIR_STARTUP_TIMING"), os.Stderr, time.Now)
}
