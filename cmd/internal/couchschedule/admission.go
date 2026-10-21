package couchschedule

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"golang.org/x/sys/unix"
)

func executionLease(path string) (func(), bool, error) {
	if err := durablefile.EnsureDirectory(filepath.Dir(path)); err != nil {
		return nil, false, err
	}
	if err := buildSafePath(filepath.Dir(path), filepath.Dir(path)); err != nil {
		return nil, false, err
	}
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, false, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		f.Close()
		return nil, false, errors.New("unsafe execution lease")
	}
	if err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		if err == unix.EWOULDBLOCK || err == unix.EAGAIN {
			return nil, false, nil
		}
		return nil, false, err
	}
	var once sync.Once
	return func() { once.Do(func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = f.Close() }) }, true, nil
}

func acquireExecutionPermits(ctx context.Context, r ExecutionRequest) (func(), error) {
	taskPath := filepath.Join(r.PermitsDir, "tasks", (TaskKey{Hub: r.HubRoot, Task: r.Task}).ID()+".lock")
	releaseTask, ok, err := executionLease(taskPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("scheduled task is already running")
	}
	for {
		if err = ctx.Err(); err != nil {
			releaseTask()
			return nil, err
		}
		for i := 0; i < 2; i++ {
			releasePermit, taken, e := executionLease(filepath.Join(r.PermitsDir, fmt.Sprintf("permit%d.lock", i)))
			if e != nil {
				releaseTask()
				return nil, e
			}
			if taken {
				return func() { releasePermit(); releaseTask() }, nil
			}
		}
		select {
		case <-ctx.Done():
			releaseTask()
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func executionEnvironment(r ExecutionRequest, heavyAcquired bool) []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "WF_TEST_") || strings.HasPrefix(key, "WF_SDLC_QUEUE_") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "WF_TEST_SUITES=2", "WF_SDLC_QUEUE_STOP_IF_PAUSED=1")
	if r.TestLockFile != "" {
		env = append(env, "WF_TEST_LOCK_FILE="+r.TestLockFile, "WF_SDLC_QUEUE_ROOT="+r.TestLockFile+".queue-state")
	}
	if heavyAcquired {
		env = append(env, "WF_TEST_LOCK_HELD_BY="+strconv.Itoa(os.Getpid()))
	}
	return env
}
