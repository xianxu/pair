package procutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
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
func Snapshot(ctx context.Context) ([]Observation, error) {
	return snapshotCapture(ctx, nativeReader{}, nil, os.Getenv("PAIR_STARTUP_TIMING"), os.Stderr, time.Now)
}

// SnapshotNamed enriches only rows with the exact short kernel name or an
// unavailable name. Metadata selects candidates; native executable and argv
// evidence still determine their meaning. Unknown names retain uncertainty.
func SnapshotNamed(ctx context.Context, name string) ([]Observation, error) {
	return snapshotCapture(ctx, nativeReader{}, func(p Process) bool { return processNameCandidate(p, name) }, os.Getenv("PAIR_STARTUP_TIMING"), os.Stderr, time.Now)
}

// Capture diagnostics count calls at the reader boundary, including failed
// reads and rows that disappear. Output contains no process-table contents or
// requested names. Disabled captures take the original path without a wrapper.
func snapshotCapture(ctx context.Context, r processReader, selectRow func(Process) bool, enabled string, out io.Writer, now func() time.Time) ([]Observation, error) {
	if enabled != "1" {
		return snapshotSelected(ctx, r, selectRow)
	}
	kind := "full"
	if selectRow != nil {
		kind = "named"
	}
	counted := &snapshotCallReader{processReader: r}
	start := now()
	defer func() {
		elapsed := now().Sub(start)
		fmt.Fprintf(out, "pair startup snapshot: pid=%d kind=%s table_calls=%d command_calls=%d identity_calls=%d duration_ms=%.3f\n", os.Getpid(), kind, counted.tables, counted.commands, counted.identities, float64(elapsed)/float64(time.Millisecond))
	}()
	return snapshotSelected(ctx, counted, selectRow)
}

type snapshotCallReader struct {
	processReader
	tables, commands, identities int
}

func (r *snapshotCallReader) table() ([]Process, error) {
	r.tables++
	return r.processReader.table()
}

func (r *snapshotCallReader) command(pid int) (CommandInfo, error) {
	r.commands++
	return r.processReader.command(pid)
}

func (r *snapshotCallReader) identity(pid int) string {
	r.identities++
	return r.processReader.identity(pid)
}

func processNameCandidate(p Process, name string) bool {
	return p.Name == name || p.Name == ""
}

func snapshot(ctx context.Context, r processReader) ([]Observation, error) {
	return snapshotSelected(ctx, r, nil)
}

func snapshotSelected(ctx context.Context, r processReader, selectRow func(Process) bool) ([]Observation, error) {
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
		if p.Zombie || (selectRow != nil && !selectRow(p)) {
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
