package procutil

import (
	"context"
	"os"
	"testing"
	"time"
)

// This reader measures actual native calls rather than estimating calls from
// the number of observations (terminated processes can disappear mid-read).
type startupTimingReader struct {
	nativeReader
	rows, commands, identities           int
	tableTime, commandTime, identityTime time.Duration
}

func (r *startupTimingReader) table() ([]Process, error) {
	start := time.Now()
	rows, err := r.nativeReader.table()
	r.tableTime += time.Since(start)
	r.rows += len(rows)
	return rows, err
}
func (r *startupTimingReader) command(pid int) (CommandInfo, error) {
	start := time.Now()
	value, err := r.nativeReader.command(pid)
	r.commandTime += time.Since(start)
	r.commands++
	return value, err
}
func (r *startupTimingReader) identity(pid int) string {
	start := time.Now()
	value := r.nativeReader.identity(pid)
	r.identityTime += time.Since(start)
	r.identities++
	return value
}

func TestStartupNativeSnapshotTimingLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" || os.Getenv("PAIR_LIVE_STARTUP_PERF") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	r := &startupTimingReader{}
	start := time.Now()
	observations, err := snapshot(ctx, r)
	wall := time.Since(start)
	t.Logf("native snapshot: rows=%d observations=%d command_calls=%d identity_calls=%d table=%s argv=%s identity=%s wall=%s residual=%s", r.rows, len(observations), r.commands, r.identities, r.tableTime, r.commandTime, r.identityTime, wall, wall-r.tableTime-r.commandTime-r.identityTime)
	if err != nil {
		t.Fatal(err)
	}
}
