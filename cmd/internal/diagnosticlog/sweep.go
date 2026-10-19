package diagnosticlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// SweepSnapshot observes all candidate open files and runtimes once. Its result
// belongs to one sweep only; a later apply must capture a fresh observation.
var errLsofUnavailable = errors.New("lsof unavailable or incomplete")

type SweepSnapshot interface {
	Snapshot(context.Context, []string) (Inspection, error)
}
type sweepKey struct{}
type sweepObservation struct {
	once       sync.Once
	paths      []string
	inspection Inspection
	err        error
}

func SweepContext(ctx context.Context, paths []string) context.Context {
	return context.WithValue(ctx, sweepKey{}, &sweepObservation{paths: append([]string(nil), paths...)})
}
func InspectionProof(inspector Inspection) Proof {
	return func(ctx context.Context, path string, registered []Registration) error {
		effective := inspector
		if sweep, ok := ctx.Value(sweepKey{}).(*sweepObservation); ok {
			if provider, ok := inspector.(SweepSnapshot); ok {
				sweep.once.Do(func() { sweep.inspection, sweep.err = provider.Snapshot(ctx, sweep.paths) })
				if sweep.err != nil {
					return sweep.err
				}
				effective = sweep.inspection
			}
		}
		return VerifyWriters(ctx, path, registered, effective)
	}
}

type observedInspection struct {
	holders             map[string][]Registration
	runtimes            []Runtime
	fileErr, runtimeErr error
}

func (s *observedInspection) OpenFiles(ctx context.Context, path string) ([]Registration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.fileErr != nil {
		return nil, s.fileErr
	}
	holders, ok := s.holders[path]
	if !ok {
		return nil, ErrUnknownWriters
	}
	return holders, nil
}
func (s *observedInspection) Runtimes(ctx context.Context) ([]Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.runtimes, s.runtimeErr
}

func (o OSInspection) Snapshot(ctx context.Context, paths []string) (Inspection, error) {
	result := &observedInspection{holders: map[string][]Registration{}}
	wanted := map[string]bool{}
	aliases := map[string]string{}
	args := []string{"-nP", "-F", "pn"}
	for _, original := range paths {
		path, resolveErr := canonical(original)
		if resolveErr != nil {
			if !os.IsNotExist(resolveErr) {
				result.fileErr = resolveErr
			}
			continue
		}
		aliases[original] = path
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if wanted[path] {
			continue
		}
		wanted[path] = true
		result.holders[path] = nil

	}
	if len(wanted) > 0 {
		run := o.openFileCommand
		if run == nil {
			run = snapshotOpenFiles
		}
		out, stderr, err := run(ctx, args...)
		// lsof's exit1 means no matching open files only with completely empty output
		// and stderr. Partial output or any warning is uncertainty for the whole sweep.
		if err != nil && !(isNoOpenFiles(err) && len(out) == 0 && len(stderr) == 0) || len(stderr) > 0 {
			if err != nil {
				result.fileErr = fmt.Errorf("%w: %w", errLsofUnavailable, err)
			} else {
				result.fileErr = errors.New("lsof reported incomplete inspection")
			}
		} else {
			result.holders, result.fileErr = parseOpenFileSnapshot(ctx, out, wanted)
		}
	}
	for original, path := range aliases {
		if holders, ok := result.holders[path]; ok {
			result.holders[original] = holders
		}
	}
	result.runtimes, result.runtimeErr = o.Runtimes(ctx)
	return result, ctx.Err()
}
func parseOpenFileSnapshot(ctx context.Context, data []byte, wanted map[string]bool) (map[string][]Registration, error) {
	holders := map[string][]Registration{}
	for path := range wanted {
		holders[path] = nil
	}
	births := map[int]string{}
	pid := 0
	pending := false
	for _, line := range strings.Split(string(data), "\n") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			if pending {
				return nil, ErrUnknownWriters
			}
			pending = true
			var err error
			pid, err = strconv.Atoi(line[1:])
			if err != nil || pid <= 0 {
				return nil, ErrUnknownWriters
			}
		case 'f':
			pending = true
			if pid == 0 {
				return nil, ErrUnknownWriters
			}
		case 'n':
			if pid == 0 {
				return nil, ErrUnknownWriters
			}
			pending = false
			owner := line[1:]
			if !wanted[owner] {
				index := strings.Index(owner, ".pair-diagnostics/segments/")
				if index < 0 {
					continue
				}
				logPath := owner[:index]
				name := filepath.Base(owner)
				if !wanted[logPath] || !validSegment(name) || segmentPath(logPath, name) != owner {
					continue
				}
				owner = logPath
			}
			birth := births[pid]
			if birth == "" {
				birth = procutil.StrictIdentity(strconv.Itoa(pid))
				if birth == "" {
					return nil, ErrUnknownWriters
				}
				births[pid] = birth
			}
			pending = false
			holders[owner] = append(holders[owner], Registration{PID: pid, Birth: birth})
		default:
			return nil, ErrUnknownWriters
		}
	}
	if pending {
		return nil, ErrUnknownWriters
	}
	return holders, nil
}

func isNoOpenFiles(err error) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == 1
}

// Bound both native output streams; truncation is an inspection failure, never
// a partially successful open-file inventory.
type snapshotBuffer struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (b *snapshotBuffer) Write(p []byte) (int, error) {
	n := len(p)
	room := max(0, b.limit-b.Len())
	if len(p) > room {
		b.overflow = true
		p = p[:room]
	}
	_, _ = b.Buffer.Write(p)
	return n, nil
}
func snapshotOpenFiles(parent context.Context, args ...string) ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "lsof", args...)
	cmd.WaitDelay = time.Second
	out := &snapshotBuffer{limit: 16 << 20}
	stderr := &snapshotBuffer{limit: 64 << 10}
	cmd.Stdout = out
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if out.overflow || stderr.overflow {
		err = errors.New("diagnostic open-file inventory exceeds output budget")
	}
	return out.Bytes(), stderr.Bytes(), err
}
