package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/crashreport"
	pairterminal "github.com/xianxu/pair/cmd/internal/terminal"
)

// TestConsoleTerminalFailureIsRecordedAsTheExitReason: a console that ended
// on a parent-output write past its deadline (as presenter_stall_test proves a
// stalled terminal produces) leaves that reason for the next start, which
// reports it apart from another run's panic (pair#409). Anything else leaves
// the crash file empty, so a clean end still removes it.
func TestConsoleTerminalFailureIsRecordedAsTheExitReason(t *testing.T) {
	dir := t.TempDir()
	dead := func(int) bool { return false }
	if _, _, err := crashreport.Install(dir, time.Date(2026, 10, 7, 4, 31, 0, 0, time.UTC), 21, dead); err != nil {
		t.Fatal(err)
	}
	// Another run's panic, left beside this one's file.
	if err := os.WriteFile(filepath.Join(dir, "20261007T043000Z-7.log"), []byte("panic: boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stalled := &pairterminal.WriteFailure{Op: "parent output", Accepted: 1024, Total: 17424, Err: context.DeadlineExceeded}
	recordConsoleExit(errors.Join(fmt.Errorf("present: %w", stalled), errors.New("restore: deadline exceeded")))
	if err := crashreport.Finish(); err != nil {
		t.Fatal(err)
	}
	_, reports, err := crashreport.Install(dir, time.Date(2026, 10, 7, 4, 40, 0, 0, time.UTC), 22, dead)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[crashreport.Kind]int{}
	for _, r := range reports {
		kinds[r.Kind]++
	}
	summary := crashreport.Summary(reports)
	if kinds[crashreport.Exited] != 1 || kinds[crashreport.Crashed] != 1 || !strings.Contains(summary, "previous couch exited: terminal stopped accepting output (a write waited up to "+pairterminal.WriteTimeout.String()+"; wrote 1024 of 17424 bytes) — see ") {
		t.Fatalf("reports %+v\nsummary %q", reports, summary)
	}

	// No terminal failure: nothing is recorded and the capture file goes.
	recordConsoleExit(nil)
	recordConsoleExit(errors.New("operator quit"))
	if err := crashreport.Finish(); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Fatalf("a clean end left a file: %v", entries)
	}
}
