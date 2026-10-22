package launcher

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// Captured read-only from installed Zellij 0.45.1 on 2026-10-18 (#672).
// Multi-session missing-target action prints the colored full session list,
// then exits zero inside list_sessions before its caller's intended exit one.
func TestNativeMissingSessionListingAcceptsOnlyNameEvidence(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		b, e := os.ReadFile("testdata/zellij-0.45.1-name-absence/" + name)
		if e != nil {
			t.Fatal(e)
		}
		return string(b)
	}
	stdout, stderr := read("stdout"), read("stderr")
	const target = "📁1-292"
	clients := read("clients.stdout")
	if _, err := zellijreply.ParseClients([]byte(stdout)); err == nil {
		t.Fatal("session listing became a clients table")
	}
	if got, err := zellijreply.ParseClients([]byte(clients)); err != nil || len(got) != 0 {
		t.Fatalf("native clients header: %v %v", got, err)
	}
	cases := []struct {
		name, out, diagnostic string
		failure               error
		accept                bool
		wantCalls             int // Zero means the ordinary single attempt.
	}{
		{name: "captured native absence", out: stdout, diagnostic: stderr, accept: true},
		{name: "missing heading", out: stdout},
		{name: "foreign target heading", out: stdout, diagnostic: strings.Replace(stderr, target, "foreign", 1)},
		{name: "extra diagnostic", out: stdout, diagnostic: stderr + "extra\n"},
		{name: "empty listing", diagnostic: stderr},
		{name: "client table", out: clients, diagnostic: stderr},
		{name: "JSON", out: `[{"id":1}]`, diagnostic: stderr},
		{name: "unrelated", out: "unrelated output\n", diagnostic: stderr},
		{name: "truncated", out: stdout[:len(stdout)-20], diagnostic: stderr},
		{name: "missing final newline", out: strings.TrimSuffix(stdout, "\n"), diagnostic: stderr},
		{name: "malformed duration", out: strings.Replace(stdout, "11h 6m 17s", "nonsense", 1), diagnostic: stderr},
		{name: "unknown terminal escape", out: strings.Replace(stdout, "\x1b[32;1m", "\x1b[2J", 1), diagnostic: stderr},
		{name: "invalid UTF8", out: stdout + "\xff", diagnostic: stderr},
		{name: "duplicate identity", out: stdout + strings.SplitAfter(stdout, "\n")[0], diagnostic: stderr},
		{name: "requested identity present", out: strings.Replace(stdout, "📁1-291", target, 1), diagnostic: stderr, wantCalls: 3},
		{name: "collection failure", out: stdout, diagnostic: stderr, failure: errors.New("collection failed")},
		{name: "joined cancellation", out: stdout, diagnostic: stderr, failure: errors.Join(errors.New("collection"), context.Canceled)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			rt := OSRuntime{nameQueryRun: func(cmd *exec.Cmd) error {
				calls++
				if cmd.Args[len(cmd.Args)-1] != "list-clients" {
					t.Fatalf("wrong query %v", cmd.Args)
				}
				if _, e := io.WriteString(cmd.Stdout, tc.out); e != nil {
					t.Fatal(e)
				}
				if _, e := io.WriteString(cmd.Stderr, tc.diagnostic); e != nil {
					t.Fatal(e)
				}
				return tc.failure
			}}
			err := rt.ProbeSessionNameContext(context.Background(), target)
			if (err == nil) != tc.accept {
				t.Fatalf("accepted=%v want=%v error=%v", err == nil, tc.accept, err)
			}
			wantCalls := tc.wantCalls
			if wantCalls == 0 {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("probe calls=%d want=%d", calls, wantCalls)
			}
		})
	}
}

func TestNativeMissingSessionListingCannotOutliveCaller(t *testing.T) {
	stdout, err := os.ReadFile("testdata/zellij-0.45.1-name-absence/stdout")
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := os.ReadFile("testdata/zellij-0.45.1-name-absence/stderr")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt := OSRuntime{nameQueryRun: func(cmd *exec.Cmd) error {
		_, _ = cmd.Stdout.Write(stdout)
		_, _ = cmd.Stderr.Write(stderr)
		cancel()
		return nil
	}}
	if err := rt.ProbeSessionNameContext(ctx, "📁1-292"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled reply admitted: %v", err)
	}
}
