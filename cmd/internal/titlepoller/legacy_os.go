package titlepoller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

var ErrLegacySampleUnsupported = errors.New("complete legacy-title sampling is supported only on Darwin; other platforms keep all processes")

type LegacyReport struct {
	CapturedAt time.Time             `json:"captured_at"`
	Complete   bool                  `json:"complete"`
	Platform   string                `json:"platform"`
	Cutoff     LegacyCutoff          `json:"disclosed_activation_cutoff"`
	Executable string                `json:"exact_title_executable"`
	DataRoot   string                `json:"artifact_owner_data_root"`
	Titles     []LegacyTitleDecision `json:"titles"`
}

// WriteLegacyTitleReport is report-only. Calling it never signals a process or
// writes PID/owner metadata. The supplied cutoff is a disclosed operator input,
// not a build mtime, guessed deployment time, or inferred generation.
func WriteLegacyTitleReport(w io.Writer, sample LegacySample, policy LegacyPolicy) error {
	return json.NewEncoder(w).Encode(LegacyReport{CapturedAt: sample.At, Complete: sample.Complete, Platform: sample.Platform, Cutoff: policy.Cutoff, Executable: policy.Executable, DataRoot: policy.DataRoot, Titles: DecideLegacyTitles(sample, policy)})
}

// ObserveLegacyTitles uses one process snapshot and generation-fenced resource
// reads. Darwin's table is one complete kern.proc.all read. Linux's current
// procutil.Table silently skips unreadable stat rows, so it cannot establish
// complete wrapper absence; fail closed until that shared seam is strict.
func ObserveLegacyTitles(ctx context.Context) (LegacySample, error) {
	reader := procutil.NativeResourceReader{}
	return captureLegacyTitles(ctx, runtime.GOOS, procutil.Snapshot, func(ctx context.Context, p procutil.Process) (LegacyResource, error) {
		usage, _, err := reader.Usage(ctx, p)
		return LegacyResource{RSSBytes: usage.RSSBytes, RSSKnown: err == nil}, err
	}, time.Now)
}
func captureLegacyTitles(ctx context.Context, platform string, snapshot func(context.Context) ([]procutil.Observation, error), usage func(context.Context, procutil.Process) (LegacyResource, error), now func() time.Time) (LegacySample, error) {
	sample := LegacySample{Platform: platform, Resources: map[int]LegacyResource{}}
	if ctx == nil {
		return sample, errors.New("nil sample context")
	}
	if err := ctx.Err(); err != nil {
		return sample, err
	}
	if platform != "darwin" {
		sample.At = now().UTC()
		return sample, ErrLegacySampleUnsupported
	}
	rows, err := snapshot(ctx)
	sample.At = now().UTC()
	if err != nil {
		return sample, err
	}
	sample.Rows = rows
	sample.Complete = true
	for _, row := range rows {
		if ctx.Err() != nil {
			sample.Complete = false
			return sample, ctx.Err()
		}
		if row.Err != nil || row.Zombie || filepath.Base(row.Executable) != "pair" || len(row.Args) != 5 || row.Args[1] != "title" {
			continue
		}
		resource, err := usage(ctx, row.Process)
		if err == nil {
			sample.Resources[row.PID] = resource
		}
	}
	if err := ctx.Err(); err != nil {
		sample.Complete = false
		return sample, err
	}
	return sample, nil
}

// LegacyNativeCleanupIO is intentionally separate from observation/reporting.
// The caller must first obtain TL approval of the exact reported generations.
type LegacyNativeCleanupIO struct{}

func (LegacyNativeCleanupIO) Identity(pid int) (string, error) {
	if pid <= 1 || pid > 1<<31-1 {
		return "", errors.New("invalid legacy title PID")
	}
	if stamp := procutil.Identity(strconv.Itoa(pid)); stamp != "" {
		return stamp, nil
	}
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return "", os.ErrNotExist
	}
	return "", errors.New("legacy title identity unavailable")
}
func (LegacyNativeCleanupIO) Signal(pid int, signal syscall.Signal) error {
	if pid <= 1 || pid > 1<<31-1 || (signal != syscall.SIGTERM && signal != syscall.SIGKILL) {
		return errors.New("invalid legacy title signal")
	}
	return syscall.Kill(pid, signal)
}
func (LegacyNativeCleanupIO) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
