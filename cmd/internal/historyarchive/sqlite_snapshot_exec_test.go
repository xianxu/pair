package historyarchive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecSQLiteBoundsStreamsAndPreservesUnknownFailure(t *testing.T) {
	for _, behavior := range []string{"overflow", "exit"} {
		t.Run(behavior, func(t *testing.T) {
			t.Setenv("PAIR688_FAKE_SQLITE", behavior)
			work := t.TempDir()
			lease, err := snapshotLease(filepath.Join(work, "lease"), true)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			result, err := (ExecSQLiteRunner{SQLite: os.Args[0]}).Run(context.Background(), SQLiteRequest{Work: work, Database: "unused", Lease: lease})
			if err == nil || errors.Is(err, ErrSQLiteDeadline) {
				t.Fatalf("tool failure misclassified: %+v %v", result, err)
			}
			if len(result.Stderr) > 8192 || len(result.Stdout) > 65536 {
				t.Fatal("unbounded output")
			}
			if behavior == "overflow" && !result.Overflow {
				t.Fatal("stdout truncation accepted")
			}
			if behavior == "exit" && !strings.Contains(result.Stderr, "locked") {
				t.Fatal("diagnostic dropped")
			}
		})
	}
}

func TestExecSQLiteCancellationJoinsStartedChild(t *testing.T) {
	work := t.TempDir()
	ready := filepath.Join(work, "ready")
	t.Setenv("PAIR688_FAKE_SQLITE", "blocked")
	t.Setenv("PAIR688_SQLITE_READY", ready)
	lease, err := snapshotLease(filepath.Join(work, "lease"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completed := make(chan error, 1)
	go func() {
		_, err := (ExecSQLiteRunner{SQLite: os.Args[0]}).Run(ctx, SQLiteRequest{Work: work, Database: "unused", Lease: lease})
		completed <- err
	}()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-completed:
			t.Fatalf("child ended before readiness: %v", err)
		case <-tick.C:
			if _, err := os.Stat(ready); err == nil {
				cancel()
				if err := <-completed; !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation not propagated: %v", err)
				}
				return
			}
		}
	}
}

func TestExecSQLiteRejectsOutOfPolicyBudgetBeforeExecution(t *testing.T) {
	for _, seconds := range []int{-1, 4, 31, 600} {
		_, err := (ExecSQLiteRunner{SQLite: "/must-not-be-executed"}).Run(context.Background(), SQLiteRequest{ReadSeconds: seconds})
		if err == nil || !strings.Contains(err.Error(), "budget") {
			t.Fatalf("out of policy budget %d reached exec: %v", seconds, err)
		}
	}
}
