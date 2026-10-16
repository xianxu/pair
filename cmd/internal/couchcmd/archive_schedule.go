package couchcmd

import (
	"context"
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
	store  string // $PAIR_DATA_DIR/archive
	now    func() time.Time
	launch func(ctx context.Context) error
	notify func(string)
	failed bool
}

const (
	archiveCheckEvery = time.Hour
	archiveEvery      = 24 * time.Hour
)

func newArchiveScheduler(pairData string, notify func(string)) *archiveScheduler {
	store := filepath.Join(pairData, "archive")
	return &archiveScheduler{store: store, now: time.Now, notify: notify, launch: func(ctx context.Context) error {
		pair, err := exec.LookPath("pair")
		if err != nil {
			return err
		}
		logFile, err := os.OpenFile(filepath.Join(store, "continuous.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		defer logFile.Close()
		cmd := exec.CommandContext(ctx, "nice", "-n", "10", pair, "archive", "continuous")
		// The command resolves its store from PAIR_DATA_DIR, which a Pair
		// session scopes to one repository: name the global one explicitly.
		cmd.Env = append(os.Environ(), "PAIR_DATA_DIR="+pairData)
		cmd.Stdout, cmd.Stderr = logFile, logFile
		cmd.WaitDelay = time.Second
		return cmd.Run()
	}}
}

// due reports whether a pass should start now: configured, and no completed
// pass within the last day.
func (s *archiveScheduler) due() bool {
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
	err := s.launch(ctx)
	if err != nil && ctx.Err() == nil && !s.failed {
		s.failed = true
		s.notify("history backup pass failed; see " + filepath.Join(s.store, "continuous.log"))
	}
	if err == nil {
		s.failed = false
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
