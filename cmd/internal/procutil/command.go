package procutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// ErrUnsupported identifies a platform without native process inspection.
var ErrUnsupported = errors.New("native process inspection unsupported")

// IsProcessGone classifies individual native process reads, never table-wide
// failures. The kernel may discard argv before publishing a zombie table row.
func IsProcessGone(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) || errors.Is(err, syscall.EINVAL)
}

// CommandInfo keeps the executable independently from the process-controlled argv.
type CommandInfo struct {
	Executable string
	Args       []string
}

// Observation describes one process generation. Err makes unavailable command
// evidence explicit; neither executable nor argv is retained on failure.
type Observation struct {
	Process
	CommandInfo
	Err error
}

type processReader interface {
	table() ([]Process, error)
	command(int) (CommandInfo, error)
	identity(int) string
}
type nativeReader struct{}

func (nativeReader) table() ([]Process, error)            { return Table() }
func (nativeReader) command(pid int) (CommandInfo, error) { return nativeCommand(pid) }
func (nativeReader) identity(pid int) string              { return Identity(strconv.Itoa(pid)) }

// Snapshot reads the metadata table once and enriches every row with native
// command evidence, excluding kernel-confirmed terminated rows. Start identities
// fence PID reuse during each read. It keeps
// per-row failures; cancellation or table failure aborts the whole observation.
func Snapshot(ctx context.Context) ([]Observation, error) { return snapshot(ctx, nativeReader{}) }
func snapshot(ctx context.Context, r processReader) ([]Observation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.table()
	if err != nil {
		return nil, err
	}
	out := make([]Observation, 0, len(rows))
	for _, p := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if p.Zombie {
			continue
		}
		o := Observation{Process: p}
		if p.PID <= 0 || p.Identity == "" {
			o.Err = errors.New("process identity unavailable")
		} else {
			command, err := r.command(p.PID)
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if IsProcessGone(err) {
				continue
			}
			identity := r.identity(p.PID)
			if identity != "" && identity != p.Identity {
				continue // Only the old snapshot generation is gone; never adopt its replacement.
			}
			if err == nil && identity == "" {
				err = errors.New("process identity unavailable after command read")
			}
			if err != nil {
				o.Err = fmt.Errorf("process %d: %w", p.PID, err)
			} else {
				o.CommandInfo = command
			}
		}
		out = append(out, o)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Command returns space-joined native argv for the same process generation, or
// empty when the PID or its command evidence is unavailable.
func Command(pid string) string {
	n, ok := positivePID(pid)
	if !ok {
		return ""
	}
	before := Identity(pid)
	if before == "" {
		return ""
	}
	info, err := nativeCommand(n)
	if err != nil || Identity(pid) != before {
		return ""
	}
	return strings.Join(info.Args, " ")
}
