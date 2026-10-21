package launcher

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestZellijQueryObserverReportsActualDeadlineContexts(t *testing.T) {
	for _, budget := range []time.Duration{0, time.Second, time.Minute} {
		t.Run(budget.String(), func(t *testing.T) {
			ctx := context.Background()
			var wantCaller time.Time
			if budget != 0 {
				wantCaller = time.Now().Add(budget)
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, wantCaller)
				defer cancel()
			}
			var event ZellijQueryObservation
			ctx = WithZellijQueryObserver(ctx, func(e ZellijQueryObservation) { event = e })
			source := ZellijSource{queryRun: func(cmd *exec.Cmd) error {
				return nil
			}}
			before := time.Now()
			_, _ = source.runContext(ctx, "list-sessions", "--short")
			after := time.Now()
			if !event.CallerDeadline.Equal(wantCaller) {
				t.Fatalf("deadlines caller=%v want=%v effective=%v", event.CallerDeadline, wantCaller, event.EffectiveDeadline)
			}
			if budget == time.Second {
				if !event.EffectiveDeadline.Equal(wantCaller) {
					t.Fatal("query lost earlier caller deadline")
				}
			} else if event.EffectiveDeadline.Before(before.Add(zellijQueryTimeout)) || event.EffectiveDeadline.After(after.Add(zellijQueryTimeout)) {
				t.Fatalf("query cap deadline=%v outside call interval", event.EffectiveDeadline)
			}
		})
	}
}

func TestZellijQueryObserverRetainsExactResponsesWithoutChangingClassification(t *testing.T) {
	dir := t.TempDir()
	path, count := filepath.Join(dir, "zellij"), filepath.Join(dir, "count")
	script := `#!/bin/sh
case "$*" in
 'list-sessions --short') ` + zellijtest.ShellCase(t, "launcher-observer-short") + ` ;;
 'list-sessions --no-formatting') ` + zellijtest.ShellCase(t, "launcher-observer-full") + ` ;;
 '--session pair-observed action list-clients')
 n=0
 if [ -f '` + count + `' ]; then read -r n < '` + count + `'; fi
 n=$((n+1)); printf '%s\n' "$n" > '` + count + `'
 case "$n" in
  1|2|3) ` + zellijtest.ShellCase(t, "launcher-observer-unanswered") + ` ;;
  4) ` + zellijtest.ShellCase(t, "launcher-observer-empty-clients") + ` ;;
  5) ` + zellijtest.ShellCase(t, "launcher-observer-attached") + ` ;;
  *) ` + zellijtest.ShellCase(t, "launcher-observer-partial-failure") + ` ;;
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
	for i, want := range []SessionState{"", SessionDetached, SessionAttached, ""} {
		rows, err := source.SnapshotSessionsContext(ctx, []string{"pair-observed"})
		if i == 0 {
			if !errors.Is(err, ErrZellijNoReply) || rows != nil {
				t.Fatalf("empty reply became usable rows: %+v %v", rows, err)
			}
		} else if i == 3 {
			if err == nil || rows != nil {
				t.Fatalf("failure became usable rows: %+v %v", rows, err)
			}
		} else if err != nil || len(rows) != 1 || rows[0].State != want {
			t.Fatalf("attempt%d rows=%+v err=%v", i, rows, err)
		}
	}
	if len(events) != 10 {
		t.Fatalf("observed %d calls, want 10 (four listings, three exhausted attempts and three later client queries)", len(events))
	}
	for i, want := range []string{"", "", "", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_0 sh\n", "partial\n"} {
		e := events[[]int{1, 2, 3, 5, 7, 9}[i]]
		if string(e.Stdout) != want || len(e.Args) != 4 || e.Args[1] != "pair-observed" || e.Started.IsZero() || e.Finished.Before(e.Started) || e.Duration < 0 {
			t.Fatalf("attempt%d evidence=%+v", i, e)
		}
		if i == 5 {
			if e.ExitCode != 7 || string(e.Stderr) != "query failed\n" || e.Err == nil {
				t.Fatalf("failure evidence=%+v", e)
			}
		} else if i < 3 {
			if e.ExitCode != 0 || !errors.Is(e.Err, ErrZellijNoReply) {
				t.Fatalf("unanswered observation lost exit0: %+v", e)
			}
		} else if e.ExitCode != 0 || e.Err != nil {
			t.Fatalf("success evidence=%+v", e)
		}
	}
	if raw, err := os.ReadFile(count); err != nil || string(raw) != "6\n" {
		t.Fatalf("client query count: %q %v", raw, err)
	}
	// A context observer must not escape into unrelated calls.
	_, _ = source.runContext(t.Context(), "unobserved")
	if len(events) != 10 {
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

type queryObservedOwnerWorld struct{ *ownerWorld }

func (w *queryObservedOwnerWorld) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	return (osSessionOwnerIO{}).SessionPanes(ctx, name)
}

func TestZellijQueryObserverRetainsMalformedAuthoritativePaneResponse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '{malformed pane JSON'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	world := &queryObservedOwnerWorld{&ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}}
	var events []ZellijQueryObservation
	ctx := WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { events = append(events, e) })
	got, err := (SessionOwnerProbe{IO: world}).Probe(ctx, "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err != nil || got.State != SessionOwnerUnknown || got.Diagnostic != "invalid live pane JSON" {
		t.Fatalf("ownership classification changed: %+v %v", got, err)
	}
	if len(events) != 1 || string(events[0].Stdout) != "{malformed pane JSON" || events[0].ExitCode != 0 || events[0].Err != nil || strings.Join(events[0].Args, " ") != "--session pair-fixture action list-panes --json --command" {
		t.Fatalf("authoritative response not retained: %+v", events)
	}
}
