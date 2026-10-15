package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestZellijQueryObserverRetainsExactResponsesWithoutChangingClassification(t *testing.T) {
	dir := t.TempDir()
	path, count := filepath.Join(dir, "zellij"), filepath.Join(dir, "count")
	script := `#!/bin/sh
case "$*" in
 'list-sessions --short') printf 'pair-observed\n' ;;
 'list-sessions --no-formatting') printf 'pair-observed [Created]\n' ;;
 '--session pair-observed action list-clients')
 n=0
 if [ -f '` + count + `' ]; then read -r n < '` + count + `'; fi
 n=$((n+1)); printf '%s\n' "$n" > '` + count + `'
 case "$n" in
  1) : ;;
  2) printf 'CLIENTS\n' ;;
  3) printf 'CLIENTS\n1\n' ;;
  *) printf 'partial\n'; printf 'query failed\n' >&2; exit 7 ;;
 esac ;;
 *) exit 9 ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	var events []ZellijQueryObservation
	ctx := WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { events = append(events, e) })
	source := ZellijSource{Path: path}
	for i, want := range []SessionState{SessionDetached, SessionDetached, SessionAttached, ""} {
		rows, err := source.SnapshotSessionsContext(ctx, []string{"pair-observed"})
		if i == 3 {
			if err == nil || rows != nil {
				t.Fatalf("failure became usable rows: %+v %v", rows, err)
			}
		} else if err != nil || len(rows) != 1 || rows[0].State != want {
			t.Fatalf("attempt%d rows=%+v err=%v", i, rows, err)
		}
	}
	if len(events) != 12 {
		t.Fatalf("observed %d calls, want original 12", len(events))
	}
	for i, want := range []string{"", "CLIENTS\n", "CLIENTS\n1\n", "partial\n"} {
		e := events[i*3+2]
		if string(e.Stdout) != want || len(e.Args) != 4 || e.Args[1] != "pair-observed" || e.Started.IsZero() || e.Finished.Before(e.Started) || e.Duration < 0 {
			t.Fatalf("attempt%d evidence=%+v", i, e)
		}
		if i == 3 {
			if e.ExitCode != 7 || string(e.Stderr) != "query failed\n" || e.Err == nil {
				t.Fatalf("failure evidence=%+v", e)
			}
		} else if e.ExitCode != 0 || e.Err != nil {
			t.Fatalf("success evidence=%+v", e)
		}
	}
	if raw, err := os.ReadFile(count); err != nil || string(raw) != "4\n" {
		t.Fatalf("query duplicated: %q %v", raw, err)
	}
	// A context observer must not escape into unrelated calls.
	_, _ = source.runContext(t.Context(), "unobserved")
	if len(events) != 12 {
		t.Fatal("observer escaped its context")
	}
}

func TestZellijQueryObserverBoundsCopiesWithoutChangingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zellij")
	raw := strings.Repeat("x", 70*1024)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s' '"+raw+"'\nprintf '%s' '"+raw+"' >&2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	var event ZellijQueryObservation
	ctx := WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { event = e; e.Stdout[0] = 'z'; e.Stderr[0] = 'z'; e.Args[0] = "changed" })
	args := []string{"original"}
	out, err := (ZellijSource{Path: path}).runContext(ctx, args...)
	if err != nil || string(out) != raw || args[0] != "original" {
		t.Fatalf("observer changed original query: len=%d err=%v args=%v", len(out), err, args)
	}
	if len(event.Stdout) != 64*1024 || len(event.Stderr) != 64*1024 || event.StdoutBytes != len(raw) || event.StderrBytes != len(raw) || !event.StdoutTruncated || !event.StderrTruncated {
		t.Fatalf("capture bounds: stdout=%d/%d stderr=%d/%d", len(event.Stdout), event.StdoutBytes, len(event.Stderr), event.StderrBytes)
	}
}

func TestZellijQueryObserverPreservesCancellation(t *testing.T) {
	dir := t.TempDir()
	path, ready := filepath.Join(dir, "zellij"), filepath.Join(dir, "ready")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf ready > '"+ready+"'\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	events := make(chan ZellijQueryObservation, 1)
	ctx = WithZellijQueryObserver(ctx, func(e ZellijQueryObservation) { events <- e })
	done := make(chan error, 1)
	go func() { _, err := (ZellijSource{Path: path}).runContext(ctx, "blocked"); done <- err }()
	deadline := time.After(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("query never started")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("original direct query did not join")
	}
	e := <-events
	if !errors.Is(e.Err, context.Canceled) || e.ExitCode == 0 {
		t.Fatalf("canceled query evidence=%+v", e)
	}
}
