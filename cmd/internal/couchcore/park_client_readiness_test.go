package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

type parkClientReply struct {
	body string
	exit int
}

const parkClientHeader = "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"

// The fake retains its invocation index across processes. An unexpected query
// fails instead of falling through to any installed native binary.
func parkClientCLI(t *testing.T, root string, replies []parkClientReply) string {
	t.Helper()
	state := filepath.Join(root, "query-count")
	if err := os.WriteFile(state, []byte("0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var script strings.Builder
	script.WriteString("#!/bin/sh\ncase \"$*\" in '--session pair-readiness action list-clients') ;; *) exit 99 ;; esac\nIFS= read -r n < \"$PAIR_TEST_CLIENT_REPLY_STATE\" || exit 98\nprintf '%s\\n' \"$((n+1))\" > \"$PAIR_TEST_CLIENT_REPLY_STATE\"\ncase $n in\n")
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	for i, reply := range replies {
		redirect := ""
		if reply.exit != 0 {
			redirect = " >&2"
		}
		fmt.Fprintf(&script, "%d) printf '%%s' %s%s; exit %d ;;\n", i, quote(reply.body), redirect, reply.exit)
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
		reply           parkClientReply
		want            int
		noReply, failed bool
	}{
		{name: "empty", noReply: true},
		{name: "whitespace", reply: parkClientReply{body: " \n\t"}, noReply: true},
		{name: "CLI failure", reply: parkClientReply{body: "query failed", exit: 2}, failed: true},
		{name: "header only", reply: parkClientReply{body: parkClientHeader}},
		{name: "one client", reply: parkClientReply{body: parkClientHeader + "1 0 /bin/sh\n"}, want: 1},
		{name: "two clients", reply: parkClientReply{body: parkClientHeader + "1 0 /bin/sh\n2 1 /bin/sh\n"}, want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			state := parkClientCLI(t, root, []parkClientReply{tc.reply})
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
			if readErr != nil || strings.TrimSpace(string(raw)) != "1" {
				t.Fatalf("query count=%q err=%v; want exactly one invocation", raw, readErr)
			}
		})
	}
}

func TestParkHelperRejectsUnknownBaselineBeforeStarting(t *testing.T) {
	root := t.TempDir()
	// A later header must never turn an unanswered baseline into attachment.
	state := parkClientCLI(t, root, []parkClientReply{{}, {body: parkClientHeader}})
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
	if n, err := strconv.Atoi(strings.TrimSpace(string(raw))); err != nil || n != 1 {
		t.Fatalf("baseline refusal made more queries: %q, %v", raw, err)
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
