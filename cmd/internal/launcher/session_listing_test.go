package launcher

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListedTerminalObservationIsBoundedAndPositive(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rows  []string
		errs  []error
		want  bool
		calls int
	}{
		{"live", []string{"chosen [Created 1s ago]"}, nil, true, 1},
		{"attached", []string{"chosen [Created 1s ago] (current)"}, nil, true, 1},
		{"exited", []string{"chosen [Created 1s ago] (EXITED - attach to resurrect)"}, nil, false, 1},
		{"other exact name", []string{"chosen-more [Created 1s ago]"}, nil, false, 1},
		{"missing", []string{""}, nil, false, 1},
		{"failed", []string{"chosen [Created 1s ago]"}, []error{errors.New("listing failed")}, false, 1},
		{"unknown is final despite later success", []string{"", "chosen [Created 1s ago]"}, []error{errors.New("transient"), nil}, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			query := func(context.Context) ([]byte, error) {
				i := calls
				calls++
				row := tc.rows[min(i, len(tc.rows)-1)]
				var err error
				if len(tc.errs) > 0 {
					err = tc.errs[min(i, len(tc.errs)-1)]
				}
				return []byte(row), err
			}
			err := requireListedLiveTerminal(context.Background(), "chosen", query, noLiveServers)
			if (err == nil) != tc.want || calls != tc.calls {
				t.Fatalf("err=%v calls=%d wantSuccess=%v calls=%d", err, calls, tc.want, tc.calls)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	if err := requireListedLiveTerminal(ctx, "chosen", func(context.Context) ([]byte, error) { calls++; return nil, nil }, noLiveServers); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("cancelled observation err=%v calls=%d", err, calls)
	}
}

type listedAttachWorld struct {
	*handoffWorld
	attempt *terminalOSAttempt
}

func (w *listedAttachWorld) Check(ctx context.Context) error {
	w.event("check")
	return w.attempt.Check(ctx)
}
func (w *listedAttachWorld) Admit(ctx context.Context) error { return w.attempt.Admit(ctx) }

func TestWarmAttachUsesOnlyPositiveListingAndNeverReplaysClient(t *testing.T) {
	for _, live := range []bool{true, false} {
		t.Run(map[bool]string{true: "listed", false: "query-failure"}[live], func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\n" + zellijtest.ShellCase(t, "launcher-warm-attach-full")
			if !live {
				script = "#!/bin/sh\n" + zellijtest.ShellCase(t, "launcher-warm-attach-query-failure")
			}
			if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			// No Pair tag, readiness, launch policy, socket or custom ownership
			// evidence exists. The live listing is the complete attach prerequisite.
			w := newHandoffWorld()
			attempt := &terminalOSAttempt{name: "chosen"}
			done := make(chan error, 1)
			go func() {
				_, err := CoordinateTerminalHandoff(context.Background(), &listedAttachWorld{w, attempt}, time.Second)
				done <- err
			}()
			if live {
				select {
				case <-w.released:
				case <-time.After(2 * time.Second):
					t.Fatal("live listing did not admit attach")
				}
				w.exit <- HandoffExit{Code: 0}
			}
			select {
			case err := <-done:
				if (err == nil) != live {
					t.Fatalf("attach error=%v live=%v", err, live)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("unbounded attach")
			}
			starts := 0
			for _, event := range w.events {
				if event == "start" {
					starts++
				}
			}
			if live && starts != 1 || !live && starts != 0 {
				t.Fatalf("client starts=%d live=%v events=%v", starts, live, w.events)
			}
		})
	}
}

func TestListedTerminalCancelledObservationCannotAuthorizeAttach(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := requireListedLiveTerminal(ctx, "chosen", func(context.Context) ([]byte, error) { cancel(); return []byte("chosen [Created 1s ago]"), nil }, noLiveServers)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled listing authorized attach: %v", err)
	}
}

func noLiveServers(context.Context, string) ([]SessionServerIdentity, error) { return nil, nil }

// A listing can leave out a session whose server is live: Zellij 0.45.1 drops
// it when the ConnStatus probe loses its reply to a concurrent connection
// (pair#692, zellij-org/zellij#5270). Process evidence of the exact server
// makes that omission unknown, a refusal of its own class, never "not listed
// live". Only a listing naming the session live admits; nothing retries.
func TestListedTerminalOmissionOfALiveServerIsUnknown(t *testing.T) {
	live := func(context.Context, string) ([]SessionServerIdentity, error) {
		return []SessionServerIdentity{{PID: 42}}, nil
	}
	scanFailed := func(context.Context, string) ([]SessionServerIdentity, error) {
		return nil, errors.New("process table unreadable")
	}
	for _, tc := range []struct {
		name, listing string
		servers       func(context.Context, string) ([]SessionServerIdentity, error)
		omitted       bool
		absent        bool
	}{
		{"empty listing, live server", "", live, true, false},
		{"other sessions only, live server", "other [Created 1s ago]\n", live, true, false},
		{"empty listing, no server", "", noLiveServers, false, true},
		{"empty listing, scan failed", "", scanFailed, false, false},
		{"exited, live server", "chosen [Created 1s ago] (EXITED - attach to resurrect)\n", live, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listings, scans := 0, 0
			err := requireListedLiveTerminal(t.Context(), "chosen",
				func(context.Context) ([]byte, error) { listings++; return []byte(tc.listing), nil },
				func(ctx context.Context, name string) ([]SessionServerIdentity, error) {
					scans++
					return tc.servers(ctx, name)
				})
			if err == nil || listings != 1 {
				t.Fatalf("err=%v listings=%d", err, listings)
			}
			if errors.Is(err, ErrListingOmittedLiveServer) != tc.omitted {
				t.Fatalf("omitted class=%v want %v: %v", errors.Is(err, ErrListingOmittedLiveServer), tc.omitted, err)
			}
			if absent := strings.Contains(err.Error(), "is not listed live"); absent != tc.absent {
				t.Fatalf("absent class=%v want %v: %v", absent, tc.absent, err)
			}
		})
	}
	scans := 0
	err := requireListedLiveTerminal(t.Context(), "chosen",
		func(context.Context) ([]byte, error) { return []byte("chosen [Created 1s ago]\n"), nil },
		func(context.Context, string) ([]SessionServerIdentity, error) { scans++; return nil, nil })
	if err != nil || scans != 0 {
		t.Fatalf("a listed live session: err=%v scans=%d", err, scans)
	}
}
