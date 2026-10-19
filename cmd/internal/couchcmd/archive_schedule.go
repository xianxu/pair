package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

// The history-backup scheduler (pair#592): once a day, when continuous
// archiving is configured, Couch runs `pair archive continuous` at low
// priority. The command is standalone and holds its own lock, so a manual run
// and this one never overlap; this loop only decides when to start it, from
// the pass's own last-run marker, so a Couch restart neither skips nor repeats
// a day.
type archiveScheduler struct {
	store    string // $PAIR_DATA_DIR/archive
	now      func() time.Time
	launch   func(ctx context.Context) (historyarchive.ContinuousCommandReport, error)
	warn     func(context.Context, string) (string, error)
	warnings archiveWarningGate
	notify   func(string)
	failed   bool
	failedAt time.Time
}

const (
	archiveCheckEvery = time.Hour
	archiveEvery      = 24 * time.Hour
	archiveRetryAfter = 6 * time.Hour // a failed pass is not retried hourly
)

func newArchiveScheduler(pairData string, notify func(string)) *archiveScheduler {
	store := filepath.Join(pairData, "archive")
	s := &archiveScheduler{store: store, now: time.Now, notify: notify}
	s.launch = func(ctx context.Context) (historyarchive.ContinuousCommandReport, error) {
		return runArchivePass(ctx, pairData, store)
	}
	s.warn = func(_ context.Context, text string) (string, error) { s.notify(text); return text, nil }
	return s
}

// archiveReportCapture bounds the protocol independently of the existing log.
// A write error closes exec's stdout copier rather than retaining unbounded data.
type archiveReportCapture struct {
	buffer   bytes.Buffer
	overflow bool
}

func (b *archiveReportCapture) Len() int      { return b.buffer.Len() }
func (b *archiveReportCapture) Bytes() []byte { return b.buffer.Bytes() }

func (b *archiveReportCapture) Write(p []byte) (int, error) {
	left := historyarchive.MaxContinuousReportBytes - b.Len()
	if len(p) > left {
		b.overflow = true
		n, _ := b.buffer.Write(p[:left])
		return n, io.ErrShortBuffer
	}
	return b.buffer.Write(p)
}
func runArchivePass(ctx context.Context, pairData, store string) (historyarchive.ContinuousCommandReport, error) {
	var report historyarchive.ContinuousCommandReport
	pair, err := exec.LookPath("pair")
	if err != nil {
		return report, err
	}
	logFile, err := openArchiveLog(store)
	if err != nil {
		return report, err
	}
	defer logFile.Close()
	cmd := exec.CommandContext(ctx, "nice", "-n", "10", pair, "archive", "continuous", "--json")
	cmd.Env = append(os.Environ(), "PAIR_DATA_DIR="+pairData)
	var raw archiveReportCapture
	cmd.Stdout, cmd.Stderr = &raw, logFile
	cmd.WaitDelay = time.Second
	processErr := cmd.Run()
	_, logErr := logFile.Write(raw.Bytes()) // retain original protocol bytes, even on failure
	if raw.overflow {
		return report, errors.Join(processErr, logErr, errors.New("continuous report exceeds byte limit"))
	}
	report, decodeErr := historyarchive.DecodeContinuousReport(raw.Bytes())
	if decodeErr != nil {
		decodeErr = fmt.Errorf("continuous report: %w", decodeErr)
	}
	if decodeErr == nil && report.Outcome == historyarchive.ContinuousFailed && processErr == nil {
		processErr = errors.New("continuous pass reported failure")
	}
	return report, errors.Join(processErr, decodeErr, logErr)
}

// openArchiveLog opens the pass log fresh: it holds the latest pass only, so it
// never grows without bound.
func openArchiveLog(store string) (*os.File, error) {
	return os.OpenFile(filepath.Join(store, "continuous.log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
}

// due reports whether a pass should start now: configured, and no completed
// pass within the last day.
func (s *archiveScheduler) due() bool {
	if s.failed && s.now().Sub(s.failedAt) < archiveRetryAfter {
		return false
	}
	if _, err := os.Stat(filepath.Join(s.store, historyarchive.ContinuousConfigName)); err != nil {
		return false
	}
	raw, err := os.ReadFile(filepath.Join(s.store, historyarchive.LastRunName))
	if err != nil {
		return true
	}
	last, err := time.Parse(time.RFC3339, strings.TrimSpace(string(raw)))
	return err != nil || s.now().Sub(last) >= archiveEvery
}

// tick starts a pass when due and waits for it, so passes never overlap.
func (s *archiveScheduler) tick(ctx context.Context) {
	if !s.due() {
		return
	}
	report, err := s.launch(ctx)
	if ctx.Err() == nil {
		for _, row := range report.Report.Databases {
			if ctx.Err() != nil {
				break
			}
			now := s.now()
			if !row.NeedsAttention(now, report.MaxDatabaseAge) {
				continue
			}
			s.warnOnce(ctx, now, archiveWarningKey{Source: row.Source, Kind: row.Kind, Path: row.Path}, row.Warning(now, report.MaxDatabaseAge))
		}
		if report.Report.Drive != nil && ctx.Err() == nil {
			observation := report.Report.Drive.Inventory
			if text := observation.Warning(report.MaxDriveFiles); text != "" {
				s.warnOnce(ctx, s.now(), archiveWarningKey{Destination: observation.Destination}, text)
			}
		}
	}
	if err != nil && ctx.Err() == nil {
		if !s.failed {
			s.notify("history backup pass failed; see " + filepath.Join(s.store, "continuous.log"))
		}
		s.failed, s.failedAt = true, s.now()
	}
	if err == nil && report.Outcome == historyarchive.ContinuousComplete {
		s.failed = false
	}
}

// warnOnce reserves before output; uncertain delivery never rearms a daily key.
func (s *archiveScheduler) warnOnce(ctx context.Context, now time.Time, key archiveWarningKey, text string) {
	admit, overflow := s.warnings.Reserve(now, key)
	if !admit && !overflow {
		return
	}
	if overflow {
		text = "History backup alert admission is incomplete: daily source limit exceeded. Ask the hub to repair the backup check."
	}
	text += " Details: " + filepath.Join(s.store, "continuous.log")
	text, warningErr := s.warn(ctx, text)
	if warningErr != nil {
		text += "; notification outcome uncertain: " + warningErr.Error()
	}
	file, logErr := os.OpenFile(filepath.Join(s.store, "continuous.log"), os.O_WRONLY|os.O_APPEND, 0600)
	if logErr == nil {
		_, logErr = file.WriteString("\n" + text + "\n")
		logErr = errors.Join(logErr, file.Close())
	}
	if logErr != nil {
		s.notify("history backup warning could not be retained: " + logErr.Error())
	}
}

func (s *archiveScheduler) run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		s.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
