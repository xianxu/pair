package couchcore

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// The fake retains its invocation index across processes. An unexpected query
// fails instead of falling through to any installed native binary.
func parkClientCLI(t *testing.T, root string, replies []string) string {
	t.Helper()
	state := filepath.Join(root, "query-count")
	if err := os.WriteFile(state, []byte("0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var script strings.Builder
	script.WriteString("#!/bin/sh\ncase \"$*\" in '--session pair-readiness action list-clients') ;; *) exit 99 ;; esac\nIFS= read -r n < \"$PAIR_TEST_CLIENT_REPLY_STATE\" || exit 98\nprintf '%s\\n' \"$((n+1))\" > \"$PAIR_TEST_CLIENT_REPLY_STATE\"\ncase $n in\n")
	for i, name := range replies {
		fmt.Fprintf(&script, "%d) %s ;;\n", i, zellijtest.ShellCase(t, name))
	}

	script.WriteString("*) exit 97 ;;\nesac\n")
	if err := os.WriteFile(filepath.Join(root, "zellij"), []byte(script.String()), 0700); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestParkClientCountQueryContract(t *testing.T) {
	for _, tc := range []struct {
		name            string
		reply           string
		want            int
		noReply, failed bool
	}{
		{name: "empty", reply: "park.empty", noReply: true},
		{name: "whitespace", reply: "park.whitespace", noReply: true},
		{name: "CLI failure", reply: "park.cli-failure", failed: true},
		{name: "header only", reply: "park.zero"},
		{name: "one client", reply: "park.one", want: 1},
		{name: "two clients", reply: "park.two", want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			replies := []string{tc.reply}
			if tc.noReply {
				replies = append(replies, tc.reply, tc.reply)
			}
			state := parkClientCLI(t, root, replies)
			testpair.ApplyEnvironment(t, root, "PATH="+root+":/usr/bin:/bin", "PAIR_TEST_CLIENT_REPLY_STATE="+state)
			got, err := zellijClientCount(t.Context(), "pair-readiness")
			if tc.noReply {
				if !errors.Is(err, launcher.ErrZellijNoReply) {
					t.Fatalf("count=%d err=%v; want unanswered query", got, err)
				}
			} else if tc.failed {
				if err == nil {
					t.Fatal("CLI failure granted a known baseline")
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("count=%d err=%v; want %d", got, err, tc.want)
			}
			raw, readErr := os.ReadFile(state)
			if readErr != nil || strings.TrimSpace(string(raw)) != strconv.Itoa(len(replies)) {
				t.Fatalf("query count=%q err=%v; want %d invocations", raw, readErr, len(replies))
			}
		})
	}
}

func TestParkHelperRejectsUnknownBaselineBeforeStarting(t *testing.T) {
	root := t.TempDir()
	// A header after the exhausted read budget must not establish attachment.
	state := parkClientCLI(t, root, []string{"park.empty", "park.empty", "park.empty", "park.zero"})
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestParkUnknownBaselineProbe$")
	cmd.Env = testpair.Environment(t, root, "PATH="+root+":/usr/bin:/bin", "PAIR_TEST_CLIENT_REPLY_STATE="+state, "PAIR_TEST_UNKNOWN_BASELINE_PROBE=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), launcher.ErrZellijNoReply.Error()) {
		t.Fatalf("helper did not refuse the unanswered baseline: err=%v output=%s", err, out)
	}
	raw, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := strconv.Atoi(strings.TrimSpace(string(raw))); err != nil || n != 3 {
		t.Fatalf("baseline refusal exceeded its three-attempt budget: %q, %v", raw, err)
	}
}

func TestParkUnknownBaselineProbe(t *testing.T) {
	if os.Getenv("PAIR_TEST_UNKNOWN_BASELINE_PROBE") != "1" {
		t.Skip("self-executed baseline probe")
	}
	root := os.Getenv("HOME")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd, terminal, wait := startParkLifecycleHelper(t, ctx, parkHelperConfig{
		GlobalDataDir: filepath.Join(root, "data"), Scope: "0123456789abcdef", Tag: "counter", Nonce: "regression", Session: "pair-readiness", PairHome: root, StagePath: filepath.Join(root, "stage"), Attempt: 1, CompletedAt: time.Unix(100, 0),
	})
	// Only the broken counter reaches here. Collect that owned helper before
	// returning the deliberately failing observation to the parent test.
	_ = terminal.Close()
	_ = cmd.Process.Kill()
	<-wait
	t.Fatal("header-only response falsely established helper attachment")
}

// Exercise the exact readiness loop used after the helper starts. A complete
// unchanged observation cannot establish attachment; only client growth can.
func TestParkHelperAttachmentRequiresClientGrowth(t *testing.T) {
	for _, baselineClients := range []int{0, 1} {
		t.Run(strconv.Itoa(baselineClients), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), testwait.Budget(t))
			defer cancel()
			growthRequested := make(chan struct{})
			releaseGrowth := make(chan struct{})
			done := make(chan struct{})
			calls := 0
			query := func(ctx context.Context, session string) (int, error) {
				if session != "pair-readiness" {
					return 0, fmt.Errorf("unexpected session %q", session)
				}
				calls++
				switch calls {
				case 1, 2:
					return baselineClients, nil // baseline, then a complete unchanged observation
				case 3:
					return 0, launcher.ErrZellijNoReply // unknown is not attachment either
				case 4:
					close(growthRequested)
					select {
					case <-releaseGrowth:
						return baselineClients + 1, nil
					case <-ctx.Done():
						return 0, ctx.Err()
					}
				default:
					return 0, fmt.Errorf("unexpected query %d", calls)
				}
			}
			go func() {
				defer close(done)
				count, err := query(ctx, "pair-readiness")
				if err != nil || count != baselineClients {
					t.Errorf("baseline=%d err=%v; %s", count, err, testwait.HostState())
					return
				}
				drained := make(chan struct{})
				close(drained)
				waitParkHelperAttachmentWithQuery(t, ctx, "pair-readiness", count, nil, drained, new(parkHelperCapture), func() bool { return true }, query)
			}()
			t.Cleanup(func() { cancel(); <-done })
			select {
			case <-growthRequested:
			case <-done:
				t.Fatalf("readiness stopped before querying growth; %s", testwait.HostState())
			case <-ctx.Done():
				t.Fatalf("readiness did not query growth: %v; %s", ctx.Err(), testwait.HostState())
			}
			select {
			case <-done:
				t.Fatalf("unchanged or unanswered query established attachment; %s", testwait.HostState())
			default:
			}
			close(releaseGrowth)
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatalf("readiness did not accept released growth: %v; %s", ctx.Err(), testwait.HostState())
			}
			if calls != 4 {
				t.Fatalf("readiness query count=%d; want baseline, unchanged, unknown, growth; %s", calls, testwait.HostState())
			}
		})
	}
}

// The readiness loop opens no session connection until OS evidence shows the
// helper's client started (pair#692): before that the helper is inside its
// one list-sessions fence, which a concurrent connection can make miss a live
// session. Once started, the existing growth query decides as before.
func TestParkReadinessQueriesOnlyAfterTheHelperClientStarts(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	checks, started := 0, false
	queriedBeforeStart := false
	queries := 0
	query := func(context.Context, string) (int, error) {
		if !started {
			queriedBeforeStart = true
		}
		queries++
		return 2, nil
	}
	drained := make(chan struct{})
	close(drained)
	waitParkHelperAttachmentWithQuery(t, ctx, "pair-readiness", 1, nil, drained, new(parkHelperCapture), func() bool {
		checks++
		started = checks > 3
		return started
	}, query)
	if queriedBeforeStart || queries != 1 || checks != 4 {
		t.Fatalf("queried before the client started=%v queries=%d checks=%d", queriedBeforeStart, queries, checks)
	}
}
