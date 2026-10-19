package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SQLitePurpose restricts the external protocol to source backup and private validation.
type SQLitePurpose uint8

const (
	SQLiteAcquire SQLitePurpose = iota
	SQLiteValidate
)

// SQLiteRequest owns one SQLite invocation and its inherited scratch lease.
type SQLiteRequest struct {
	Purpose        SQLitePurpose
	Database, Work string
	Lease          *os.File
	ReadSeconds    int
}

// SQLiteRunResult retains bounded tool output; overflow is never integrity evidence.
type SQLiteRunResult struct {
	Stdout, Stderr string
	Overflow       bool
}

// SQLiteRunner is the shared boundary for the stateful fake and real CLI.
type SQLiteRunner interface {
	Run(context.Context, SQLiteRequest) (SQLiteRunResult, error)
}

// ExecSQLiteRunner invokes this binary's kernel-alarm launcher, then SQLite.
// pair:m5-concept integration
type ExecSQLiteRunner struct{ SQLite, Launcher string }

func (r ExecSQLiteRunner) Run(ctx context.Context, req SQLiteRequest) (SQLiteRunResult, error) {
	seconds, script := 5, ".timeout 250\nBEGIN;\nPRAGMA schema_version;\n.backup main snapshot.sqlite\nROLLBACK;\n"
	if req.Purpose == SQLiteAcquire && req.ReadSeconds != 0 {
		if req.ReadSeconds < 5 || req.ReadSeconds > 30 {
			return SQLiteRunResult{}, errors.New("SQLite source budget must be 5 through 30 seconds")
		}
		seconds = req.ReadSeconds
	}
	if req.Purpose == SQLiteValidate {
		seconds, script = 600, "PRAGMA journal_mode=DELETE; PRAGMA integrity_check;\n"
	} else if req.Purpose != SQLiteAcquire {
		return SQLiteRunResult{}, errors.New("unknown SQLite purpose")
	}
	bin := r.SQLite
	if bin == "" {
		bin = "sqlite3"
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return SQLiteRunResult{}, err
	}
	launcher := r.Launcher
	if launcher == "" {
		launcher, err = os.Executable()
		if err != nil {
			return SQLiteRunResult{}, err
		}
	}
	args := []string{"internal", "exec-alarm", strconv.Itoa(seconds), "--", resolved, "-bail", "-batch", "-init", "/dev/null", "-nofollow"}
	if req.Purpose == SQLiteAcquire {
		args = append(args, "-readonly")
	}
	args = append(args, req.Database)
	bounded, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, launcher, args...)
	cmd.Dir = req.Work
	cmd.Stdin = strings.NewReader(script)
	if req.Lease != nil {
		cmd.ExtraFiles = []*os.File{req.Lease}
	}
	cmd.WaitDelay = time.Second
	out := &boundedRcloneOutput{limit: 64 << 10}
	diag := &codecLimitedBuffer{limit: 8 << 10}
	cmd.Stdout, cmd.Stderr = out, diag
	err = cmd.Run()
	result := SQLiteRunResult{Stdout: out.buffer.String(), Stderr: diag.String(), Overflow: out.exceeded}
	if bounded.Err() != nil {
		return result, bounded.Err()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() && status.Signal() == syscall.SIGALRM {
			return result, ErrSQLiteDeadline
		}
	}
	if err == nil && result.Overflow {
		err = fmt.Errorf("SQLite output exceeds %d bytes", 64<<10)
	}
	return result, err
}
