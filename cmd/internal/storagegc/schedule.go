package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// Scheduler runs one bounded batch at a time. Batch owns its own root-lock
// transactions; the separate, permanent scheduler lock serializes whole batches.
type Scheduler struct {
	Coordinator *Coordinator
	Now         func() time.Time
	Interval    time.Duration
	Batch       func(context.Context, string, int) (next string, complete bool, err error)
	Limit       int
	WorkBudget  time.Duration
}

type scheduleState struct {
	Version   int       `json:"version"`
	Cursor    string    `json:"cursor"`
	Completed time.Time `json:"completed"`
}

// RunOnce reports whether Batch ran, including when that batch failed.
func (s *Scheduler) RunOnce(ctx context.Context) (bool, error) {
	ran, _, err := s.runOnce(ctx)
	return ran, err
}

func (s *Scheduler) runOnce(ctx context.Context) (ran bool, delay time.Duration, err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	if s.Coordinator == nil || s.Batch == nil {
		return false, 0, errors.New("scheduler requires coordinator and batch")
	}
	interval := s.Interval
	if interval == 0 {
		interval = 24 * time.Hour
	}
	limit := s.Limit
	if limit == 0 {
		limit = 100
	}
	if interval < 0 || limit < 1 || limit > 10000 {
		return false, 0, errors.New("invalid scheduler budget")
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	c := s.Coordinator
	dir := filepath.Join(c.Root, ".retention")
	if err = checkDirectory(dir, true); err != nil {
		return
	}
	if err = c.syncDirectory(c.Root); err != nil {
		return
	}
	fd, openErr := unix.Open(filepath.Join(dir, "scheduler.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if openErr != nil {
		return false, 0, openErr
	}
	defer unix.Close(fd)
	var stat unix.Stat_t
	if err = unix.Fstat(fd, &stat); err != nil {
		return
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return false, 0, errors.New("unsafe scheduler lock")
	}
	if err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return false, time.Minute, nil
		}
		return
	}
	defer func() { err = errors.Join(err, unix.Flock(fd, unix.LOCK_UN)) }()
	state := scheduleState{Version: 1}
	path := filepath.Join(dir, "schedule.json")
	if err = readStateJSON(path, &state); errors.Is(err, os.ErrNotExist) {
		err = nil
	} else if err != nil {
		return
	}
	if state.Version != 1 || len(state.Cursor) > 4096 {
		return false, 0, errors.New("invalid scheduler state")
	}
	if state.Cursor == "" && !state.Completed.IsZero() {
		delay = state.Completed.Add(interval).Sub(now())
		if delay > 0 {
			return false, delay, nil
		}
	}
	if err = ctx.Err(); err != nil {
		return false, 0, err
	}
	budget := s.WorkBudget
	if budget == 0 {
		budget = 2 * time.Second
	}
	if budget < 0 {
		return false, 0, errors.New("invalid maintenance time budget")
	}
	workCtx, cancel := context.WithTimeout(ctx, budget)
	next, complete, batchErr := s.Batch(workCtx, state.Cursor, limit)
	cancel()
	if ctx.Err() == nil && (errors.Is(batchErr, context.DeadlineExceeded) || errors.Is(batchErr, ErrCoordinatorBusy) || errors.Is(batchErr, ErrMaintenanceYield)) {
		return true, time.Minute, nil
	}
	if batchErr != nil {
		return true, 0, batchErr
	}
	if err = ctx.Err(); err != nil {
		return true, 0, err
	}
	if len(next) > 4096 || (!complete && (next == "" || next == state.Cursor)) {
		return true, 0, errors.New("batch did not advance scheduler cursor")
	}
	state.Cursor = next
	if complete {
		state.Cursor = ""
		state.Completed = now()
		delay = interval
	} else {
		delay = 0
	}
	err = c.TryWithLock(ctx, func(l *Locked) error { return l.atomicJSON(path, state) })
	if errors.Is(err, ErrCoordinatorBusy) {
		return true, time.Minute, nil
	}
	return true, delay, err
}

const schedulerBurstLimit = 16

// ScheduleWorker belongs to the calling runtime. Stop cancels it; Wait joins it.
// A batch error terminates the worker and preserves the last durable cursor.
type ScheduleWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func (s *Scheduler) Start(ctx context.Context) *ScheduleWorker {
	ctx, cancel := context.WithCancel(ctx)
	w := &ScheduleWorker{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(w.done)
		burst := 0
		for {
			_, delay, err := s.runOnce(ctx)
			if err != nil {
				if ctx.Err() == nil {
					w.err = err
				}
				return
			}
			burst++
			if delay <= 0 && burst < schedulerBurstLimit {
				continue
			}
			if delay <= 0 {
				delay = time.Minute
			}
			burst = 0
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return w
}
func (w *ScheduleWorker) Stop() {
	if w != nil {
		w.cancel()
	}
}
func (w *ScheduleWorker) Wait() error {
	if w == nil {
		return nil
	}
	<-w.done
	return w.err
}
